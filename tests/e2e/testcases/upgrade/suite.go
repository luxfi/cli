// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package lpm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/luxfi/cli/cmd/chaincmd/upgradecmd"
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/binutils"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/tests/e2e/commands"
	"github.com/luxfi/cli/tests/e2e/utils"
	"github.com/luxfi/evm/params/extras"
	"github.com/luxfi/ids"
	luxlog "github.com/luxfi/log"
	anr_utils "github.com/luxfi/netrunner/utils"
	"github.com/luxfi/sdk/models"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

const (
	subnetName       = "e2eSubnetTest"
	secondSubnetName = "e2eSecondSubnetTest"

	evmVersion1 = "v0.4.7"
	evmVersion2 = "v0.4.8"

	luxRPC1Version = "v1.9.5"
	luxRPC2Version = "v1.9.8"

	controlKeys = "P-custom18jma8ppw3nhx5r4ap8clazz0dps7rv5u9xde7p"
	keyName     = "ewoq"

	upgradeBytesPath = "tests/e2e/assets/test_upgrade.json"

	upgradeBytesPath2 = "tests/e2e/assets/test_upgrade_2.json"
)

var (
	binaryToVersion map[string]string
	err             error
)

// need to have this outside the normal suite because of the BeforeEach
var _ = ginkgo.Describe("[Upgrade expect network failure]", ginkgo.Ordered, func() {
	ginkgo.AfterEach(func() {
		commands.CleanNetworkHard()
		err := utils.DeleteConfigs(subnetName)
		gomega.Expect(err).Should(gomega.BeNil())
	})

	ginkgo.It("fails on stopped network", func() {
		commands.CreateEVMConfig(subnetName, utils.EVMGenesisPath)

		_, err = commands.ImportUpgradeBytes(subnetName, upgradeBytesPath)
		gomega.Expect(err).Should(gomega.BeNil())

		// we want to simulate a situation here where the subnet has been deployed
		// but the network is stopped
		// the code would detect it hasn't been deployed yet so report that error first
		// therefore we can just manually edit the file to fake it had been deployed
		app := application.New()
		app.Setup(utils.GetBaseDir(), luxlog.NewNoOpLogger(), nil, nil, nil)
		sc := models.Sidecar{
			Name:     subnetName,
			Subnet:   subnetName,
			Networks: make(map[string]models.NetworkData),
		}
		sc.Networks[models.Local.String()] = models.NetworkData{
			SubnetID:     ids.GenerateTestID(),
			BlockchainID: ids.GenerateTestID(),
		}
		err = app.UpdateSidecar(&sc)
		gomega.Expect(err).Should(gomega.BeNil())

		out, err := commands.ApplyUpgradeLocal(subnetName)
		gomega.Expect(err).Should(gomega.HaveOccurred())
		gomega.Expect(out).Should(gomega.ContainSubstring(binutils.ErrGRPCTimeout.Error()))
	})
})

// upgrade a public network
// the approach is rather simple: import the upgrade file,
// call the apply command which "just" installs the file at an expected path,
// and then check the file is there and has the correct content.
var _ = ginkgo.Describe("[Upgrade public network]", ginkgo.Ordered, func() {
	ginkgo.AfterEach(func() {
		commands.CleanNetworkHard()
		err := utils.DeleteConfigs(subnetName)
		gomega.Expect(err).Should(gomega.BeNil())
	})

	ginkgo.It("can create and apply to public node", func() {
		commands.CreateEVMConfig(subnetName, utils.EVMGenesisPath)

		// simulate as if this had already been deployed to testnet
		// by just entering fake data into the struct
		app := application.New()
		app.Setup(utils.GetBaseDir(), luxlog.NewNoOpLogger(), nil, nil, nil)

		sc, err := app.LoadSidecar(subnetName)
		gomega.Expect(err).Should(gomega.BeNil())

		blockchainID := ids.GenerateTestID()
		sc.Networks = make(map[string]models.NetworkData)
		sc.Networks[models.Testnet.String()] = models.NetworkData{
			SubnetID:     ids.GenerateTestID(),
			BlockchainID: blockchainID,
		}
		err = app.UpdateSidecar(&sc)
		gomega.Expect(err).Should(gomega.BeNil())

		// import the upgrade bytes file so have one
		_, err = commands.ImportUpgradeBytes(subnetName, upgradeBytesPath)
		gomega.Expect(err).Should(gomega.BeNil())

		// we'll set a fake chain config dir to not mess up with a potential real one
		// in the system
		nodeConfigDir, err := os.MkdirTemp("", "cli-tmp-lux-conf-dir")
		gomega.Expect(err).Should(gomega.BeNil())
		defer func() { _ = os.RemoveAll(nodeConfigDir) }()

		// now we try to apply
		_, err = commands.ApplyUpgradeToPublicNode(subnetName, nodeConfigDir)
		gomega.Expect(err).Should(gomega.BeNil())

		// we expect the file to be present at the expected location and being
		// the same content as the original one
		expectedPath := filepath.Join(nodeConfigDir, blockchainID.String(), constants.UpgradeBytesFileName)
		gomega.Expect(expectedPath).Should(gomega.BeARegularFile())
		ori, err := os.ReadFile(upgradeBytesPath)
		gomega.Expect(err).Should(gomega.BeNil())
		cp, err := os.ReadFile(expectedPath) //nolint:gosec // G304: Test code reading from test directories
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(ori).Should(gomega.Equal(cp))
	})
})

