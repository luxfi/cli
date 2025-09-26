// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package nodecmd

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

type validatorConfig struct {
	Name        string    `json:"name"`
	Seed        string    `json:"seed"`
	Account     int       `json:"account"`
	HTTPPort    int       `json:"http_port"`
	StakingPort int       `json:"staking_port"`
	Bootstrap   string    `json:"bootstrap"`
	Group       string    `json:"group"`
	NetworkID   uint32    `json:"network_id"`
	Created     time.Time `json:"created"`
}

type validatorRuntime struct {
	PID     int       `json:"pid"`
	Started time.Time `json:"started"`
	HTTPUrl string    `json:"http_url"`
	RPCUrl  string    `json:"rpc_url"`
	WSUrl   string    `json:"ws_url"`
}

func newValidatorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validator",
		Short: "Manage validator nodes with flexible wallet configurations",
		Long: `Manage multiple validator nodes, each with its own wallet configuration.
Each validator can have its own seed phrase and account index, allowing
complete flexibility in deployment scenarios.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// Add subcommands
	cmd.AddCommand(newValidatorAddCmd())
	cmd.AddCommand(newValidatorStartCmd())
	cmd.AddCommand(newValidatorStopCmd())
	cmd.AddCommand(newValidatorStatusCmd())
	cmd.AddCommand(newValidatorListCmd())
	cmd.AddCommand(newValidatorRemoveCmd())
	cmd.AddCommand(newValidatorExportCmd())
	cmd.AddCommand(newValidatorImportCmd())

	return cmd
}

func newValidatorAddCmd() *cobra.Command {
	var (
		name        string
		seed        string
		account     int
		httpPort    int
		stakingPort int
		bootstrap   string
		group       string
		networkID   uint32
	)

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new validator configuration",
		Long: `Add a new validator with its own wallet configuration.
Each validator can have its own seed phrase and account index.`,
		Example: `  # Add validator with specific seed and account
  lux node validator add --name mainnet-0 --seed "your seed phrase" --account 0 --http-port 9630

  # Add multiple validators with same seed, different accounts
  lux node validator add --name mainnet-1 --seed "your seed phrase" --account 1 --http-port 9640 --bootstrap "127.0.0.1:9631"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return addValidator(name, seed, account, httpPort, stakingPort, bootstrap, group, networkID)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Validator name (required)")
	cmd.Flags().StringVar(&seed, "seed", "", "Wallet seed phrase (required)")
	cmd.Flags().IntVar(&account, "account", 0, "Wallet account index")
	cmd.Flags().IntVar(&httpPort, "http-port", 0, "HTTP port (auto-assigned if 0)")
	cmd.Flags().IntVar(&stakingPort, "staking-port", 0, "Staking port (default: http-port + 1)")
	cmd.Flags().StringVar(&bootstrap, "bootstrap", "", "Bootstrap nodes (comma-separated)")
	cmd.Flags().StringVar(&group, "group", "default", "Validator group")
	cmd.Flags().Uint32Var(&networkID, "network-id", 96369, "Network ID")

	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("seed")

	return cmd
}

func newValidatorStartCmd() *cobra.Command {
	var (
		name  string
		group string
	)

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start validator(s)",
		Long:  `Start one or more validators by name or group.`,
		Example: `  # Start specific validator
  lux node validator start --name mainnet-0

  # Start all validators in a group
  lux node validator start --group mainnet`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" && group == "" {
				return fmt.Errorf("either --name or --group required")
			}
			return startValidator(name, group)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Validator name")
	cmd.Flags().StringVar(&group, "group", "", "Validator group")

	return cmd
}

func newValidatorStopCmd() *cobra.Command {
	var (
		name  string
		group string
	)

	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop validator(s)",
		Long:  `Stop one or more validators by name or group.`,
		Example: `  # Stop specific validator
  lux node validator stop --name mainnet-0

  # Stop all validators in a group
  lux node validator stop --group mainnet`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" && group == "" {
				return fmt.Errorf("either --name or --group required")
			}
			return stopValidator(name, group)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Validator name")
	cmd.Flags().StringVar(&group, "group", "", "Validator group")

	return cmd
}

func newValidatorStatusCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Check validator status",
		Long:  `Check the status of all validators or a specific validator.`,
		Example: `  # Check all validators
  lux node validator status

  # Check specific validator
  lux node validator status --name mainnet-0`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return checkValidatorStatus(name)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Validator name (optional)")

	return cmd
}

func newValidatorListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all configured validators",
		Long:  `List all configured validators organized by group.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return listValidators()
		},
	}

	return cmd
}

func newValidatorRemoveCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:     "remove",
		Short:   "Remove a validator configuration",
		Long:    `Remove a validator configuration and optionally its data.`,
		Example: `  lux node validator remove --name mainnet-0`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return removeValidator(name)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Validator name (required)")
	cmd.MarkFlagRequired("name")

	return cmd
}

func newValidatorExportCmd() *cobra.Command {
	var file string

	cmd := &cobra.Command{
		Use:     "export",
		Short:   "Export validator configurations",
		Long:    `Export validator configurations to a JSON file for backup or migration.`,
		Example: `  lux node validator export --file validators-backup.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return exportValidators(file)
		},
	}

	cmd.Flags().StringVar(&file, "file", "validators-export.json", "Output file")

	return cmd
}

func newValidatorImportCmd() *cobra.Command {
	var file string

	cmd := &cobra.Command{
		Use:     "import",
		Short:   "Import validator configurations",
		Long:    `Import validator configurations from a JSON file.`,
		Example: `  lux node validator import --file validators-backup.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return importValidators(file)
		},
	}

	cmd.Flags().StringVar(&file, "file", "", "Input file (required)")
	cmd.MarkFlagRequired("file")

	return cmd
}

// Implementation functions

func getValidatorConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".lux", "validators")
}

func addValidator(name, seed string, account, httpPort, stakingPort int, bootstrap, group string, networkID uint32) error {
	configDir := getValidatorConfigDir()
	valDir := filepath.Join(configDir, name)

	// Check if already exists
	if _, err := os.Stat(valDir); err == nil {
		return fmt.Errorf("validator %s already exists", name)
	}

	// Auto-assign ports if not specified
	if httpPort == 0 {
		httpPort = 9630
		// Find next available port
		for {
			inUse := false
			// Check all validator configs
			entries, _ := ioutil.ReadDir(configDir)
			for _, entry := range entries {
				if entry.IsDir() {
					configFile := filepath.Join(configDir, entry.Name(), "config.json")
					if data, err := ioutil.ReadFile(configFile); err == nil {
						var config validatorConfig
						if json.Unmarshal(data, &config) == nil && config.HTTPPort == httpPort {
							inUse = true
							break
						}
					}
				}
			}
			if !inUse {
				break
			}
			httpPort += 10
		}
	}

	if stakingPort == 0 {
		stakingPort = httpPort + 1
	}

	// Create validator directory
	if err := os.MkdirAll(valDir, 0755); err != nil {
		return err
	}

	// Save configuration
	config := validatorConfig{
		Name:        name,
		Seed:        seed,
		Account:     account,
		HTTPPort:    httpPort,
		StakingPort: stakingPort,
		Bootstrap:   bootstrap,
		Group:       group,
		NetworkID:   networkID,
		Created:     time.Now(),
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(filepath.Join(valDir, "config.json"), data, 0600); err != nil {
		return err
	}

	ux.Logger.PrintToUser("Added validator %s", name)
	ux.Logger.PrintToUser("  Account: %d", account)
	ux.Logger.PrintToUser("  HTTP Port: %d", httpPort)
	ux.Logger.PrintToUser("  Staking Port: %d", stakingPort)
	ux.Logger.PrintToUser("  Group: %s", group)

	return nil
}

func startValidator(name, group string) error {
	configDir := getValidatorConfigDir()

	if group != "" && name == "" {
		// Start all validators in group
		ux.Logger.PrintToUser("Starting validators in group: %s", group)
		count := 0
		entries, _ := ioutil.ReadDir(configDir)
		for _, entry := range entries {
			if entry.IsDir() {
				configFile := filepath.Join(configDir, entry.Name(), "config.json")
				if data, err := ioutil.ReadFile(configFile); err == nil {
					var config validatorConfig
					if json.Unmarshal(data, &config) == nil && config.Group == group {
						if err := startSingleValidator(entry.Name()); err != nil {
							ux.Logger.PrintToUser("Warning: Failed to start %s: %v", entry.Name(), err)
						} else {
							count++
						}
						time.Sleep(2 * time.Second)
					}
				}
			}
		}
		ux.Logger.PrintToUser("Started %d validators in group %s", count, group)
		return nil
	}

	// Start single validator
	return startSingleValidator(name)
}

func startSingleValidator(name string) error {
	configDir := getValidatorConfigDir()
	valDir := filepath.Join(configDir, name)
	configFile := filepath.Join(valDir, "config.json")

	// Load configuration
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config: %v", err)
	}

	var config validatorConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config: %v", err)
	}

	// Check if already running
	pidFile := filepath.Join(valDir, "validator.pid")
	if data, err := ioutil.ReadFile(pidFile); err == nil {
		pid, _ := strconv.Atoi(strings.TrimSpace(string(data)))
		// Check if process is still running
		if err := syscall.Kill(pid, 0); err == nil {
			ux.Logger.PrintToUser("Validator %s is already running (PID: %d)", name, pid)
			return nil
		}
	}

	// Create data directory
	dataDir := filepath.Join(valDir, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}

	// Generate staking certificates if not exist
	stakingDir := filepath.Join(dataDir, "staking")
	if err := os.MkdirAll(stakingDir, 0755); err != nil {
		return err
	}

	stakingKeyPath := filepath.Join(stakingDir, "staker.key")
	stakingCertPath := filepath.Join(stakingDir, "staker.crt")

	if _, err := os.Stat(stakingKeyPath); os.IsNotExist(err) {
		// Generate new staking credentials
		if err := generateStakingCreds(stakingKeyPath, stakingCertPath); err != nil {
			return fmt.Errorf("failed to generate staking credentials: %v", err)
		}
	}

	// Build luxd command
	luxdPath := filepath.Join(app.GetBaseDir(), "..", "..", "node", "build", "luxd")
	if _, err := os.Stat(luxdPath); os.IsNotExist(err) {
		return fmt.Errorf("luxd binary not found at %s", luxdPath)
	}

	args := []string{
		"--network-id", fmt.Sprintf("%d", config.NetworkID),
		"--data-dir", dataDir,
		"--db-dir", filepath.Join(dataDir, "db"),
		"--staking-tls-key-file", stakingKeyPath,
		"--staking-tls-cert-file", stakingCertPath,
		"--http-port", fmt.Sprintf("%d", config.HTTPPort),
		"--staking-port", fmt.Sprintf("%d", config.StakingPort),
		"--http-host", "0.0.0.0",
		"--staking-enabled", "true",
		"--api-admin-enabled", "true",
		"--api-keystore-enabled", "true",
		"--api-metrics-enabled", "true",
		"--index-enabled", "true",
		"--log-level", "info",
	}

	// Add bootstrap nodes if provided
	if config.Bootstrap != "" {
		args = append(args, "--bootstrap-ips", config.Bootstrap)
	}

	// Set environment variables for wallet
	env := os.Environ()
	env = append(env, fmt.Sprintf("WALLET_SEED=%s", config.Seed))
	env = append(env, fmt.Sprintf("WALLET_ACCOUNT=%d", config.Account))

	// Create log file
	logFile := filepath.Join(valDir, "validator.log")
	log, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer log.Close()

	// Start the process
	cmd := exec.Command(luxdPath, args...)
	cmd.Env = env
	cmd.Stdout = log
	cmd.Stderr = log

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start luxd: %v", err)
	}

	// Save PID
	if err := ioutil.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0644); err != nil {
		cmd.Process.Kill()
		return err
	}

	// Save runtime info
	runtime := validatorRuntime{
		PID:     cmd.Process.Pid,
		Started: time.Now(),
		HTTPUrl: fmt.Sprintf("http://localhost:%d", config.HTTPPort),
		RPCUrl:  fmt.Sprintf("http://localhost:%d/ext/bc/C/rpc", config.HTTPPort),
		WSUrl:   fmt.Sprintf("ws://localhost:%d/ext/bc/C/ws", config.HTTPPort),
	}

	runtimeData, _ := json.MarshalIndent(runtime, "", "  ")
	ioutil.WriteFile(filepath.Join(valDir, "runtime.json"), runtimeData, 0644)

	ux.Logger.PrintToUser("Started validator %s", name)
	ux.Logger.PrintToUser("  PID: %d", cmd.Process.Pid)
	ux.Logger.PrintToUser("  HTTP: %s", runtime.HTTPUrl)
	ux.Logger.PrintToUser("  RPC: %s", runtime.RPCUrl)
	ux.Logger.PrintToUser("  Logs: %s", logFile)

	return nil
}

func stopValidator(name, group string) error {
	configDir := getValidatorConfigDir()

	if group != "" && name == "" {
		// Stop all validators in group
		ux.Logger.PrintToUser("Stopping validators in group: %s", group)
		count := 0
		entries, _ := ioutil.ReadDir(configDir)
		for _, entry := range entries {
			if entry.IsDir() {
				configFile := filepath.Join(configDir, entry.Name(), "config.json")
				if data, err := ioutil.ReadFile(configFile); err == nil {
					var config validatorConfig
					if json.Unmarshal(data, &config) == nil && config.Group == group {
						if err := stopSingleValidator(entry.Name()); err != nil {
							ux.Logger.PrintToUser("Warning: Failed to stop %s: %v", entry.Name(), err)
						} else {
							count++
						}
					}
				}
			}
		}
		ux.Logger.PrintToUser("Stopped %d validators in group %s", count, group)
		return nil
	}

	// Stop single validator
	return stopSingleValidator(name)
}

func stopSingleValidator(name string) error {
	configDir := getValidatorConfigDir()
	pidFile := filepath.Join(configDir, name, "validator.pid")

	data, err := ioutil.ReadFile(pidFile)
	if err != nil {
		return fmt.Errorf("validator %s is not running", name)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return fmt.Errorf("invalid PID in %s", pidFile)
	}

	// Check if process exists
	if err := syscall.Kill(pid, 0); err != nil {
		// Process doesn't exist, clean up PID file
		os.Remove(pidFile)
		return fmt.Errorf("validator %s is not running (stale PID file)", name)
	}

	// Send SIGTERM
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to stop validator: %v", err)
	}

	// Wait for process to exit (up to 10 seconds)
	for i := 0; i < 10; i++ {
		if err := syscall.Kill(pid, 0); err != nil {
			// Process has exited
			break
		}
		time.Sleep(time.Second)
	}

	// Clean up files
	os.Remove(pidFile)
	os.Remove(filepath.Join(configDir, name, "runtime.json"))

	ux.Logger.PrintToUser("Stopped validator %s (PID: %d)", name, pid)
	return nil
}

func checkValidatorStatus(name string) error {
	configDir := getValidatorConfigDir()

	// If no name specified, show all validators
	if name == "" {
		entries, err := ioutil.ReadDir(configDir)
		if err != nil {
			return err
		}

		ux.Logger.PrintToUser("=== Validator Status ===")
		for _, entry := range entries {
			if entry.IsDir() {
				showValidatorStatus(entry.Name())
			}
		}
		return nil
	}

	// Show specific validator
	return showValidatorStatus(name)
}

func showValidatorStatus(name string) error {
	configDir := getValidatorConfigDir()
	valDir := filepath.Join(configDir, name)

	// Load config
	configFile := filepath.Join(valDir, "config.json")
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		ux.Logger.PrintToUser("  %s: Not configured", name)
		return nil
	}

	var config validatorConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	// Check if running
	pidFile := filepath.Join(valDir, "validator.pid")
	runtimeFile := filepath.Join(valDir, "runtime.json")

	if pidData, err := ioutil.ReadFile(pidFile); err == nil {
		pid, _ := strconv.Atoi(strings.TrimSpace(string(pidData)))

		// Check if process is still running
		if err := syscall.Kill(pid, 0); err == nil {
			// Running - load runtime info
			var runtime validatorRuntime
			if rtData, err := ioutil.ReadFile(runtimeFile); err == nil {
				json.Unmarshal(rtData, &runtime)
			}

			// Try to check health
			healthStatus := "Unknown"
			resp, err := http.Get(fmt.Sprintf("http://localhost:%d/ext/health", config.HTTPPort))
			if err == nil {
				defer resp.Body.Close()
				var health map[string]interface{}
				if json.NewDecoder(resp.Body).Decode(&health) == nil {
					if healthy, ok := health["healthy"].(bool); ok && healthy {
						healthStatus = "Healthy"
					} else {
						healthStatus = "Unhealthy"
					}
				}
			}

			ux.Logger.PrintToUser("\n%s:", name)
			ux.Logger.PrintToUser("  Status: Running")
			ux.Logger.PrintToUser("  PID: %d", pid)
			ux.Logger.PrintToUser("  Health: %s", healthStatus)
			ux.Logger.PrintToUser("  Account: %d", config.Account)
			ux.Logger.PrintToUser("  Group: %s", config.Group)
			ux.Logger.PrintToUser("  HTTP Port: %d", config.HTTPPort)
			ux.Logger.PrintToUser("  RPC URL: %s", runtime.RPCUrl)
			ux.Logger.PrintToUser("  Started: %s", runtime.Started.Format("2006-01-02 15:04:05"))
		} else {
			// Process died, clean up
			os.Remove(pidFile)
			os.Remove(runtimeFile)
			ux.Logger.PrintToUser("\n%s: Stopped (stale PID file)", name)
		}
	} else {
		ux.Logger.PrintToUser("\n%s: Stopped", name)
		ux.Logger.PrintToUser("  Account: %d", config.Account)
		ux.Logger.PrintToUser("  Group: %s", config.Group)
		ux.Logger.PrintToUser("  HTTP Port: %d", config.HTTPPort)
	}

	return nil
}

func listValidators() error {
	configDir := getValidatorConfigDir()

	// Group validators by group
	groups := make(map[string][]validatorConfig)

	entries, err := ioutil.ReadDir(configDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			configFile := filepath.Join(configDir, entry.Name(), "config.json")
			if data, err := ioutil.ReadFile(configFile); err == nil {
				var config validatorConfig
				if json.Unmarshal(data, &config) == nil {
					groups[config.Group] = append(groups[config.Group], config)
				}
			}
		}
	}

	ux.Logger.PrintToUser("=== Configured Validators ===")
	for group, validators := range groups {
		ux.Logger.PrintToUser("\nGroup: %s", group)
		for _, val := range validators {
			ux.Logger.PrintToUser("  - %s (account: %d, port: %d)", val.Name, val.Account, val.HTTPPort)
		}
	}

	return nil
}

func removeValidator(name string) error {
	configDir := getValidatorConfigDir()
	valDir := filepath.Join(configDir, name)

	// Check if validator exists
	if _, err := os.Stat(valDir); os.IsNotExist(err) {
		return fmt.Errorf("validator %s does not exist", name)
	}

	// Check if running
	pidFile := filepath.Join(valDir, "validator.pid")
	if data, err := ioutil.ReadFile(pidFile); err == nil {
		pid, _ := strconv.Atoi(strings.TrimSpace(string(data)))
		if err := syscall.Kill(pid, 0); err == nil {
			return fmt.Errorf("validator %s is still running - stop it first", name)
		}
	}

	// Remove directory
	if err := os.RemoveAll(valDir); err != nil {
		return fmt.Errorf("failed to remove validator: %v", err)
	}

	ux.Logger.PrintToUser("Removed validator %s", name)
	return nil
}

func exportValidators(file string) error {
	configDir := getValidatorConfigDir()

	// Collect all validator configs
	var validators []validatorConfig

	entries, err := ioutil.ReadDir(configDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			configFile := filepath.Join(configDir, entry.Name(), "config.json")
			if data, err := ioutil.ReadFile(configFile); err == nil {
				var config validatorConfig
				if json.Unmarshal(data, &config) == nil {
					validators = append(validators, config)
				}
			}
		}
	}

	// Write to file
	data, err := json.MarshalIndent(validators, "", "  ")
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(file, data, 0644); err != nil {
		return err
	}

	ux.Logger.PrintToUser("Exported %d validators to %s", len(validators), file)
	return nil
}

func importValidators(file string) error {
	// Read the file
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	var validators []validatorConfig
	if err := json.Unmarshal(data, &validators); err != nil {
		return fmt.Errorf("failed to parse JSON: %v", err)
	}

	imported := 0
	for _, val := range validators {
		if err := addValidator(val.Name, val.Seed, val.Account, val.HTTPPort, val.StakingPort, val.Bootstrap, val.Group, val.NetworkID); err != nil {
			ux.Logger.PrintToUser("Warning: Failed to import %s: %v", val.Name, err)
		} else {
			imported++
		}
	}

	ux.Logger.PrintToUser("Imported %d validators from %s", imported, file)
	return nil
}

// Helper function to generate staking credentials
func generateStakingCreds(keyPath, certPath string) error {
	// Generate a new private key
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"Lux"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"lux"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour * 10), // 10 years
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// Create the certificate
	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return err
	}

	// Write key
	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer keyOut.Close()

	privKeyBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return err
	}

	if err := pem.Encode(keyOut, &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privKeyBytes,
	}); err != nil {
		return err
	}

	// Write certificate
	certOut, err := os.Create(certPath)
	if err != nil {
		return err
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	}); err != nil {
		return err
	}

	return nil
}
