// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// docgen renders the CLI reference straight from the live cobra tree, so the
// docs never drift from the binary. Prose comes from each command's Long field.
//
//	go run ./tools/docgen out.md          # one aggregated Markdown file
//	go run ./tools/docgen path/to/dir/    # one MDX page per command suite + meta.json
//
// (build standalone: GOWORK=off go run ./tools/docgen ...)
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/luxfi/cli/cmd"
	"github.com/spf13/cobra"
)

func main() {
	out := "cmd/commands.md"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	root := cmd.NewRootCmd()
	if strings.HasSuffix(out, "/") || filepath.Ext(out) == "" {
		if err := writeMDXDir(root, strings.TrimRight(out, "/")); err != nil {
			fail(err)
		}
		return
	}
	var b strings.Builder
	b.WriteString("# Lux CLI reference\n\n_Generated from the command tree by `make docs` — do not edit by hand._\n\n")
	for _, c := range root.Commands() {
		render(c, &b, false)
	}
	if err := os.WriteFile(out, []byte(b.String()), 0o644); err != nil {
		fail(err)
	}
	fmt.Println("docgen: wrote", out)
}

// writeMDXDir emits one MDX page per top-level command suite plus meta.json,
// matching the docs site's content/docs/cli/commands/ layout.
func writeMDXDir(root *cobra.Command, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	var order []string
	for _, c := range root.Commands() {
		if c.Hidden || c.Name() == "help" || !c.IsAvailableCommand() {
			continue
		}
		var b strings.Builder
		fmt.Fprintf(&b, "---\ntitle: lux %s\ndescription: %s\n---\n\n", c.Name(), oneline(c.Short))
		fmt.Fprintf(&b, "{/* Generated from the command tree — edit the command's Long/Short, not this file. */}\n\n")
		render(c, &b, true)
		if err := os.WriteFile(filepath.Join(dir, c.Name()+".mdx"), []byte(b.String()), 0o644); err != nil {
			return err
		}
		order = append(order, c.Name())
	}
	meta := `{"pages":["` + strings.Join(order, `","`) + `"]}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, "meta.json"), []byte(meta), 0o644); err != nil {
		return err
	}
	fmt.Printf("docgen: wrote %d suites + meta.json to %s\n", len(order), dir)
	return nil
}

func render(c *cobra.Command, b *strings.Builder, mdx bool) {
	if c.Hidden || c.Name() == "help" {
		return
	}
	depth := strings.Count(c.CommandPath(), " ")
	hashes := strings.Repeat("#", min(depth+1, 6))
	if mdx {
		fmt.Fprintf(b, "%s %s\n\n", hashes, c.CommandPath())
	} else {
		fmt.Fprintf(b, "<a id=%q></a>\n%s %s\n\n", strings.ReplaceAll(c.CommandPath(), " ", "-"), hashes, c.CommandPath())
	}
	if desc := c.Long; desc != "" {
		b.WriteString(prose(desc, mdx) + "\n\n")
	} else if c.Short != "" {
		b.WriteString(prose(c.Short, mdx) + "\n\n")
	}
	if c.Runnable() {
		fmt.Fprintf(b, "**Usage:**\n\n```bash\n%s\n```\n\n", c.UseLine())
	}
	if f := c.LocalFlags(); f != nil && f.HasAvailableFlags() {
		fmt.Fprintf(b, "**Flags:**\n\n```\n%s```\n\n", f.FlagUsages())
	}
	for _, sub := range c.Commands() {
		render(sub, b, mdx)
	}
}

// prose makes command text safe to drop into MDX: outside code fences, `<` and
// `{` start JSX/expressions, so escape them. Fenced code is left untouched.
func prose(s string, mdx bool) string {
	if !mdx {
		return s
	}
	lines := strings.Split(s, "\n")
	fenced := false
	for i, ln := range lines {
		if strings.HasPrefix(strings.TrimSpace(ln), "```") {
			fenced = !fenced
			continue
		}
		if fenced {
			continue
		}
		ln = strings.ReplaceAll(ln, "<", "&lt;")
		ln = strings.ReplaceAll(ln, "{", "&#123;")
		lines[i] = ln
	}
	return strings.Join(lines, "\n")
}

func oneline(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, `"`, `'`)
	return strings.ReplaceAll(s, "<", "&lt;") // frontmatter description → MDX-safe
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "docgen:", err)
	os.Exit(1)
}
