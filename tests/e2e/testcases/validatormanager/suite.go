// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package packageman

import (
	"fmt"
	"os"
	"os/exec"
	"path"

	"github.com/luxfi/cli/pkg/chainvalidators"
	"github.com/luxfi/cli/pkg/key"
	"github.com/luxfi/cli/tests/e2e/commands"
	"github.com/luxfi/cli/tests/e2e/utils"
	"github.com/luxfi/constants"
	"github.com/luxfi/ids"
	luxlog "github.com/luxfi/log"
	"github.com/luxfi/protocol/p/txs"
	"github.com/luxfi/sdk/api/info"
	blockchainSDK "github.com/luxfi/sdk/blockchain"
	"github.com/luxfi/sdk/evm"
	"github.com/luxfi/sdk/models"

	"github.com/luxfi/geth/common"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

const (
	CLIBinary            = "./bin/lux"
	keyName              = "ewoq"
	ewoqEVMAddress       = "0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC"
	ewoqPChainAddress    = "P-custom18jma8ppw3nhx5r4ap8clazz0dps7rv5u9xde7p"
	ProxyContractAddress = "0xFEEDC0DE0000000000000000000000000000000"
)

var err error

func createEtnaEVMConfig() error {
	// Check config does not already exist
	_, err = utils.ChainConfigExists(utils.BlockchainName)
	if err != nil {
		return err
	}

	// Create config
	cmd := exec.Command( //nolint:gosec // G204: Running our own CLI binary in tests
		CLIBinary,
		"blockchain",
		"create",
		utils.BlockchainName,
		"--evm",
		"--proof-of-authority",
		"--validator-manager-owner",
		ewoqEVMAddress,
		"--proxy-contract-owner",
		ewoqEVMAddress,
		"--production-defaults",
		"--evm-chain-id=99999",
		"--evm-token=TOK",
		"--"+constants.SkipUpdateFlag,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(cmd.String())
		utils.PrintStdErr(err)
	}
	fmt.Println(string(output))
	return err
}

func createSovereignChain() (string, string, error) {
	if err := createEtnaEVMConfig(); err != nil {
		return "", "", err
	}
	// Deploy chain on etna local network with local machine as bootstrap validator
	cmd := exec.Command( //nolint:gosec // G204: Running our own CLI binary in tests
		CLIBinary,
		"blockchain",
		"deploy",
		utils.BlockchainName,
		"--local",
		"--num-bootstrap-validators=1",
		"--ewoq",
		"--convert-only",
		"--change-owner-address",
		ewoqPChainAddress,
		"--"+constants.SkipUpdateFlag,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(cmd.String())
		utils.PrintStdErr(err)
	}
	fmt.Println(string(output))
	chainID, err := utils.ParsePublicDeployOutput(string(output), utils.ChainIDParseType)
	if err != nil {
		return "", "", err
	}
	blockchainID, err := utils.ParsePublicDeployOutput(string(output), utils.BlockchainIDParseType)
	if err != nil {
		return "", "", err
	}
	return chainID, blockchainID, err
}

func destroyLocalNode() {
	_, err := os.Stat(utils.TestLocalNodeName)
	if os.IsNotExist(err) {
		return
	}
	cmd := exec.Command( //nolint:gosec // G204: Running our own CLI binary in tests
		CLIBinary,
		"node",
		"local",
		"destroy",
		utils.TestLocalNodeName,
		"--"+constants.SkipUpdateFlag,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(cmd.String())
		fmt.Println(string(output))
		utils.PrintStdErr(err)
	}
}

func getBootstrapValidator(uri string) ([]*txs.ConvertChainToL1Validator, error) {
	infoClient := info.NewClient(uri)
	ctx, cancel := utils.GetAPILargeContext()
	defer cancel()
	nodeID, proofOfPossession, err := infoClient.GetNodeID(ctx)
	if err != nil {
		return nil, err
	}
	publicKey := "0x" + proofOfPossession.PublicKey
	pop := "0x" + proofOfPossession.ProofOfPossession

	bootstrapValidator := models.ChainValidator{
		NodeID:               nodeID.String(),
		Weight:               constants.BootstrapValidatorWeight,
		Balance:              constants.BootstrapValidatorBalanceNanoLUX,
		BLSPublicKey:         publicKey,
		BLSProofOfPossession: pop,
		ChangeOwnerAddr:      ewoqPChainAddress,
	}
	luxdBootstrapValidators, err := chainvalidators.ToL1Validators([]models.ChainValidator{bootstrapValidator})
	if err != nil {
		return nil, err
	}

	return luxdBootstrapValidators, nil
}

var _ = ginkgo.Describe("[Validator Manager POA Set Up]", ginkgo.Ordered, func() {
	ginkgo.BeforeEach(func() {
		// key
		_ = utils.DeleteKey(keyName)
		output, err := commands.CreateKeyFromPath(keyName, utils.LocalKeyPath)
		if err != nil {
			fmt.Println(output)
			utils.PrintStdErr(err)
		}
		gomega.Expect(err).Should(gomega.BeNil())
		// chain config
		_ = utils.DeleteConfigs(utils.BlockchainName)
		destroyLocalNode()
	})

	ginkgo.AfterEach(func() {
		destroyLocalNode()
		commands.DeleteChainConfig(utils.BlockchainName)
		err := utils.DeleteKey(keyName)
		gomega.Expect(err).Should(gomega.BeNil())
		commands.CleanNetwork()
	})
	ginkgo.It("Set Up POA Validator Manager", func() {
		chainIDStr, blockchainIDStr, err := createSovereignChain()
		gomega.Expect(err).Should(gomega.BeNil())
		uris, err := utils.GetLocalClusterUris()
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(len(uris)).Should(gomega.Equal(1))
		_, err = commands.TrackLocalEtnaChain(utils.TestLocalNodeName, utils.BlockchainName)
		gomega.Expect(err).Should(gomega.BeNil())
		keyPath := path.Join(utils.GetBaseDir(), constants.KeyDir, fmt.Sprintf("chain_%s_airdrop", utils.BlockchainName)+constants.KeySuffix)
		k, err := key.LoadSoft(models.NewLocalNetwork().ID(), keyPath)
		gomega.Expect(err).Should(gomega.BeNil())
		rpcURL := fmt.Sprintf("%s/ext/bc/%s/rpc", uris[0], blockchainIDStr)
		client, err := evm.GetClient(rpcURL)
		gomega.Expect(err).Should(gomega.BeNil())
		err = client.WaitForEVMBootstrapped(0)
		gomega.Expect(err).Should(gomega.BeNil())

		network := models.NewNetworkFromCluster(models.NewLocalNetwork(), utils.TestLocalNodeName)

		chainID, err := ids.FromString(chainIDStr)
		gomega.Expect(err).Should(gomega.BeNil())

		blockchainID, err := ids.FromString(blockchainIDStr)
		gomega.Expect(err).Should(gomega.BeNil())

		luxdBootstrapValidators, err := getBootstrapValidator(uris[0])
		gomega.Expect(err).Should(gomega.BeNil())
		// Convert validators to interface slice for SDK
		var validators []interface{}
		for _, v := range luxdBootstrapValidators {
			validators = append(validators, v)
		}
		ownerAddress := common.HexToAddress(ewoqEVMAddress)
		netSDK := blockchainSDK.Net{
			NetworkID:           networkID,
			BlockchainID:        blockchainID,
			OwnerAddress:        &ownerAddress,
			RPC:                 rpcURL,
			BootstrapValidators: validators,
		}

		_, cancel := utils.GetSignatureAggregatorContext()
		defer cancel()
		err = netSDK.InitializeProofOfAuthority(
			luxlog.NoLog{},
			network.SDKNetwork(),
			k.PrivKeyHex(),
			luxlog.NoLog{},
			ProxyContractAddress,
			true,
			"",
		)
		gomega.Expect(err).Should(gomega.BeNil())
	})
})
