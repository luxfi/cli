// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package route

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/luxfi/constants"
)

// gone is the segment chains used to answer on, and served is the one they
// answer on now. Both are written in parts, so this file spells neither
// address whole and is not the first thing its own guard catches.
const (
	gone   = "bc"
	served = "chain"
)

// TestChainAddressesTheServedSegment states the wire contract outright: a
// composed address is under /v1/chain, and nowhere near the segment a current
// node no longer serves. This is the assertion that would have caught the 404.
func TestChainAddressesTheServedSegment(t *testing.T) {
	require := require.New(t)

	live := "/v1/" + served
	dead := "/v1/" + gone

	address := Chain("http://127.0.0.1:9650", "C") + "/rpc"

	require.Equal("http://127.0.0.1:9650"+live+"/C/rpc", address)
	require.Contains(address, live+"/")
	require.NotContains(address, dead)
}

// TestChainDerivesFromTheConstant pins Chain to the constant rather than to a
// copy of its current value, so renaming the segment renames the address.
func TestChainDerivesFromTheConstant(t *testing.T) {
	require := require.New(t)

	require.Equal(served, constants.ChainAliasPrefix, "the segment moved; this package should have moved with it")
	require.Equal(base+"/"+constants.ChainAliasPrefix+"/P", Chain("", "P"))
	require.Equal("http://n:9630"+base+"/"+constants.ChainAliasPrefix+"/C", Chain("http://n:9630", "C"))
}

// TestAliasReadsBackWhatChainWrites measures the two against each other, so
// neither can drift onto a segment the other does not use.
func TestAliasReadsBackWhatChainWrites(t *testing.T) {
	require := require.New(t)

	for _, alias := range []string{"P", "X", "C", "2G8mK7VCZX1dV8iPjkkTDMpYGZDCNLLVdTJVLmMsG5ZV7zKVmB"} {
		got, ok := Alias(Chain("http://127.0.0.1:9650", alias) + "/rpc")
		require.True(ok, "Chain wrote an address Alias does not recognise: %s", alias)
		require.Equal(alias, got)
	}

	// An address under the segment that is no longer served names no chain.
	_, ok := Alias("http://127.0.0.1:9650/v1/" + gone + "/C/rpc")
	require.False(ok, "the segment that is no longer served still reads as a chain")

	for _, notAChain := range []string{"", "http://127.0.0.1:9650/v1/info", Chain("http://127.0.0.1:9650", "")} {
		_, ok := Alias(notAChain)
		require.False(ok, "%q read as a chain address", notAChain)
	}
}

// TestChainAddressBuiltOnlyHere fails if any Go source in this module writes a
// chain address out by hand instead of calling [Chain].
//
// This is what stops the segment drifting back. The CLI had spelled it at 112
// sites across 34 files; when the node renamed the segment, every one of them
// went on addressing a name the router had already left behind, and nothing
// failed to compile. They simply answered 404.
//
// It reads string literals, so prose describing a route stays free to name it;
// only building one is the error.
func TestChainAddressBuiltOnlyHere(t *testing.T) {
	bad, err := scan(moduleRoot(t))
	require.NoError(t, err)
	require.Empty(t, bad, "a chain address is built in exactly one place, [Chain]:\n%s", strings.Join(bad, "\n"))
}

// TestGuardCatchesEachShape is the guard's own test. A guard nobody has seen
// fail is not known to work, so each shape it forbids is planted in a source
// file of its own and the guard is required to find it — and the same route in
// prose is required to pass, since prose is not an address.
func TestGuardCatchesEachShape(t *testing.T) {
	for _, shape := range forbidden() {
		t.Run(shape.frag, func(t *testing.T) {
			require := require.New(t)

			// Assembled, never written, so this file plants nothing it would
			// itself be caught for.
			planted := shape.frag + "/C/rpc"
			file := filepath.Join(t.TempDir(), "planted.go")

			require.NoError(os.WriteFile(file, []byte(
				"package p\n\nvar u = \""+planted+"\"\n"), 0o600))
			bad, err := scan(filepath.Dir(file))
			require.NoError(err)
			require.Len(bad, 1, "guard missed %q", planted)
			require.Contains(bad[0], shape.why)

			require.NoError(os.WriteFile(file, []byte(
				"package p\n\n// A chain answers at "+planted+".\nvar u = \"\"\n"), 0o600))
			bad, err = scan(filepath.Dir(file))
			require.NoError(err)
			require.Empty(bad, "guard flagged prose, which names no address")
		})
	}
}

// forbidden is every shape that builds a chain address without [Chain]: the
// segment that is no longer served, the one that is, and the hole left for it.
// The fragments are assembled, so this file spells none of them.
func forbidden() []struct{ frag, why string } {
	return []struct{ frag, why string }{
		{base + "/" + gone, "addresses a chain under a segment that is not served"},
		{base + "/" + constants.ChainAliasPrefix, "spells the chain segment; call route.Chain(uri, alias)"},
		{base + "/%s", "interpolates the chain segment; call route.Chain(uri, alias)"},
	}
}

// scan reports every string literal under root that builds a chain address.
func scan(root string) ([]string, error) {
	var bad []string
	shapes := forbidden()
	fset := token.NewFileSet()

	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "vendor", "testdata", "node_modules":
				return fs.SkipDir
			}
			return nil
		}
		if filepath.Ext(p) != ".go" {
			return nil
		}
		// A file that will not parse is the build's problem, not this test's.
		f, err := parser.ParseFile(fset, p, nil, 0)
		if err != nil {
			return nil
		}
		ast.Inspect(f, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			text, err := strconv.Unquote(lit.Value)
			if err != nil {
				return true
			}
			for _, shape := range shapes {
				if strings.Contains(text, shape.frag) {
					at := fset.Position(lit.Pos())
					rel, _ := filepath.Rel(root, at.Filename)
					bad = append(bad, fmt.Sprintf("%s:%d: %s", filepath.ToSlash(rel), at.Line, shape.why))
				}
			}
			return true
		})
		return nil
	})
	return bad, err
}

// moduleRoot walks up from the package directory to the directory holding
// go.mod, so the scan covers the module rather than this package.
func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, parent, dir, "no go.mod above the package directory")
		dir = parent
	}
}
