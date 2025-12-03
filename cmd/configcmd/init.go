// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package configcmd

import (
	"fmt"

	"github.com/luxfi/cli/pkg/globalconfig"
	"github.com/spf13/cobra"
)

var (
	initProject bool
	initForce   bool
)

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize configuration with smart defaults",
		Long: `Initialize a new configuration file with smart defaults based on your environment.

By default, creates a global configuration at ~/.lux/config.json.
Use --project to create a project-local configuration at .luxconfig.json.

The command auto-detects your environment (CI, Codespace, development, production)
and suggests optimal defaults.`,
		RunE: runInit,
	}

	cmd.Flags().BoolVar(&initProject, "project", false, "Create project-local config instead of global")
	cmd.Flags().BoolVar(&initForce, "force", false, "Overwrite existing config file")

	return cmd
}

func runInit(_ *cobra.Command, _ []string) error {
	// Get smart defaults based on environment
	smart := globalconfig.GetSmartDefaults()

	fmt.Printf("Detected environment: %s\n", smart.Environment)
	fmt.Printf("Suggested nodes: %d\n", smart.SuggestedNumNodes)
	fmt.Printf("Suggested instance: %s\n", smart.SuggestedInstance)

	if initProject {
		return initProjectConfig(smart)
	}
	return initGlobalConfig(smart)
}

func initGlobalConfig(smart *globalconfig.SmartDefaults) error {
	baseDir := app.GetBaseDir()

	// Check if config exists
	existing, err := globalconfig.LoadGlobalConfig(baseDir)
	if err != nil {
		return fmt.Errorf("failed to check existing config: %w", err)
	}
	if existing != nil && !initForce {
		return fmt.Errorf("config already exists at %s/config.json (use --force to overwrite)", baseDir)
	}

	// Create config with smart defaults
	config := globalconfig.DefaultGlobalConfig()
	config.Local.NumNodes = &smart.SuggestedNumNodes
	config.Node.DefaultInstanceType = smart.SuggestedInstance

	if err := globalconfig.SaveGlobalConfig(baseDir, &config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Created global config at %s/config.json\n", baseDir)
	return nil
}

func initProjectConfig(smart *globalconfig.SmartDefaults) error {
	// Check if project config exists
	existing, err := globalconfig.LoadProjectConfig()
	if err != nil {
		return fmt.Errorf("failed to check existing config: %w", err)
	}
	if existing != nil && !initForce {
		return fmt.Errorf("project config already exists (use --force to overwrite)")
	}

	// Create config with smart defaults
	config := &globalconfig.ProjectConfig{
		GlobalConfig: globalconfig.DefaultGlobalConfig(),
		ProjectName:  "",
	}
	config.Local.NumNodes = &smart.SuggestedNumNodes
	config.Node.DefaultInstanceType = smart.SuggestedInstance

	if err := globalconfig.SaveProjectConfig(config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println("Created project config at .luxconfig.json")
	return nil
}
