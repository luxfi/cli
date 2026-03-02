// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package devcmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/constants"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Stack directory layout under ~/.lux/dev/
const (
	stackConfigFile = "stack.yaml"
	stackPIDDir     = "pids"
	stackLogDir     = "logs"
	stackDataDir    = "data"
	stackBinDir     = "bin"
	stackChainsFile = "chains.json"

	portStride      = 100 // chain i gets port_base + i*100
	shutdownTimeout = 10 * time.Second
	healthTimeout   = 60 * time.Second
	healthInterval  = 1 * time.Second
)

// StackConfig is the on-disk stack.yaml schema.
type StackConfig struct {
	Chains  int        `yaml:"chains"`
	Apps    []AppEntry `yaml:"apps"`
	DataDir string     `yaml:"data_dir"`
	LogDir  string     `yaml:"log_dir"`
}

// AppEntry describes one application in the stack.
type AppEntry struct {
	Name     string `yaml:"name"`
	PortBase int    `yaml:"port_base"`
	Enabled  bool   `yaml:"enabled"`
	Binary   string `yaml:"binary,omitempty"` // image ref or binary name
}

// ChainInfo is written to chains.json for peer discovery.
type ChainInfo struct {
	Index    int    `json:"index"`
	RPCHTTP  string `json:"rpc_http"`
	RPCWS    string `json:"rpc_ws"`
	StakingP int    `json:"staking_port"`
	PID      int    `json:"pid,omitempty"`
}

// ChainsManifest is the top-level chains.json structure.
type ChainsManifest struct {
	UpdatedAt string      `json:"updated_at"`
	Chains    []ChainInfo `json:"chains"`
}

var stackChains int
var stackConfigPath string

func newStackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stack",
		Short: "Orchestrate full local dev stack",
		Long: `Manage a multi-app local development stack.

The stack runs luxd (one or more nodes) plus companion apps:
explorer, bridge, exchange, safe, dao, wallet, faucet.

Config lives at ~/.lux/dev/stack.yaml and is auto-created on first run.

Examples:
  lux dev stack up                # Start stack with defaults
  lux dev stack up --chains 3     # Start 3 luxd nodes + apps
  lux dev stack down              # Graceful shutdown
  lux dev stack status            # Show running processes
  lux dev stack logs explorer     # Tail explorer logs`,
		RunE: cobrautils.CommandSuiteUsage,
	}

	cmd.AddCommand(newStackUpCmd())
	cmd.AddCommand(newStackDownCmd())
	cmd.AddCommand(newStackStatusCmd())
	cmd.AddCommand(newStackLogsCmd())

	return cmd
}

// --- up ---

func newStackUpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up",
		Short: "Start the dev stack",
		Long: `Start all enabled apps in the dev stack.

luxd nodes start first and must pass health checks before companion
apps are launched. Port deconfliction for multi-chain: chain i gets
ports at port_base + 100*i.`,
		RunE:         stackUp,
		Args:         cobra.ExactArgs(0),
		SilenceUsage: true,
	}

	cmd.Flags().IntVar(&stackChains, "chains", 0, "number of luxd nodes (overrides stack.yaml)")
	cmd.Flags().StringVar(&stackConfigPath, "config", "", "path to stack.yaml (default: ~/.lux/dev/stack.yaml)")

	return cmd
}

