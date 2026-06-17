// Package ctxcmd implements `lux ctx` — list discoverable brand/env
// tuples from $LUX_BRAND_PATH.
package ctxcmd

import (
	"fmt"

	"github.com/luxfi/cli/pkg/brand"
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ctx",
		Short: "List discoverable network/env contexts",
		Long: `Scans $LUX_NETWORK_PATH (default ~/work/{lux,zoo,hanzo,pars,osage,adnexus}/universe)
for chain.yaml files and prints every (network, env) tuple they declare.`,
		Args: cobra.ExactArgs(0),
		RunE: func(*cobra.Command, []string) error {
			reg, err := brand.Discover()
			if err != nil {
				return err
			}
			for _, ref := range reg.Refs() {
				fmt.Println(ref)
			}
			return nil
		},
		SilenceUsage: true,
	}
}
