// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package lpmintegration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/prompts"
	"github.com/luxfi/constants"
	luxlog "github.com/luxfi/log"
	"github.com/stretchr/testify/require"
)

const (
	org1 = "org1"
	org2 = "org2"

	repo1 = "repo1"
	repo2 = "repo2"

	chain1 = "testchain1"
	chain2 = "testchain2"

	vm = "testvm"

	testChainYaml = `chain:
  id: "abcd"
  alias: "testchain"
  homepage: "https://chain.com"
  description: It's a chain
  maintainers:
    - "dev@chain.com"
  vms:
    - "testvm1"
    - "testvm2"
`

	testVMYaml = `vm:
  id: "efgh"
  alias: "testvm"
  description: "Virtual machine"
  binary: "build/sqja3uK17MJxfC7AN8nGadBw9JK5BcrsNwNynsqP5Gih8M5Bm"
  url: "https://github.com/org/repo/archive/refs/tags/v1.0.0.tar.gz"
  checksum: "1ac250f6c40472f22eaf0616fc8c886078a4eaa9b2b85fbb4fb7783a1db6af3f"
  version: "v1.0.0"
`
)

func newTestApp(t *testing.T, testDir string) *application.Lux {
	tempDir := t.TempDir()
	app := application.New()
	app.Setup(tempDir, luxlog.NewNoOpLogger(), nil, prompts.NewPrompter(), application.NewDownloader())
	app.LpmDir = testDir
	return app
}

func TestGetRepos(t *testing.T) {
	type test struct {
		name  string
		orgs  []string
		repos []string
	}

	tests := []test{
		{
			name:  "Single",
			orgs:  []string{org1},
			repos: []string{repo1},
		},
		{
			name:  "Multiple",
			orgs:  []string{org1, org2},
			repos: []string{repo1, repo2},
		},
		{
			name:  "Empty",
			orgs:  []string{},
			repos: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)

			testDir := t.TempDir()
			app := newTestApp(t, testDir)

			repositoryDir := filepath.Join(testDir, "repositories")
			err := os.Mkdir(repositoryDir, constants.DefaultPerms755)
			require.NoError(err)

			// create repos
			for _, org := range tt.orgs {
				for _, repo := range tt.repos {
					repoPath := filepath.Join(repositoryDir, org, repo)
					err = os.MkdirAll(repoPath, constants.DefaultPerms755)
					require.NoError(err)
				}
			}

			// test function
			repos, err := GetRepos(app)
			require.NoError(err)

			// check results
			numRepos := len(tt.orgs) * len(tt.repos)
			require.Equal(numRepos, len(repos))

			index := 0
			for _, org := range tt.orgs {
				for _, repo := range tt.repos {
					require.Equal(org+"/"+repo, repos[index])
					index++
				}
			}
		})
	}
}

func TestGetChains(t *testing.T) {
	type test struct {
		name       string
		org        string
		repo       string
		chainNames []string
	}

	tests := []test{
		{
			name:       "Single",
			org:        org1,
			repo:       repo1,
			chainNames: []string{chain1},
		},
		{
			name:       "Multiple",
			org:        org1,
			repo:       repo1,
			chainNames: []string{chain1, chain2},
		},
		{
			name:       "Empty",
			org:        org1,
			repo:       repo1,
			chainNames: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)

			testDir := t.TempDir()
			app := newTestApp(t, testDir)

			// Setup chain directory
			chainPath := filepath.Join(testDir, "repositories", tt.org, tt.repo, "chains")
			err := os.MkdirAll(chainPath, constants.DefaultPerms755)
			require.NoError(err)

			// Create chain files
			for _, chain := range tt.chainNames {
				chainFile := filepath.Join(chainPath, chain+".yaml")
				err = os.WriteFile(chainFile, []byte(testChainYaml), constants.DefaultPerms755)
				require.NoError(err)
			}

			chains, err := GetChains(app, makeAlias(tt.org, tt.repo))
			require.NoError(err)

			// check results
			require.Equal(len(tt.chainNames), len(chains))
			for i, chain := range tt.chainNames {
				require.Equal(tt.chainNames[i], chain)
			}
		})
	}
}

