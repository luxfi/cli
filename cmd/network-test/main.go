// Simple test program for network command
package main

import (
	"os"
	
	"github.com/luxfi/cli/v2/v2/cmd/networkcmd"
	"github.com/luxfi/cli/v2/v2/pkg/application"
)

func main() {
	app := application.New()
	cmd := networkcmd.NewCmd(app)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}