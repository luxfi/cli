// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package nodecmd

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"

	"github.com/luxfi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

var (
	joinServer string
	joinToken  string
	joinRole   string
	joinName   string
	joinInit   bool
	joinPrint  bool
)

func newJoinCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "join",
		Short: "Join this box to the Lux node pool (or --init to start one)",
		Long: `Run this on any box and it joins the pool; the operator schedules a
compact validator onto it and rebalances the fleet. Drain a box and its
validator reschedules onto the others — run a node anywhere, the network spreads.

  # First box — become the control plane and hold history:
  lux node join --init --role archive

  # Any other box — join and run a compact validator:
  lux node join --server https://<control>:6443 --token <token>

Off-LAN boxes: put them on one tailnet first (tailscale up, or hanzozt) and pass
the tailnet IP as --server, so "anywhere" really means anywhere. The role is a
node label (lux.cloud/validator|archive=true) the NodeFleet schedules against —
see operator/spec/examples/nodefleet-lab.yaml.`,
		RunE: func(_ *cobra.Command, _ []string) error { return runJoin() },
	}
	cmd.Flags().BoolVar(&joinInit, "init", false, "make THIS box the pool's control plane (k3s server)")
	cmd.Flags().StringVar(&joinServer, "server", "", "control-plane API URL, e.g. https://<ip>:6443 (agent mode)")
	cmd.Flags().StringVar(&joinToken, "token", "", "node token from the control plane (agent mode)")
	cmd.Flags().StringVar(&joinRole, "role", "validator", "node role: validator (compact) or archive (full history)")
	cmd.Flags().StringVar(&joinName, "name", "", "node name (default: hostname)")
	cmd.Flags().BoolVar(&joinPrint, "print", false, "print the k3s command instead of running it")
	return cmd
}

func runJoin() error {
	if joinRole != "validator" && joinRole != "archive" {
		return fmt.Errorf("--role must be validator or archive, got %q", joinRole)
	}
	label := "lux.cloud/" + joinRole + "=true"

	var sh string
	if joinInit {
		sh = fmt.Sprintf("curl -sfL https://get.k3s.io | sh -s - server --write-kubeconfig-mode 644 --node-label %s", label)
	} else {
		if joinServer == "" || joinToken == "" {
			return fmt.Errorf("agent mode needs --server and --token (a control plane started with `lux node join --init` prints both)")
		}
		sh = fmt.Sprintf("curl -sfL https://get.k3s.io | K3S_URL=%s K3S_TOKEN=%s sh -s - agent --node-label %s", joinServer, joinToken, label)
	}
	if joinName != "" {
		sh += " --node-name " + joinName
	}

	if joinPrint {
		ux.Logger.PrintToUser("%s", sh)
		return nil
	}

	ux.Logger.PrintToUser("Joining the pool as %s ...", joinRole)
	c := exec.Command("sh", "-c", sh)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("k3s join failed: %w", err)
	}

	if joinInit {
		token, _ := os.ReadFile("/var/lib/rancher/k3s/server/node-token")
		ux.Logger.PrintToUser("")
		ux.Logger.PrintToUser("Control plane up. Add any other box with:")
		ux.Logger.PrintToUser("  lux node join --server https://%s:6443 --token %s", primaryIP(), strings.TrimSpace(string(token)))
		ux.Logger.PrintToUser("")
		ux.Logger.PrintToUser("Then spread the network:  kubectl apply -f ~/work/lux/operator/spec/examples/nodefleet-lab.yaml")
		return nil
	}
	ux.Logger.PrintToUser("Joined. The operator will schedule a %s on this box shortly.", joinRole)
	return nil
}

// primaryIP returns this box's primary outbound IPv4 without sending a packet.
func primaryIP() string {
	c, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "<this-box-ip>"
	}
	defer c.Close()
	return c.LocalAddr().(*net.UDPAddr).IP.String()
}
