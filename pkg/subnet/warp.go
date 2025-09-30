// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package subnet

// WarpSpec contains configuration for Warp deployments
type WarpSpec struct {
	SkipWarpDeploy               bool
	SkipRelayerDeploy            bool
	WarpVersion                  string
	RelayerVersion               string
	RelayerBinPath               string
	RelayerLogLevel              string
	MessengerContractAddressPath string
	MessengerDeployerAddressPath string
	MessengerDeployerTxPath      string
	FundedAddress                string
	RegistryBydecodePath         string
}
