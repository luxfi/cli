// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package ux

import (
	"bytes"
	"strings"
	"testing"
)

// TestCompactChainEndpointsAreSquareAndComposed renders the endpoint box and
// measures it: every address is the one the composer builds, and every framed
// line is the same width.
//
// The box used to be padded by hand against a segment two characters long, and
// the addresses inside it were written out. A rename moved the addresses and
// would have left the frame ragged; both are now derived.
func TestCompactChainEndpointsAreSquareAndComposed(t *testing.T) {
	saved := Logger
	t.Cleanup(func() { Logger = saved })

	var buf bytes.Buffer
	Logger = &UserLog{writer: &buf}
	PrintCompactChainEndpoints(9650)
	out := buf.String()

	for _, c := range GetNativeChains() {
		scheme := "http"
		if c.Type == "WS" {
			scheme = "ws"
		}
		want := c.Address(scheme + "://localhost:9650")
		if !strings.Contains(out, want) {
			t.Errorf("compact endpoints missing %s", want)
		}
	}

	width := -1
	framed := 0
	for _, line := range strings.Split(out, "\n") {
		edge := strings.TrimSpace(line)
		if edge == "" || !strings.ContainsAny(edge[:3], "┌│└") {
			continue
		}
		framed++
		switch n := len([]rune(edge)); {
		case width < 0:
			width = n
		case n != width:
			t.Errorf("box line is %d runes, want %d: %q", n, width, edge)
		}
	}
	if framed != len(GetNativeChains())+2 {
		t.Fatalf("framed %d lines, want %d rows plus two edges", framed, len(GetNativeChains()))
	}
}
