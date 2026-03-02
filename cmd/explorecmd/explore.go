// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package explorecmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/luxfi/cli/pkg/application"
	"github.com/spf13/cobra"
)

var app *application.Lux

// NewCmd creates the explore command for running the block explorer.
func NewCmd(injectedApp *application.Lux) *cobra.Command {
	app = injectedApp
	cmd := &cobra.Command{
		Use:   "explore",
		Short: "Run a local block explorer",
		Long: `The explore command starts a local block explorer that indexes
chain data and serves the explorer API + frontend.

USAGE:

  lux explore                     Start explorer for the running local network
  lux explore --rpc <url>         Start explorer for a specific RPC endpoint
  lux explore --chain cchain      Index a specific chain (default: cchain)
  lux explore --port 8090         API port (default: 8090)

The explorer runs as a background process. Use 'lux explore stop' to stop it.
Data is stored in ~/.lux/explorer/ and persists across restarts.

ENDPOINTS:

  http://localhost:8090/v1/explorer/stats     Chain statistics
  http://localhost:8090/v1/explorer/blocks     Block list
  http://localhost:8090/v1/explorer/search     Search
  http://localhost:8090/health                 Health check`,
		RunE: startExplorer,
	}

	cmd.Flags().String("rpc", "", "RPC endpoint (auto-detected from running network if not set)")
	cmd.Flags().String("chain", "cchain", "Chain to index (cchain, xchain, pchain, or subnet name)")
	cmd.Flags().Int("port", 8090, "HTTP port for explorer API")
	cmd.Flags().String("data", "", "Data directory (default: ~/.lux/explorer/)")
	cmd.Flags().Bool("open", true, "Open browser after starting")

	cmd.AddCommand(newStopCmd())
	cmd.AddCommand(newStatusCmd())

	return cmd
}

func newStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the running explorer",
		RunE: func(cmd *cobra.Command, args []string) error {
			return stopExplorer()
		},
	}
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show explorer status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return showStatus()
		},
	}
}

func startExplorer(cmd *cobra.Command, args []string) error {
	rpc, _ := cmd.Flags().GetString("rpc")
	chain, _ := cmd.Flags().GetString("chain")
	port, _ := cmd.Flags().GetInt("port")
	dataDir, _ := cmd.Flags().GetString("data")
	openBrowser, _ := cmd.Flags().GetBool("open")

	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, ".lux", "explorer")
	}

	// Auto-detect RPC from running local network
	if rpc == "" {
		rpc = detectRPC(chain)
		if rpc == "" {
			return fmt.Errorf("no RPC endpoint specified and no local network detected.\nUse: lux explore --rpc http://localhost:9650/ext/bc/C/rpc")
		}
	}

	// Find the explorer binary
	explorerBin := findExplorerBinary()
	if explorerBin == "" {
		return fmt.Errorf("explorer binary not found. Install with:\n  go install github.com/luxfi/explorer/cmd/indexer@latest")
	}

	// Check if already running
	if pid := readPID(); pid > 0 {
		if isRunning(pid) {
			fmt.Printf("Explorer already running (PID %d) on port %d\n", pid, port)
			fmt.Printf("  API:     http://localhost:%d/v1/explorer/stats\n", port)
			fmt.Printf("  Health:  http://localhost:%d/health\n", port)
			return nil
		}
	}

	// Start explorer in background
	explorerArgs := []string{
		"--chain", chain,
		"--rpc", rpc,
		"--port", fmt.Sprintf("%d", port),
		"--data", dataDir,
	}

	process := exec.Command(explorerBin, explorerArgs...)
	process.Stdout = nil
	process.Stderr = nil

	logFile := filepath.Join(dataDir, "explorer.log")
	os.MkdirAll(dataDir, 0755)
	f, err := os.Create(logFile)
	if err == nil {
		process.Stdout = f
		process.Stderr = f
	}

	if err := process.Start(); err != nil {
		return fmt.Errorf("failed to start explorer: %w", err)
	}

	// Save PID
	writePID(process.Process.Pid)

	fmt.Printf("Explorer started (PID %d)\n", process.Process.Pid)
	fmt.Printf("  Chain:   %s\n", chain)
	fmt.Printf("  RPC:     %s\n", rpc)
	fmt.Printf("  API:     http://localhost:%d/v1/explorer/\n", port)
	fmt.Printf("  Health:  http://localhost:%d/health\n", port)
	fmt.Printf("  Data:    %s\n", dataDir)
	fmt.Printf("  Logs:    %s\n", logFile)
	fmt.Printf("\n  Stop:    lux explore stop\n")
	fmt.Printf("  Status:  lux explore status\n")

	if openBrowser {
		openURL(fmt.Sprintf("http://localhost:%d", port))
	}

	return nil
}

