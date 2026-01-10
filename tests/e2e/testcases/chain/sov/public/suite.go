// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	cliutils "github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/tests/e2e/commands"
	"github.com/luxfi/cli/tests/e2e/utils"
	"github.com/luxfi/constants"
	"github.com/luxfi/ids"
	luxlog "github.com/luxfi/log"
	"github.com/luxfi/sdk/models"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

const (
	chainName      = "e2eChainTest"
	controlKeys    = "P-custom18jma8ppw3nhx5r4ap8clazz0dps7rv5u9xde7p"
	keyName        = "ewoq"
	ledger1Seed    = "ledger1"
	ledger2Seed    = "ledger2"
	ledger3Seed    = "ledger3"
	txFnamePrefix  = "lux-cli-tx-"
	mainnetChainID = 123456
)

func deployChainToTestnetSOV() (string, map[string]utils.NodeInfo) {
	// deploy
	s := commands.SimulateTestnetDeploySOV(chainName, keyName, controlKeys)
	chainID, err := utils.ParsePublicDeployOutput(s, utils.ChainIDParseType)
	gomega.Expect(err).Should(gomega.BeNil())
	// add validators to chain
	nodeInfos, err := utils.GetLocalNetworkNodesInfo()
	gomega.Expect(err).Should(gomega.BeNil())
	for _, nodeInfo := range nodeInfos {
		start := time.Now().Add(time.Second * 30).UTC().Format("2006-01-02 15:04:05")
		_ = commands.SimulateTestnetAddValidator(chainName, keyName, nodeInfo.ID, start, "24h", "20")
	}
	// join to copy vm binary and update config file
	for _, nodeInfo := range nodeInfos {
		_ = commands.SimulateTestnetJoin(chainName, nodeInfo.ConfigFile, nodeInfo.PluginDir, nodeInfo.ID)
	}
	// get and check whitelisted chains from config file
	for _, nodeInfo := range nodeInfos {
		whitelistedChains, err := utils.GetWhitelistedChainsFromConfigFile(nodeInfo.ConfigFile)
		gomega.Expect(err).Should(gomega.BeNil())
		whitelistedChainsSlice := strings.Split(whitelistedChains, ",")
		gomega.Expect(whitelistedChainsSlice).Should(gomega.ContainElement(chainID))
	}
	// restart nodes
	err = utils.RestartNodes()
	gomega.Expect(err).Should(gomega.BeNil())
	// wait for chain walidators to be up
	err = utils.WaitChainValidators(chainID, nodeInfos)
	gomega.Expect(err).Should(gomega.BeNil())
	return chainID, nodeInfos
}

