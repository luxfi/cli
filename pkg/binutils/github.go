// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package binutils

import (
	"fmt"

	"github.com/luxfi/constants"
)

const (
	linux   = "linux"
	darwin  = "darwin"
	windows = "windows"

	zipExtension = "zip"
	tarExtension = "tar.gz"
)

// GithubDownloader downloads binaries from GitHub releases.
type GithubDownloader interface {
	GetDownloadURL(version string, installer Installer) (string, string, error)
}

type (
	evmDownloader       struct{}
	nodeDownloader      struct{}
	netrunnerDownloader struct{}
)

var (
	_ GithubDownloader = (*evmDownloader)(nil)
	_ GithubDownloader = (*nodeDownloader)(nil)
	_ GithubDownloader = (*netrunnerDownloader)(nil)
)

// GetGithubLatestReleaseURL returns the GitHub API URL for the latest release.
func GetGithubLatestReleaseURL(org, repo string) string {
	return "https://api.github.com/repos/" + org + "/" + repo + "/releases/latest"
}

// NewLuxDownloader creates a new Lux node downloader.
func NewLuxDownloader() GithubDownloader {
	return &nodeDownloader{}
}

// NewLuxdDownloader is an alias for NewLuxDownloader for compatibility
func NewLuxdDownloader() GithubDownloader {
	return NewLuxDownloader()
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
			constants.LuxOrg,
			constants.LuxRepoName,
			version,
			goarch,
			version,
		)
		ext = tarExtension
	case darwin:
		nodeURL = fmt.Sprintf(
			"https://github.com/%s/%s/releases/download/%s/node-macos-%s.zip",
			constants.LuxOrg,
			constants.LuxRepoName,
			version,
			version,
		)
		ext = zipExtension
		// EXPERIMENTAL WIN, no support
	case windows:
		nodeURL = fmt.Sprintf(
			"https://github.com/%s/%s/releases/download/%s/node-win-%s-experimental.zip",
			constants.LuxOrg,
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

// NewEVMDownloader creates a new EVM downloader.
func NewEVMDownloader() GithubDownloader {
	return &evmDownloader{}
}

func (evmDownloader) GetDownloadURL(version string, installer Installer) (string, string, error) {
	// NOTE: if any of the underlying URLs change (github changes, release file names, etc.) this fails
	goarch, goos := installer.GetArch()

	var evmURL string
	ext := tarExtension

	switch goos {
	case linux:
		evmURL = fmt.Sprintf(
			"https://github.com/%s/%s/releases/download/%s/%s_%s_linux_%s.tar.gz",
			constants.LuxOrg,
			constants.EVMRepoName,
			version,
			constants.EVMRepoName,
			version[1:], // WARN evm isn't consistent in its release naming, it's omitting the v in the file name...
			goarch,
		)
	case darwin:
		evmURL = fmt.Sprintf(
			"https://github.com/%s/%s/releases/download/%s/%s_%s_darwin_%s.tar.gz",
			constants.LuxOrg,
			constants.EVMRepoName,
			version,
			constants.EVMRepoName,
			version[1:],
			goarch,
		)
	default:
		return "", "", fmt.Errorf("OS not supported: %s", goos)
	}

	return evmURL, ext, nil
}

// NewNetrunnerDownloader creates a new downloader for netrunner binaries
func NewNetrunnerDownloader() GithubDownloader {
	return &netrunnerDownloader{}
}

func (netrunnerDownloader) GetDownloadURL(version string, installer Installer) (string, string, error) {
	goarch, goos := installer.GetArch()

	var netrunnerURL string
	ext := tarExtension

	switch goos {
	case linux:
		netrunnerURL = fmt.Sprintf(
			"https://github.com/%s/%s/releases/download/%s/%s_%s_%s.tar.gz",
			constants.LuxOrg,
			constants.NetrunnerRepoName,
			version,
			constants.NetrunnerRepoName,
			goos,
			goarch,
		)
	case darwin:
		netrunnerURL = fmt.Sprintf(
			"https://github.com/%s/%s/releases/download/%s/%s_%s_%s.tar.gz",
			constants.LuxOrg,
			constants.NetrunnerRepoName,
			version,
			constants.NetrunnerRepoName,
			goos,
			goarch,
		)
	default:
		return "", "", fmt.Errorf("OS not supported: %s", goos)
	}

	return netrunnerURL, ext, nil
}
