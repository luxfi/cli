// Copyright (C) 2022, Lux Partners Limited, All rights reserved.
// See the file LICENSE for licensing terms.

package binutils

import (
	"fmt"

	"github.com/luxdefi/cli/pkg/constants"
)

const (
	linux   = "linux"
	darwin  = "darwin"
	windows = "windows"

	zipExtension = "zip"
	tarExtension = "tar.gz"
)

type GithubDownloader interface {
	GetDownloadURL(version string, installer Installer) (string, string, error)
}

type (
	subnetEVMDownloader   struct{}
	luxGoDownloader struct{}
)

var (
	_ GithubDownloader = (*subnetEVMDownloader)(nil)
	_ GithubDownloader = (*luxGoDownloader)(nil)
)

func GetGithubLatestReleaseURL(org, repo string) string {
	return "https://api.github.com/repos/" + org + "/" + repo + "/releases/latest"
}

func NewAvagoDownloader() GithubDownloader {
	return &luxGoDownloader{}
}

func (luxGoDownloader) GetDownloadURL(version string, installer Installer) (string, string, error) {
	// NOTE: if any of the underlying URLs change (github changes, release file names, etc.) this fails
	goarch, goos := installer.GetArch()

	var luxgoURL string
	var ext string

	switch goos {
	case linux:
		luxgoURL = fmt.Sprintf(
			"https://github.com/%s/%s/releases/download/%s/luxgo-linux-%s-%s.tar.gz",
			constants.LuxDeFiOrg,
			constants.LuxGoRepoName,
			version,
			goarch,
			version,
		)
		ext = tarExtension
	case darwin:
		luxgoURL = fmt.Sprintf(
			"https://github.com/%s/%s/releases/download/%s/luxgo-macos-%s.zip",
			constants.LuxDeFiOrg,
			constants.LuxGoRepoName,
			version,
			version,
		)
		ext = zipExtension
		// EXPERIMENTAL WIN, no support
	case windows:
		luxgoURL = fmt.Sprintf(
			"https://github.com/%s/%s/releases/download/%s/luxgo-win-%s-experimental.zip",
			constants.LuxDeFiOrg,
			constants.LuxGoRepoName,
			version,
			version,
		)
		ext = zipExtension
	default:
		return "", "", fmt.Errorf("OS not supported: %s", goos)
	}

	return luxgoURL, ext, nil
}

func NewSubnetEVMDownloader() GithubDownloader {
	return &subnetEVMDownloader{}
}

func (subnetEVMDownloader) GetDownloadURL(version string, installer Installer) (string, string, error) {
	// NOTE: if any of the underlying URLs change (github changes, release file names, etc.) this fails
	goarch, goos := installer.GetArch()

	var subnetEVMURL string
	ext := tarExtension

	switch goos {
	case linux:
		subnetEVMURL = fmt.Sprintf(
			"https://github.com/%s/%s/releases/download/%s/%s_%s_linux_%s.tar.gz",
			constants.LuxDeFiOrg,
			constants.SubnetEVMRepoName,
			version,
			constants.SubnetEVMRepoName,
			version[1:], // WARN subnet-evm isn't consistent in its release naming, it's omitting the v in the file name...
			goarch,
		)
	case darwin:
		subnetEVMURL = fmt.Sprintf(
			"https://github.com/%s/%s/releases/download/%s/%s_%s_darwin_%s.tar.gz",
			constants.LuxDeFiOrg,
			constants.SubnetEVMRepoName,
			version,
			constants.SubnetEVMRepoName,
			version[1:],
			goarch,
		)
	default:
		return "", "", fmt.Errorf("OS not supported: %s", goos)
	}

	return subnetEVMURL, ext, nil
}
