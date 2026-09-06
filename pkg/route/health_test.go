// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package route

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestHealthAddressesTheServedPaths states the wire contract. The health
// service moved onto typed ops, and the address it used to answer on now
// returns 404, so a probe written against the old one reports every healthy
// node as unhealthy.
func TestHealthAddressesTheServedPaths(t *testing.T) {
	require := require.New(t)

	const node = "http://127.0.0.1:9630"

	require.Equal(node+"/v1/health/ops/health", Health(node))
	require.Equal(node+"/v1/health/ops/liveness", Liveness(node))
	require.Equal(node+"/v1/health/ops/readiness", Readiness(node))

	// The path alone, for callers that hold the host separately.
	require.Equal("/v1/health/ops/liveness", Liveness(""))
}

// TestHealthSharesTheChainBase pins the three to the same /v1 every other
// route hangs from, so the node's addresses move together or not at all.
func TestHealthSharesTheChainBase(t *testing.T) {
	require := require.New(t)

	for _, address := range []string{Health(""), Liveness(""), Readiness("")} {
		require.Equal(base+"/health/ops", address[:len(base)+len("/health/ops")])
	}
}
