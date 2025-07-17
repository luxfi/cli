// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package binutils

import (
	"fmt"

	"github.com/luxfi/cli/pkg/constants"
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
	nodeDownloader struct{}
)

var (
	_ GithubDownloader = (*subnetEVMDownloader)(nil)
	_ GithubDownloader = (*nodeDownloader)(nil)
)

func GetGithubLatestReleaseURL(org, repo string) string {
	return "https://api.github.com/repos/" + org + "/" + repo + "/releases/latest"
}

func NewLuxDownloader() GithubDownloader {
	return &nodeDownloader{}
}

func (nodeDownloader) GetDownloadURL(version string, installer Installer) (string, string, error) {
	// NOTE: if any of the underlying URLs change (github changes, release file names, etc.) this fails
	goarch, goos := installer.GetArch()

	var nodeURL string
	var ext string

	switch goos {
	case linux:
		nodeURL = fmt.Sprintf(
			"https://github.com/%s/%s/releases/download/%s/node-linux-%s-%s.tar.gz",
			constants.AvaLabsOrg,
			constants.LuxRepoName,
			version,
			goarch,
			version,
		)
		ext = tarExtension
	case darwin:
		nodeURL = fmt.Sprintf(
			"https://github.com/%s/%s/releases/download/%s/node-macos-%s.zip",
			constants.AvaLabsOrg,
			constants.LuxRepoName,
			version,
			version,
		)
		ext = zipExtension
		// EXPERIMENTAL WIN, no support
	case windows:
		nodeURL = fmt.Sprintf(
			"https://github.com/%s/%s/releases/download/%s/node-win-%s-experimental.zip",
			constants.AvaLabsOrg,
			constants.LuxRepoName,
			version,
			version,
		)
		ext = zipExtension
	default:
		return "", "", fmt.Errorf("OS not supported: %s", goos)
	}

	return nodeURL, ext, nil
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
			constants.AvaLabsOrg,
			constants.SubnetEVMRepoName,
			version,
			constants.SubnetEVMRepoName,
			version[1:], // WARN subnet-evm isn't consistent in its release naming, it's omitting the v in the file name...
			goarch,
		)
	case darwin:
		subnetEVMURL = fmt.Sprintf(
			"https://github.com/%s/%s/releases/download/%s/%s_%s_darwin_%s.tar.gz",
			constants.AvaLabsOrg,
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
