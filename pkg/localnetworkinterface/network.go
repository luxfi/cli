// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package localnetworkinterface

import (
	"context"
	"errors"
	"strings"

	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/node/api/info"
)

type StatusChecker interface {
	GetCurrentNetworkVersion() (string, int, bool, error)
}

type networkStatusChecker struct{}

func NewStatusChecker() StatusChecker {
	return networkStatusChecker{}
}

func (networkStatusChecker) GetCurrentNetworkVersion() (string, int, bool, error) {
	ctx := context.Background()
	infoClient := info.NewClient(constants.LocalAPIEndpoint)
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
