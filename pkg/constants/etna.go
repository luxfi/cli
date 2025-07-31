// Copyright (C) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package constants

import (
	"time"

	"github.com/luxfi/node/upgrade"
	luxdconstants "github.com/luxfi/node/utils/constants"
)

var EtnaActivationTime = map[uint32]time.Time{
	luxdconstants.TestnetID:    time.Date(2024, time.November, 25, 16, 0, 0, 0, time.UTC),
	luxdconstants.MainnetID: time.Date(2024, time.December, 16, 17, 0, 0, 0, time.UTC),
	LocalNetworkID:           upgrade.Default.EtnaTime,
}
