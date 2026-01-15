// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/luxfi/cli/cmd/ammcmd"
	"github.com/luxfi/cli/cmd/configcmd"
	"github.com/luxfi/log/level"

	"github.com/luxfi/cli/cmd/backendcmd"
	"github.com/luxfi/cli/cmd/chaincmd"
	"github.com/luxfi/cli/cmd/contractcmd"
	"github.com/luxfi/cli/cmd/devcmd"
	"github.com/luxfi/cli/cmd/dexcmd"
	"github.com/luxfi/cli/cmd/gpucmd"
	"github.com/luxfi/cli/cmd/keycmd"
	"github.com/luxfi/cli/cmd/networkcmd"
	"github.com/luxfi/cli/cmd/primarycmd"
	"github.com/luxfi/cli/cmd/rpccmd"
	"github.com/luxfi/cli/cmd/updatecmd"
	"github.com/luxfi/cli/cmd/validatorcmd"
	"github.com/luxfi/cli/cmd/vmcmd"
	"github.com/luxfi/cli/cmd/warpcmd"
	"github.com/luxfi/cli/internal/migrations"
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/config"
	"github.com/luxfi/cli/pkg/lpmintegration"
	"github.com/luxfi/cli/pkg/prompts"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/constants"
	"github.com/luxfi/filesystem/perms"
	luxlog "github.com/luxfi/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	app        *application.Lux
	logFactory luxlog.Factory

	logLevel       string
	Version        = "1.22.5"
	cfgFile        string
	skipCheck      bool
	nonInteractive bool
	verboseFlag    bool
	debugFlag      bool
	quietFlag      bool
)

func NewRootCmd() *cobra.Command {
	// rootCmd represents the base command when called without any subcommands
	rootCmd := &cobra.Command{
		Use: "lux",
		Long: `Lux CLI - Developer toolchain for blockchain development and deployment.

The Lux CLI provides a complete toolkit for creating, testing, and deploying
blockchains on the Lux network. It supports local development, testnet
deployment, and mainnet operations with a unified command structure.

COMMAND OVERVIEW:

  network     Manage local network runtime (start/stop/status/clean)
  chain       Blockchain lifecycle (create/deploy/import/export)
  key         Key and wallet management
  validator   Validator operations
  config      CLI configuration

ARCHITECTURE:

  L1 (Sovereign)  - Independent validator set, own tokenomics
  L2 (Rollup)     - Based on L1 sequencing (Lux, Ethereum, etc.)
  L3 (App Chain)  - Built on L2 for application-specific use

SEQUENCING OPTIONS:

  lux       100ms blocks, lowest cost (default)
  ethereum  12s blocks, highest security
  op        OP Stack compatible

NETWORK TYPES:

  --mainnet   Production network (5 validators, port 9630)
  --testnet   Test network (5 validators, port 9640)
  --devnet    Development network (5 validators, port 9650)
  --dev       Single-node dev mode with K=1 consensus

QUICK START:

  # Start a local development network
  lux network start --devnet

  # Create a new blockchain
  lux chain create mychain

  # Deploy to local network
  lux chain deploy mychain

  # Check status
  lux network status
  lux chain list

For detailed command help, use: lux <command> --help`,
		PersistentPreRunE: createApp,
		Version:           Version,
		PersistentPostRun: handleTracking,
	}

	// Disable printing the completion command
	rootCmd.CompletionOptions.HiddenDefaultCmd = true

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.lux/cli.json)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "ERROR", "log level for the application")
	rootCmd.PersistentFlags().BoolVar(&skipCheck, constants.SkipUpdateFlag, false, "skip check for new versions")
	rootCmd.PersistentFlags().BoolVar(&nonInteractive, "non-interactive", false,
		"Disable prompts; fail if required values are missing (also enabled when stdin is not a TTY or CI=1)")
	rootCmd.PersistentFlags().Bool("verbose", false, "Show verbose output (info level logs)")
	rootCmd.PersistentFlags().Bool("debug", false, "Show debug output (debug level logs)")
	rootCmd.PersistentFlags().Bool("quiet", false, "Show only errors (quiet mode)")

	// add sub commands
	rootCmd.AddCommand(devcmd.NewCmd(app))        // dev (local dev environment)
	rootCmd.AddCommand(networkcmd.NewCmd(app))    // network (local network management)
	rootCmd.AddCommand(networkcmd.NewStatusCmd()) // status alias (new version)
	rootCmd.AddCommand(primarycmd.NewCmd(app))
	rootCmd.AddCommand(chaincmd.NewCmd(app)) // unified chain command (l1/l2/l3)

	// add transaction command

	// add config command
	rootCmd.AddCommand(configcmd.NewCmd(app))

	// add update command
	rootCmd.AddCommand(updatecmd.NewCmd(app, Version))

	// add warp command (cross-chain messaging)
	rootCmd.AddCommand(warpcmd.NewCmd(app))

	// add dex command (decentralized exchange)
	rootCmd.AddCommand(dexcmd.NewCmd(app))

	// add amm command (Uniswap-style AMM trading)
	rootCmd.AddCommand(ammcmd.NewCmd(app))

	// add contract command
	rootCmd.AddCommand(contractcmd.NewCmd(app))

	// add validator command
	rootCmd.AddCommand(validatorcmd.NewCmd(app))

	// add key management command
	rootCmd.AddCommand(keycmd.NewCmd(app))

	// add vm management command
	rootCmd.AddCommand(vmcmd.NewCmd(app))

	// add gpu management command
	rootCmd.AddCommand(gpucmd.NewCmd(app))

	// add rpc command for direct RPC calls
	rootCmd.AddCommand(rpccmd.NewCmd(app))

	// add hidden backend command (base)
	rootCmd.AddCommand(backendcmd.NewCmd(app))

	// add network-specific backend commands (lux-mainnet-grpc, lux-testnet-grpc, etc.)
	for _, cmd := range backendcmd.NewAllNetworkCmds(app) {
		rootCmd.AddCommand(cmd)
	}

	return rootCmd
}

