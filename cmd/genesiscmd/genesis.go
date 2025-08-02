// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package genesiscmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/luxfi/cli/v2/pkg/ux"
	"github.com/spf13/cobra"
)

// NewCmd returns the genesis command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "genesis",
		Short: "Genesis data generation and management",
		Long: `The genesis command provides integration with the Lux genesis tool for 
managing genesis data, extracting blockchain state, and preparing mainnet launches.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		Args: cobra.NoArgs,
	}

	// Add subcommands
	cmd.AddCommand(newGenerateCmd())
	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newExtractCmd())
	cmd.AddCommand(newMigrateCmd())
	cmd.AddCommand(newValidatorsCmd())
	cmd.AddCommand(newAnalyzeCmd())

	return cmd
}

// findGenesisTool finds the genesis binary
func findGenesisTool() (string, error) {
	// Try common locations
	paths := []string{
		"genesis",                                          // In PATH
		"./genesis",                                        // Current directory
		filepath.Join(os.Getenv("HOME"), "work/lux/genesis/bin/genesis"), // Development path
		"/usr/local/bin/genesis",                           // Installed
	}

	for _, path := range paths {
		if _, err := exec.LookPath(path); err == nil {
			return path, nil
		}
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("genesis tool not found. Install from github.com/luxfi/genesis")
}

// runGenesisTool executes the genesis tool with given arguments
func runGenesisTool(args ...string) error {
	genesisTool, err := findGenesisTool()
	if err != nil {
		return err
	}

	cmd := exec.Command(genesisTool, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	ux.Logger.PrintToUser("🔄 Running: %s %s", genesisTool, strings.Join(args, " "))
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("genesis tool failed: %w", err)
	}

	return nil
}

// newGenerateCmd creates the generate subcommand
func newGenerateCmd() *cobra.Command {
	var (
		network string
		output  string
	)

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate genesis files for a network",
		Long: `Generate all genesis files (P-Chain, C-Chain, X-Chain) for the specified network.
This creates the complete genesis configuration including validators and initial allocations.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ux.Logger.PrintToUser("🌟 Generating Genesis Files")
			ux.Logger.PrintToUser("=" + strings.Repeat("=", 50))
			
			cmdArgs := []string{"generate"}
			if network != "" {
				cmdArgs = append(cmdArgs, "--network", network)
			}
			if output != "" {
				cmdArgs = append(cmdArgs, "--output", output)
			}
			
			return runGenesisTool(cmdArgs...)
		},
		Args: cobra.NoArgs,
		SilenceUsage: true,
	}

	cmd.Flags().StringVar(&network, "network", "mainnet", "Network to generate genesis for (mainnet, testnet)")
	cmd.Flags().StringVar(&output, "output", "", "Output directory (default: configs/<network>)")

	return cmd
}

// newStatusCmd creates the status subcommand
func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show genesis configuration status",
		Long:  "Display current genesis configuration and readiness status",
		RunE: func(cmd *cobra.Command, args []string) error {
			ux.Logger.PrintToUser("📊 Genesis Configuration Status")
			ux.Logger.PrintToUser("=" + strings.Repeat("=", 50))
			
			// Check for existing genesis files
			homeDir, _ := os.UserHomeDir()
			luxdHome := filepath.Join(homeDir, ".luxd")
			
			networks := []string{"mainnet", "testnet", "local"}
			for _, network := range networks {
				genesisPath := filepath.Join(luxdHome, "configs", network, "C", "genesis.json")
				if _, err := os.Stat(genesisPath); err == nil {
					ux.Logger.PrintToUser("✅ %s genesis: Found", strings.Title(network))
				} else {
					ux.Logger.PrintToUser("❌ %s genesis: Not found", strings.Title(network))
				}
			}
			
			// Run genesis tool status if available
			if genesisTool, err := findGenesisTool(); err == nil {
				ux.Logger.PrintToUser("\n🔧 Genesis Tool: %s", genesisTool)
				return runGenesisTool("status")
			}
			
			return nil
		},
		Args: cobra.NoArgs,
		SilenceUsage: true,
	}

	return cmd
}