func stackUp(*cobra.Command, []string) error {
	cfg, err := loadOrCreateConfig()
	if err != nil {
		return err
	}

	if stackChains > 0 {
		cfg.Chains = stackChains
	}
	if cfg.Chains < 1 {
		cfg.Chains = 1
	}

	baseDir := stackBaseDir()
	pidDir := filepath.Join(baseDir, stackPIDDir)
	logDir := expandPath(cfg.LogDir)

	for _, dir := range []string{pidDir, logDir, expandPath(cfg.DataDir)} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}

	// Phase 1: start luxd nodes
	luxdEntry := findApp(cfg, "luxd")
	if luxdEntry == nil {
		return fmt.Errorf("luxd not found in stack config")
	}

	manifest := ChainsManifest{
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	for i := 0; i < cfg.Chains; i++ {
		portOffset := i * portStride
		httpPort := luxdEntry.PortBase + portOffset
		stakingPort := httpPort + 1
		name := chainInstanceName("luxd", i)

		if isRunning(pidDir, name) {
			ux.Logger.PrintToUser("%s already running, skipping", name)
			pid, _ := readPID(pidDir, name)
			manifest.Chains = append(manifest.Chains, ChainInfo{
				Index:    i,
				RPCHTTP:  fmt.Sprintf("http://127.0.0.1:%d", httpPort),
				RPCWS:    fmt.Sprintf("ws://127.0.0.1:%d/ext/bc/C/ws", httpPort),
				StakingP: stakingPort,
				PID:      pid,
			})
			continue
		}

		binary, err := resolveBinary("luxd")
		if err != nil {
			return err
		}

		dataDir := filepath.Join(expandPath(cfg.DataDir), name)
		nodeLogDir := filepath.Join(logDir, name)
		for _, d := range []string{dataDir, nodeLogDir} {
			if err := os.MkdirAll(d, 0o750); err != nil {
				return fmt.Errorf("mkdir %s: %w", d, err)
			}
		}

		args := []string{
			"--dev",
			fmt.Sprintf("--network-id=%d", 1337),
			"--http-host=0.0.0.0",
			fmt.Sprintf("--http-port=%d", httpPort),
			fmt.Sprintf("--staking-port=%d", stakingPort),
			fmt.Sprintf("--data-dir=%s", dataDir),
			fmt.Sprintf("--log-dir=%s", nodeLogDir),
			"--log-level=info",
			"--api-admin-enabled=true",
			"--index-enabled=true",
		}

		logFile, err := openLogFile(logDir, name)
		if err != nil {
			return err
		}

		ux.Logger.PrintToUser("Starting %s on port %d...", name, httpPort)
		pid, err := spawnProcess(binary, args, logFile, pidDir, name)
		if err != nil {
			_ = logFile.Close()
			return fmt.Errorf("start %s: %w", name, err)
		}

		manifest.Chains = append(manifest.Chains, ChainInfo{
			Index:    i,
			RPCHTTP:  fmt.Sprintf("http://127.0.0.1:%d", httpPort),
			RPCWS:    fmt.Sprintf("ws://127.0.0.1:%d/ext/bc/C/ws", httpPort),
			StakingP: stakingPort,
			PID:      pid,
		})
	}

	// Write chains.json before waiting (apps can poll it)
	if err := writeChainsManifest(baseDir, &manifest); err != nil {
		return err
	}

	// Wait for all luxd nodes to become healthy
	for i := 0; i < cfg.Chains; i++ {
		httpPort := luxdEntry.PortBase + i*portStride
		name := chainInstanceName("luxd", i)
		ux.Logger.PrintToUser("Waiting for %s health...", name)
		if err := waitForHealth(httpPort); err != nil {
			return fmt.Errorf("%s health check failed: %w", name, err)
		}
		ux.Logger.GreenCheckmarkToUser("%s healthy", name)
	}

	// Phase 2: start companion apps
	for _, app := range cfg.Apps {
		if app.Name == "luxd" || !app.Enabled {
			continue
		}

		for i := 0; i < cfg.Chains; i++ {
			portOffset := i * portStride
			port := app.PortBase + portOffset
			name := chainInstanceName(app.Name, i)

			if isRunning(pidDir, name) {
				ux.Logger.PrintToUser("%s already running, skipping", name)
				continue
			}

			binary, err := resolveBinary(app.Name)
			if err != nil {
				ux.Logger.PrintToUser("Warning: %s - skipping %s", err, name)
				continue
			}

			logFile, err := openLogFile(logDir, name)
			if err != nil {
				return err
			}

			// Pass port and chains.json path via env
			appArgs := []string{
				fmt.Sprintf("--port=%d", port),
			}

			ux.Logger.PrintToUser("Starting %s on port %d...", name, port)
			if _, err := spawnProcessWithEnv(binary, appArgs, logFile, pidDir, name, []string{
				fmt.Sprintf("PORT=%d", port),
				fmt.Sprintf("LUX_CHAINS_FILE=%s", filepath.Join(baseDir, stackChainsFile)),
				fmt.Sprintf("LUX_RPC_URL=http://127.0.0.1:%d", luxdEntry.PortBase+i*portStride),
			}); err != nil {
				_ = logFile.Close()
				ux.Logger.PrintToUser("Warning: failed to start %s: %v", name, err)
				continue
			}
			ux.Logger.GreenCheckmarkToUser("%s started", name)
		}
	}

	// Refresh manifest with final PID info
	manifest.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := writeChainsManifest(baseDir, &manifest); err != nil {
		return err
	}

	printStackSummary(cfg)
	return nil
}