func createApp(cmd *cobra.Command, _ []string) error {
	baseDir, err := setupEnv()
	if err != nil {
		return err
	}
	log, err := setupLogging(baseDir)
	if err != nil {
		return err
	}

	// Adjust log level based on flags (must be done after flags are parsed)
	if cmd.Flags().Changed("debug") {
		logFactory.SetDisplayLevel("lux", luxlog.Level(-4)) // DEBUG
	} else if cmd.Flags().Changed("verbose") {
		logFactory.SetDisplayLevel("lux", luxlog.Level(0)) // INFO
	} else if cmd.Flags().Changed("quiet") {
		logFactory.SetDisplayLevel("lux", luxlog.Level(8)) // ERROR
	} else if logLevel != "" {
		level, err := luxlog.ToLevel(logLevel)
		if err == nil {
			logFactory.SetDisplayLevel("lux", level)
		}
	}

	cf := config.New()

	// Adjust log level based on flags BEFORE any logging happens
	// Use only luxlog types to avoid mixing log libraries
	if cmd.Flags().Changed("debug") {
		logFactory.SetLogLevel("lux", luxlog.Level(level.Debug))
		logFactory.SetDisplayLevel("lux", luxlog.Level(level.Debug))
	} else if cmd.Flags().Changed("verbose") {
		logFactory.SetLogLevel("lux", luxlog.Level(level.Info))
		logFactory.SetDisplayLevel("lux", luxlog.Level(level.Info))
	} else if cmd.Flags().Changed("quiet") {
		logFactory.SetLogLevel("lux", luxlog.Level(level.Error))
		logFactory.SetDisplayLevel("lux", luxlog.Level(level.Error))
	} else if logLevel != "" {
		level, err := luxlog.ToLevel(logLevel)
		if err == nil {
			logFactory.SetLogLevel("lux", level)
			logFactory.SetDisplayLevel("lux", level)
		}
	}

	// If --non-interactive flag is set, propagate to env so IsInteractive() sees it
	// This allows TTY detection to work automatically while still respecting the flag
	if nonInteractive {
		_ = os.Setenv(prompts.EnvNonInteractive, "1")
	}

	// Interactive by default on TTY, non-interactive when:
	// LUX_NON_INTERACTIVE=1, CI=1, --non-interactive flag, or stdin is piped
	prompter := prompts.NewPrompterForMode(nonInteractive)
	app.Setup(baseDir, log, cf, prompter, application.NewDownloader())

	// Setup LPM, skip if running a hidden command
	if !cmd.Hidden {
		usr, err := user.Current()
		if err != nil {
			app.Log.Error("unable to get system user")
			return err
		}
		lpmBaseDir := filepath.Join(usr.HomeDir, constants.LPMDir)
		if err = lpmintegration.SetupLpm(app, lpmBaseDir); err != nil {
			return err
		}
	}

	initConfig()

	if err := migrations.RunMigrations(app); err != nil {
		return err
	}

	// Skip metrics prompt in non-interactive mode, E2E tests, or if config exists
	if os.Getenv("RUN_E2E") == "" && prompts.IsInteractive() && !app.ConfigFileExists() {
		err = utils.HandleUserMetricsPreference(app)
		if err != nil {
			return err
		}
	}
	if err := checkForUpdates(cmd, app); err != nil {
		return err
	}

	return nil
}

