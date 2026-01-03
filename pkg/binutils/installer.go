// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package binutils

import (
	"runtime"
)

// Installer provides system architecture information.
type Installer interface {
	GetArch() (string, string)
}

type installerImpl struct{}

// NewInstaller creates a new installer.
func NewInstaller() Installer {
	return &installerImpl{}
}

func (installerImpl) GetArch() (string, string) {
	return runtime.GOARCH, runtime.GOOS
}
