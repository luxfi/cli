// Copyright (C) 2024, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package subnetcmd

// getBlockTime returns the block time in milliseconds for the given sequencer/chain
func getBlockTime(sequencer string) int {
	switch sequencer {
	case "lux":
		return 100 // 100ms
	case "ethereum":
		return 12000 // 12s
	case "avalanche":
		return 2000 // 2s
	case "op":
		return 2000 // 2s (OP Stack block time)
	case "external":
		return 1000 // 1s default for external sequencers
	default:
		return 100 // Default to Lux timing
	}
}

// isBasedRollup returns true if the sequencer represents a based rollup (L1-sequenced)
func isBasedRollup(sequencer string) bool {
	switch sequencer {
	case "lux", "ethereum", "avalanche":
		return true // These are L1s, so it's a based rollup
	case "op", "external":
		return false // OP Stack and external sequencers are not based rollups
	default:
		return false
	}
}