// checkForUpdates evaluates first if the user is maybe wanting to skip the update check
// if there's no skip, it runs the update check
func checkForUpdates(cmd *cobra.Command, app *application.Lux) error {
	// If skip-update-check is enabled (via flag or config), skip silently
	if skipCheck {
		return nil
	}

	var (
		lastActs *application.LastActions
		err      error
	)
	// we store a timestamp of the last skip check in a file
	lastActs, err = app.ReadLastActionsFile()
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			app.Log.Warn("failed to read last-actions file! This is non-critical but is logged", "error", err)
		}
		lastActs = &application.LastActions{}
	}

	// if the user had requested to skipCheck less than 24 hrs ago via flag, we skip
	if lastActs.LastSkipCheck != (time.Time{}) &&
		time.Now().Before(lastActs.LastSkipCheck.Add(24*time.Hour)) {
		return nil
	}

	// at this point we want to run the check
	isUserCalled := false
	commandList := strings.Fields(cmd.CommandPath())
	if len(commandList) <= 1 || commandList[1] != "update" {
		if err := updatecmd.Update(cmd, isUserCalled, Version); err != nil {
			if errors.Is(err, updatecmd.ErrUserAbortedInstallation) {
				return nil
			}
			if errors.Is(err, updatecmd.ErrNoVersion) {
				ux.Logger.PrintToUser(
					"Attempted to check if a new version is available, but couldn't find the currently running version information")
				ux.Logger.PrintToUser(
					"Make sure to follow official instructions, or automatic updates won't be available for you")
				return nil
			}
			return err
		}
	}
	return nil
}

func handleTracking(cmd *cobra.Command, _ []string) {
	utils.HandleTracking(cmd, app, nil)
}

func setupEnv() (string, error) {
	// Set base dir
	usr, err := user.Current()
	if err != nil {
		// no logger here yet
		fmt.Printf("unable to get system user %s\n", err)
		return "", err
	}
	baseDir := filepath.Join(usr.HomeDir, constants.BaseDirName)

	// Create base dir if it doesn't exist
	err = os.MkdirAll(baseDir, 0o750)
	if err != nil {
		// no logger here yet
		fmt.Printf("failed creating the basedir %s: %s\n", baseDir, err)
		return "", err
	}

	// Create snapshots dir if it doesn't exist
	snapshotsDir := filepath.Join(baseDir, constants.SnapshotsDirName)
	if err = os.MkdirAll(snapshotsDir, 0o750); err != nil {
		fmt.Printf("failed creating the snapshots dir %s: %s\n", snapshotsDir, err)
		os.Exit(1)
	}

	// Create key dir if it doesn't exist
	keyDir := filepath.Join(baseDir, constants.KeyDir)
	if err = os.MkdirAll(keyDir, 0o750); err != nil {
		fmt.Printf("failed creating the key dir %s: %s\n", keyDir, err)
		os.Exit(1)
	}

	// Create custom vm dir if it doesn't exist
	vmDir := filepath.Join(baseDir, constants.CustomVMDir)
	if err = os.MkdirAll(vmDir, 0o750); err != nil {
		fmt.Printf("failed creating the vm dir %s: %s\n", vmDir, err)
		os.Exit(1)
	}

	// Create chain dir if it doesn't exist
	chainDir := filepath.Join(baseDir, constants.ChainsDir)
	if err = os.MkdirAll(chainDir, 0o750); err != nil {
		fmt.Printf("failed creating the chain dir %s: %s\n", chainDir, err)
		os.Exit(1)
	}

	// Create repos dir if it doesn't exist
	repoDir := filepath.Join(baseDir, constants.ReposDir)
	if err = os.MkdirAll(repoDir, 0o750); err != nil {
		fmt.Printf("failed creating the repo dir %s: %s\n", repoDir, err)
		os.Exit(1)
	}

	pluginDir := filepath.Join(baseDir, constants.PluginDir)
	if err = os.MkdirAll(pluginDir, 0o750); err != nil {
		fmt.Printf("failed creating the plugin dir %s: %s\n", pluginDir, err)
		os.Exit(1)
	}

	return baseDir, nil
}

