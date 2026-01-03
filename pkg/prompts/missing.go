// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package prompts

import (
	"errors"
	"fmt"
	"strings"
)

// MissingOpt describes a required option that was not provided.
type MissingOpt struct {
	Flag    string // e.g., "--chain-id"
	Env     string // e.g., "LUX_CHAIN_ID" (optional)
	Prompt  string // e.g., "EVM chain ID" - used for interactive prompts
	Note    string // optional additional context
	Default string // optional default value hint
}

// MissingError creates a clear, actionable error listing all missing options.
// The error message follows UNIX conventions and guides users to the right flags.
func MissingError(cmd string, missing []MissingOpt) error {
	if len(missing) == 0 {
		return nil
	}

	var b strings.Builder
	b.WriteString("missing required options:\n")
	for _, m := range missing {
		if m.Env != "" {
			fmt.Fprintf(&b, "  %s (or %s)", m.Flag, m.Env)
		} else {
			fmt.Fprintf(&b, "  %s", m.Flag)
		}
		if m.Note != "" {
			fmt.Fprintf(&b, " - %s", m.Note)
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "\nrun '%s --help' to see all options", cmd)
	if IsInteractive() {
		b.WriteString("\nor run on a TTY to be prompted interactively")
	}
	return errors.New(b.String())
}

// PromptOrFail handles the common pattern of prompting for missing values.
// In interactive mode, prompts for each missing option.
// In non-interactive mode, returns an error listing all missing options.
//
// Usage:
//
//	missing := []prompts.MissingOpt{}
//	if chainID == "" {
//	    missing = append(missing, prompts.MissingOpt{Flag: "--chain-id", Env: "LUX_CHAIN_ID", Prompt: "EVM chain ID"})
//	}
//	if err := prompts.PromptOrFail("lux chain create", missing, func(m MissingOpt) (string, error) {
//	    return app.Prompt.CaptureString(m.Prompt)
//	}, &chainID); err != nil {
//	    return err
//	}
func PromptOrFail(cmd string, missing []MissingOpt, promptFn func(MissingOpt) (string, error), targets ...*string) error {
	if len(missing) == 0 {
		return nil
	}

	if len(missing) != len(targets) {
		return fmt.Errorf("internal error: %d missing options but %d targets", len(missing), len(targets))
	}

	// Non-interactive: fail with complete error message
	if !IsInteractive() {
		return MissingError(cmd, missing)
	}

	// Interactive: prompt for each missing value
	for i, m := range missing {
		val, err := promptFn(m)
		if err != nil {
			return fmt.Errorf("failed to get %s: %w", m.Flag, err)
		}
		*targets[i] = val
	}
	return nil
}

// Validator holds options being collected and tracks missing ones.
// Use this for clean, declarative option handling.
type Validator struct {
	cmd     string
	missing []MissingOpt
	values  []*string
}

// NewValidator creates a validator for a command.
func NewValidator(cmd string) *Validator {
	return &Validator{cmd: cmd}
}

// Require marks a value as required. If empty, adds to missing list.
func (v *Validator) Require(target *string, opt MissingOpt) *Validator {
	if *target == "" {
		v.missing = append(v.missing, opt)
		v.values = append(v.values, target)
	}
	return v
}

// RequireWithDefault marks a value as required with a default.
// Uses the default if empty and non-interactive, otherwise prompts.
func (v *Validator) RequireWithDefault(target *string, opt MissingOpt, defaultVal string) *Validator {
	if *target == "" {
		if !IsInteractive() {
			*target = defaultVal
		} else {
			opt.Default = defaultVal
			v.missing = append(v.missing, opt)
			v.values = append(v.values, target)
		}
	}
	return v
}

// Optional sets a default if the value is empty (no prompting).
func (v *Validator) Optional(target *string, defaultVal string) *Validator {
	if *target == "" {
		*target = defaultVal
	}
	return v
}

// Missing returns the list of missing options.
func (v *Validator) Missing() []MissingOpt {
	return v.missing
}

// HasMissing returns true if any required options are missing.
func (v *Validator) HasMissing() bool {
	return len(v.missing) > 0
}

// Resolve prompts for missing values (interactive) or returns error (non-interactive).
func (v *Validator) Resolve(promptFn func(MissingOpt) (string, error)) error {
	if !v.HasMissing() {
		return nil
	}

	if !IsInteractive() {
		return MissingError(v.cmd, v.missing)
	}

	for i, m := range v.missing {
		val, err := promptFn(m)
		if err != nil {
			return fmt.Errorf("failed to get %s: %w", m.Flag, err)
		}
		*v.values[i] = val
	}
	return nil
}
