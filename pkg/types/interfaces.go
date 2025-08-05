// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package types

// ConfigWriter provides methods for writing configuration
type ConfigWriter interface {
	WriteConfigFile(data []byte) error
}

// ConfigLoader provides methods for loading configuration
type ConfigLoader interface {
	LoadConfig() (Config, error)
}

// PrompterInterface provides methods for user interaction
type PrompterInterface interface {
	CaptureYesNo(prompt string) (bool, error)
}
