// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package net

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/luxfi/cli/internal/mocks"
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/binutils"
	"github.com/luxfi/cli/pkg/config"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/prompts"
	"github.com/luxfi/cli/pkg/ux"
	luxlog "github.com/luxfi/log"
	"github.com/luxfi/netrunner/client"
	"github.com/luxfi/netrunner/rpcpb"
	"github.com/luxfi/node/utils/perms"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

var (
	testBlockChainID1 = "S4mBqKYypXfnCX7drcacHmSJFneYYrqCTfq3fkVwPnPHVqQ2y"
	testBlockChainID2 = "11111111111111111111111111111111LpoYY"
	testSubnetID1     = "XDnPSGJr2XmkkFaBEGcKFmJgtH8Fv7rNa6YFRKxCHQsUV6Egp"
	testSubnetID2     = "2LSwchh6dK64RtGRXVdjyDd9YPu89mXB2MMjpZ1dDvnKZDYyro"

	testVMID      = "tGBrM2SXkAdNsqzb3SaFZZWMNdzjjFEUKteheTa4dhUwnfQyu" // VM ID of "test"
	testChainName = "test"

	fakeWaitForHealthyResponse = &rpcpb.WaitForHealthyResponse{
		ClusterInfo: &rpcpb.ClusterInfo{
			Healthy:             true, // currently actually not checked, should it, if CustomVMsHealthy already is?
			CustomChainsHealthy: true,
			NodeNames:           []string{"testNode1", "testNode2"},
			NodeInfos: map[string]*rpcpb.NodeInfo{
				"testNode1": {
					Name: "testNode1",
					Uri:  "http://fake.localhost:12345",
				},
				"testNode2": {
					Name: "testNode2",
					Uri:  "http://fake.localhost:12345",
				},
			},
			CustomChains: map[string]*rpcpb.CustomChainInfo{
				"bchain1": {
					ChainId: testBlockChainID1,
				},
				"bchain2": {
					ChainId: testBlockChainID2,
				},
			},
			Subnets: map[string]*rpcpb.SubnetInfo{
				testSubnetID1: {},
				testSubnetID2: {},
			},
		},
	}
)

func setupTest(t *testing.T) *require.Assertions {
	// use io.Discard to not print anything
	ux.NewUserLog(luxlog.NewNoOpLogger(), io.Discard)
	return require.New(t)
}

