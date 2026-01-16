// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package localnetworkinterface provides local network status checking.
package localnetworkinterface

import (
	"context"
	"errors"
	"strings"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/constants"
	sdkinfo "github.com/luxfi/sdk/info"
)

// StatusChecker provides network status checking operations.
type StatusChecker interface {
	GetCurrentNetworkVersion() (string, int, bool, error)
}

// networkStatusChecker checks the status of the running network
// It uses the network state file to determine the correct API endpoint
type networkStatusChecker struct {
	app *application.Lux
}

// NewStatusChecker creates a new status checker
// If app is nil, it uses the default LocalAPIEndpoint
func NewStatusChecker() StatusChecker {
	return &networkStatusChecker{app: nil}
}

// NewStatusCheckerWithApp creates a new status checker with app context
// This allows it to read the network state and use the correct endpoint
func NewStatusCheckerWithApp(app *application.Lux) StatusChecker {
	return &networkStatusChecker{app: app}
}

func (n *networkStatusChecker) GetCurrentNetworkVersion() (string, int, bool, error) {
	ctx := context.Background()

	// Use dynamic endpoint if app is available
	endpoint := constants.LocalAPIEndpoint
	if n.app != nil {
		endpoint = n.app.GetRunningNetworkEndpoint()
	}

	infoClient := sdkinfo.NewClient(endpoint)
	versionResponse, err := infoClient.GetNodeVersion(ctx)
	if err != nil {
		// not actually an error, network just not running
		return "", 0, false, nil
	}

	// version is in format lux/x.y.z, need to turn to semantic
	splitVersion := strings.Split(versionResponse.Version, "/")
	if len(splitVersion) != 2 {
		return "", 0, false, errors.New("unable to parse node version " + versionResponse.Version)
	}
	// index 0 should be lux, index 1 will be version
	parsedVersion := "v" + splitVersion[1]

	return parsedVersion, int(versionResponse.RPCProtocolVersion), true, nil
}