// --- down ---

func newStackDownCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "down",
		Short:        "Stop the dev stack",
		Long:         "Gracefully stop all running stack processes. Sends SIGTERM, waits 10s, then SIGKILL.",
		RunE:         stackDown,
		Args:         cobra.ExactArgs(0),
		SilenceUsage: true,
	}
}

func stackDown(*cobra.Command, []string) error {
	pidDir := filepath.Join(stackBaseDir(), stackPIDDir)

	entries, err := os.ReadDir(pidDir)
	if err != nil {
		if os.IsNotExist(err) {
			ux.Logger.PrintToUser("No stack running")
			return nil
		}
		return err
	}

	if len(entries) == 0 {
		ux.Logger.PrintToUser("No stack running")
		return nil
	}

	// Stop companion apps first (reverse order), then luxd nodes
	var luxdPIDs []string
	var appPIDs []string
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".pid") {
			continue
		}
		if strings.HasPrefix(e.Name(), "luxd") {
			luxdPIDs = append(luxdPIDs, e.Name())
		} else {
			appPIDs = append(appPIDs, e.Name())
		}
	}

	// Stop apps first
	for _, pidFile := range appPIDs {
		name := strings.TrimSuffix(pidFile, ".pid")
		stopOne(pidDir, name)
	}
	// Then luxd
	for _, pidFile := range luxdPIDs {
		name := strings.TrimSuffix(pidFile, ".pid")
		stopOne(pidDir, name)
	}

	// Clean chains.json
	_ = os.Remove(filepath.Join(stackBaseDir(), stackChainsFile))

	ux.Logger.PrintToUser("Stack stopped")
	return nil
}

func stopOne(pidDir, name string) {
	pid, err := readPID(pidDir, name)
	if err != nil {
		ux.Logger.PrintToUser("%s: no PID file", name)
		return
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		removePIDFile(pidDir, name)
		return
	}

	// Check if process is actually running
	if err := process.Signal(syscall.Signal(0)); err != nil {
		ux.Logger.PrintToUser("%s (PID %d): not running", name, pid)
		removePIDFile(pidDir, name)
		return
	}

	ux.Logger.PrintToUser("Stopping %s (PID %d)...", name, pid)
	if err := process.Signal(syscall.SIGTERM); err != nil {
		ux.Logger.PrintToUser("%s: SIGTERM failed: %v, trying SIGKILL", name, err)
		_ = process.Signal(syscall.SIGKILL)
		removePIDFile(pidDir, name)
		return
	}

	// Wait for exit with timeout
	done := make(chan struct{})
	go func() {
		_, _ = process.Wait()
		close(done)
	}()

	select {
	case <-done:
		ux.Logger.GreenCheckmarkToUser("%s stopped", name)
	case <-time.After(shutdownTimeout):
		ux.Logger.PrintToUser("%s: shutdown timeout, sending SIGKILL", name)
		_ = process.Signal(syscall.SIGKILL)
		_, _ = process.Wait()
	}

	removePIDFile(pidDir, name)
}

// --- status ---

func newStackStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "status",
		Short:        "Show dev stack status",
		Long:         "Display a table of all stack processes with PID, port, state, and uptime.",
		RunE:         stackStatus,
		Args:         cobra.ExactArgs(0),
		SilenceUsage: true,
	}
}

func stackStatus(*cobra.Command, []string) error {
	cfg, err := loadConfig()
	if err != nil {
		ux.Logger.PrintToUser("No stack configured (run `lux dev stack up` first)")
		return nil
	}

	pidDir := filepath.Join(stackBaseDir(), stackPIDDir)

	table := ux.NewCompatTable()
	table.SetHeader([]string{"APP", "INSTANCE", "PID", "PORT", "STATE", "UPTIME"})

	for _, app := range cfg.Apps {
		for i := 0; i < cfg.Chains; i++ {
			name := chainInstanceName(app.Name, i)
			port := app.PortBase + i*portStride

			pid, err := readPID(pidDir, name)
			state := "stopped"
			uptime := "-"
			pidStr := "-"

			if err == nil {
				pidStr = strconv.Itoa(pid)
				if processAlive(pid) {
					state = "running"
					uptime = processUptime(pidDir, name)
				} else {
					state = "crashed"
				}
			}

			if !app.Enabled && state == "stopped" {
				state = "disabled"
			}

			_ = table.Append([]string{app.Name, name, pidStr, strconv.Itoa(port), state, uptime})
		}
	}

	_ = table.Render()
	return nil
}

