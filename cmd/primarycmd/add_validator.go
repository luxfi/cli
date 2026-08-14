// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package primarycmd provides commands for managing primary network validators.
package primarycmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/luxfi/cli/pkg/chain"
	"github.com/luxfi/cli/pkg/cobrautils"
	"github.com/luxfi/cli/pkg/keychain"
	"github.com/luxfi/cli/pkg/networkoptions"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/address"
	"github.com/luxfi/constants"
	"github.com/luxfi/ids"
	"github.com/luxfi/proto/p/signer"
	"github.com/luxfi/sdk/platformvm"
	"github.com/luxfi/utxo/secp256k1fx"
	"github.com/spf13/cobra"
)

var (
	globalNetworkFlags networkoptions.NetworkFlags
	keyName            string
	useLedger          bool
	ledgerAddresses    []string
	nodeIDStr          string
	publicKey          string
	pop                string
	stake              uint64
	duration           time.Duration
	delegationFee      uint32
	rewardAddress      string
)

// lux primary addValidator
func newAddValidatorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "addValidator",
		Short: "Register a validator on the primary network",
		Long: `Issues an AddPermissionlessValidatorTx for a node identity.

Create the identity first — it prints every value this command needs:

  lux key staker ~/.luxd/staking
  lux primary addValidator --mainnet \
      --node-id NodeID-... --public-key 0x... --proof-of-possession 0x... \
      --stake 2000000000 --duration 336h

The stake is in nLUX (1 LUX = 1e9 nLUX) and the chain reads the start time
from its own clock, so --duration measures from when the tx is accepted.`,
		RunE: addValidator,
		Args: cobrautils.ExactArgs(0),
	}
	networkoptions.AddNetworkFlagsToCmd(cmd, &globalNetworkFlags, false, networkoptions.NonLocalSupportedNetworkOptions)
	cmd.Flags().StringVarP(&keyName, "key", "k", "", "name of the stored key that pays and owns the rewards")
	cmd.Flags().BoolVarP(&useLedger, "ledger", "g", false, "sign with a ledger device instead of a stored key")
	cmd.Flags().StringSliceVar(&ledgerAddresses, "ledger-addrs", []string{}, "ledger addresses to search")
	cmd.Flags().StringVar(&nodeIDStr, "node-id", "", "NodeID of the validator")
	cmd.Flags().StringVar(&publicKey, "public-key", "", "BLS public key of the validator")
	cmd.Flags().StringVar(&pop, "proof-of-possession", "", "BLS proof of possession of the validator")
	cmd.Flags().Uint64Var(&stake, "stake", 0, "amount to stake, in nLUX")
	cmd.Flags().DurationVar(&duration, "duration", 0, "how long the validator stays in the set")
	cmd.Flags().Uint32Var(&delegationFee, "delegation-fee", 20_000, "share of delegation rewards the validator keeps, out of 1,000,000")
	cmd.Flags().StringVar(&rewardAddress, "reward-address", "", "P-Chain address to own the staking reward (default: the paying key)")
	return cmd
}

func addValidator(_ *cobra.Command, _ []string) error {
	network, err := networkoptions.GetNetworkFromCmdLineFlags(
		app,
		"",
		globalNetworkFlags,
		true,
		false,
		networkoptions.NonLocalSupportedNetworkOptions,
		"",
	)
	if err != nil {
		return err
	}

	nodeID, err := ids.NodeIDFromString(nodeIDStr)
	if err != nil {
		return fmt.Errorf("--node-id: %w", err)
	}
	// The BLS key travels as the pair the node's info API and `lux key staker`
	// both print, so it round-trips through the same JSON either one emits.
	blsKey := &signer.ProofOfPossession{}
	blob, err := json.Marshal(map[string]string{"publicKey": publicKey, "proofOfPossession": pop})
	if err != nil {
		return err
	}
	if err := json.Unmarshal(blob, blsKey); err != nil {
		return fmt.Errorf("--public-key/--proof-of-possession: %w", err)
	}
	if err := blsKey.Verify(); err != nil {
		return fmt.Errorf("--proof-of-possession does not prove ownership of --public-key: %w", err)
	}

	rewardsOwner, err := rewardOwner(rewardAddress)
	if err != nil {
		return err
	}

	// The chain is the authority on its own staking floor; a table compiled in
	// here would be a second answer that drifts.
	ctx, cancel := context.WithTimeout(context.Background(), constants.DefaultConfirmTxTimeout)
	defer cancel()
	minStake, _, err := platformvm.NewClient(network.Endpoint()).GetMinStake(ctx, constants.PrimaryNetworkID)
	if err != nil {
		return fmt.Errorf("reading the staking floor from %s: %w", network.Endpoint(), err)
	}
	if stake < minStake {
		return fmt.Errorf("--stake %d nLUX is below %s's minimum of %d nLUX", stake, network.Name(), minStake)
	}

	kc, err := keychain.GetKeychain(app, false, useLedger, ledgerAddresses, keyName, network, stake)
	if err != nil {
		return err
	}

	deployer := chain.NewPublicDeployer(app, useLedger, kc.Keychain, network)
	txID, err := deployer.AddValidator(
		nodeID,
		blsKey,
		stake,
		time.Now().Add(duration),
		rewardsOwner, // nil falls back to the paying wallet's first address
		delegationFee,
	)
	if err != nil {
		return err
	}
	ux.Logger.PrintToUser("%s registered on %s", nodeID, network.Name())
	ux.Logger.PrintToUser("  txID: %s", txID)
	return nil
}

// rewardOwner resolves --reward-address into the owner of the staking reward.
//
// Who funds the bond and who earns from it are different questions. They
// coincide by default, and an operator staking on someone else's behalf has to
// be able to say so — otherwise the reward silently accrues to whoever paid.
// An empty address means "unstated", which the caller passes through as nil so
// the wallet's own first address owns the reward.
func rewardOwner(addr string) (*secp256k1fx.OutputOwners, error) {
	if addr == "" {
		return nil, nil
	}
	addrs, err := address.ParseToIDs([]string{addr})
	if err != nil {
		return nil, fmt.Errorf("--reward-address %q: %w", addr, err)
	}
	return &secp256k1fx.OutputOwners{Threshold: 1, Addrs: addrs}, nil
}
