// Package callcmd implements `lux call` — every operation a node publishes, as
// a command, with none of them written down here.
//
// The node derives an OpenAPI document from its typed-op registry; this asks a
// running node for that document and projects it back into a command tree. So
// an operation registered on the node this morning is a command this afternoon
// with nothing rebuilt and no release of this binary, and a command cannot
// address an operation the node does not serve, because there is no second
// place for one to be written.
//
// It replaces the shape it grew out of: a hand-maintained list of methods, each
// one a string somebody typed. That is how this SDK came to send
// exchangevm.getBlock to a node that registers the X-chain under xvm.
package callcmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zap-proto/zip"
)

func NewCmd() *cobra.Command {
	var at string
	cmd := &cobra.Command{
		Use:   "call [service] [operation] [flags]",
		Short: "Run one of a node's operations",
		Long: `Asks a node what it can do, and runs one of the answers.

With no arguments it lists every service the node publishes; with a
service it lists that service's operations; with both it runs one. The
flags of an operation are the fields of its input, and its help is the
prose the handler carries — all of it read from the node, none of it
written here.

Examples:
  lux call
  lux call platform
  lux call platform get-height
  lux call --at node.lux.svc:9653 platform get-validators --net-id 8675309`,
		DisableFlagParsing: true, // an operation's flags belong to the operation
		SilenceUsage:       true,
		RunE: func(c *cobra.Command, args []string) error {
			addr, args := address(at, args)
			node := zip.Remote{Base: addr}
			ctx := context.Background()

			spec, err := node.Spec(ctx)
			if err != nil {
				// A service with no typed ops does not serve the document at
				// all, so the honest reading of a 404 here is "nothing to
				// derive from" and not "wrong address". It is the state a node
				// is in until its handlers are typed, and it is the first thing
				// anyone running this will hit.
				if strings.Contains(err.Error(), "404") {
					return fmt.Errorf("%s publishes no operations: it serves no document, so there is nothing to derive commands from", addr)
				}
				return fmt.Errorf("asking %s what it serves: %w", addr, err)
			}
			cmds, err := zip.CommandsFromSpec(spec)
			if err != nil {
				return err
			}
			if len(cmds) == 0 {
				return fmt.Errorf("%s publishes no operations: its document has no paths in it", addr)
			}
			return (&zip.CLI{
				Name:     "lux call",
				Commands: cmds,
				Invoke:   node.Invoke,
				Out:      c.OutOrStdout(),
			}).Run(ctx, args)
		},
	}
	cmd.Flags().StringVar(&at, "at", zip.SocketPath("luxd"), "the node to ask (a socket path, host:port, or URL)")
	return cmd
}

// address pulls --at out of the argv by hand, because flag parsing is off: the
// remaining arguments are the operation's, and cobra would claim any of them
// that happened to share a name with one of ours.
func address(def string, args []string) (string, []string) {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--at" && i+1 < len(args):
			def = args[i+1]
			i++
		case len(args[i]) > 5 && args[i][:5] == "--at=":
			def = args[i][5:]
		default:
			out = append(out, args[i])
		}
	}
	return def, out
}
