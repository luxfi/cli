// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package nodecmd

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

const (
	defaultLuxdVersion = "v1.13.3"
	luxdDownloadURL = "https://github.com/luxfi/node/releases/download/%s/luxd-linux-%s-%s.tar.gz"
)

type versionFlags struct {
	version string
	force   bool
}

func newVersionCmd() *cobra.Command {
	flags := &versionFlags{}
	
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Manage luxd node versions",
		Long:  "Download and manage different versions of the luxd node binary",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// Add subcommands
	cmd.AddCommand(newInstallCmd(flags))
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newUseCmd())
	
	return cmd
}

func newInstallCmd(flags *versionFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install [version]",
		Short: "Install a specific version of luxd",
		Long:  "Download and install a specific version of the luxd node binary",
		Example: `  # Install default version
  lux node version install

  # Install specific version
  lux node version install v1.13.3

  # Force reinstall
  lux node version install v1.13.3 --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			version := defaultLuxdVersion
			if len(args) > 0 {
				version = args[0]
			}
			if !strings.HasPrefix(version, "v") {
				version = "v" + version
			}
			return installLuxd(version, flags.force)
		},
	}
	
	cmd.Flags().BoolVar(&flags.force, "force", false, "Force reinstall even if version exists")
	
	return cmd
}

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List installed luxd versions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return listVersions()
		},
	}
}

func newUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use [version]",
		Short: "Switch to a specific luxd version",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			version := args[0]
			if !strings.HasPrefix(version, "v") {
				version = "v" + version
			}
			return useVersion(version)
		},
	}
}

func installLuxd(version string, force bool) error {
	binDir := filepath.Join(app.GetBaseDir(), "bin")
	versionDir := filepath.Join(binDir, "versions", version)
	luxdPath := filepath.Join(versionDir, "luxd")
	
	// Check if already installed
	if _, err := os.Stat(luxdPath); err == nil && !force {
		ux.Logger.PrintToUser("Version %s is already installed", version)
		return useVersion(version)
	}
	
	// Create directories
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		return fmt.Errorf("failed to create version directory: %w", err)
	}
	
	// Determine architecture
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "amd64"
	} else if arch == "arm64" {
		arch = "arm64"
	}
	
	// Download URL
	url := fmt.Sprintf(luxdDownloadURL, version, runtime.GOOS, arch)
	
	ux.Logger.PrintToUser("Downloading luxd %s from %s...", version, url)
	
	// Download file
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}
	
	// Create temp file
	tmpFile, err := os.CreateTemp("", "luxd-*.tar.gz")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	
	// Download to temp file
	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	tmpFile.Close()
	
	// Extract tar.gz
	ux.Logger.PrintToUser("Extracting luxd...")
	if err := extractTarGz(tmpFile.Name(), versionDir); err != nil {
		return fmt.Errorf("failed to extract: %w", err)
	}
	
	// Make executable
	if err := os.Chmod(luxdPath, 0755); err != nil {
		return fmt.Errorf("failed to make executable: %w", err)
	}
	
	ux.Logger.PrintToUser("Successfully installed luxd %s", version)
	
	// Set as current version
	return useVersion(version)
}

func extractTarGz(src, dst string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	
	tr := tar.NewReader(gz)
	
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		
		target := filepath.Join(dst, header.Name)
		
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			// Extract only the luxd binary
			if filepath.Base(header.Name) == "luxd" {
				outFile, err := os.Create(filepath.Join(dst, "luxd"))
				if err != nil {
					return err
				}
				if _, err := io.Copy(outFile, tr); err != nil {
					outFile.Close()
					return err
				}
				outFile.Close()
			}
		}
	}
	
	return nil
}

func listVersions() error {
	binDir := filepath.Join(app.GetBaseDir(), "bin")
	versionsDir := filepath.Join(binDir, "versions")
	
	// Create if not exists
	if err := os.MkdirAll(versionsDir, 0755); err != nil {
		return err
	}
	
	// List versions
	entries, err := os.ReadDir(versionsDir)
	if err != nil {
		return fmt.Errorf("failed to list versions: %w", err)
	}
	
	// Get current version
	currentPath, _ := os.Readlink(filepath.Join(binDir, "luxd"))
	currentVersion := ""
	if currentPath != "" {
		currentVersion = filepath.Base(filepath.Dir(currentPath))
	}
	
	ux.Logger.PrintToUser("Installed luxd versions:")
	for _, entry := range entries {
		if entry.IsDir() {
			version := entry.Name()
			if version == currentVersion {
				ux.Logger.PrintToUser("  * %s (current)", version)
			} else {
				ux.Logger.PrintToUser("    %s", version)
			}
		}
	}
	
	return nil
}

func useVersion(version string) error {
	binDir := filepath.Join(app.GetBaseDir(), "bin")
	versionDir := filepath.Join(binDir, "versions", version)
	luxdPath := filepath.Join(versionDir, "luxd")
	linkPath := filepath.Join(binDir, "luxd")
	
	// Check if version exists
	if _, err := os.Stat(luxdPath); err != nil {
		return fmt.Errorf("version %s is not installed. Run 'lux node version install %s' first", version, version)
	}
	
	// Remove existing symlink
	os.Remove(linkPath)
	
	// Create new symlink
	if err := os.Symlink(luxdPath, linkPath); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}
	
	ux.Logger.PrintToUser("Now using luxd %s", version)
	return nil
}