// Copyright (C) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package transactioncmd

import (
	"fmt"

	"github.com/luxfi/cli/v2/v2/pkg/txutils"
	"github.com/luxfi/cli/v2/v2/pkg/ux"
	"github.com/luxfi/ids"
	"github.com/luxfi/node/v2/v2/utils/units"
	"github.com/luxfi/node/v2/v2/vms/platformvm/txs"

	"github.com/ethereum/go-ethereum/common"
)

func validateConvertOperation(tx *txs.Tx, action string) (bool, error) {
	network, err := txutils.GetNetwork(tx)
	if err != nil {
		return false, err
	}
	convertToL1Tx, ok := tx.Unsigned.(*txs.ConvertSubnetToL1Tx)
	if !ok {
		return false, fmt.Errorf("expected tx to be of type txs.ConvertSubnetToL1Tx, found %T", tx.Unsigned)
	}
	ux.Logger.PrintToUser("You are about to %s a ConvertSubnetToL1Tx for %s with the following content:", action, network.Name())
	ux.Logger.PrintToUser("  Subnet ID: %s", convertToL1Tx.Subnet)
	ux.Logger.PrintToUser("  Blockchain ID: %s", convertToL1Tx.ChainID)
	ux.Logger.PrintToUser("  Manager Address: %s", common.BytesToAddress(convertToL1Tx.Address).Hex())
	ux.Logger.PrintToUser("  Validators:")
	for _, val := range convertToL1Tx.Validators {
		nodeID, err := ids.ToNodeID(val.NodeID)
		if err != nil {
			return false, fmt.Errorf("unexpected node ID on tx")
		}
		ux.Logger.PrintToUser("    Node ID: %s", nodeID)
		ux.Logger.PrintToUser("    Weight: %d", val.Weight)
		ux.Logger.PrintToUser("    Balance: %.5f", float64(val.Balance)/float64(units.Lux))
	}
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("Please review the details of the ConvertSubnetToL1 Transaction")
	ux.Logger.PrintToUser("")
	return app.Prompt.CaptureYesNo(fmt.Sprintf("Do you want to %s the transaction?", action))
}
