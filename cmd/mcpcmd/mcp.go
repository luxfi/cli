// Package mcpcmd implements `lux mcp` — an agent's door onto a running node.
//
// The node serves MCP over ZAP: frames on a socket, no HTTP listener, no
// request, no status code. An MCP client (Claude Code, an editor, an agent
// runtime) speaks the same protocol over stdin and stdout. Those are two
// renderings of ONE value — zapmcp.Frame marshals as the ZAP wire and as the
// JSON-RPC 2.0 message — so this command is a relay and not a translation: it
// reads a frame from one side and writes it to the other, and reads nothing
// inside it.
//
// That is why there is no HTTP anywhere in here, and why there is no tool list
// either. The tools are the node's typed ops, projected by the node, and a copy
// of them on this side would be one more thing to drift.
package mcpcmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	zapmcp "github.com/zap-proto/mcp"
	"github.com/zap-proto/zip"
)

// The JSON-RPC codes this relay can raise on its own. Everything else comes
// from the node and is passed through untouched.
const (
	parseError    = -32700
	internalError = -32603
)

func NewCmd() *cobra.Command {
	var at string
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Serve a node's tools to an MCP client over stdio",
		Long: `Relays MCP between an agent on stdio and a node's door on ZAP.

The node's tools ARE its typed operations — one tool per op, named by the
op — so nothing here enumerates them and nothing here can fall behind
them. Every frame is passed whole.

Point an MCP client at this command:

  {"command": "lux", "args": ["mcp", "--at", "/run/zip/luxd.sock"]}

Examples:
  lux mcp
  lux mcp --at /run/zip/luxd.sock
  lux mcp --at node.lux.svc:9655`,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(c *cobra.Command, _ []string) error {
			return relay(zapmcp.Dial(zip.NetworkOf(at), at), c.InOrStdin(), c.OutOrStdout())
		},
	}
	cmd.Flags().StringVar(&at, "at", zip.SocketPath("luxd"), "the node's MCP address (a socket path, or host:port)")
	return cmd
}

// relay moves frames between an MCP client and a node until the client stops
// sending. It is one exchange at a time because stdio is one stream: the client
// owns the ordering, and a relay that reordered would be inventing a semantics
// neither end asked for.
func relay(to *zapmcp.Transport, in io.Reader, out io.Writer) error {
	dec := json.NewDecoder(in)
	enc := json.NewEncoder(out)

	for {
		var f zapmcp.Frame
		switch err := dec.Decode(&f); {
		case errors.Is(err, io.EOF):
			return nil
		case err != nil:
			// A message we cannot read still gets an answer, or the client
			// waits forever on a request it believes it sent. It has no id we
			// could correlate, which is what -32700 means.
			if werr := enc.Encode(fault("", parseError, err.Error())); werr != nil {
				return werr
			}
			return fmt.Errorf("lux mcp: unreadable message: %w", err)
		}

		answer, err := to.Do(&f)
		if err != nil {
			// The far end failed, not this hop. The client is told so in its
			// own vocabulary rather than being left holding an id nothing will
			// ever answer.
			if werr := enc.Encode(fault(f.ID, internalError, err.Error())); werr != nil {
				return werr
			}
			continue
		}
		if answer == nil {
			continue // a notification expects none
		}
		if err := enc.Encode(answer); err != nil {
			return err
		}
	}
}

// fault is a refusal this relay raised itself, correlated to the message that
// caused it.
func fault(id string, code int32, msg string) *zapmcp.Frame {
	f := &zapmcp.Frame{Kind: zapmcp.Response, ID: id}
	return f.Fail(code, msg)
}
