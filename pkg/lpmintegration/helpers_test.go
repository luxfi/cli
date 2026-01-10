// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package lpmintegration

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetGithubOrg(t *testing.T) {
	type test struct {
		name        string
		url         string
		expectedOrg string
		expectedErr bool
	}

	tests := []test{
		{
			name:        "Success",
			url:         "https://github.com/luxfi/plugins-core.git",
			expectedOrg: "luxfi",
			expectedErr: false,
		},
		{
			name:        "Success",
			url:         "https://github.com/luxfi/plugins-core",
			expectedOrg: "luxfi",
			expectedErr: false,
		},
		{
			name:        "No org",
			url:         "https://github.com/plugins-core",
			expectedOrg: "",
			expectedErr: true,
		},
		{
			name:        "No url path",
			url:         "https://github.com/",
			expectedOrg: "",
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)
			parsedURL, err := url.ParseRequestURI(tt.url)
			require.NoError(err)
			org, err := getGitOrg(parsedURL)
			require.Equal(tt.expectedOrg, org)
			if tt.expectedErr {
				require.Error(err)
			} else {
				require.NoError(err)
			}
		})
	}
}

func TestGetGithubRepo(t *testing.T) {
	type test struct {
		name         string
		url          string
		expectedRepo string
		expectedErr  bool
	}

	tests := []test{
		{
			name:         "Success",
			url:          "https://github.com/luxfi/plugins-core.git",
			expectedRepo: "plugins-core",
			expectedErr:  false,
		},
		{
			name:         "Success",
			url:          "https://github.com/luxfi/plugins-core",
			expectedRepo: "plugins-core",
			expectedErr:  false,
		},
		{
			name:         "No org",
			url:          "https://github.com/plugins-core",
			expectedRepo: "",
			expectedErr:  true,
		},
		{
			name:         "No url path",
			url:          "https://github.com/",
			expectedRepo: "",
			expectedErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)
			parsedURL, err := url.ParseRequestURI(tt.url)
			require.NoError(err)
			repo, err := getGitRepo(parsedURL)
			require.Equal(tt.expectedRepo, repo)
			if tt.expectedErr {
				require.Error(err)
			} else {
				require.NoError(err)
			}
		})
	}
}

func TestGetAlias(t *testing.T) {
	type test struct {
		name          string
		url           string
		expectedAlias string
		expectedErr   bool
	}

	tests := []test{
		{
			name:          "Success",
			url:           "https://github.com/luxfi/plugins-core.git",
			expectedAlias: "luxfi/plugins-core",
			expectedErr:   false,
		},
		{
			name:          "Success",
			url:           "https://github.com/luxfi/plugins-core",
			expectedAlias: "luxfi/plugins-core",
			expectedErr:   false,
		},
		{
			name:          "No org",
			url:           "https://github.com/plugins-core",
			expectedAlias: "",
			expectedErr:   true,
		},
		{
			name:          "No url path",
			url:           "https://github.com/",
			expectedAlias: "",
			expectedErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)
			parsedURL, err := url.ParseRequestURI(tt.url)
			require.NoError(err)
			alias, err := getAlias(parsedURL)
			require.Equal(tt.expectedAlias, alias)
			if tt.expectedErr {
				require.Error(err)
			} else {
				require.NoError(err)
			}
		})
	}
}

func TestSplitKey(t *testing.T) {
	require := require.New(t)

	key := "luxfi/plugins-core:wagmi"
	expectedAlias := "luxfi/plugins-core"
	expectedChain := "wagmi"

	alias, chain, err := splitKey(key)
	require.NoError(err)
	require.Equal(expectedAlias, alias)
	require.Equal(expectedChain, chain)
}

func TestSplitKey_Errpr(t *testing.T) {
	require := require.New(t)

	key := "luxfi/plugins-core_wagmi"

	_, _, err := splitKey(key)
	require.ErrorContains(err, "invalid chain key:")
}
