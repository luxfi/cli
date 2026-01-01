// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

/*
Package prompts provides user interaction primitives following UNIX conventions.

# Design Philosophy

The CLI follows standard UNIX behavior for interactive mode:

  - If stdin is a TTY → prompting is allowed for missing values
  - If stdin is not a TTY → never prompt (piped/scripted)
  - Explicit overrides (LUX_NON_INTERACTIVE, CI) force non-interactive

This gives predictable scripting behavior without quirky mode toggles.

# Mode Detection

Non-interactive mode is enabled when ANY of these is true:

  - LUX_NON_INTERACTIVE=1/true/yes/on environment variable
  - CI=1/true environment variable (GitHub Actions, GitLab CI, etc.)
  - stdin is not a TTY (piped/redirected/scripted)

Interactive mode is enabled otherwise (stdin is TTY, no overrides).

# Option Precedence

Values are resolved in this order (UNIX-standard):

 1. Flags (--chain-id=12345)
 2. Environment variables (LUX_CHAIN_ID=12345)
 3. Config file (~/.lux/cli.json)
 4. Defaults
 5. Prompts (only if interactive/TTY)

Prompts should only fill values that remain empty after 1-4.

# Usage Pattern: Validator

The recommended pattern uses Validator for clean, declarative option handling:

	func createChain(cmd *cobra.Command, args []string) error {
	    chainName := args[0]

	    // 1. Resolve from flags/env/config (cobra already did this)

	    // 2. Validate and collect missing required options
	    v := prompts.NewValidator("lux chain create")
	    v.RequireWithDefault(&chainID, prompts.MissingOpt{
	        Flag:   "--chain-id",
	        Env:    "LUX_CHAIN_ID",
	        Prompt: "EVM chain ID",
	    }, "200200")
	    v.Require(&tokenName, prompts.MissingOpt{
	        Flag:   "--token-name",
	        Prompt: "Native token name",
	    })

	    // 3. Prompt for missing (interactive) or fail with error (non-interactive)
	    if err := v.Resolve(func(m prompts.MissingOpt) (string, error) {
	        prompt := m.Prompt
	        if m.Default != "" {
	            prompt = fmt.Sprintf("%s (default: %s)", m.Prompt, m.Default)
	        }
	        return app.Prompt.CaptureString(prompt)
	    }); err != nil {
	        return err
	    }

	    // 4. All values are now populated - proceed with command
	    ...
	}

# Error Messages

When non-interactive and required values are missing, errors look like:

	missing required options:
	  --chain-id (or LUX_CHAIN_ID)
	  --token-symbol

	run 'lux chain create --help' to see all options

# Adding New Commands

When adding commands that require user input:

 1. Accept flags for all promptable values
 2. Use Validator pattern to collect missing options
 3. Call Resolve() to prompt or fail
 4. Never call prompt methods directly without validation

# Operations Requiring Interaction

For operations that absolutely require interaction (ledger signing):

	prompts.MustInteractive("ledger signing")

This panics if called in non-interactive mode, preventing silent failures.

# Checking Mode

	if prompts.IsInteractive() {
	    // can prompt for optional values
	}
*/
package prompts