func TestLoadChainFile_Success(t *testing.T) {
	require := require.New(t)

	testDir := t.TempDir()
	app := newTestApp(t, testDir)

	// Setup chain directory
	chainPath := filepath.Join(testDir, "repositories", org1, repo1, "chains")
	err := os.MkdirAll(chainPath, constants.DefaultPerms755)
	require.NoError(err)

	// Create chain files
	chainFile := filepath.Join(chainPath, chain1+".yaml")
	err = os.WriteFile(chainFile, []byte(testChainYaml), constants.DefaultPerms755)
	require.NoError(err)

	expectedChain := Chain{
		ID:          "abcd",
		Alias:       "testchain",
		Description: "It's a chain",
		VMs:         []string{"testvm1", "testvm2"},
	}

	loadedChain, err := LoadChainFile(app, MakeKey(makeAlias(org1, repo1), chain1))
	require.NoError(err)
	require.Equal(expectedChain, loadedChain)
}

func TestLoadChainFile_BadKey(t *testing.T) {
	require := require.New(t)

	testDir := t.TempDir()
	app := newTestApp(t, testDir)

	// Setup chain directory
	chainPath := filepath.Join(testDir, "repositories", org1, repo1, "chains")
	err := os.MkdirAll(chainPath, constants.DefaultPerms755)
	require.NoError(err)

	// Create chain files
	chainFile := filepath.Join(chainPath, chain1+".yaml")
	err = os.WriteFile(chainFile, []byte(testChainYaml), constants.DefaultPerms755)
	require.NoError(err)

	_, err = LoadChainFile(app, chain1)
	require.ErrorContains(err, "invalid chain key")
}

func TestGetVMsInChain(t *testing.T) {
	require := require.New(t)

	testDir := t.TempDir()
	app := newTestApp(t, testDir)

	// Setup chain directory
	chainPath := filepath.Join(testDir, "repositories", org1, repo1, "chains")
	err := os.MkdirAll(chainPath, constants.DefaultPerms755)
	require.NoError(err)

	// Create chain files
	chainFile := filepath.Join(chainPath, chain1+".yaml")
	err = os.WriteFile(chainFile, []byte(testChainYaml), constants.DefaultPerms755)
	require.NoError(err)

	expectedVMs := []string{"testvm1", "testvm2"}

	loadedVMs, err := getVMsInChain(app, MakeKey(makeAlias(org1, repo1), chain1))
	require.NoError(err)
	require.Equal(expectedVMs, loadedVMs)
}

func TestLoadVMFile(t *testing.T) {
	require := require.New(t)

	testDir := t.TempDir()
	app := newTestApp(t, testDir)

	// Setup vm directory
	vmPath := filepath.Join(testDir, "repositories", org1, repo1, "vms")
	err := os.MkdirAll(vmPath, constants.DefaultPerms755)
	require.NoError(err)

	// Create chain files
	vmFile := filepath.Join(vmPath, vm+".yaml")
	err = os.WriteFile(vmFile, []byte(testVMYaml), constants.DefaultPerms755)
	require.NoError(err)

	expectedVM := VM{
		ID:          "efgh",
		Alias:       vm,
		Description: "Virtual machine",
		Binary:      "build/sqja3uK17MJxfC7AN8nGadBw9JK5BcrsNwNynsqP5Gih8M5Bm",
		URL:         "https://github.com/org/repo/archive/refs/tags/v1.0.0.tar.gz",
		Checksum:    "1ac250f6c40472f22eaf0616fc8c886078a4eaa9b2b85fbb4fb7783a1db6af3f",
		Version:     "v1.0.0",
	}

	loadedVM, err := LoadVMFile(app, makeAlias(org1, repo1), vm)
	require.NoError(err)
	require.Equal(expectedVM, loadedVM)
}
