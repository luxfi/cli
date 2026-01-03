// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package networkcmd

import (
	"fmt"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/ids"
	"github.com/luxfi/sdk/contract"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/sdk/prompts"
)

// GetProxyOwnerPrivateKey retrieves the private key for a proxy contract owner.
// If not found in managed keys, prompts the user.
func GetProxyOwnerPrivateKey(
	app *application.Lux,
	network models.Network,
	proxyContractOwner string,
	printFunc func(msg string, args ...interface{}),
) (string, error) {
	found, _, _, proxyOwnerPrivateKey, err := contract.SearchForManagedKey(
		app.GetSDKApp(),
		network,
		proxyContractOwner,
		true,
	)
	if err != nil {
		return "", err
	}
	if !found {
		printFunc("Private key for proxy owner address %s was not found", proxyContractOwner)
		proxyOwnerPrivateKey, err = prompts.PromptPrivateKey(
			app.Prompt,
			"configure validator manager proxy for PoS",
		)
		if err != nil {
			return "", err
		}
	}
	return proxyOwnerPrivateKey, nil
}

// PromptNodeID prompts the user to enter a node ID for the specified goal.
func PromptNodeID(goal string) (ids.NodeID, error) {
	txt := fmt.Sprintf("What is the NodeID of the node you want to %s?", goal)
	return app.Prompt.CaptureNodeID(txt)
}