var _ = ginkgo.Describe("[Public Chain SOV]", func() {
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
		_ = utils.DeleteConfigs(chainName)
		_, luxdVersion := commands.CreateEVMConfigSOV(chainName, utils.EVMGenesisPath)

		// local network
		commands.StartNetworkWithVersion(luxdVersion)
	})

	ginkgo.AfterEach(func() {
		commands.DeleteChainConfig(chainName)
		err := utils.DeleteKey(keyName)
		gomega.Expect(err).Should(gomega.BeNil())
		commands.CleanNetwork()
	})

	ginkgo.It("deploy chain to testnet SOV", func() {
		deployChainToTestnetSOV()
	})

	ginkgo.It("deploy chain to mainnet SOV", func() {
		var interactionEndCh, ledgerSimEndCh chan struct{}
		if os.Getenv("LEDGER_SIM") != "" {
			interactionEndCh, ledgerSimEndCh = utils.StartLedgerSim(7, ledger1Seed, true)
		}
		// fund ledger address
		// Estimate fee: CreateChainTxFee + CreateChainTxFee + TxFee
		fee := estimateDeploymentFee(3)
		err := utils.FundLedgerAddress(fee)
		gomega.Expect(err).Should(gomega.BeNil())
		fmt.Println()
		fmt.Println(luxlog.LightRed.Wrap("DEPLOYING CHAIN. VERIFY LEDGER ADDRESS HAS CUSTOM HRP BEFORE SIGNING"))
		s := commands.SimulateMainnetDeploySOV(chainName, 0, false)
		// deploy
		chainID, err := utils.ParsePublicDeployOutput(s, utils.ChainIDParseType)
		gomega.Expect(err).Should(gomega.BeNil())
		// add validators to chain
		nodeInfos, err := utils.GetLocalNetworkNodesInfo()
		gomega.Expect(err).Should(gomega.BeNil())
		nodeIdx := 1
		for _, nodeInfo := range nodeInfos {
			fmt.Println(luxlog.LightRed.Wrap(
				fmt.Sprintf("ADDING VALIDATOR %d of %d. VERIFY LEDGER ADDRESS HAS CUSTOM HRP BEFORE SIGNING", nodeIdx, len(nodeInfos))))
			start := time.Now().Add(time.Second * 30).UTC().Format("2006-01-02 15:04:05")
			_ = commands.SimulateMainnetAddValidator(chainName, nodeInfo.ID, start, "24h", "20")
			nodeIdx++
		}
		if os.Getenv("LEDGER_SIM") != "" {
			close(interactionEndCh)
			<-ledgerSimEndCh
		}
		fmt.Println(luxlog.LightBlue.Wrap("EXECUTING NON INTERACTIVE PART OF THE TEST: JOIN/WHITELIST/WAIT/HARDHAT"))
		// join to copy vm binary and update config file
		for _, nodeInfo := range nodeInfos {
			_ = commands.SimulateMainnetJoin(chainName, nodeInfo.ConfigFile, nodeInfo.PluginDir, nodeInfo.ID)
		}
		// get and check whitelisted chains from config file
		for _, nodeInfo := range nodeInfos {
			whitelistedChains, err := utils.GetWhitelistedChainsFromConfigFile(nodeInfo.ConfigFile)
			gomega.Expect(err).Should(gomega.BeNil())
			whitelistedChainsSlice := strings.Split(whitelistedChains, ",")
			gomega.Expect(whitelistedChainsSlice).Should(gomega.ContainElement(chainID))
		}
		// restart nodes
		err = utils.RestartNodes()
		gomega.Expect(err).Should(gomega.BeNil())
		// wait for chain walidators to be up
		err = utils.WaitChainValidators(chainID, nodeInfos)
		gomega.Expect(err).Should(gomega.BeNil())

		// this is a simulation, so app is probably saving the info in the
		// `local network` section of the sidecar instead of the `testnet` section...
		// ...need to manipulate the `testnet` section of the sidecar to contain the chainID info
		// so that the `stats` command for `testnet` can find it
		output := commands.SimulateGetChainStatsTestnet(chainName, chainID)
		gomega.Expect(output).Should(gomega.Not(gomega.BeNil()))
		gomega.Expect(output).Should(gomega.ContainSubstring("Current validators"))
		gomega.Expect(output).Should(gomega.ContainSubstring("NodeID-"))
	})

	ginkgo.It("deploy chain with new chain id SOV", func() {
		chainMainnetChainID, err := utils.GetEVMMainnetChainID(chainName)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(chainMainnetChainID).Should(gomega.Equal(uint(0)))
		_ = commands.SimulateMainnetDeploySOV(chainName, mainnetChainID, true)
		chainMainnetChainID, err = utils.GetEVMMainnetChainID(chainName)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(chainMainnetChainID).Should(gomega.Equal(uint(mainnetChainID)))
	})

	ginkgo.It("remove validator testnet SOV", func() {
		chainIDStr, nodeInfos := deployChainToTestnetSOV()

		// pick a validator to remove
		var validatorToRemove string
		for _, nodeInfo := range nodeInfos {
			validatorToRemove = nodeInfo.ID
			break
		}

		// confirm current validator set
		chainID, err := ids.FromString(chainIDStr)
		gomega.Expect(err).Should(gomega.BeNil())
		validators, err := utils.GetChainValidators(chainID)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(len(validators)).Should(gomega.Equal(5))

		// Check that the validatorToRemove is in the chain validator set
		var found bool
		for _, validator := range validators {
			if validator == validatorToRemove {
				found = true
				break
			}
		}
		gomega.Expect(found).Should(gomega.BeTrue())

		// remove validator
		_ = commands.SimulateTestnetRemoveValidator(chainName, keyName, validatorToRemove)

		// confirm current validator set
		validators, err = utils.GetChainValidators(chainID)
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(len(validators)).Should(gomega.Equal(4))

		// Check that the validatorToRemove is NOT in the chain validator set
		found = false
		for _, validator := range validators {
			if validator == validatorToRemove {
				found = true
				break
			}
		}
		gomega.Expect(found).Should(gomega.BeFalse())
	})

	ginkgo.It("mainnet multisig deploy SOV", func() {
		// this is not expected to be executed with real ledgers
		// as that will complicate too much the test flow
		gomega.Expect(os.Getenv("LEDGER_SIM")).Should(gomega.Equal("true"), "multisig test not designed for real ledgers: please set env var LEDGER_SIM to true")

		txPath, err := utils.GetTmpFilePath(txFnamePrefix)
		gomega.Expect(err).Should(gomega.BeNil())

		// obtain ledger2 addr
		interactionEndCh, ledgerSimEndCh := utils.StartLedgerSim(0, ledger2Seed, false)
		ledger2Addr, err := utils.GetLedgerAddress(models.NewLocalNetwork(), 0)
		gomega.Expect(err).Should(gomega.BeNil())
		close(interactionEndCh)
		<-ledgerSimEndCh

		// obtain ledger3 addr
		interactionEndCh, ledgerSimEndCh = utils.StartLedgerSim(0, ledger3Seed, false)
		ledger3Addr, err := utils.GetLedgerAddress(models.NewLocalNetwork(), 0)
		gomega.Expect(err).Should(gomega.BeNil())
		close(interactionEndCh)
		<-ledgerSimEndCh

		// ledger4 addr
		// will not be used to sign, only as a extra control key, so no sim is needed to generate it
		ledger4Addr := "P-custom18g2tekxzt60j3sn8ymjx6qvk96xunhctkyzckt"

		// start the deploy process with ledger1
		interactionEndCh, ledgerSimEndCh = utils.StartLedgerSim(2, ledger1Seed, true)

		// obtain ledger1 addr
		ledger1Addr, err := utils.GetLedgerAddress(models.NewLocalNetwork(), 0)
		gomega.Expect(err).Should(gomega.BeNil())

		// multisig deploy from unfunded ledger1 should not create any chain/blockchain
		gomega.Expect(err).Should(gomega.BeNil())
		s := commands.SimulateMultisigMainnetDeploySOV(
			chainName,
			[]string{ledger2Addr, ledger3Addr, ledger4Addr},
			[]string{ledger2Addr, ledger3Addr},
			txPath,
			true,
		)
		toMatch := "(?s).+Not enough funds in the first (?s).+ indices of Ledger(?s).+Error: not enough funds on ledger(?s).+"
		matched, err := regexp.MatchString(toMatch, cliutils.RemoveLineCleanChars(s))
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(matched).Should(gomega.Equal(true), "no match between command output %q and pattern %q", s, toMatch)

		// let's fund the ledger
		// Estimate fee: CreateChainTxFee + CreateChainTxFee + TxFee
		fee := estimateDeploymentFee(3)
		err = utils.FundLedgerAddress(fee)
		gomega.Expect(err).Should(gomega.BeNil())

		// multisig deploy from funded ledger1 should create the chain but not deploy the blockchain,
		// instead signing only its tx fee as it is not a chain auth key,
		// and creating the tx file to wait for chain auths from ledger2 and ledger3
		s = commands.SimulateMultisigMainnetDeploySOV(
			chainName,
			[]string{ledger2Addr, ledger3Addr, ledger4Addr},
			[]string{ledger2Addr, ledger3Addr},
			txPath,
			false,
		)
		toMatch = "(?s).+Ledger addresses:(?s).+  " + ledger1Addr + "(?s).+Chain has been created with ID(?s).+" +
			"0 of 2 required Blockchain Creation signatures have been signed\\. Saving tx to disk to enable remaining signing\\.(?s).+" +
			"Addresses remaining to sign the tx\\s+" + ledger2Addr + "(?s).+" + ledger3Addr + "(?s).+"
		matched, err = regexp.MatchString(toMatch, cliutils.RemoveLineCleanChars(s))
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(matched).Should(gomega.Equal(true), "no match between command output %q and pattern %q", s, toMatch)

		// try to commit before signature is complete (no funded wallet needed for commit)
		s = commands.TransactionCommit(
			chainName,
			txPath,
			true,
		)
		toMatch = "(?s).*0 of 2 required signatures have been signed\\.(?s).+" +
			"Addresses remaining to sign the tx\\s+" + ledger2Addr + "(?s).+" + ledger3Addr + "(?s).+" +
			"(?s).+Error: tx is not fully signed(?s).+"
		matched, err = regexp.MatchString(toMatch, cliutils.RemoveLineCleanChars(s))
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(matched).Should(gomega.Equal(true), "no match between command output %q and pattern %q", s, toMatch)

		// try to sign using unauthorized ledger1
		s = commands.TransactionSignWithLedger(
			chainName,
			txPath,
			true,
		)
		toMatch = "(?s).+Ledger addresses:(?s).+  " + ledger1Addr + "(?s).+There are no required chain auth keys present in the wallet(?s).+" +
			"Expected one of:\\s+" + ledger2Addr + "(?s).+" + ledger3Addr + "(?s).+Error: no remaining signer address present in wallet.*"
		matched, err = regexp.MatchString(toMatch, cliutils.RemoveLineCleanChars(s))
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(matched).Should(gomega.Equal(true), "no match between command output %q and pattern %q", s, toMatch)

		// wait for end of ledger1 simulation
		close(interactionEndCh)
		<-ledgerSimEndCh

		// try to commit before signature is complete
		s = commands.TransactionCommit(
			chainName,
			txPath,
			true,
		)
		toMatch = "(?s).*0 of 2 required signatures have been signed\\.(?s).+" +
			"Addresses remaining to sign the tx\\s+" + ledger2Addr + "(?s).+" + ledger3Addr + "(?s).+" +
			"(?s).+Error: tx is not fully signed(?s).+"
		matched, err = regexp.MatchString(toMatch, cliutils.RemoveLineCleanChars(s))
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(matched).Should(gomega.Equal(true), "no match between command output %q and pattern %q", s, toMatch)

		// sign using ledger2
		interactionEndCh, ledgerSimEndCh = utils.StartLedgerSim(1, ledger2Seed, true)
		s = commands.TransactionSignWithLedger(
			chainName,
			txPath,
			false,
		)
		toMatch = "(?s).+Ledger addresses:(?s).+  " + ledger2Addr + "(?s).+1 of 2 required Tx signatures have been signed\\.(?s).+" +
			"Addresses remaining to sign the tx\\s+" + ledger3Addr + ".*"
		matched, err = regexp.MatchString(toMatch, cliutils.RemoveLineCleanChars(s))
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(matched).Should(gomega.Equal(true), "no match between command output %q and pattern %q", s, toMatch)

		// try to sign using ledger2 which already signed
		s = commands.TransactionSignWithLedger(
			chainName,
			txPath,
			true,
		)
		toMatch = "(?s).+Ledger addresses:(?s).+  " + ledger2Addr + "(?s).+There are no required chain auth keys present in the wallet(?s).+" +
			"Expected one of:\\s+" + ledger3Addr + "(?s).+Error: no remaining signer address present in wallet.*"
		matched, err = regexp.MatchString(toMatch, cliutils.RemoveLineCleanChars(s))
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(matched).Should(gomega.Equal(true), "no match between command output %q and pattern %q", s, toMatch)

		// wait for end of ledger2 simulation
		close(interactionEndCh)
		<-ledgerSimEndCh

		// try to commit before signature is complete
		s = commands.TransactionCommit(
			chainName,
			txPath,
			true,
		)
		toMatch = "(?s).*1 of 2 required signatures have been signed\\.(?s).+" +
			"Addresses remaining to sign the tx\\s+" + ledger3Addr +
			"(?s).+Error: tx is not fully signed(?s).+"
		matched, err = regexp.MatchString(toMatch, cliutils.RemoveLineCleanChars(s))
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(matched).Should(gomega.Equal(true), "no match between command output %q and pattern %q", s, toMatch)

		// sign with ledger3
		interactionEndCh, ledgerSimEndCh = utils.StartLedgerSim(1, ledger3Seed, true)
		s = commands.TransactionSignWithLedger(
			chainName,
			txPath,
			false,
		)
		toMatch = "(?s).+Ledger addresses:(?s).+  " + ledger3Addr + "(?s).+Tx is fully signed, and ready to be committed(?s).+"
		matched, err = regexp.MatchString(toMatch, cliutils.RemoveLineCleanChars(s))
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(matched).Should(gomega.Equal(true), "no match between command output %q and pattern %q", s, toMatch)

		// try to sign using ledger3 which already signedtx is already fully signed"
		s = commands.TransactionSignWithLedger(
			chainName,
			txPath,
			true,
		)
		toMatch = "(?s).*Tx is fully signed, and ready to be committed(?s).+Error: tx is already fully signed"
		matched, err = regexp.MatchString(toMatch, cliutils.RemoveLineCleanChars(s))
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(matched).Should(gomega.Equal(true), "no match between command output %q and pattern %q", s, toMatch)

		// wait for end of ledger3 simulation
		close(interactionEndCh)
		<-ledgerSimEndCh

		// commit after complete signature
		s = commands.TransactionCommit(
			chainName,
			txPath,
			false,
		)
		toMatch = "(?s).+DEPLOYMENT RESULTS(?s).+Blockchain ID(?s).+"
		matched, err = regexp.MatchString(toMatch, cliutils.RemoveLineCleanChars(s))
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(matched).Should(gomega.Equal(true), "no match between command output %q and pattern %q", s, toMatch)

		// try to commit again
		s = commands.TransactionCommit(
			chainName,
			txPath,
			true,
		)
		toMatch = "(?s).*Error: error issuing tx with ID(?s).+: failed to decode client response: couldn't issue tx: failed to read consumed(?s).+"
		matched, err = regexp.MatchString(toMatch, cliutils.RemoveLineCleanChars(s))
		gomega.Expect(err).Should(gomega.BeNil())
		gomega.Expect(matched).Should(gomega.Equal(true), "no match between command output %q and pattern %q", s, toMatch)
	})
})

func estimateDeploymentFee(txCount int) uint64 {
	// Base fee per transaction type
	return uint64(txCount) * constants.Lux //nolint:gosec // G115: txCount is bounded by test parameters
}