// --- logs ---

func newStackLogsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logs <app>",
		Short: "Tail logs for a stack app",
		Long: `Tail the log file for a stack application.

The app name can be a base name (e.g., "explorer") which tails instance 0,
or a full instance name (e.g., "explorer-1") for a specific chain instance.`,
		RunE:         stackLogs,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
	}
}

func stackLogs(_ *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("no stack configured: %w", err)
	}

	appName := args[0]
	// If bare name, assume instance 0
	if !strings.Contains(appName, "-") || !isDigitSuffix(appName) {
		appName = chainInstanceName(appName, 0)
	}

	logDir := expandPath(cfg.LogDir)

	// For luxd, logs are in a subdirectory
	if strings.HasPrefix(appName, "luxd") {
		logPath := filepath.Join(logDir, appName)
		// luxd writes to main.log inside its log dir
		mainLog := filepath.Join(logPath, "main.log")
		if _, err := os.Stat(mainLog); err == nil {
			return tailFile(mainLog)
		}
		// Fallback: first .log file in the dir
		entries, err := os.ReadDir(logPath)
		if err != nil {
			return fmt.Errorf("no logs for %s: %w", appName, err)
		}
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".log") {
				return tailFile(filepath.Join(logPath, e.Name()))
			}
		}
		return fmt.Errorf("no log files found in %s", logPath)
	}

	logPath := filepath.Join(logDir, appName+".log")
	if _, err := os.Stat(logPath); err != nil {
		return fmt.Errorf("no logs for %s at %s", appName, logPath)
	}
	return tailFile(logPath)
}

// --- config ---

func defaultConfig() *StackConfig {
	return &StackConfig{
		Chains: 1,
		Apps: []AppEntry{
			{Name: "luxd", PortBase: 9650, Enabled: true},
			{Name: "explorer", PortBase: 3001, Enabled: true, Binary: "ghcr.io/luxfi/explorer:local"},
			{Name: "bridge", PortBase: 3002, Enabled: true},
			{Name: "exchange", PortBase: 3003, Enabled: false},
			{Name: "safe", PortBase: 3004, Enabled: true},
			{Name: "dao", PortBase: 3005, Enabled: true},
			{Name: "wallet", PortBase: 3006, Enabled: true},
			{Name: "faucet", PortBase: 3007, Enabled: true},
		},
		DataDir: "~/.lux/dev/data",
		LogDir:  "~/.lux/dev/logs",
	}
}

func stackBaseDir() string {
	return filepath.Join(os.Getenv("HOME"), constants.BaseDirName, constants.DevDir)
}

func configPath() string {
	if stackConfigPath != "" {
		return expandPath(stackConfigPath)
	}
	return filepath.Join(stackBaseDir(), stackConfigFile)
}

func loadOrCreateConfig() (*StackConfig, error) {
	cfg, err := loadConfig()
	if err == nil {
		return cfg, nil
	}

	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read config: %w", err)
	}

	// Create default config
	cfg = defaultConfig()
	if err := saveConfig(cfg); err != nil {
		return nil, err
	}
	ux.Logger.PrintToUser("Created default stack config at %s", configPath())
	return cfg, nil
}

func loadConfig() (*StackConfig, error) {
	data, err := os.ReadFile(configPath()) //nolint:gosec // G304: Reading from app's config directory
	if err != nil {
		return nil, err
	}
	var cfg StackConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", configPath(), err)
	}
	if err := validateStackConfig(&cfg); err != nil {
		return nil, fmt.Errorf("%s: %w", configPath(), err)
	}
	return &cfg, nil
}

