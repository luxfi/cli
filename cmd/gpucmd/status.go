// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package gpucmd

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var printJSON bool

// GPUStatus represents the GPU status information.
type GPUStatus struct {
	Available    bool   `json:"available"`
	Backend      string `json:"backend"`
	Platform     string `json:"platform"`
	Architecture string `json:"architecture"`
	CGOEnabled   bool   `json:"cgo_enabled"`
	Features     struct {
		NTTAcceleration bool `json:"ntt_acceleration"`
		FHEAcceleration bool `json:"fhe_acceleration"`
	} `json:"features"`
	DefaultConfig struct {
		Enabled     bool   `json:"enabled"`
		Backend     string `json:"backend"`
		DeviceIndex int    `json:"device_index"`
		LogLevel    string `json:"log_level"`
	} `json:"default_config"`
}

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show GPU acceleration status",
		Long: `Show the current GPU acceleration status including:
  - GPU availability on this system
  - Active backend (Metal, CUDA, or CPU)
  - Platform and architecture information
  - Available GPU-accelerated features
  - Default configuration settings`,
		RunE: statusCmd,
	}

	cmd.Flags().BoolVar(&printJSON, "json", false, "output status in JSON format")
	return cmd
}

func statusCmd(_ *cobra.Command, _ []string) error {
	status := getGPUStatus()

	if printJSON {
		jsonBytes, err := json.MarshalIndent(status, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal status: %w", err)
		}
		fmt.Println(string(jsonBytes))
		return nil
	}

	// Print table format
	printStatusTable(status)
	return nil
}

func getGPUStatus() GPUStatus {
	status := GPUStatus{
		Platform:     runtime.GOOS,
		Architecture: runtime.GOARCH,
		CGOEnabled:   isCGOEnabled(),
	}

	// Determine expected backend based on platform
	switch runtime.GOOS {
	case "darwin":
		status.Backend = "Metal"
		status.Available = status.CGOEnabled // Metal requires CGO
	case "linux":
		status.Backend = "CUDA"
		status.Available = status.CGOEnabled // CUDA requires CGO
	default:
		status.Backend = "CPU"
		status.Available = true // CPU always available
	}

	// If CGO is disabled, fall back to CPU
	if !status.CGOEnabled {
		status.Backend = "CPU (CGO disabled)"
		status.Available = true
	}

	// Set feature availability (requires CGO for GPU acceleration)
	status.Features.NTTAcceleration = status.CGOEnabled
	status.Features.FHEAcceleration = status.CGOEnabled

	// Default configuration
	status.DefaultConfig.Enabled = true
	status.DefaultConfig.Backend = "auto"
	status.DefaultConfig.DeviceIndex = 0
	status.DefaultConfig.LogLevel = "warn"

	return status
}

func printStatusTable(status GPUStatus) {
	fmt.Println("GPU Acceleration Status")
	fmt.Println("=======================")
	fmt.Println()

	// System info
	fmt.Println("System Information:")
	fmt.Printf("  Platform:     %s\n", status.Platform)
	fmt.Printf("  Architecture: %s\n", status.Architecture)
	fmt.Printf("  CGO Enabled:  %v\n", status.CGOEnabled)
	fmt.Println()

	// GPU status
	availableStr := "Yes"
	if !status.Available {
		availableStr = "No"
	}
	fmt.Println("GPU Status:")
	fmt.Printf("  Available:    %s\n", availableStr)
	fmt.Printf("  Backend:      %s\n", status.Backend)
	fmt.Println()

	// Features
	fmt.Println("Accelerated Features:")
	fmt.Printf("  NTT (Ringtail consensus): %v\n", status.Features.NTTAcceleration)
	fmt.Printf("  FHE (ThresholdVM):        %v\n", status.Features.FHEAcceleration)
	fmt.Println()

	// Default config
	fmt.Println("Default Configuration:")
	fmt.Printf("  Enabled:      %v\n", status.DefaultConfig.Enabled)
	fmt.Printf("  Backend:      %s\n", status.DefaultConfig.Backend)
	fmt.Printf("  Device Index: %d\n", status.DefaultConfig.DeviceIndex)
	fmt.Printf("  Log Level:    %s\n", status.DefaultConfig.LogLevel)
	fmt.Println()

	// Hints
	if !status.CGOEnabled {
		fmt.Println("Note: GPU acceleration requires CGO. Build with CGO_ENABLED=1 for full GPU support.")
	} else if status.Platform == "darwin" {
		fmt.Println("Note: Metal GPU acceleration is available on Apple Silicon.")
	} else if status.Platform == "linux" {
		fmt.Println("Note: CUDA GPU acceleration requires NVIDIA GPU with CUDA toolkit.")
	}
}

// isCGOEnabled returns whether CGO was enabled at build time.
// This is determined at compile time via build tags.
func isCGOEnabled() bool {
	return cgoEnabled
}
