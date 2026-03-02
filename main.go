// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	"crypto/tls"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/luxfi/cli/cmd"
)

func init() {
	// Configure the global HTTP transport with short per-IP dial timeouts.
	// Remote API endpoints (api.lux-dev.network, api.lux-test.network, etc.)
	// may resolve to multiple IPs where some are unreachable. Without this,
	// Go's default dialer waits 30s per dead IP before trying the next.
	// Allow skipping TLS verification for devnet/internal endpoints
	// that may use self-signed or staging certificates.
	// Set LUX_INSECURE_TLS=1 to skip verification.
	skipTLS := os.Getenv("LUX_INSECURE_TLS") == "1"
	http.DefaultTransport = &http.Transport{
		DialContext:           (&net.Dialer{Timeout: 5 * time.Second}).DialContext,
		TLSHandshakeTimeout:  10 * time.Second,
		TLSClientConfig:      &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: skipTLS},
		MaxIdleConns:         100,
		MaxIdleConnsPerHost:  10,
		IdleConnTimeout:      90 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:    true,
	}
}

func main() {
	cmd.Execute()
}
