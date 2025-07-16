// Copyright (C) 2024, Lux Industries Inc. All rights reserved.
// Placeholder for APM config
package apm

type Config struct {
	RepositoryURL   string
	Auth            string
	RegistryURL     string
}

type Credential struct {
	RegistryURL string `yaml:"registry_url"`
	Token       string `yaml:"token"`
}

func DefaultConfig() *Config {
	return &Config{
		RepositoryURL: "https://apm.lux.network",
		RegistryURL:   "https://registry.lux.network",
	}
}