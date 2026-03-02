// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package transfer

import (
	"fmt"
	"time"

	"github.com/luxfi/cli/tests/e2e/commands"
	"github.com/luxfi/cli/tests/e2e/utils"
	"github.com/luxfi/constants"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

const (
	keyName            = "e2eKey"
	treasuryKeyName    = "treasury"
	chainName          = "e2eChainTest"
	treasuryEVMAddress = "0x9011E888251AB053B7bD1cdB598Db4f9DEd94714"
)

var _ = ginkgo.Describe("[Key] transfer", func() {
	ginkgo.AfterEach(func() {
		err := utils.DeleteKey(keyName)
		gomega.Expect(err).Should(gomega.BeNil())
	})

	ginkgo.Context("with valid input", func() {
		ginkgo.BeforeEach(func() {
			_, err := commands.CreateKey(keyName)
			gomega.Expect(err).Should(gomega.BeNil())

			commands.StartNetwork()
		})

		ginkgo.AfterEach(func() {
			commands.CleanNetwork()
			err := utils.DeleteConfigs(chainName)
			gomega.Expect(err).Should(gomega.BeNil())
			utils.DeleteCustomBinary(chainName)
		})

		ginkgo.It("can transfer from P-chain to P-chain with treasury key and local key", func() {
			amount := 0.2
			amountStr := fmt.Sprintf("%.2f", amount)
			amountNLux := uint64(amount * float64(constants.Lux))
			commandArguments := []string{
				"--local",
				"--key",
				treasuryKeyName,
				"--destination-key",
				keyName,
				"--p-chain-sender",
				"--p-chain-receiver",
				"--amount",
				amountStr,
			}

			output, err := commands.ListKeys("local", true, "", "")
			gomega.Expect(err).Should(gomega.BeNil())
			_, keyBalance1, err := utils.ParseAddrBalanceFromKeyListOutput(output, keyName, "P-Chain")
			gomega.Expect(err).Should(gomega.BeNil())
			_, treasuryKeyBalance1, err := utils.ParseAddrBalanceFromKeyListOutput(output, treasuryKeyName, "P-Chain")
			gomega.Expect(err).Should(gomega.BeNil())

			output, err = commands.KeyTransferSend(commandArguments)
			gomega.Expect(err).Should(gomega.BeNil())

			feeNLux, err := utils.GetKeyTransferFee(output, "P-Chain")
			gomega.Expect(err).Should(gomega.BeNil())

			output, err = commands.ListKeys("local", true, "", "")
			gomega.Expect(err).Should(gomega.BeNil())
			_, keyBalance2, err := utils.ParseAddrBalanceFromKeyListOutput(output, keyName, "P-Chain")
			gomega.Expect(err).Should(gomega.BeNil())
			_, treasuryKeyBalance2, err := utils.ParseAddrBalanceFromKeyListOutput(output, treasuryKeyName, "P-Chain")
			gomega.Expect(err).Should(gomega.BeNil())
			gomega.Expect(feeNLux + amountNLux).
				Should(gomega.Equal(treasuryKeyBalance1 - treasuryKeyBalance2))
			gomega.Expect(keyBalance2 - keyBalance1).Should(gomega.Equal(amountNLux))
		})

		ginkgo.It("can transfer from P-chain to C-chain with treasury key and local key", func() {
			amount := 0.2
			amountStr := fmt.Sprintf("%.2f", amount)
			amountNLux := uint64(amount * float64(constants.Lux))
			commandArguments := []string{
				"--local",
				"--key",
				treasuryKeyName,
				"--destination-key",
				keyName,
				"--p-chain-sender",
				"--c-chain-receiver",
				"--amount",
				amountStr,
			}

			output, err := commands.ListKeys("local", true, "", "")
			gomega.Expect(err).Should(gomega.BeNil())
			_, keyBalance1, err := utils.ParseAddrBalanceFromKeyListOutput(output, keyName, "C-Chain")
			gomega.Expect(err).Should(gomega.BeNil())
			_, treasuryKeyBalance1, err := utils.ParseAddrBalanceFromKeyListOutput(output, treasuryKeyName, "P-Chain")
			gomega.Expect(err).Should(gomega.BeNil())

			output, err = commands.KeyTransferSend(commandArguments)
			gomega.Expect(err).Should(gomega.BeNil())

			pChainFee, err := utils.GetKeyTransferFee(output, "P-Chain")
			gomega.Expect(err).Should(gomega.BeNil())

			cChainFee, err := utils.GetKeyTransferFee(output, "C-Chain")
			gomega.Expect(err).Should(gomega.BeNil())

			output, err = commands.ListKeys("local", true, "", "")
			gomega.Expect(err).Should(gomega.BeNil())
			_, keyBalance2, err := utils.ParseAddrBalanceFromKeyListOutput(output, keyName, "C-Chain")
			gomega.Expect(err).Should(gomega.BeNil())
			_, treasuryKeyBalance2, err := utils.ParseAddrBalanceFromKeyListOutput(output, treasuryKeyName, "P-Chain")
			gomega.Expect(err).Should(gomega.BeNil())
			gomega.Expect(pChainFee + amountNLux).
				Should(gomega.Equal(treasuryKeyBalance1 - treasuryKeyBalance2))
			gomega.Expect(keyBalance2 - keyBalance1).Should(gomega.Equal(amountNLux - cChainFee))
		})

		ginkgo.It("can transfer from C-chain to P-chain with treasury key and local key", func() {
			amount := 0.2
			amountStr := fmt.Sprintf("%.2f", amount)
			amountNLux := uint64(amount * float64(constants.Lux))
			commandArguments := []string{
				"--local",
				"--key",
				treasuryKeyName,
				"--destination-key",
				keyName,
				"--c-chain-sender",
				"--p-chain-receiver",
				"--amount",
				amountStr,
			}

			// send/receive without recovery
			output, err := commands.ListKeys("local", true, "", "")
			gomega.Expect(err).Should(gomega.BeNil())
			_, keyBalance1, err := utils.ParseAddrBalanceFromKeyListOutput(output, treasuryKeyName, "P-Chain")
			gomega.Expect(err).Should(gomega.BeNil())
			_, treasuryKeyBalance1, err := utils.ParseAddrBalanceFromKeyListOutput(output, treasuryKeyName, "C-Chain")
			gomega.Expect(err).Should(gomega.BeNil())

			output, err = commands.KeyTransferSend(commandArguments)
			gomega.Expect(err).Should(gomega.BeNil())

			pChainFee, err := utils.GetKeyTransferFee(output, "P-Chain")
			gomega.Expect(err).Should(gomega.BeNil())

			cChainFee, err := utils.GetKeyTransferFee(output, "C-Chain")
			gomega.Expect(err).Should(gomega.BeNil())

			output, err = commands.ListKeys("local", true, "", "")
			gomega.Expect(err).Should(gomega.BeNil())
			_, keyBalance2, err := utils.ParseAddrBalanceFromKeyListOutput(output, treasuryKeyName, "P-Chain")
			gomega.Expect(err).Should(gomega.BeNil())
			_, treasuryKeyBalance2, err := utils.ParseAddrBalanceFromKeyListOutput(output, treasuryKeyName, "C-Chain")
			gomega.Expect(err).Should(gomega.BeNil())
			gomega.Expect(cChainFee + amountNLux).
				Should(gomega.Equal(treasuryKeyBalance1 - treasuryKeyBalance2))
			gomega.Expect(keyBalance2 - keyBalance1).Should(gomega.Equal(amountNLux - pChainFee))
		})

		ginkgo.It("can transfer from P-chain to X-chain with treasury key", func() {
			amount := 0.2
			amountStr := fmt.Sprintf("%.2f", amount)
			amountNLux := uint64(amount * float64(constants.Lux))
			commandArguments := []string{
				"--local",
				"--key",
				treasuryKeyName,
				"--destination-key",
				keyName,
				"--p-chain-sender",
				"--x-chain-receiver",
				"--amount",
				amountStr,
			}

			// send/receive without recovery
			output, err := commands.ListKeys("local", true, "", "")
			gomega.Expect(err).Should(gomega.BeNil())
			_, keyBalance1, err := utils.ParseAddrBalanceFromKeyListOutput(output, treasuryKeyName, "X-Chain")
			gomega.Expect(err).Should(gomega.BeNil())
			_, treasuryKeyBalance1, err := utils.ParseAddrBalanceFromKeyListOutput(output, treasuryKeyName, "P-Chain")
			gomega.Expect(err).Should(gomega.BeNil())

			output, err = commands.KeyTransferSend(commandArguments)
			gomega.Expect(err).Should(gomega.BeNil())

			xChainFee, err := utils.GetKeyTransferFee(output, "X-Chain")
			gomega.Expect(err).Should(gomega.BeNil())

			pChainFee, err := utils.GetKeyTransferFee(output, "P-Chain")
			gomega.Expect(err).Should(gomega.BeNil())

			output, err = commands.ListKeys("local", true, "", "")
			gomega.Expect(err).Should(gomega.BeNil())
			_, keyBalance2, err := utils.ParseAddrBalanceFromKeyListOutput(output, treasuryKeyName, "X-Chain")
			gomega.Expect(err).Should(gomega.BeNil())
			_, treasuryKeyBalance2, err := utils.ParseAddrBalanceFromKeyListOutput(output, treasuryKeyName, "P-Chain")
			gomega.Expect(err).Should(gomega.BeNil())
			gomega.Expect(pChainFee + amountNLux).
				Should(gomega.Equal(treasuryKeyBalance1 - treasuryKeyBalance2))
			gomega.Expect(keyBalance2 - keyBalance1).Should(gomega.Equal(amountNLux - xChainFee))
		})

		ginkgo.It("can transfer from C-chain to C-chain with treasury key and local key", func() {
			amount := 0.2
			amountStr := fmt.Sprintf("%.2f", amount)
			amountNLux := uint64(amount * float64(constants.Lux))
			commandArguments := []string{
				"--local",
				"--key",
				treasuryKeyName,
				"--destination-key",
				keyName,
				"--c-chain-sender",
				"--c-chain-receiver",
				"--amount",
				amountStr,
			}

			output, err := commands.ListKeys("local", true, "", "")
			gomega.Expect(err).Should(gomega.BeNil())
			_, keyBalance1, err := utils.ParseAddrBalanceFromKeyListOutput(output, keyName, "C-Chain")
			gomega.Expect(err).Should(gomega.BeNil())
			_, treasuryKeyBalance1, err := utils.ParseAddrBalanceFromKeyListOutput(output, treasuryKeyName, "C-Chain")
			gomega.Expect(err).Should(gomega.BeNil())

			output, err = commands.KeyTransferSend(commandArguments)
			gomega.Expect(err).Should(gomega.BeNil())

			feeNLux, err := utils.GetKeyTransferFee(output, "C-Chain")
			gomega.Expect(err).Should(gomega.BeNil())

			output, err = commands.ListKeys("local", true, "", "")
			gomega.Expect(err).Should(gomega.BeNil())
			_, keyBalance2, err := utils.ParseAddrBalanceFromKeyListOutput(output, keyName, "C-Chain")
			gomega.Expect(err).Should(gomega.BeNil())
			_, treasuryKeyBalance2, err := utils.ParseAddrBalanceFromKeyListOutput(output, treasuryKeyName, "C-Chain")
			gomega.Expect(err).Should(gomega.BeNil())
			gomega.Expect(feeNLux + amountNLux).
				Should(gomega.Equal(treasuryKeyBalance1 - treasuryKeyBalance2))
			gomega.Expect(keyBalance2 - keyBalance1).Should(gomega.Equal(amountNLux))
		})

		ginkgo.It("can transfer from Chain to Chain with treasury key", func() {
			amount := 0.2
			amountStr := fmt.Sprintf("%.2f", amount)
			amountNLux := uint64(amount * float64(constants.Lux))
			commandArguments := []string{
				"--local",
				"--key",
				treasuryKeyName,
				"--destination-key",
				keyName,
				"--sender-blockchain",
				chainName,
				"--receiver-blockchain",
				chainName,
				"--amount",
				amountStr,
			}

			commands.CreateEVMConfigNonSOV(chainName, utils.EVMGenesisPath, false)
			commands.DeployChainLocallyNonSOV(chainName)

			output, err := commands.ListKeys("local", true, "c,"+chainName, "")
			gomega.Expect(err).Should(gomega.BeNil())
			_, keyBalance1, err := utils.ParseAddrBalanceFromKeyListOutput(output, keyName, chainName)
			gomega.Expect(err).Should(gomega.BeNil())
			_, treasuryKeyBalance1, err := utils.ParseAddrBalanceFromKeyListOutput(output, treasuryKeyName, chainName)
			gomega.Expect(err).Should(gomega.BeNil())

			output, err = commands.KeyTransferSend(commandArguments)
			gomega.Expect(err).Should(gomega.BeNil())

			feeNLux, err := utils.GetKeyTransferFee(output, chainName)
			gomega.Expect(err).Should(gomega.BeNil())

			output, err = commands.ListKeys("local", true, "c,"+chainName, "")
			gomega.Expect(err).Should(gomega.BeNil())
			_, keyBalance2, err := utils.ParseAddrBalanceFromKeyListOutput(output, keyName, chainName)
			gomega.Expect(err).Should(gomega.BeNil())
			_, treasuryKeyBalance2, err := utils.ParseAddrBalanceFromKeyListOutput(output, treasuryKeyName, chainName)
			gomega.Expect(err).Should(gomega.BeNil())
			gomega.Expect(feeNLux + amountNLux).
				Should(gomega.Equal(treasuryKeyBalance1 - treasuryKeyBalance2))
			gomega.Expect(keyBalance2 - keyBalance1).Should(gomega.Equal(amountNLux))
		})

		ginkgo.It("can transfer from C-Chain to Chain with treasury key and local key", func() {
			commands.CreateEVMConfigNonSOV(chainName, utils.EVMGenesisPath, true)
			commands.DeployChainLocallyNonSOV(chainName)
			_, err := commands.SendWarpMessage([]string{"cchain", chainName, "hello world"}, utils.TestFlags{"key": treasuryKeyName})
			gomega.Expect(err).Should(gomega.BeNil())
			output := commands.DeployERC20Contract("--local", treasuryKeyName, "TEST", "100000", treasuryEVMAddress, "--c-chain")
			erc20Address, err := utils.GetERC20TokenAddress(output)
			gomega.Expect(err).Should(gomega.BeNil())
			icctArgs := []string{
				"--local",
				"--c-chain-home",
				"--remote-blockchain",
				chainName,
				"--deploy-erc20-home",
				erc20Address,
				"--home-genesis-key",
				"--remote-genesis-key",
			}

			output = commands.DeployInterchainTokenTransferrer(icctArgs)
			gomega.Expect(err).Should(gomega.BeNil())
			homeAddress, remoteAddress, err := utils.GetTokenTransferrerAddresses(output)
			gomega.Expect(err).Should(gomega.BeNil())

			// Get ERC20 balances
			output, err = commands.ListKeys("local", true, "c,"+chainName, fmt.Sprintf("%s,%s", erc20Address, remoteAddress))
			gomega.Expect(err).Should(gomega.BeNil())
			_, keyERCBalance1, err := utils.ParseAddrBalanceFromKeyListOutput(output, keyName, chainName)
			gomega.Expect(err).Should(gomega.BeNil())
			_, treasuryKeyERCBalance1, err := utils.ParseAddrBalanceFromKeyListOutput(output, treasuryKeyName, "C-Chain")
			gomega.Expect(err).Should(gomega.BeNil())

			amount := uint64(500)
			amountStr := fmt.Sprintf("%d", amount)
			transferArgs := []string{
				"--local",
				"--key",
				treasuryKeyName,
				"--destination-key",
				keyName,
				"--c-chain-sender",
				"--receiver-blockchain",
				chainName,
				"--amount",
				amountStr,
				"--origin-transferrer-address",
				homeAddress,
				"--destination-transferrer-address",
				remoteAddress,
			}

			_, err = commands.KeyTransferSend(transferArgs)
			gomega.Expect(err).Should(gomega.BeNil())
			time.Sleep(5 * time.Second) // Wait for warp transaction confirmation

			// Verify ERC20 balances
			output, err = commands.ListKeys("local", true, "c,"+chainName, fmt.Sprintf("%s,%s", erc20Address, remoteAddress))
			gomega.Expect(err).Should(gomega.BeNil())
			_, keyBalance2, err := utils.ParseAddrBalanceFromKeyListOutput(output, keyName, chainName)
			gomega.Expect(err).Should(gomega.BeNil())
			_, treasuryKeyERCBalance2, err := utils.ParseAddrBalanceFromKeyListOutput(output, treasuryKeyName, "C-Chain")
			gomega.Expect(err).Should(gomega.BeNil())
			gomega.Expect(amount).
				Should(gomega.Equal(treasuryKeyERCBalance1 - treasuryKeyERCBalance2))
			gomega.Expect(keyBalance2 - keyERCBalance1).Should(gomega.Equal(amount))
		})

		ginkgo.It("can transfer from Chain to C-chain with treasury key and local key", func() {
			commands.CreateEVMConfigNonSOV(chainName, utils.EVMGenesisPath, true)
			commands.DeployChainLocallyNonSOV(chainName)
			// commands.SendWarpMessage("--local", "cchain", chainName, "hello world", treasuryKeyName)
			output := commands.DeployERC20Contract("--local", treasuryKeyName, "TEST", "100000", treasuryEVMAddress, chainName)
			erc20Address, err := utils.GetERC20TokenAddress(output)
			gomega.Expect(err).Should(gomega.BeNil())
			icctArgs := []string{
				"--local",
				"--c-chain-remote",
				"--home-blockchain",
				chainName,
				"--deploy-erc20-home",
				erc20Address,
				"--home-genesis-key",
				"--remote-genesis-key",
			}

			output = commands.DeployInterchainTokenTransferrer(icctArgs)
			gomega.Expect(err).Should(gomega.BeNil())
			homeAddress, remoteAddress, err := utils.GetTokenTransferrerAddresses(output)
			gomega.Expect(err).Should(gomega.BeNil())

			// Get ERC20 balances
			output, err = commands.ListKeys("local", true, "c,"+chainName, fmt.Sprintf("%s,%s", erc20Address, remoteAddress))
			gomega.Expect(err).Should(gomega.BeNil())
			_, keyERCBalance1, err := utils.ParseAddrBalanceFromKeyListOutput(output, keyName, "C-Chain")
			gomega.Expect(err).Should(gomega.BeNil())
			_, treasuryKeyERCBalance1, err := utils.ParseAddrBalanceFromKeyListOutput(output, treasuryKeyName, chainName)
			gomega.Expect(err).Should(gomega.BeNil())

			amount := uint64(500)
			amountStr := fmt.Sprintf("%d", amount)
			transferArgs := []string{
				"--local",
				"--key",
				treasuryKeyName,
				"--destination-key",
				keyName,
				"--c-chain-receiver",
				"--sender-blockchain",
				chainName,
				"--amount",
				amountStr,
				"--origin-transferrer-address",
				homeAddress,
				"--destination-transferrer-address",
				remoteAddress,
			}

			_, err = commands.KeyTransferSend(transferArgs)
			gomega.Expect(err).Should(gomega.BeNil())
			time.Sleep(5 * time.Second) // Wait for warp transaction confirmation

			// Verify ERC20 balances
			output, err = commands.ListKeys("local", true, "c,"+chainName, fmt.Sprintf("%s,%s", erc20Address, remoteAddress))
			gomega.Expect(err).Should(gomega.BeNil())
			_, keyBalance2, err := utils.ParseAddrBalanceFromKeyListOutput(output, keyName, "C-Chain")
			gomega.Expect(err).Should(gomega.BeNil())
			_, treasuryKeyERCBalance2, err := utils.ParseAddrBalanceFromKeyListOutput(output, treasuryKeyName, chainName)
			gomega.Expect(err).Should(gomega.BeNil())
			gomega.Expect(amount).
				Should(gomega.Equal(treasuryKeyERCBalance1 - treasuryKeyERCBalance2))
			gomega.Expect(keyBalance2 - keyERCBalance1).Should(gomega.Equal(amount))
		})
	})
	ginkgo.Context("with invalid input", func() {
		ginkgo.It("should fail when both key and ledger index were provided", func() {
			commandArguments := []string{
				"--local",
				"--key",
				"test",
				"--ledger",
				"10",
				"--destination-key",
				"test",
				"--amount",
				"0.1",
			}
			output, err := commands.KeyTransferSend(commandArguments)

			gomega.Expect(err).Should(gomega.HaveOccurred())
			gomega.Expect(output).
				Should(gomega.ContainSubstring("only one between a keyname or a ledger index must be given"))
		})

		ginkgo.Context("Within intraEvmSend", func() {
			ginkgo.It("should fail when keyName (not treasury) is provided but no key is found", func() {
				keyName := "nokey"
				commandArguments := []string{
					"--local",
					"--key",
					keyName,
					"--amount",
					"0.1",
					"--c-chain-sender",
					"--c-chain-receiver",
				}
				output, err := commands.KeyTransferSend(commandArguments)

				gomega.Expect(err).Should(gomega.HaveOccurred())
				gomega.Expect(output).
					Should(gomega.ContainSubstring(fmt.Sprintf(".lux-cli/e2e/key/%s.pk: no such file or directory", keyName)))
			})

			ginkgo.It("should fail when destinationKeyName (not treasury) is provided but no key is found", func() {
				keyName := "nokey"
				commandArguments := []string{
					"--local",
					"--key",
					treasuryKeyName,
					"--destination-key",
					keyName,
					"--amount",
					"0.1",
					"--c-chain-sender",
					"--c-chain-receiver",
				}

				output, err := commands.KeyTransferSend(commandArguments)

				gomega.Expect(err).Should(gomega.HaveOccurred())
				gomega.Expect(output).
					Should(gomega.ContainSubstring(fmt.Sprintf(".lux-cli/e2e/key/%s.pk: no such file or directory", keyName)))
			})

			ginkgo.It("should fail when amount provided amount is negative", func() {
				commandArguments := []string{
					"--local",
					"--key",
					treasuryKeyName,
					"--destination-key",
					treasuryKeyName,
					"--amount",
					"-0.1",
					"--c-chain-sender",
					"--c-chain-receiver",
				}
				output, err := commands.KeyTransferSend(commandArguments)

				gomega.Expect(err).Should(gomega.HaveOccurred())
				gomega.Expect(output).
					Should(gomega.ContainSubstring("amount must be positive"))
			})

			ginkgo.It("should fail to load sidecar when blockchain does not exist in chains directory", func() {
				blockhainName := "NonExistingBlockchain"
				commandArguments := []string{
					"--local",
					"--key",
					treasuryKeyName,
					"--destination-key",
					treasuryKeyName,
					"--amount",
					"0.1",
					"--sender-blockchain",
					blockhainName,
					"--c-chain-receiver",
				}
				output, err := commands.KeyTransferSend(commandArguments)

				gomega.Expect(err).Should(gomega.HaveOccurred())
				gomega.Expect(output).
					Should(gomega.ContainSubstring("failed to load sidecar"))
			})
		})
	})
	ginkgo.Context("with unsupported paths", func() {
		ginkgo.It("should fail when transferring from X-Chain to X-Chain", func() {
			commandArguments := []string{
				"--local",
				"--key",
				treasuryKeyName,
				"--destination-key",
				treasuryKeyName,
				"--amount",
				"0.1",
				"--x-chain-sender",
				"--x-chain-receiver",
			}
			output, err := commands.KeyTransferSend(commandArguments)

			gomega.Expect(err).Should(gomega.HaveOccurred())
			gomega.Expect(output).
				Should(gomega.ContainSubstring("transfer from X-Chain to X-Chain is not supported"))
		})

		ginkgo.It("should fail when transferring from X-Chain to C-Chain", func() {
			commandArguments := []string{
				"--local",
				"--key",
				treasuryKeyName,
				"--destination-key",
				treasuryKeyName,
				"--amount",
				"0.1",
				"--x-chain-sender",
				"--c-chain-receiver",
			}
			output, err := commands.KeyTransferSend(commandArguments)

			gomega.Expect(err).Should(gomega.HaveOccurred())
			gomega.Expect(output).
				Should(gomega.ContainSubstring("transfer from X-Chain to C-Chain is not supported"))
		})

		ginkgo.It("should fail when transferring from X-Chain to P-Chain", func() {
			commandArguments := []string{
				"--local",
				"--key",
				treasuryKeyName,
				"--destination-key",
				treasuryKeyName,
				"--amount",
				"0.1",
				"--x-chain-sender",
				"--p-chain-receiver",
			}
			output, err := commands.KeyTransferSend(commandArguments)

			gomega.Expect(err).Should(gomega.HaveOccurred())
			gomega.Expect(output).
				Should(gomega.ContainSubstring("transfer from X-Chain to P-Chain is not supported"))
		})

		ginkgo.It("should fail when transferring from X-Chain to Chain", func() {
			commandArguments := []string{
				"--local",
				"--key",
				treasuryKeyName,
				"--destination-key",
				treasuryKeyName,
				"--amount",
				"0.1",
				"--x-chain-sender",
				"--receiver-blockchain",
				"Test-Chain",
			}
			output, err := commands.KeyTransferSend(commandArguments)

			gomega.Expect(err).Should(gomega.HaveOccurred())
			gomega.Expect(output).
				Should(gomega.ContainSubstring("transfer from X-Chain to Test-Chain is not supported"))
		})

		ginkgo.It("should fail when transferring from Chain to X-Chain", func() {
			commandArguments := []string{
				"--local",
				"--key",
				treasuryKeyName,
				"--destination-key",
				treasuryKeyName,
				"--amount",
				"0.1",
				"--x-chain-receiver",
				"--sender-blockchain",
				"Test-Chain",
			}
			output, err := commands.KeyTransferSend(commandArguments)

			gomega.Expect(err).Should(gomega.HaveOccurred())
			gomega.Expect(output).
				Should(gomega.ContainSubstring("transfer from Test-Chain to X-Chain is not supported"))
		})

		ginkgo.It("should fail when transferring from Chain to P-Chain", func() {
			commandArguments := []string{
				"--local",
				"--key",
				treasuryKeyName,
				"--destination-key",
				treasuryKeyName,
				"--amount",
				"0.1",
				"--p-chain-receiver",
				"--sender-blockchain",
				"Test-Chain",
			}
			output, err := commands.KeyTransferSend(commandArguments)

			gomega.Expect(err).Should(gomega.HaveOccurred())
			gomega.Expect(output).
				Should(gomega.ContainSubstring("transfer from Test-Chain to P-Chain is not supported"))
		})

		ginkgo.It("should fail when transferring from P-Chain to Chain", func() {
			commandArguments := []string{
				"--local",
				"--key",
				treasuryKeyName,
				"--destination-key",
				treasuryKeyName,
				"--amount",
				"0.1",
				"--p-chain-sender",
				"--receiver-blockchain",
				"Test-Chain",
			}
			output, err := commands.KeyTransferSend(commandArguments)

			gomega.Expect(err).Should(gomega.HaveOccurred())
			gomega.Expect(output).
				Should(gomega.ContainSubstring("transfer from P-Chain to Test-Chain is not supported"))
		})
	})
})