// appNamePattern restricts stack.yaml `name` fields to a safe alphabet
// — they become PID file names, log file names, and process labels.
var appNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,31}$`)

// validateStackConfig catches obviously-wrong stack.yaml values before
// they become command-line arguments or filesystem paths. Each field
// is validated for the shape it actually needs to take.
func validateStackConfig(cfg *StackConfig) error {
	if cfg.Chains < 1 {
		return fmt.Errorf("chains: must be >= 1, got %d", cfg.Chains)
	}
	if cfg.Chains > 32 {
		return fmt.Errorf("chains: must be <= 32, got %d", cfg.Chains)
	}
	for i, app := range cfg.Apps {
		if !appNamePattern.MatchString(app.Name) {
			return fmt.Errorf("apps[%d].name %q: must match %s", i, app.Name, appNamePattern)
		}
		if app.PortBase < 1 || app.PortBase > 65535 {
			return fmt.Errorf("apps[%d].port_base %d: out of TCP range", i, app.PortBase)
		}
		if _, err := PortForAppChecked(app.PortBase, cfg.Chains-1); err != nil {
			return fmt.Errorf("apps[%d]: %w", i, err)
		}
		if app.Binary != "" && strings.ContainsAny(app.Binary, " \t\n\r\"'`$;&|<>") {
			return fmt.Errorf("apps[%d].binary: contains shell metacharacters", i)
		}
	}
	return nil
}

func saveConfig(cfg *StackConfig) error {
	dir := filepath.Dir(configPath())
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), data, 0o644) //nolint:gosec // G306: Config file needs to be readable
}

// --- process management ---

func resolveBinary(name string) (string, error) {
	// Priority 1: $APP_BIN env var (uppercase, dashes to underscores)
	envKey := strings.ToUpper(strings.ReplaceAll(name, "-", "_")) + "_BIN"
	if p := os.Getenv(envKey); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	// Priority 2: ~/.lux/dev/bin/{name}
	devBin := filepath.Join(stackBaseDir(), stackBinDir, name)
	if _, err := os.Stat(devBin); err == nil {
		return devBin, nil
	}

	// Priority 3: $PATH lookup
	if p, err := exec.LookPath(name); err == nil {
		return p, nil
	}

	// For luxd, also check the standard CLI install location
	if name == "luxd" {
		cliBin := filepath.Join(os.Getenv("HOME"), constants.BaseDirName, constants.LuxCliBinDir, "luxd")
		if _, err := os.Stat(cliBin); err == nil {
			return cliBin, nil
		}
	}

	return "", fmt.Errorf("binary not found: %s (install it or set %s)", name, envKey)
}

func spawnProcess(binary string, args []string, logFile *os.File, pidDir, name string) (int, error) {
	return spawnProcessWithEnv(binary, args, logFile, pidDir, name, nil)
}

func spawnProcessWithEnv(binary string, args []string, logFile *os.File, pidDir, name string, extraEnv []string) (int, error) {
	cmd := exec.Command(binary, args...) //nolint:gosec // G204: Running configured dev binaries
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} // Detach from parent process group

	if len(extraEnv) > 0 {
		cmd.Env = append(os.Environ(), extraEnv...)
	}

	if err := cmd.Start(); err != nil {
		return 0, err
	}

	pid := cmd.Process.Pid
	if err := writePID(pidDir, name, pid); err != nil {
		return pid, fmt.Errorf("save PID: %w", err)
	}

	// Release the process so it survives CLI exit
	_ = cmd.Process.Release()

	return pid, nil
}

func openLogFile(logDir, name string) (*os.File, error) {
	path := filepath.Join(logDir, name+".log")
	// O_NOFOLLOW blocks the classic symlink-to-arbitrary-file attack on
	// the log directory (Red #7 vector). If an attacker replaces the
	// expected log path with a symlink pointing at ~/.ssh/authorized_keys
	// or similar, OpenFile returns ELOOP instead of truncating / appending
	// to the real target. Combined with the restrictive 0o600 mode, a
	// shared /tmp-style logDir cannot be weaponised.
	return os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND|syscall.O_NOFOLLOW, 0o600) //nolint:gosec // G304: validated within log dir; O_NOFOLLOW blocks symlink races
}

// --- PID file operations ---

func writePID(pidDir, name string, pid int) error {
	if err := os.MkdirAll(pidDir, 0o750); err != nil {
		return err
	}
	path := filepath.Join(pidDir, name+".pid")
	return os.WriteFile(path, []byte(strconv.Itoa(pid)), 0o644) //nolint:gosec // G306: PID file needs to be readable
}

