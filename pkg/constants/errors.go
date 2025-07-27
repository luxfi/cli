// Copyright (C) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package constants

import "errors"

var (
	ErrNoBlockchainID                 = errors.New("\n\nNo blockchainID found. To resolve this:\n- Use 'lux blockchain deploy' to deploy the blockchain and generate a blockchainID.\n- Or use 'lux blockchain import' to import an existing configuration.\n") //nolint:stylecheck
	ErrNoSubnetID                     = errors.New("\n\nNo subnetID found. To resolve this:\n- Use 'lux blockchain deploy' to create the subnet and generate a subnetID.\n- Or use 'lux blockchain import' to import an existing configuration.\n")             //nolint:stylecheck
	ErrInvalidValidatorManagerAddress = errors.New("invalid validator manager address")
	ErrKeyNotFoundOnMap               = errors.New("key not found on map")
)
