// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package lpmintegration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/sdk/prompts"
	luxlog "github.com/luxfi/log"
	"github.com/stretchr/testify/require"
)

const (
	org1 = "org1"
	org2 = "org2"

	repo1 = "repo1"
	repo2 = "repo2"

	subnet1 = "testsubnet1"
	subnet2 = "testsubnet2"

	vm = "testvm"

	testSubnetYaml = `subnet:
  id: "abcd"
  alias: "testsubnet"
  homepage: "https://subnet.com"
  description: It's a subnet
  maintainers:
    - "dev@subnet.com"
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

func TestGetSubnets(t *testing.T) {
	type test struct {
		name        string
		org         string
		repo        string
		subnetNames []string
	}

	tests := []test{
		{
			name:        "Single",
			org:         org1,
			repo:        repo1,
			subnetNames: []string{subnet1},
		},
		{
			name:        "Multiple",
			org:         org1,
			repo:        repo1,
			subnetNames: []string{subnet1, subnet2},
		},
		{
			name:        "Empty",
			org:         org1,
			repo:        repo1,
			subnetNames: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)

			testDir := t.TempDir()
			app := newTestApp(t, testDir)

			// Setup subnet directory
			subnetPath := filepath.Join(testDir, "repositories", tt.org, tt.repo, "subnets")
			err := os.MkdirAll(subnetPath, constants.DefaultPerms755)
			require.NoError(err)

			// Create subnet files
			for _, subnet := range tt.subnetNames {
				subnetFile := filepath.Join(subnetPath, subnet+".yaml")
				err = os.WriteFile(subnetFile, []byte(testSubnetYaml), constants.DefaultPerms755)
				require.NoError(err)
			}

			subnets, err := GetSubnets(app, makeAlias(tt.org, tt.repo))
			require.NoError(err)

			// check results
			require.Equal(len(tt.subnetNames), len(subnets))
			for i, subnet := range tt.subnetNames {
				require.Equal(tt.subnetNames[i], subnet)
			}
		})
	}
}

func TestLoadSubnetFile_Success(t *testing.T) {
	require := require.New(t)

	testDir := t.TempDir()
	app := newTestApp(t, testDir)

	// Setup subnet directory
	subnetPath := filepath.Join(testDir, "repositories", org1, repo1, "subnets")
	err := os.MkdirAll(subnetPath, constants.DefaultPerms755)
	require.NoError(err)

	// Create subnet files
	subnetFile := filepath.Join(subnetPath, subnet1+".yaml")
	err = os.WriteFile(subnetFile, []byte(testSubnetYaml), constants.DefaultPerms755)
	require.NoError(err)

	expectedSubnet := Subnet{
		ID:          "abcd",
		Alias:       "testsubnet",
		Description: "It's a subnet",
		VMs:         []string{"testvm1", "testvm2"},
	}

	loadedSubnet, err := LoadSubnetFile(app, MakeKey(makeAlias(org1, repo1), subnet1))
	require.NoError(err)
	require.Equal(expectedSubnet, loadedSubnet)
}

func TestLoadSubnetFile_BadKey(t *testing.T) {
	require := require.New(t)

	testDir := t.TempDir()
	app := newTestApp(t, testDir)

	// Setup subnet directory
	subnetPath := filepath.Join(testDir, "repositories", org1, repo1, "subnets")
	err := os.MkdirAll(subnetPath, constants.DefaultPerms755)
	require.NoError(err)

	// Create subnet files
	subnetFile := filepath.Join(subnetPath, subnet1+".yaml")
	err = os.WriteFile(subnetFile, []byte(testSubnetYaml), constants.DefaultPerms755)
	require.NoError(err)

	_, err = LoadSubnetFile(app, subnet1)
	require.ErrorContains(err, "invalid subnet key")
}

func TestGetVMsInSubnet(t *testing.T) {
	require := require.New(t)

	testDir := t.TempDir()
	app := newTestApp(t, testDir)

	// Setup subnet directory
	subnetPath := filepath.Join(testDir, "repositories", org1, repo1, "subnets")
	err := os.MkdirAll(subnetPath, constants.DefaultPerms755)
	require.NoError(err)

	// Create subnet files
	subnetFile := filepath.Join(subnetPath, subnet1+".yaml")
	err = os.WriteFile(subnetFile, []byte(testSubnetYaml), constants.DefaultPerms755)
	require.NoError(err)

	expectedVMs := []string{"testvm1", "testvm2"}

	loadedVMs, err := getVMsInSubnet(app, MakeKey(makeAlias(org1, repo1), subnet1))
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

	// Create subnet files
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