func setupLogging(baseDir string) (luxlog.Logger, error) {
	var err error

	config := luxlog.Config{}
	config.LogLevel = luxlog.Level(-6) // Info level for file logging

	// Set default display level to WARN (quiet by default)
	config.DisplayLevel, _ = luxlog.ToLevel("WARN")

	// Log level can be overridden by flags, but we'll handle that in createApp
	// after flags are parsed, by adjusting the logger level dynamically

	config.Directory = filepath.Join(baseDir, constants.LogDir)
	if err := os.MkdirAll(config.Directory, perms.ReadWriteExecute); err != nil {
		return nil, fmt.Errorf("failed creating log directory: %w", err)
	}

	// some logging config params
	config.LogFormat = luxlog.Colors
	config.MaxSize = constants.MaxLogFileSize
	config.MaxFiles = constants.MaxNumOfLogFiles
	config.MaxAge = constants.RetainOldFiles

	// Register ux package as internal so caller tracking shows actual source, not the wrapper
	luxlog.RegisterInternalPackages("github.com/luxfi/cli/pkg/ux")

	factory := luxlog.NewFactoryWithConfig(config)
	log, err := factory.Make("lux")
	if err != nil {
		factory.Close()
		return nil, fmt.Errorf("failed setting up logging, exiting: %w", err)
	}
	// Store factory globally so we can adjust levels later
	logFactory = factory
	// create the user facing logger as a global var
	// User output goes to stdout, logs go to stderr
	ux.NewUserLog(log, os.Stdout)
	return log, nil
}

// initConfig reads in config file and ENV variables if set.
// Priority: flags > env vars > config file > defaults
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Search for config in ~/.lux/ directory
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)
		luxDir := filepath.Join(home, constants.BaseDirName) // ~/.lux/
		viper.AddConfigPath(luxDir)
		viper.SetConfigType(constants.DefaultConfigFileType)
		viper.SetConfigName(constants.DefaultConfigFileName) // cli.json
	}

	// Bind environment variables for binary paths
	// LUX_NODE_PATH -> node-path, etc.
	_ = viper.BindEnv(constants.ConfigNodePath, constants.EnvNodePath)
	_ = viper.BindEnv(constants.ConfigNetrunnerPath, constants.EnvNetrunnerPath)
	_ = viper.BindEnv(constants.ConfigEVMPath, constants.EnvEVMPath)
	_ = viper.BindEnv(constants.ConfigPluginsDir, constants.EnvPluginsDir)

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		app.Log.Debug("using config file", "config-file", viper.ConfigFileUsed())

		// Read skip-update-check from config file if not already set by flag
		if !skipCheck && viper.IsSet(constants.SkipUpdateFlag) {
			skipCheck = viper.GetBool(constants.SkipUpdateFlag)
		}
	}
	// No config file is normal - most users don't have one, so we silently continue
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	app = application.New()
	rootCmd := NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err)
		os.Exit(1)
	}
}
