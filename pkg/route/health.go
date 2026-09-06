// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package route

// health is where the node's health service answers, and ops is the endpoint
// its typed operations hang from. Every service on the node is reachable the
// same way — /v1/info/ops, /v1/chain/P/ops — so the two are named once here
// and composed below.
const (
	health = base + "/health"
	ops    = "/ops"
)

// Health is the node's full health report: every registered check, and the
// verdict over all of them. What to read when the answer matters more than
// the status code.
//
// uri is a node's base URL, or empty for the path alone.
func Health(uri string) string { return uri + health + ops + "/health" }

// Liveness is the probe that answers whether the process is up. Cheapest of
// the three, and the one to poll in a loop.
func Liveness(uri string) string { return uri + health + ops + "/liveness" }

// Readiness is the probe that answers whether the node will serve — up, and
// past bootstrap. What to wait on before calling a node started.
func Readiness(uri string) string { return uri + health + ops + "/readiness" }
