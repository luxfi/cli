// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package binutils

import "testing"

func TestGetGRPCPorts(t *testing.T) {
	tests := []struct {
		name        string
		networkType string
		wantServer  int
		wantGateway int
	}{
		{
			name:        "mainnet ports",
			networkType: "mainnet",
			wantServer:  GRPCPortMainnet,
			wantGateway: GRPCGatewayPortMainnet,
		},
		{
			name:        "testnet ports",
			networkType: "testnet",
			wantServer:  GRPCPortTestnet,
			wantGateway: GRPCGatewayPortTestnet,
		},
		{
			name:        "devnet ports",
			networkType: "devnet",
			wantServer:  GRPCPortDevnet,
			wantGateway: GRPCGatewayPortDevnet,
		},
		{
			name:        "custom ports",
			networkType: "custom",
			wantServer:  GRPCPortCustom,
			wantGateway: GRPCGatewayPortCustom,
		},
		{
			name:        "local ports (alias for custom)",
			networkType: "local",
			wantServer:  GRPCPortLocal,
			wantGateway: GRPCGatewayPortLocal,
		},
		{
			name:        "unknown network defaults to custom",
			networkType: "unknown",
			wantServer:  GRPCPortCustom,
			wantGateway: GRPCGatewayPortCustom,
		},
		{
			name:        "empty string defaults to custom",
			networkType: "",
			wantServer:  GRPCPortCustom,
			wantGateway: GRPCGatewayPortCustom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ports := GetGRPCPorts(tt.networkType)
			if ports.Server != tt.wantServer {
				t.Errorf("GetGRPCPorts(%q).Server = %d, want %d", tt.networkType, ports.Server, tt.wantServer)
			}
			if ports.Gateway != tt.wantGateway {
				t.Errorf("GetGRPCPorts(%q).Gateway = %d, want %d", tt.networkType, ports.Gateway, tt.wantGateway)
			}
		})
	}
}

func TestPortsAreUnique(t *testing.T) {
	// Verify all network types have unique ports to avoid conflicts
	// Note: "local" is an alias for "custom", so they share ports intentionally
	ports := map[int]string{}
	networks := []string{"mainnet", "testnet", "devnet", "custom"}

	for _, net := range networks {
		p := GetGRPCPorts(net)
		if existing, ok := ports[p.Server]; ok {
			t.Errorf("Server port %d is shared between %s and %s", p.Server, existing, net)
		}
		ports[p.Server] = net + "-server"

		if existing, ok := ports[p.Gateway]; ok {
			t.Errorf("Gateway port %d is shared between %s and %s", p.Gateway, existing, net)
		}
		ports[p.Gateway] = net + "-gateway"
	}
}

func TestLocalIsAliasForCustom(t *testing.T) {
	// Verify "local" returns the same ports as "custom"
	localPorts := GetGRPCPorts("local")
	customPorts := GetGRPCPorts("custom")

	if localPorts.Server != customPorts.Server {
		t.Errorf("local server port (%d) should equal custom server port (%d)", localPorts.Server, customPorts.Server)
	}
	if localPorts.Gateway != customPorts.Gateway {
		t.Errorf("local gateway port (%d) should equal custom gateway port (%d)", localPorts.Gateway, customPorts.Gateway)
	}
}

func TestPortConstants(t *testing.T) {
	// Verify the actual port values match the documented configuration
	// Port scheme: aligned with chain IDs (8368-8371 for gRPC)
	// - 8368: testnet (chain ID 96368)
	// - 8369: mainnet (chain ID 96369)
	// - 8370: devnet (chain ID 96370)
	// - 8371: custom/local (chain ID 1337)
	if GRPCPortMainnet != 8369 {
		t.Errorf("GRPCPortMainnet = %d, want 8369", GRPCPortMainnet)
	}
	if GRPCPortTestnet != 8368 {
		t.Errorf("GRPCPortTestnet = %d, want 8368", GRPCPortTestnet)
	}
	if GRPCPortLocal != 8371 {
		t.Errorf("GRPCPortLocal = %d, want 8371", GRPCPortLocal)
	}
	if GRPCPortCustom != 8371 {
		t.Errorf("GRPCPortCustom = %d, want 8371", GRPCPortCustom)
	}
	// Verify local is an alias for custom
	if GRPCPortLocal != GRPCPortCustom {
		t.Errorf("GRPCPortLocal (%d) should equal GRPCPortCustom (%d)", GRPCPortLocal, GRPCPortCustom)
	}
}