var _ = ginkgo.Describe("[Upgrade local network]", ginkgo.Ordered, func() {
	_ = ginkgo.BeforeAll(func() {
		mapper := utils.NewVersionMapper()
		binaryToVersion, err = utils.GetVersionMapping(mapper)
		gomega.Expect(err).Should(gomega.BeNil())
	})

	ginkgo.BeforeEach(func() {
		output, err := commands.CreateKeyFromPath(keyName, utils.LocalKeyPath)
		if err != nil {
			fmt.Println(output)
			utils.PrintStdErr(err)
		}
		gomega.Expect(err).Should(gomega.BeNil())
	})

	ginkgo.AfterEach(func() {
		commands.CleanNetworkHard()
		err := utils.DeleteConfigs(subnetName)
		gomega.Expect(err).Should(gomega.BeNil())
		err = utils.DeleteConfigs(secondSubnetName)
		gomega.Expect(err).Should(gomega.BeNil())
		_ = utils.DeleteKey(keyName)
		utils.DeleteCustomBinary(subnetName)
	})

	ginkgo.It("fails on undeployed subnet", func() {
		commands.CreateEVMConfig(subnetName, utils.EVMGenesisPath)

		_, err = commands.ImportUpgradeBytes(subnetName, upgradeBytesPath)
		gomega.Expect(err).Should(gomega.BeNil())

		_ = commands.StartNetwork()

		out, err := commands.ApplyUpgradeLocal(subnetName)
		gomega.Expect(err).Should(gomega.HaveOccurred())
		gomega.Expect(out).Should(gomega.ContainSubstring(upgradecmd.ErrSubnetNotDeployedOutput))
	})

	ginkgo.It("can create and apply to locally running subnet", func() {
		commands.CreateEVMConfig(subnetName, utils.EVMGenesisPath)

		deployOutput := commands.DeploySubnetLocally(subnetName)

		_, err = commands.ImportUpgradeBytes(subnetName, upgradeBytesPath)
		gomega.Expect(err).Should(gomega.BeNil())

		_, err = commands.ApplyUpgradeLocal(subnetName)
		gomega.Expect(err).Should(gomega.BeNil())

		upgradeBytes, err := os.ReadFile(upgradeBytesPath)
		gomega.Expect(err).Should(gomega.BeNil())

		var precmpUpgrades extras.UpgradeConfig
		err = json.Unmarshal(upgradeBytes, &precmpUpgrades)
		gomega.Expect(err).Should(gomega.BeNil())

		rpcs, err := utils.ParseRPCsFromOutput(deployOutput)
		if err != nil {
			fmt.Println(deployOutput)
		}
		err = utils.CheckUpgradeIsDeployed(rpcs[0], precmpUpgrades)
		gomega.Expect(err).Should(gomega.BeNil())

		app := application.New()
		app.Setup(utils.GetBaseDir(), luxlog.NewNoOpLogger(), nil, nil, nil)

		stripped := stripWhitespaces(string(upgradeBytes))
		lockUpgradeBytes, err := app.ReadLockUpgradeFile(subnetName)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect([]byte(stripped)).Should(gomega.Equal(lockUpgradeBytes))
	})

	ginkgo.It("can't upgrade transactionAllowList precompile because admin address doesn't have enough token", func() {
		commands.CreateEVMConfig(subnetName, utils.EVMGenesisPath)

		commands.DeploySubnetLocally(subnetName)

		_, err = commands.ImportUpgradeBytes(subnetName, upgradeBytesPath2)
		gomega.Expect(err).Should(gomega.BeNil())

		_, err = commands.ApplyUpgradeLocal(subnetName)
		gomega.Expect(err).Should(gomega.HaveOccurred())
	})

	ginkgo.It("can upgrade transactionAllowList precompile because admin address has enough tokens", func() {
		commands.CreateEVMConfig(subnetName, utils.EVMGenesisPath)

		commands.DeploySubnetLocally(subnetName)

		_, err = commands.ImportUpgradeBytes(subnetName, upgradeBytesPath)
		gomega.Expect(err).Should(gomega.BeNil())

		_, err = commands.ApplyUpgradeLocal(subnetName)
		gomega.Expect(err).Should(gomega.BeNil())
	})

	ginkgo.It("can create and update future", func() {
		evmVersion1 := binaryToVersion[utils.SoloEVMKey1]
		evmVersion2 := binaryToVersion[utils.SoloEVMKey2]
		commands.CreateEVMConfigWithVersion(subnetName, utils.EVMGenesisPath, evmVersion1)

		// check version
		output, err := commands.DescribeSubnet(subnetName)
		gomega.Expect(err).Should(gomega.BeNil())

		containsVersion1 := strings.Contains(output, evmVersion1)
		containsVersion2 := strings.Contains(output, evmVersion2)
		gomega.Expect(containsVersion1).Should(gomega.BeTrue())
		gomega.Expect(containsVersion2).Should(gomega.BeFalse())

		_, err = commands.UpgradeVMConfig(subnetName, evmVersion2)
		gomega.Expect(err).Should(gomega.BeNil())

		output, err = commands.DescribeSubnet(subnetName)
		gomega.Expect(err).Should(gomega.BeNil())

		containsVersion1 = strings.Contains(output, evmVersion1)
		containsVersion2 = strings.Contains(output, evmVersion2)
		gomega.Expect(containsVersion1).Should(gomega.BeFalse())
		gomega.Expect(containsVersion2).Should(gomega.BeTrue())

		commands.DeleteSubnetConfig(subnetName)
	})

	ginkgo.It("upgrade EVM local deployment", func() {
		commands.CreateEVMConfigWithVersion(subnetName, utils.EVMGenesisPath, evmVersion1)
		deployOutput := commands.DeploySubnetLocally(subnetName)
		rpcs, err := utils.ParseRPCsFromOutput(deployOutput)
		if err != nil {
			fmt.Println(deployOutput)
		}

		// check running version
		// remove string suffix starting with /ext
		nodeURI := strings.Split(rpcs[0], "/ext")[0]
		vmid, err := anr_utils.VMID(subnetName)
		gomega.Expect(err).Should(gomega.BeNil())
		version, err := utils.GetNodeVMVersion(nodeURI, vmid.String())
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(version).Should(gomega.Equal(evmVersion1))

		// stop network
		_ = commands.StopNetwork()

		// upgrade
		commands.UpgradeVMLocal(subnetName, evmVersion2)

		// restart network
		commands.StartNetwork()

		// check running version
		version, err = utils.GetNodeVMVersion(nodeURI, vmid.String())
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(version).Should(gomega.Equal(evmVersion2))

		commands.DeleteSubnetConfig(subnetName)
	})

	ginkgo.It("upgrade custom vm local deployment", func() {
		// download vm bins
		customVMPath1, err := utils.DownloadCustomVMBin(evmVersion1)
		gomega.Expect(err).Should(gomega.BeNil())
		customVMPath2, err := utils.DownloadCustomVMBin(evmVersion2)
		gomega.Expect(err).Should(gomega.BeNil())

		// create and deploy
		commands.CreateCustomVMConfig(subnetName, utils.EVMGenesisPath, customVMPath1)
		// need to set lux version manually since VMs are custom
		commands.StartNetworkWithVersion(luxRPC1Version)
		deployOutput := commands.DeploySubnetLocally(subnetName)
		rpcs, err := utils.ParseRPCsFromOutput(deployOutput)
		if err != nil {
			fmt.Println(deployOutput)
		}

		// check running version
		// remove string suffix starting with /ext from rpc url to get node uri
		nodeURI := strings.Split(rpcs[0], "/ext")[0]
		vmid, err := anr_utils.VMID(subnetName)
		gomega.Expect(err).Should(gomega.BeNil())
		version, err := utils.GetNodeVMVersion(nodeURI, vmid.String())
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(version).Should(gomega.Equal(evmVersion1))

		// stop network
		_ = commands.StopNetwork()

		// upgrade
		commands.UpgradeCustomVMLocal(subnetName, customVMPath2)

		// restart network
		commands.StartNetworkWithVersion(luxRPC2Version)

		// check running version
		version, err = utils.GetNodeVMVersion(nodeURI, vmid.String())
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(version).Should(gomega.Equal(evmVersion2))

		commands.DeleteSubnetConfig(subnetName)
	})

	ginkgo.It("can update a evm to a custom VM", func() {
		customVMPath, err := utils.DownloadCustomVMBin(binaryToVersion[utils.SoloEVMKey2])
		gomega.Expect(err).Should(gomega.BeNil())

		commands.CreateEVMConfigWithVersion(
			subnetName,
			utils.EVMGenesisPath,
			binaryToVersion[utils.SoloEVMKey1],
		)

		// check version
		output, err := commands.DescribeSubnet(subnetName)
		gomega.Expect(err).Should(gomega.BeNil())

		containsVersion1 := strings.Contains(output, binaryToVersion[utils.SoloEVMKey1])
		containsVersion2 := strings.Contains(output, binaryToVersion[utils.SoloEVMKey2])
		gomega.Expect(containsVersion1).Should(gomega.BeTrue())
		gomega.Expect(containsVersion2).Should(gomega.BeFalse())

		_, err = commands.UpgradeCustomVM(subnetName, customVMPath)
		gomega.Expect(err).Should(gomega.BeNil())

		output, err = commands.DescribeSubnet(subnetName)
		gomega.Expect(err).Should(gomega.BeNil())

		containsVersion2 = strings.Contains(output, binaryToVersion[utils.SoloEVMKey2])
		gomega.Expect(containsVersion2).Should(gomega.BeFalse())
		// the following indicates it is a custom VM
		containsCustomVM := strings.Contains(output, "Printing genesis")
		gomega.Expect(containsCustomVM).Should(gomega.BeTrue())

		commands.DeleteSubnetConfig(subnetName)
	})

	ginkgo.It("can upgrade evm on public deployment", func() {
		_ = commands.StartNetworkWithVersion(binaryToVersion[utils.SoloLuxKey])
		commands.CreateEVMConfigWithVersion(subnetName, utils.EVMGenesisPath, binaryToVersion[utils.SoloEVMKey1])

		// Simulate testnet deployment
		s := commands.SimulateTestnetDeploy(subnetName, keyName, controlKeys)
		subnetID, err := utils.ParsePublicDeployOutput(s, utils.SubnetIDParseType)
		gomega.Expect(err).Should(gomega.BeNil())
		// add validators to subnet
		nodeInfos, err := utils.GetNodesInfo()
		gomega.Expect(err).Should(gomega.BeNil())
		for _, nodeInfo := range nodeInfos {
			start := time.Now().Add(time.Second * 30).UTC().Format("2006-01-02 15:04:05")
			_ = commands.SimulateTestnetAddValidator(subnetName, keyName, nodeInfo.ID, start, "24h", "20")
		}
		// join to copy vm binary and update config file
		for _, nodeInfo := range nodeInfos {
			_ = commands.SimulateTestnetJoin(subnetName, nodeInfo.ConfigFile, nodeInfo.PluginDir, nodeInfo.ID)
		}
		// get and check whitelisted subnets from config file
		var whitelistedSubnets string
		for _, nodeInfo := range nodeInfos {
			whitelistedSubnets, err = utils.GetWhitelistedSubnetsFromConfigFile(nodeInfo.ConfigFile)
			gomega.Expect(err).Should(gomega.BeNil())
			whitelistedSubnetsSlice := strings.Split(whitelistedSubnets, ",")
			gomega.Expect(whitelistedSubnetsSlice).Should(gomega.ContainElement(subnetID))
		}
		// update nodes whitelisted subnets
		err = utils.RestartNodesWithWhitelistedChains(whitelistedSubnets)
		gomega.Expect(err).Should(gomega.BeNil())
		// wait for subnet walidators to be up
		err = utils.WaitSubnetValidators(subnetID, nodeInfos)
		gomega.Expect(err).Should(gomega.BeNil())

		var originalHash string

		// upgrade the vm on each node
		vmid, err := anr_utils.VMID(subnetName)
		gomega.Expect(err).Should(gomega.BeNil())

		for _, nodeInfo := range nodeInfos {
			originalHash, err = utils.GetFileHash(filepath.Join(nodeInfo.PluginDir, vmid.String()))
			gomega.Expect(err).Should(gomega.BeNil())
		}

		// stop network
		_ = commands.StopNetwork()

		for _, nodeInfo := range nodeInfos {
			_, err := commands.UpgradeVMPublic(subnetName, binaryToVersion[utils.SoloEVMKey2], nodeInfo.PluginDir)
			gomega.Expect(err).Should(gomega.BeNil())
		}

		for _, nodeInfo := range nodeInfos {
			measuredHash, err := utils.GetFileHash(filepath.Join(nodeInfo.PluginDir, vmid.String()))
			gomega.Expect(err).Should(gomega.BeNil())

			gomega.Expect(measuredHash).ShouldNot(gomega.Equal(originalHash))
		}

		commands.DeleteSubnetConfig(subnetName)
	})
})

func stripWhitespaces(str string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			// if the character is a space, drop it
			return -1
		}
		// else keep it in the string
		return r
	}, str)
}
