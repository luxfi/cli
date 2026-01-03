// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

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
	anrutils "github.com/luxfi/netrunner/utils"
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
					BlockchainId: testBlockChainID1,
				},
				"bchain2": {
					BlockchainId: testBlockChainID2,
				},
			},
			Chains: map[string]*rpcpb.ChainInfo{
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
	f, err := os.Create(filepath.Join(binDir, "node")) //nolint:gosec // G304: Test file creation
	require.NoError(err)
	defer func() {
		_ = f.Close()
	}()

	// Create a dummy VM binary that passes validation
	// Must be executable and at least 1KB to pass preflight checks
	vmBinPath := filepath.Join(testDir, "test-vm-binary")
	dummyVMContent := make([]byte, 2048)                 // 2KB of zeros
	err = os.WriteFile(vmBinPath, dummyVMContent, 0o755) //nolint:gosec // G306: Test binary needs to be executable
	require.NoError(err)

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
		vmBin:              vmBinPath, // Set the VM binary path for preflight validation
	}

	// create a simple genesis for the test
	genesis := `{"config":{"chainId":9999},"gasLimit":"0x0","difficulty":"0x0","alloc":{}}`
	// create a dummy genesis file, deploy will check it exists
	testGenesis, err := os.CreateTemp(tmpDir, "test-genesis.json")
	require.NoError(err)
	err = os.WriteFile(testGenesis.Name(), []byte(genesis), constants.DefaultPerms755)
	require.NoError(err)
	// create dummy sidecar file, also checked by deploy
	// Use Custom VM so chainName "test" is used for VM ID computation (matches testVMID)
	sidecar := `{"VM": "Custom"}`
	testSubnetDir := filepath.Join(testDir, constants.ChainsDir, testChainName)
	err = os.MkdirAll(testSubnetDir, constants.DefaultPerms755)
	require.NoError(err)
	testSidecar, err := os.Create(filepath.Join(testSubnetDir, constants.SidecarFileName)) //nolint:gosec // G304: Test file creation
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
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
	fakeHealthResponse := &rpcpb.HealthResponse{
		ClusterInfo: fakeWaitForHealthyResponse.ClusterInfo,
	}
	c.On("LoadSnapshot", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fakeLoadSnapshotResponse, nil)
	c.On("SaveSnapshot", mock.Anything, mock.Anything).Return(fakeSaveSnapshotResponse, nil)
	c.On("RemoveSnapshot", mock.Anything, mock.Anything).Return(fakeRemoveSnapshotResponse, nil)
	// CreateChains takes context and blockchain specs (2 args)
	c.On("CreateChains", mock.Anything, mock.Anything).Return(fakeCreateBlockchainsResponse, nil)
	c.On("URIs", mock.Anything).Return([]string{"fakeUri"}, nil)
	// Health is called by formatChainHealthError for diagnostics
	c.On("Health", mock.Anything).Return(fakeHealthResponse, nil)
	// When fake deploying, the first response needs to have a bogus subnet ID, because
	// otherwise the doDeploy function "aborts" when checking if the subnet had already been deployed.
	// Afterwards, we can set the actual VM ID so that the test returns an expected subnet ID...

	// Return a fake wait for healthy response twice
	c.On("WaitForHealthy", mock.Anything).Return(fakeWaitForHealthyResponse, nil).Twice()
	// Afterwards, change the VmId so that TestDeployToLocal has the correct ID to check
	alteredFakeResponse := proto.Clone(fakeWaitForHealthyResponse).(*rpcpb.WaitForHealthyResponse) // new(rpcpb.WaitForHealthyResponse)
	alteredFakeResponse.ClusterInfo.CustomChains["bchain2"].VmId = testVMID
	alteredFakeResponse.ClusterInfo.CustomChains["bchain2"].ChainName = testChainName
	alteredFakeResponse.ClusterInfo.CustomChains["bchain2"].PchainId = testSubnetID2 // Set the subnet ID
	alteredFakeResponse.ClusterInfo.CustomChains["bchain1"].ChainName = "bchain1"
	c.On("WaitForHealthy", mock.Anything).Return(alteredFakeResponse, nil)
	// Status is called for quick status checks - uses the altered response with VmId set
	fakeStatusResponse := &rpcpb.StatusResponse{
		ClusterInfo: alteredFakeResponse.ClusterInfo,
	}
	c.On("Status", mock.Anything).Return(fakeStatusResponse, nil)
	c.On("Close").Return(nil)
	return c, nil
}

func fakeSetDefaultSnapshot(string, bool) error {
	return nil
}

func TestValidateVMBinary(t *testing.T) {
	require := setupTest(t)

	tmpDir, err := os.MkdirTemp("", "vm-validate-test")
	require.NoError(err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	app := application.New()
	app.Setup(tmpDir, luxlog.NewNoOpLogger(), config.New(), prompts.NewPrompter(), application.NewDownloader())

	deployer := &LocalDeployer{app: app}

	t.Run("valid binary passes", func(_ *testing.T) {
		// Create a valid VM binary (executable, >1KB)
		vmPath := filepath.Join(tmpDir, "valid-vm")
		err := os.WriteFile(vmPath, make([]byte, 2048), 0o755) //nolint:gosec // G306: Test binary needs to be executable
		require.NoError(err)

		vmID, _ := anrutils.VMID("test")
		err = deployer.validateVMBinary(vmPath, vmID)
		require.NoError(err)
	})

	t.Run("missing binary fails", func(_ *testing.T) {
		vmID, _ := anrutils.VMID("test")
		err := deployer.validateVMBinary("/nonexistent/path", vmID)
		require.Error(err)
		require.Contains(err.Error(), "not found")
	})

	t.Run("non-executable binary fails", func(_ *testing.T) {
		vmPath := filepath.Join(tmpDir, "nonexec-vm")
		err := os.WriteFile(vmPath, make([]byte, 2048), 0o644) //nolint:gosec // G306: Test file, intentionally not executable
		require.NoError(err)

		vmID, _ := anrutils.VMID("test")
		err = deployer.validateVMBinary(vmPath, vmID)
		require.Error(err)
		require.Contains(err.Error(), "not executable")
	})

	t.Run("too small binary fails", func(_ *testing.T) {
		vmPath := filepath.Join(tmpDir, "small-vm")
		err := os.WriteFile(vmPath, make([]byte, 100), 0o755) //nolint:gosec // G306: Test binary needs to be executable
		require.NoError(err)

		vmID, _ := anrutils.VMID("test")
		err = deployer.validateVMBinary(vmPath, vmID)
		require.Error(err)
		require.Contains(err.Error(), "too small")
	})
}

func TestDeploymentError(t *testing.T) {
	require := setupTest(t)

	t.Run("error with healthy network", func(_ *testing.T) {
		err := &DeploymentError{
			ChainName:      "mychain",
			Cause:          fmt.Errorf("VM failed to load"),
			NetworkHealthy: true,
		}
		require.Contains(err.Error(), "mychain")
		require.Contains(err.Error(), "network still running")
		require.Contains(err.Error(), "VM failed to load")
	})

	t.Run("error with crashed network", func(_ *testing.T) {
		err := &DeploymentError{
			ChainName:      "mychain",
			Cause:          fmt.Errorf("node stopped unexpectedly"),
			NetworkHealthy: false,
		}
		require.Contains(err.Error(), "mychain")
		require.Contains(err.Error(), "network crashed")
	})

	t.Run("unwrap returns cause", func(_ *testing.T) {
		cause := fmt.Errorf("root cause")
		err := &DeploymentError{ChainName: "test", Cause: cause}
		require.Equal(cause, err.Unwrap())
	})
}