func readPID(pidDir, name string) (int, error) {
	path := filepath.Join(pidDir, name+".pid")
	data, err := os.ReadFile(path) //nolint:gosec // G304: Reading from app's PID directory
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

func removePIDFile(pidDir, name string) {
	_ = os.Remove(filepath.Join(pidDir, name+".pid"))
}

func isRunning(pidDir, name string) bool {
	pid, err := readPID(pidDir, name)
	if err != nil {
		return false
	}
	return processAlive(pid)
}

func processAlive(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return process.Signal(syscall.Signal(0)) == nil
}

// processUptime returns approximate uptime based on PID file mtime.
func processUptime(pidDir, name string) string {
	path := filepath.Join(pidDir, name+".pid")
	info, err := os.Stat(path)
	if err != nil {
		return "-"
	}
	d := time.Since(info.ModTime())
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	default:
		return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
	}
}

// --- chains.json ---

func writeChainsManifest(baseDir string, m *ChainsManifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(baseDir, stackChainsFile), data, 0o644) //nolint:gosec // G306: Chains manifest needs to be readable
}

// --- health check ---

func waitForHealth(httpPort int) error {
	healthURL := fmt.Sprintf("http://127.0.0.1:%d/ext/health", httpPort)
	ctx, cancel := context.WithTimeout(context.Background(), healthTimeout)
	defer cancel()

	client := &http.Client{Timeout: 2 * time.Second}
	ticker := time.NewTicker(healthInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout after %s waiting for health on port %d", healthTimeout, httpPort)
		case <-ticker.C:
			resp, err := client.Get(healthURL) //nolint:noctx // health check in loop with context timeout
			if err != nil {
				continue
			}
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
	}
}

// --- tail ---

func tailFile(path string) error {
	cmd := exec.Command("tail", "-f", path) //nolint:gosec // G204: Tailing known log file path
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// --- helpers ---

func chainInstanceName(app string, index int) string {
	if index == 0 {
		return app
	}
	return fmt.Sprintf("%s-%d", app, index)
}

func findApp(cfg *StackConfig, name string) *AppEntry {
	for i := range cfg.Apps {
		if cfg.Apps[i].Name == name {
			return &cfg.Apps[i]
		}
	}
	return nil
}

func expandPath(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[2:])
	}
	return p
}

func isDigitSuffix(s string) bool {
	idx := strings.LastIndex(s, "-")
	if idx < 0 || idx == len(s)-1 {
		return false
	}
	_, err := strconv.Atoi(s[idx+1:])
	return err == nil
}

// maxTCPPort is the upper bound for a valid TCP port number.
const maxTCPPort = 65535

// ErrPortOverflow is returned when port arithmetic would produce a
// number outside the valid TCP port range (e.g. --chains 1000 with a
// base of 9650 lands at 99,650 > 65535).
var ErrPortOverflow = fmt.Errorf("stack: port base + chain offset exceeds TCP port range")

// PortForApp returns the port for an app instance. Exported for testing.
// Callers should prefer PortForAppChecked when the chain index is
// user-controlled.
func PortForApp(portBase, chainIndex int) int {
	return portBase + chainIndex*portStride
}

// PortForAppChecked returns an error if the resulting port falls
// outside [1, 65535]. This is the sanctioned path for user-supplied
// --chains values.
func PortForAppChecked(portBase, chainIndex int) (int, error) {
	p := PortForApp(portBase, chainIndex)
	if p <= 0 || p > maxTCPPort {
		return 0, fmt.Errorf("%w: portBase=%d chainIndex=%d → port=%d",
			ErrPortOverflow, portBase, chainIndex, p)
	}
	return p, nil
}

func printStackSummary(cfg *StackConfig) {
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Dev stack running (%d chain(s))", cfg.Chains)
	ux.Logger.PrintToUser("")
	for _, app := range cfg.Apps {
		if !app.Enabled {
			continue
		}
		for i := 0; i < cfg.Chains; i++ {
			port := PortForApp(app.PortBase, i)
			name := chainInstanceName(app.Name, i)
			if app.Name == "luxd" {
				ux.Logger.PrintToUser("  %s  http://127.0.0.1:%d/ext/health", name, port)
			} else {
				ux.Logger.PrintToUser("  %s  http://127.0.0.1:%d", name, port)
			}
		}
	}
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Config:     %s", configPath())
	ux.Logger.PrintToUser("Chains:     %s", filepath.Join(stackBaseDir(), stackChainsFile))
	ux.Logger.PrintToUser("Logs:       %s", expandPath(cfg.LogDir))
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Stop with:  lux dev stack down")
}