func TestDeployToLocal(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	require := setupTest(t)
	luxVersion := "v1.18.0"

	// fake-return true simulating the process is running
	procChecker := &mocks.ProcessChecker{}
	procChecker.On("IsServerProcessRunning", mock.Anything).Return(true, nil)

	tmpDir := os.TempDir()
	testDir, err := os.MkdirTemp(tmpDir, "local-test")
	require.NoError(err)
	defer func() {
		err = os.RemoveAll(testDir)
		require.NoError(err)
	}()

	app := application.New()
	app.Setup(testDir, luxlog.NewNoOpLogger(), config.New(), prompts.NewPrompter(), application.NewDownloader())

	binDir := filepath.Join(app.GetLuxBinDir(), "node-"+luxVersion)

	// create a dummy plugins dir, deploy will check it exists
	binChecker := &mocks.BinaryChecker{}
	err = os.MkdirAll(filepath.Join(binDir, "plugins"), perms.ReadWriteExecute)
	require.NoError(err)

	// create a dummy node file, deploy will check it exists
	f, err := os.Create(filepath.Join(binDir, "node"))
	require.NoError(err)
	defer func() {
		_ = f.Close()
	}()

	binChecker.On("ExistsWithLatestVersion", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(true, tmpDir, nil)

	binDownloader := &mocks.PluginBinaryDownloader{}
	binDownloader.On("Download", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	binDownloader.On("InstallVM", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	testDeployer := &LocalDeployer{
		procChecker:        procChecker,
		binChecker:         binChecker,
		getClientFunc:      getTestClientFunc,
		binaryDownloader:   binDownloader,
		app:                app,
		setDefaultSnapshot: fakeSetDefaultSnapshot,
		luxVersion:         luxVersion,
	}

	// create a simple genesis for the test
	genesis := `{"config":{"chainId":9999},"gasLimit":"0x0","difficulty":"0x0","alloc":{}}`
	// create a dummy genesis file, deploy will check it exists
	testGenesis, err := os.CreateTemp(tmpDir, "test-genesis.json")
	require.NoError(err)
	err = os.WriteFile(testGenesis.Name(), []byte(genesis), constants.DefaultPerms755)
	require.NoError(err)
	// create dummy sidecar file, also checked by deploy
	sidecar := `{"VM": "EVM"}`
	testSubnetDir := filepath.Join(testDir, constants.NetDir, testChainName)
	err = os.MkdirAll(testSubnetDir, constants.DefaultPerms755)
	require.NoError(err)
	testSidecar, err := os.Create(filepath.Join(testSubnetDir, constants.SidecarFileName))
	require.NoError(err)
	err = os.WriteFile(testSidecar.Name(), []byte(sidecar), constants.DefaultPerms755)
	require.NoError(err)
	// test actual deploy
	s, b, err := testDeployer.DeployToLocalNetwork(testChainName, []byte(genesis), testGenesis.Name())
	require.NoError(err)
	require.Equal(testSubnetID2, s.String())
	require.Equal(testBlockChainID2, b.String())
}

func TestGetLatestLuxVersion(t *testing.T) {
	require := setupTest(t)

	testVersion := "v1.99.9999"
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := fmt.Sprintf(`{"some":"unimportant","fake":"data","tag_name":"%s","tag_name_was":"what we are interested in"}`, testVersion)
		_, err := w.Write([]byte(resp))
		require.NoError(err)
	})
	s := httptest.NewServer(testHandler)
	defer s.Close()

	dl := application.NewDownloader()
	v, err := dl.GetLatestReleaseVersion(s.URL)
	require.NoError(err)
	require.Equal(v, testVersion)
}

func getTestClientFunc(...binutils.GRPCClientOpOption) (client.Client, error) {
	c := &mocks.Client{}
	fakeLoadSnapshotResponse := &rpcpb.LoadSnapshotResponse{}
	fakeSaveSnapshotResponse := &rpcpb.SaveSnapshotResponse{}
	fakeRemoveSnapshotResponse := &rpcpb.RemoveSnapshotResponse{}
	fakeCreateBlockchainsResponse := &rpcpb.CreateBlockchainsResponse{}
	c.On("LoadSnapshot", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fakeLoadSnapshotResponse, nil)
	c.On("SaveSnapshot", mock.Anything, mock.Anything).Return(fakeSaveSnapshotResponse, nil)
	c.On("RemoveSnapshot", mock.Anything, mock.Anything).Return(fakeRemoveSnapshotResponse, nil)
	c.On("CreateBlockchains", mock.Anything, mock.Anything, mock.Anything).Return(fakeCreateBlockchainsResponse, nil)
	c.On("URIs", mock.Anything).Return([]string{"fakeUri"}, nil)
	// When fake deploying, the first response needs to have a bogus subnet ID, because
	// otherwise the doDeploy function "aborts" when checking if the subnet had already been deployed.
	// Afterwards, we can set the actual VM ID so that the test returns an expected subnet ID...

	// Return a fake wait for healthy response twice
	c.On("WaitForHealthy", mock.Anything).Return(fakeWaitForHealthyResponse, nil).Twice()
	// Afterwards, change the VmId so that TestDeployToLocal has the correct ID to check
	alteredFakeResponse := proto.Clone(fakeWaitForHealthyResponse).(*rpcpb.WaitForHealthyResponse) // new(rpcpb.WaitForHealthyResponse)
	alteredFakeResponse.ClusterInfo.CustomChains["bchain2"].VmId = testVMID
	alteredFakeResponse.ClusterInfo.CustomChains["bchain2"].ChainName = testChainName
	alteredFakeResponse.ClusterInfo.CustomChains["bchain1"].ChainName = "bchain1"
	c.On("WaitForHealthy", mock.Anything).Return(alteredFakeResponse, nil)
	c.On("Close").Return(nil)
	return c, nil
}

func fakeSetDefaultSnapshot(string, bool) error {
	return nil
}
