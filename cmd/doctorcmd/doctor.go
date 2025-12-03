// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package doctorcmd

import (
	"github.com/luxfi/cli/pkg/application"
	"github.com/spf13/cobra"
)

var (
	app     *application.Lux
	fixMode bool
)

// NewCmd returns a new cobra.Command for doctor operations
func NewCmd(injectedApp *application.Lux) *cobra.Command {
	app = injectedApp
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check development environment setup",
		Long: `The doctor command checks your development environment for Lux CLI compatibility.

It verifies:
  - Go version compatibility
  - Docker availability (if needed)
  - Lux node binary availability and version
  - Network connectivity to Lux endpoints
  - Disk space for state storage

Use --fix to attempt automatic remediation of detected issues.`,
		RunE: runDoctor,
	}

	cmd.Flags().BoolVar(&fixMode, "fix", false, "attempt to automatically fix detected issues")

	return cmd
}

func runDoctor(cmd *cobra.Command, args []string) error {
	doctor := NewDoctor(app, fixMode)
	return doctor.Run()
}
