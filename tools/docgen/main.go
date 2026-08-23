// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// docgen renders the CLI command reference straight from the live cobra tree,
// so docs.lux.network never drifts from the binary. Prose comes from each
// command's Long field — write docs in the command, generate the page here.
//
//	go run ./tools/docgen [out.md]      # default: cmd/commands.md
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/luxfi/cli/cmd"
	"github.com/spf13/cobra"
)

func main() {
	out := "cmd/commands.md"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	var b strings.Builder
	b.WriteString("# Lux CLI reference\n\n")
	b.WriteString("_Generated from the command tree by `make docs` — do not edit by hand._\n\n")
	walk(cmd.NewRootCmd(), &b)
	if err := os.WriteFile(out, []byte(b.String()), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "docgen:", err)
		os.Exit(1)
	}
	fmt.Println("docgen: wrote", out)
}

func walk(c *cobra.Command, b *strings.Builder) {
	if c.Hidden || c.Name() == "help" {
		return
	}
	path := c.CommandPath()
	fmt.Fprintf(b, "<a id=%q></a>\n## %s\n\n", strings.ReplaceAll(path, " ", "-"), path)
	if c.Long != "" {
		b.WriteString(c.Long + "\n\n")
	} else if c.Short != "" {
		b.WriteString(c.Short + "\n\n")
	}
	if c.Runnable() {
		fmt.Fprintf(b, "**Usage:**\n```bash\n%s\n```\n\n", c.UseLine())
	}
	if f := c.LocalFlags(); f != nil && f.HasAvailableFlags() {
		fmt.Fprintf(b, "**Flags:**\n```\n%s```\n\n", f.FlagUsages())
	}
	for _, sub := range c.Commands() {
		walk(sub, b)
	}
}
