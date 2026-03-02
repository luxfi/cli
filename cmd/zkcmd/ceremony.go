// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package zkcmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

// findCeremonyBinary locates the ceremony binary.
// Search order: PATH, $LUX_NODE_ROOT/build/ceremony, /usr/local/bin/ceremony.
func findCeremonyBinary() (string, error) {
	if p, err := exec.LookPath("ceremony"); err == nil {
		return p, nil
	}
	if root := os.Getenv("LUX_NODE_ROOT"); root != "" {
		p := filepath.Join(root, "build", "ceremony")
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	if _, err := os.Stat("/usr/local/bin/ceremony"); err == nil {
		return "/usr/local/bin/ceremony", nil
	}
	home, _ := os.UserHomeDir()
	gobin := filepath.Join(home, "go", "bin", "ceremony")
	if _, err := os.Stat(gobin); err == nil {
		return gobin, nil
	}
	return "", fmt.Errorf("ceremony binary not found\n\nBuild it with:\n  cd ~/work/lux/node && go build -o /usr/local/bin/ceremony ./cmd/ceremony/")
}

// runCeremony executes the ceremony binary with the given subcommand and args.
func runCeremony(subcmd string, args ...string) error {
	bin, err := findCeremonyBinary()
	if err != nil {
		return err
	}
	cmdArgs := append([]string{subcmd}, args...)
	cmd := exec.Command(bin, cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func newCeremonyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ceremony",
		Short: "Powers-of-tau ceremony management",
		Long: `Manage powers-of-tau ceremonies for generating trusted SRS
(Structured Reference Strings) used in Groth16 and PLONK proof systems.

The ceremony requires the 'ceremony' binary from the Lux node repo.
If not found, build it with:
  cd ~/work/lux/node && go build -o /usr/local/bin/ceremony ./cmd/ceremony/`,
	}

	cmd.AddCommand(newCeremonyInitCmd())
	cmd.AddCommand(newCeremonyContributeCmd())
	cmd.AddCommand(newCeremonyVerifyCmd())
	cmd.AddCommand(newCeremonyExportCmd())
	cmd.AddCommand(newCeremonyStatusCmd())

	return cmd
}

func newCeremonyInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new powers-of-tau ceremony",
		Long: `Create a new ceremony state file with initial powers of the BN254
generators. This is the starting point before any contributions.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			circuit, _ := cmd.Flags().GetString("circuit")
			participants, _ := cmd.Flags().GetInt("participants")
			power, _ := cmd.Flags().GetInt("power")
			output, _ := cmd.Flags().GetString("output")
			return runCeremony("init",
				"--circuit", circuit,
				"--participants", fmt.Sprintf("%d", participants),
				"--power", fmt.Sprintf("%d", power),
				"--output", output,
			)
		},
	}

	cmd.Flags().String("circuit", "", "Circuit name (required)")
	cmd.Flags().Int("participants", 3, "Expected number of participants")
	cmd.Flags().Int("power", 20, "Power of 2 for constraint count (2^power)")
	cmd.Flags().String("output", "", "Output file path (required)")
	cmd.MarkFlagRequired("circuit")
	cmd.MarkFlagRequired("output")

	return cmd
}

func newCeremonyContributeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "contribute",
		Short: "Add randomness to an existing ceremony",
		Long: `Apply a random contribution to the ceremony state. Generates
cryptographically secure random scalars (tau, alpha, beta) and mixes them
into the SRS. The random values are zeroed from memory after use.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			input, _ := cmd.Flags().GetString("input")
			output, _ := cmd.Flags().GetString("output")
			participant, _ := cmd.Flags().GetString("participant")
			return runCeremony("contribute",
				"--input", input,
				"--output", output,
				"--participant", participant,
			)
		},
	}

	cmd.Flags().String("input", "", "Input ceremony file (required)")
	cmd.Flags().String("output", "", "Output ceremony file (required)")
	cmd.Flags().String("participant", "", "Participant name (required)")
	cmd.MarkFlagRequired("input")
	cmd.MarkFlagRequired("output")
	cmd.MarkFlagRequired("participant")

	return cmd
}

func newCeremonyVerifyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify a ceremony's integrity",
		Long: `Check the consistency of a ceremony state file by verifying:
- TauG1/TauG2 form consistent geometric sequences (pairing checks)
- AlphaG1/BetaG1 use the same tau ratio
- BetaG1 and BetaG2 encode the same beta scalar
- Contribution hash chain integrity
- No points at infinity`,
		RunE: func(cmd *cobra.Command, args []string) error {
			input, _ := cmd.Flags().GetString("input")
			return runCeremony("verify", "--input", input)
		},
	}

	cmd.Flags().String("input", "", "Ceremony file to verify (required)")
	cmd.MarkFlagRequired("input")

	return cmd
}

func newCeremonyExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export the final SRS binary",
		Long: `Export the SRS (Structured Reference String) from a completed and
verified ceremony. The ceremony is verified before export. Output is
uncompressed binary (G1: 64 bytes, G2: 128 bytes per point).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			input, _ := cmd.Flags().GetString("input")
			output, _ := cmd.Flags().GetString("output")
			return runCeremony("export",
				"--input", input,
				"--output", output,
			)
		},
	}

	cmd.Flags().String("input", "", "Ceremony file to export (required)")
	cmd.Flags().String("output", "", "Output SRS binary file (required)")
	cmd.MarkFlagRequired("input")
	cmd.MarkFlagRequired("output")

	return cmd
}

// ceremonyEnvelope mirrors the ceremony binary's state file format for status display.
type ceremonyEnvelope struct {
	State     json.RawMessage `json:"state"`
	Integrity string          `json:"integrity"`
}

type ceremonyStatus struct {
	Circuit        string `json:"circuit"`
	NumConstraints int    `json:"numConstraints"`
	PowersNeeded   int    `json:"powersNeeded"`
	Participants   int    `json:"participants"`
	Contributions  []struct {
		Participant string `json:"participant"`
		Hash        string `json:"hash"`
		Timestamp   string `json:"timestamp"`
	} `json:"contributions"`
}

func newCeremonyStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show ceremony state (participants, hashes)",
		Long:  `Display the current state of a ceremony file including circuit info, contributions, and participant hashes.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			input, _ := cmd.Flags().GetString("input")
			return showCeremonyStatus(input)
		},
	}

	cmd.Flags().String("input", "", "Ceremony file to inspect (required)")
	cmd.MarkFlagRequired("input")

	return cmd
}

func showCeremonyStatus(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read ceremony file: %w", err)
	}

	var env ceremonyEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return fmt.Errorf("parse ceremony file: %w", err)
	}

	var status ceremonyStatus
	if err := json.Unmarshal(env.State, &status); err != nil {
		return fmt.Errorf("parse ceremony state: %w", err)
	}

	fmt.Printf("Ceremony: %s\n", status.Circuit)
	fmt.Printf("  Constraints:    %d\n", status.NumConstraints)
	fmt.Printf("  Powers needed:  %d\n", status.PowersNeeded)
	fmt.Printf("  Expected:       %d participants\n", status.Participants)
	fmt.Printf("  Contributions:  %d\n", len(status.Contributions))

	if len(status.Contributions) > 0 {
		fmt.Println()
		for i, c := range status.Contributions {
			hash := c.Hash
			if len(hash) > 16 {
				hash = hash[:16]
			}
			fmt.Printf("  [%d] %s at %s (hash: %s...)\n", i+1, c.Participant, c.Timestamp, hash)
		}
	}

	fmt.Printf("\nIntegrity: %s\n", env.Integrity)
	return nil
}
