// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package binutils

import "testing"

func TestGetGRPCPorts(t *testing.T) {
	tests := []struct {
		name           string
		networkType    string
		wantServer     int
		wantGateway    int
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
			name:        "local ports",
			networkType: "local",
			wantServer:  GRPCPortLocal,
			wantGateway: GRPCGatewayPortLocal,
		},
		{
			name:        "unknown network defaults to mainnet",
			networkType: "unknown",
			wantServer:  GRPCPortMainnet,
			wantGateway: GRPCGatewayPortMainnet,
		},
		{
			name:        "empty string defaults to mainnet",
			networkType: "",
			wantServer:  GRPCPortMainnet,
			wantGateway: GRPCGatewayPortMainnet,
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
	ports := map[int]string{}
	networks := []string{"mainnet", "testnet", "local"}

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

func TestPortConstants(t *testing.T) {
	// Verify the actual port values match the documented configuration
	if GRPCPortMainnet != 8097 {
		t.Errorf("GRPCPortMainnet = %d, want 8097", GRPCPortMainnet)
	}
	if GRPCPortTestnet != 8098 {
		t.Errorf("GRPCPortTestnet = %d, want 8098", GRPCPortTestnet)
	}
	if GRPCPortLocal != 8099 {
		t.Errorf("GRPCPortLocal = %d, want 8099", GRPCPortLocal)
	}
}
