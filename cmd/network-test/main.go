// Simple test program for network command
package main

import (
	"os"

	"github.com/luxfi/cli/cmd/networkcmd"
	"github.com/luxfi/cli/pkg/application"
)

func main() {
	app := application.New()
	cmd := networkcmd.NewCmd(app)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
