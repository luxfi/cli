// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package ssh provides SSH operations for remote host management.
package ssh

import (
	"strings"

	"github.com/luxfi/constantsants"
	"github.com/luxfi/sdk/models"
)

// HostInstaller handles installation operations on remote hosts.
type HostInstaller struct {
	Host *models.Host
}

// NewHostInstaller creates a new host installer.
func NewHostInstaller(host *models.Host) *HostInstaller {
	return &HostInstaller{Host: host}
}

// GetArch returns the architecture and OS of the remote host.
func (h *HostInstaller) GetArch() (string, string) {
	goArhBytes, err := h.Host.Command("dpkg --print-architecture", nil, constants.SSHScriptTimeout)
	if err != nil {
		return "", ""
	}
	goOSBytes, err := h.Host.Command("uname -s", nil, constants.SSHScriptTimeout)
	if err != nil {
		return "", ""
	}
	return strings.TrimSpace(string(goArhBytes)), strings.TrimSpace(strings.ToLower(string(goOSBytes)))
}
