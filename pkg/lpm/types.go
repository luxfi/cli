// Copyright (C) 2024, Lux Industries Inc. All rights reserved.
// Placeholder for LPM types
package lpm

type Metadata struct {
	Alias       string
	Homepage    string
	Description string
	Maintainers []string
}

type VMUpload struct {
	ID              string
	Alias           string
	Homepage        string
	Description     string
	BinaryPath      string
	InstallScript   string
	ChainConfigPath string
	GenesisPath     string
	ReadmePath      string
	LicensePath     string
	SubnetPath      string
	Versions        []string
}

type Subnet struct {
	ID          string
	Alias       string
	VM          string
	Config      string
	Genesis     string
	Description string
}

type VM struct {
	ID          string
	Alias       string
	VMType      string
	Binary      string
	ChainConfig string
	Subnet      string
	Genesis     string
	Version     string
	URL         string
	Checksum    string
	Runtime     string
	Description string
}