func stopExplorer() error {
	pid := readPID()
	if pid <= 0 {
		fmt.Println("Explorer is not running")
		return nil
	}

	p, err := os.FindProcess(pid)
	if err != nil {
		fmt.Println("Explorer is not running")
		removePID()
		return nil
	}

	if err := p.Signal(os.Interrupt); err != nil {
		fmt.Printf("Failed to stop explorer (PID %d): %v\n", pid, err)
		return nil
	}

	removePID()
	fmt.Printf("Explorer stopped (PID %d)\n", pid)
	return nil
}

func showStatus() error {
	pid := readPID()
	if pid <= 0 || !isRunning(pid) {
		fmt.Println("Explorer is not running")
		return nil
	}

	fmt.Printf("Explorer running (PID %d)\n", pid)
	fmt.Printf("  API:     http://localhost:8090/v1/explorer/stats\n")
	fmt.Printf("  Health:  http://localhost:8090/health\n")
	return nil
}

// detectRPC finds the RPC endpoint for a running local network.
func detectRPC(chain string) string {
	// Check common local network ports
	ports := []int{9650, 9630, 9640}
	chainPath := "C"
	switch strings.ToLower(chain) {
	case "pchain":
		chainPath = "P"
	case "xchain":
		chainPath = "X"
	default:
		chainPath = "C"
	}

	for _, port := range ports {
		url := fmt.Sprintf("http://localhost:%d/ext/bc/%s/rpc", port, chainPath)
		cmd := exec.Command("curl", "-s", "-o", "/dev/null", "-w", "%{http_code}", url)
		out, err := cmd.Output()
		if err == nil && strings.TrimSpace(string(out)) == "200" {
			return url
		}
	}
	return ""
}

// findExplorerBinary finds the indexer binary.
func findExplorerBinary() string {
	// Check PATH
	if p, err := exec.LookPath("indexer"); err == nil {
		return p
	}
	// Check GOPATH/bin
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		p := filepath.Join(gopath, "bin", "indexer")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	// Check ~/go/bin
	home, _ := os.UserHomeDir()
	p := filepath.Join(home, "go", "bin", "indexer")
	if _, err := os.Stat(p); err == nil {
		return p
	}
	// Check /tmp/explorer (dev build location)
	if _, err := os.Stat("/tmp/explorer"); err == nil {
		return "/tmp/explorer"
	}
	return ""
}

func pidFile() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".lux", "explorer", "explorer.pid")
}

func writePID(pid int) {
	os.MkdirAll(filepath.Dir(pidFile()), 0755)
	os.WriteFile(pidFile(), []byte(fmt.Sprintf("%d", pid)), 0644)
}

func readPID() int {
	data, err := os.ReadFile(pidFile())
	if err != nil {
		return 0
	}
	var pid int
	fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &pid)
	return pid
}

func removePID() {
	os.Remove(pidFile())
}

func isRunning(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return p.Signal(nil) == nil
}

func openURL(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return
	}
	cmd.Start()
}
