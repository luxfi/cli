// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package globalconfig

import (
	"testing"
)

func TestMergeDefaults(t *testing.T) {
	merged := Merge(nil, nil)

	if merged.Config.Local.NumNodes == nil || *merged.Config.Local.NumNodes != DefaultNumNodes {
		t.Errorf("expected default numNodes %d", DefaultNumNodes)
	}
	if merged.Sources.NumNodes != SourceDefault {
		t.Errorf("expected source %s, got %s", SourceDefault, merged.Sources.NumNodes)
	}
}

func TestMergeGlobalOverridesDefaults(t *testing.T) {
	numNodes := uint32(10)
	global := &GlobalConfig{
		Local: LocalConfig{
			NumNodes: &numNodes,
		},
	}

	merged := Merge(global, nil)

	if *merged.Config.Local.NumNodes != 10 {
		t.Errorf("expected numNodes 10, got %d", *merged.Config.Local.NumNodes)
	}
	if merged.Sources.NumNodes != SourceGlobal {
		t.Errorf("expected source %s, got %s", SourceGlobal, merged.Sources.NumNodes)
	}
}

func TestMergeProjectOverridesGlobal(t *testing.T) {
	globalNodes := uint32(10)
	global := &GlobalConfig{
		Local: LocalConfig{
			NumNodes: &globalNodes,
		},
	}

	projectNodes := uint32(3)
	project := &ProjectConfig{
		GlobalConfig: GlobalConfig{
			Local: LocalConfig{
				NumNodes: &projectNodes,
			},
		},
	}

	merged := Merge(global, project)

	if *merged.Config.Local.NumNodes != 3 {
		t.Errorf("expected numNodes 3, got %d", *merged.Config.Local.NumNodes)
	}
	if merged.Sources.NumNodes != SourceProject {
		t.Errorf("expected source %s, got %s", SourceProject, merged.Sources.NumNodes)
	}
}

func TestMergePartialOverride(t *testing.T) {
	// Global sets numNodes, project sets autoTrack
	globalNodes := uint32(7)
	global := &GlobalConfig{
		Local: LocalConfig{
			NumNodes: &globalNodes,
		},
	}

	autoTrack := false
	project := &ProjectConfig{
		GlobalConfig: GlobalConfig{
			Local: LocalConfig{
				AutoTrackSubnets: &autoTrack,
			},
		},
	}

	merged := Merge(global, project)

	// numNodes should come from global
	if *merged.Config.Local.NumNodes != 7 {
		t.Errorf("expected numNodes 7 from global, got %d", *merged.Config.Local.NumNodes)
	}
	if merged.Sources.NumNodes != SourceGlobal {
		t.Errorf("expected numNodes source %s, got %s", SourceGlobal, merged.Sources.NumNodes)
	}

	// autoTrack should come from project
	if *merged.Config.Local.AutoTrackSubnets != false {
		t.Error("expected autoTrackSubnets false from project")
	}
	if merged.Sources.AutoTrackSubnets != SourceProject {
		t.Errorf("expected autoTrack source %s, got %s", SourceProject, merged.Sources.AutoTrackSubnets)
	}
}

func TestMergeAllSettings(t *testing.T) {
	balance := float64(2000)
	weight := uint64(50)
	global := &GlobalConfig{
		Network: NetworkConfig{
			DefaultNetwork: "testnet",
			LuxdVersion:    "v1.0.0",
		},
		EVM: EVMConfig{
			DefaultTokenName:   "GLOBAL",
			DefaultTokenSymbol: "GLB",
			DefaultTokenSupply: "5000000",
		},
		Staking: StakingConfig{
			BootstrapValidatorBalance: &balance,
			BootstrapValidatorWeight:  &weight,
		},
		Node: NodeConfig{
			DefaultInstanceType: "large",
			DefaultRegion:       "eu-west-1",
		},
	}

	merged := Merge(global, nil)

	// Verify all settings applied
	if merged.Config.Network.DefaultNetwork != "testnet" {
		t.Errorf("expected network testnet, got %s", merged.Config.Network.DefaultNetwork)
	}
	if merged.Config.EVM.DefaultTokenName != "GLOBAL" {
		t.Errorf("expected token name GLOBAL, got %s", merged.Config.EVM.DefaultTokenName)
	}
	if *merged.Config.Staking.BootstrapValidatorBalance != 2000 {
		t.Errorf("expected balance 2000, got %f", *merged.Config.Staking.BootstrapValidatorBalance)
	}
	if merged.Config.Node.DefaultRegion != "eu-west-1" {
		t.Errorf("expected region eu-west-1, got %s", merged.Config.Node.DefaultRegion)
	}
}