// newExtractCmd creates the extract subcommand
func newExtractCmd() *cobra.Command {
	var (
		source      string
		destination string
		network     string
		includeState bool
	)

	cmd := &cobra.Command{
		Use:   "extract",
		Short: "Extract blockchain state from existing data",
		Long: `Extract blockchain state and account balances from an existing blockchain database.
This is used to migrate state from one network to another or to analyze blockchain data.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if source == "" {
				return fmt.Errorf("source path is required")
			}
			
			ux.Logger.PrintToUser("📤 Extracting Blockchain State")
			ux.Logger.PrintToUser("=" + strings.Repeat("=", 50))
			ux.Logger.PrintToUser("Source: %s", source)
			ux.Logger.PrintToUser("Network: %s", network)
			
			cmdArgs := []string{"extract", "state", source, destination}
			if network != "" {
				cmdArgs = append(cmdArgs, "--network", network)
			}
			if includeState {
				cmdArgs = append(cmdArgs, "--state")
			}
			
			return runGenesisTool(cmdArgs...)
		},
		Args: cobra.NoArgs,
		SilenceUsage: true,
	}

	cmd.Flags().StringVar(&source, "source", "", "Source blockchain database path")
	cmd.Flags().StringVar(&destination, "destination", "./extracted", "Destination path for extracted data")
	cmd.Flags().StringVar(&network, "network", "96369", "Network chain ID")
	cmd.Flags().BoolVar(&includeState, "state", true, "Include account state and balances")

	return cmd
}

// newMigrateCmd creates the migrate subcommand
func newMigrateCmd() *cobra.Command {
	var (
		dataDir string
		network string
		dryRun  bool
	)

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate genesis data for 2025 mainnet",
		Long: `Prepare and migrate genesis data for the 2025 mainnet launch.
This includes extracting state from existing networks and preparing final genesis files.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ux.Logger.PrintToUser("🚀 Migrating Genesis Data for 2025 Mainnet")
			ux.Logger.PrintToUser("=" + strings.Repeat("=", 50))
			
			cmdArgs := []string{"mainnet", "prepare"}
			if dataDir != "" {
				cmdArgs = append(cmdArgs, "--data-dir", dataDir)
			}
			if network != "" {
				cmdArgs = append(cmdArgs, "--network", network)
			}
			if dryRun {
				cmdArgs = append(cmdArgs, "--dry-run")
			}
			
			return runGenesisTool(cmdArgs...)
		},
		Args: cobra.NoArgs,
		SilenceUsage: true,
	}

	cmd.Flags().StringVar(&dataDir, "data-dir", "", "Data directory containing blockchain data")
	cmd.Flags().StringVar(&network, "network", "mainnet", "Target network")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Perform dry run without writing files")

	return cmd
}

// newValidatorsCmd creates the validators subcommand
func newValidatorsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validators",
		Short: "Manage genesis validators",
		Long:  "List, add, or modify validators in the genesis configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// Add subcommands
	cmd.AddCommand(newValidatorsListCmd())
	cmd.AddCommand(newValidatorsAddCmd())

	return cmd
}

// newValidatorsListCmd creates the validators list subcommand
func newValidatorsListCmd() *cobra.Command {
	var network string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List validators in genesis",
		Long:  "Display all validators configured in the genesis file",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmdArgs := []string{"validators", "list"}
			if network != "" {
				cmdArgs = append(cmdArgs, "--network", network)
			}
			
			return runGenesisTool(cmdArgs...)
		},
		Args: cobra.NoArgs,
		SilenceUsage: true,
	}

	cmd.Flags().StringVar(&network, "network", "mainnet", "Network to list validators for")

	return cmd
}

// newValidatorsAddCmd creates the validators add subcommand
func newValidatorsAddCmd() *cobra.Command {
	var (
		network    string
		configFile string
	)

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add validators to genesis",
		Long:  "Add new validators to the genesis configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			if configFile == "" {
				return fmt.Errorf("validator config file is required")
			}
			
			cmdArgs := []string{"validators", "add", "--config", configFile}
			if network != "" {
				cmdArgs = append(cmdArgs, "--network", network)
			}
			
			return runGenesisTool(cmdArgs...)
		},
		Args: cobra.NoArgs,
		SilenceUsage: true,
	}

	cmd.Flags().StringVar(&network, "network", "mainnet", "Network to add validators to")
	cmd.Flags().StringVar(&configFile, "config", "", "Validator configuration file")

	return cmd
}

// newAnalyzeCmd creates the analyze subcommand
func newAnalyzeCmd() *cobra.Command {
	var (
		dataPath string
		network  string
		account  string
	)

	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze blockchain data",
		Long: `Analyze extracted blockchain data to find accounts, balances, and other information.
This is useful for verifying migrations and understanding the state of the blockchain.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if dataPath == "" {
				return fmt.Errorf("data path is required")
			}
			
			ux.Logger.PrintToUser("🔍 Analyzing Blockchain Data")
			ux.Logger.PrintToUser("=" + strings.Repeat("=", 50))
			
			cmdArgs := []string{"analyze", dataPath}
			if network != "" {
				cmdArgs = append(cmdArgs, "--network", network)
			}
			if account != "" {
				cmdArgs = append(cmdArgs, "--account", account)
			}
			
			return runGenesisTool(cmdArgs...)
		},
		Args: cobra.NoArgs,
		SilenceUsage: true,
	}

	cmd.Flags().StringVar(&dataPath, "data", "", "Path to extracted blockchain data")
	cmd.Flags().StringVar(&network, "network", "", "Network name for analysis")
	cmd.Flags().StringVar(&account, "account", "", "Specific account to analyze")

	return cmd
}