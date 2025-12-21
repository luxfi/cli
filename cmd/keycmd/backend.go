// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package keycmd

import (
	"fmt"

	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/key"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

func newBackendCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backend",
		Short: "Manage key storage backends",
		Long: `Manage key storage backends for cryptographic keys.

Available backends:
  software       - Encrypted file storage (AES-256-GCM + Argon2id)
  keychain       - macOS Keychain with optional TouchID
  secret-service - Linux Secret Service (GNOME Keyring, KWallet)
  yubikey        - Yubikey hardware token
  zymbit         - Zymbit HSM (Raspberry Pi)
  walletconnect  - Remote signing via mobile wallet
  ledger         - Ledger hardware wallet
  env            - Environment variable storage

Examples:
  lux key backend list          # List available backends
  lux key backend set keychain  # Set default backend
  lux key backend info          # Show current backend info`,
		RunE: cobrautils.CommandSuiteUsage,
	}

	cmd.AddCommand(newBackendListCmd())
	cmd.AddCommand(newBackendSetCmd())
	cmd.AddCommand(newBackendInfoCmd())

	return cmd
}

func newBackendListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available key backends",
		Long: `List all key storage backends and their availability status.

Backends marked as 'available' can be used on this system.
Some backends require specific hardware or services to be present.`,
		Args: cobra.NoArgs,
		RunE: runBackendList,
	}
}

func runBackendList(_ *cobra.Command, _ []string) error {
	backends := key.ListAvailableBackends()

	if len(backends) == 0 {
		ux.Logger.PrintToUser("No backends available.")
		return nil
	}

	// Get default backend for comparison
	defaultBackend, _ := key.GetDefaultBackend()
	var defaultType key.BackendType
	if defaultBackend != nil {
		defaultType = defaultBackend.Type()
	}

	ux.Logger.PrintToUser("Available key backends:")
	ux.Logger.PrintToUser("")

	for _, b := range backends {
		status := "available"
		if b.Type() == defaultType {
			status = "default"
		}

		features := ""
		if b.RequiresPassword() {
			features += " [password]"
		}
		if b.RequiresHardware() {
			features += " [hardware]"
		}
		if b.SupportsRemoteSigning() {
			features += " [remote]"
		}

		ux.Logger.PrintToUser("  %-16s  %-24s  %s%s", b.Type(), b.Name(), status, features)
	}

	ux.Logger.PrintToUser("")
	return nil
}

func newBackendSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <type>",
		Short: "Set the default key backend",
		Long: `Set the default key storage backend.

The default backend is used when creating new keys.
Existing keys remain in their original backend.

Valid backend types:
  software, keychain, secret-service, yubikey, zymbit, walletconnect, ledger, env`,
		Args: cobra.ExactArgs(1),
		RunE: runBackendSet,
	}
}

func runBackendSet(_ *cobra.Command, args []string) error {
	backendType := key.BackendType(args[0])

	// Verify backend exists and is available
	backend, err := key.GetBackend(backendType)
	if err != nil {
		return fmt.Errorf("backend '%s' not available: %w", backendType, err)
	}

	// Set as default
	if err := key.SetDefaultBackend(backendType); err != nil {
		return fmt.Errorf("failed to set default backend: %w", err)
	}

	ux.Logger.PrintToUser("Default backend set to '%s' (%s).", backendType, backend.Name())
	return nil
}

func newBackendInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Show current backend information",
		Long:  `Display detailed information about the current default key storage backend.`,
		Args:  cobra.NoArgs,
		RunE:  runBackendInfo,
	}
}

func runBackendInfo(_ *cobra.Command, _ []string) error {
	backend, err := key.GetDefaultBackend()
	if err != nil {
		return fmt.Errorf("no backend available: %w", err)
	}

	ux.Logger.PrintToUser("Current Key Backend")
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("  Type:             %s", backend.Type())
	ux.Logger.PrintToUser("  Name:             %s", backend.Name())
	ux.Logger.PrintToUser("  Available:        %t", backend.Available())
	ux.Logger.PrintToUser("  Requires Password: %t", backend.RequiresPassword())
	ux.Logger.PrintToUser("  Requires Hardware: %t", backend.RequiresHardware())
	ux.Logger.PrintToUser("  Remote Signing:   %t", backend.SupportsRemoteSigning())
	ux.Logger.PrintToUser("")

	return nil
}
