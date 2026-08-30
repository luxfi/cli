// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package route addresses a node's HTTP API.
package route

import (
	"strings"

	"github.com/luxfi/constants"
)

// base is the prefix every luxd route hangs from: the node's own endpoints
// (/v1/info, /v1/health) and, below [Chain], every chain's.
const base = "/v1"

// Chain is where a chain answers, and the one place this module builds that
// address.
//
// uri is a node's base URL, or empty for the path alone: Chain("", "P") is the
// P-Chain path, and Chain("http://localhost:9630", "C")+"/rpc" is that node's
// C-Chain RPC. The segment is [constants.ChainAliasPrefix], named once, so
// every caller moves when it moves.
//
// That is the point of routing them all through here. This module used to
// spell the address out at 112 sites; when the node renamed the segment, all
// 112 went on addressing a name the router had already left behind. Nothing
// failed to compile. They simply answered 404.
func Chain(uri, alias string) string {
	return uri + base + "/" + constants.ChainAliasPrefix + "/" + alias
}

// Alias reads back what [Chain] writes: the alias in a chain address, and
// whether the address named a chain at all. Inverse of Chain, and the one
// place this module recognises the segment.
func Alias(address string) (string, bool) {
	parts := strings.Split(address, "/")
	for i, part := range parts {
		if part == constants.ChainAliasPrefix && i+1 < len(parts) && parts[i+1] != "" {
			return parts[i+1], true
		}
	}
	return "", false
}
