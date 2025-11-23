// Copyright (C) 2022-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package keycmd

import (
	"fmt"
	"math/big"
	"os"

	"github.com/luxfi/cli/cmd/blockchaincmd"
	"github.com/luxfi/cli/pkg/key"
	"github.com/luxfi/cli/pkg/networkoptions"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/ids"
	ledger "github.com/luxfi/node/utils/crypto/ledger"
	"github.com/luxfi/node/utils/formatting/address"
	"github.com/luxfi/node/utils/units"
	"github.com/luxfi/node/vms/platformvm"
	"github.com/luxfi/node/vms/exchangevm"
	"github.com/luxfi/sdk/contract"
	"github.com/luxfi/sdk/evm"
	"github.com/luxfi/sdk/models"
	sdkUtils "github.com/luxfi/sdk/utils"

	"github.com/luxfi/erc20-go/erc20"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/ethclient"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

const (
	allFlag           = "all-networks"
	pchainFlag        = "pchain"
	cchainFlag        = "cchain"
	xchainFlag        = "xchain"
	ledgerIndicesFlag = "ledger"
	useNanoLuxFlag    = "use-nano-lux"
	keysFlag          = "keys"
)

var (
	globalNetworkFlags networkoptions.NetworkFlags
	all                bool
	pchain             bool
	cchain             bool
	xchain             bool
	useNanoLux         bool
	useGwei            bool
	ledgerIndices      []uint
	keys               []string
	tokenAddresses     []string
	subnetToken        string
	subnets            []string
	showNativeToken    bool
)

// lux blockchain list
func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List stored signing keys or ledger addresses",
		Long: `The key list command prints information for all stored signing
keys or for the ledger addresses associated to certain indices.`,
		RunE: listKeys,
	}
	// Network flags handled at higher level to avoid conflicts
	cmd.Flags().BoolVarP(
		&all,
		allFlag,
		"a",
		false,
		"list all network addresses",
	)
	cmd.Flags().BoolVar(
		&pchain,
		pchainFlag,
		true,
		"list P-Chain addresses",
	)
	cmd.Flags().BoolVarP(
		&cchain,
		cchainFlag,
		"c",
		true,
		"list C-Chain addresses",
	)
	cmd.Flags().BoolVar(
		&xchain,
		xchainFlag,
		true,
		"list X-Chain addresses",
	)
	cmd.Flags().BoolVarP(
		&useNanoLux,
		useNanoLuxFlag,
		"n",
		false,
		"use nano Lux for balances",
	)
	cmd.Flags().BoolVar(
		&useGwei,
		"use-gwei",
		false,
		"use gwei for EVM balances",
	)
	cmd.Flags().UintSliceVarP(
		&ledgerIndices,
		ledgerIndicesFlag,
		"g",
		[]uint{},
		"list ledger addresses for the given indices",
	)
	cmd.Flags().StringSliceVar(
		&keys,
		keysFlag,
		[]string{},
		"list addresses for the given keys",
	)
	cmd.Flags().StringSliceVar(
		&subnets,
		"subnets",
		[]string{},
		"subnets to show information about (p=p-chain, x=x-chain, c=c-chain, and blockchain names) (default p,x,c)",
	)
	cmd.Flags().StringSliceVar(
		&subnets,
		"blockchains",
		[]string{},
		"blockchains to show information about (p=p-chain, x=x-chain, c=c-chain, and blockchain names) (default p,x,c)",
	)
	cmd.Flags().StringSliceVar(
		&tokenAddresses,
		"tokens",
		[]string{"Native"},
		"provide balance information for the given token contract addresses (Evm only)",
	)
	return cmd
}

type Clients struct {
	x             map[models.Network]*exchangevm.Client
	p             map[models.Network]*platformvm.Client
	c             map[models.Network]*ethclient.Client
	cGeth         map[models.Network]*ethclient.Client
	evm           map[models.Network]map[string]*ethclient.Client
	evmGeth       map[models.Network]map[string]*ethclient.Client
	blockchainRPC map[models.Network]map[string]string
}

func getClients(networks []models.Network, pchain bool, cchain bool, xchain bool, subnets []string) (
	*Clients,
	error,
) {
	var err error
	xClients := map[models.Network]*exchangevm.Client{}
	pClients := map[models.Network]*platformvm.Client{}
	cClients := map[models.Network]*ethclient.Client{}
	cGethClients := map[models.Network]*ethclient.Client{}
	evmClients := map[models.Network]map[string]*ethclient.Client{}
	evmGethClients := map[models.Network]map[string]*ethclient.Client{}
	blockchainRPCs := map[models.Network]map[string]string{}
	for _, network := range networks {
		if pchain {
			pClients[network] = platformvm.NewClient(network.Endpoint())
		}
		if xchain {
			xClients[network] = exchangevm.NewClient(network.Endpoint(), "X")
		}
		if cchain {
			client, err := ethclient.Dial(network.CChainEndpoint())
			if err != nil {
				return nil, err
			}
			cClients[network] = client
			if len(tokenAddresses) != 0 {
				cGethClients[network], err = ethclient.Dial(network.CChainEndpoint())
				if err != nil {
					return nil, err
				}
			}
		}
		for _, subnetName := range subnets {
			if subnetName != "p" && subnetName != "x" && subnetName != "c" {
				_, err = blockchaincmd.ValidateSubnetNameAndGetChains([]string{subnetName})
				if err != nil {
					return nil, err
				}
				b, _, err := app.HasSubnetEVMGenesis(subnetName)
				if err != nil {
					return nil, err
				}
				if b {
					sc, err := app.LoadSidecar(subnetName)
					if err != nil {
						return nil, err
					}
					subnetToken = sc.TokenSymbol
					endpoint, _, err := contract.GetBlockchainEndpoints(
						app.GetSDKApp(),
						network,
						contract.ChainSpec{
							BlockchainName: subnetName,
						},
						true,
						false,
					)
					if err == nil {
						_, b := blockchainRPCs[network]
						if !b {
							blockchainRPCs[network] = map[string]string{}
						}
						blockchainRPCs[network][subnetName] = endpoint
						_, b = evmClients[network]
						if !b {
							evmClients[network] = map[string]*ethclient.Client{}
						}
						client, err := ethclient.Dial(endpoint)
						if err != nil {
							return nil, err
						}
						evmClients[network][subnetName] = client
						if len(tokenAddresses) != 0 {
							_, b := evmGethClients[network]
							if !b {
								evmGethClients[network] = map[string]*ethclient.Client{}
							}
							evmGethClients[network][subnetName], err = ethclient.Dial(endpoint)
							if err != nil {
								return nil, err
							}
						}
					}
				}
			}
		}
	}
	return &Clients{
		p:             pClients,
		x:             xClients,
		c:             cClients,
		evm:           evmClients,
		cGeth:         cGethClients,
		evmGeth:       evmGethClients,
		blockchainRPC: blockchainRPCs,
	}, nil
}

type addressInfo struct {
	kind    string
	name    string
	chain   string
	token   string
	address string
	balance string
	network string
}

func listKeys(*cobra.Command, []string) error {
	var addrInfos []addressInfo
	networks := []models.Network{}
	if globalNetworkFlags.UseLocal || all {
		networks = append(networks, models.NewLocalNetwork())
	}
	if globalNetworkFlags.UseTestnet || all {
		networks = append(networks, models.NewTestnetNetwork())
	}
	if globalNetworkFlags.UseMainnet || all {
		networks = append(networks, models.NewMainnetNetwork())
	}
	if globalNetworkFlags.ClusterName != "" {
		network, err := app.GetClusterNetwork(globalNetworkFlags.ClusterName)
		if err != nil {
			return err
		}
		networks = append(networks, network)
	}
	if len(networks) == 0 {
		network, err := networkoptions.GetNetworkFromCmdLineFlags(
			app,
			"",
			globalNetworkFlags,
			true,
			false,
			networkoptions.DefaultSupportedNetworkOptions,
			"",
		)
		if err != nil {
			return err
		}
		networks = append(networks, network)
	}
	mainnetIsIncluded := len(utils.Filter(networks, func(n models.Network) bool { return n == models.Mainnet })) > 0
	if mainnetIsIncluded && len(keys) != 1 {
		ux.Logger.PrintToUser("For mainnet you need to specify the key name to be listed by using the --keys flag")
		return nil
	}

	if len(subnets) == 0 {
		subnets = []string{"p", "x", "c"}
	}
	if !sdkUtils.Belongs(subnets, "p") {
		pchain = false
	}
	if !sdkUtils.Belongs(subnets, "x") {
		xchain = false
	}
	if !sdkUtils.Belongs(subnets, "c") {
		cchain = false
	}
	queryLedger := len(ledgerIndices) > 0
	if queryLedger {
		pchain = true
		cchain = false
		xchain = false
	}
	if sdkUtils.Belongs(tokenAddresses, "Native") || sdkUtils.Belongs(tokenAddresses, "native") {
		showNativeToken = true
	}
	tokenAddresses = utils.RemoveFromSlice(tokenAddresses, "Native")
	clients, err := getClients(networks, pchain, cchain, xchain, subnets)
	if err != nil {
		return err
	}
	if queryLedger {
		ledgerIndicesU32 := []uint32{}
		for _, index := range ledgerIndices {
			ledgerIndicesU32 = append(ledgerIndicesU32, uint32(index))
		}
		addrInfos, err = getLedgerIndicesInfo(clients.p, ledgerIndicesU32, networks)
		if err != nil {
			return err
		}
	} else {
		addrInfos, err = getStoredKeysInfo(clients, networks)
		if err != nil {
			return err
		}
	}
	printAddrInfos(addrInfos)
	return nil
}

func getStoredKeysInfo(
	clients *Clients,
	networks []models.Network,
) ([]addressInfo, error) {
	keyNames, err := utils.GetKeyNames(app.GetKeyDir(), true)
	if err != nil {
		return nil, err
	}
	if len(keys) != 0 {
		keyNames = utils.Filter(keyNames, func(keyName string) bool { return sdkUtils.Belongs(keys, keyName) })
	}
	addrInfos := []addressInfo{}
	for _, keyName := range keyNames {
		keyAddrInfos, err := getStoredKeyInfo(clients, networks, keyName)
		if err != nil {
			return nil, err
		}
		addrInfos = append(addrInfos, keyAddrInfos...)
	}
	return addrInfos, nil
}

func getStoredKeyInfo(
	clients *Clients,
	networks []models.Network,
	keyName string,
) ([]addressInfo, error) {
	addrInfos := []addressInfo{}
	for _, network := range networks {
		keyPath := app.GetKeyPath(keyName)
		networkID, err := network.NetworkID()
		if err != nil {
			return nil, err
		}
		sk, err := key.LoadSoft(networkID, keyPath)
		if err != nil {
			return nil, err
		}
		if _, ok := clients.evm[network]; ok {
			evmAddr := sk.C()
			for subnetName := range clients.evm[network] {
				addrInfo, err := getEvmBasedChainAddrInfo(
					subnetName,
					subnetToken,
					clients.evm[network][subnetName],
					clients.evmGeth[network][subnetName],
					network,
					evmAddr,
					"stored",
					keyName,
				)
				if err != nil {
					ux.Logger.RedXToUser(
						"failure obtaining info for blockchain %s on url %s",
						subnetName,
						clients.blockchainRPC[network][subnetName],
					)
					continue
				}
				addrInfos = append(addrInfos, addrInfo...)
			}
		}
		if _, ok := clients.c[network]; ok {
			cChainAddr := sk.C()
			addrInfo, err := getEvmBasedChainAddrInfo("C-Chain", "LUX", clients.c[network], clients.cGeth[network], network, cChainAddr, "stored", keyName)
			if err != nil {
				return nil, err
			}
			addrInfos = append(addrInfos, addrInfo...)
		}
		if _, ok := clients.p[network]; ok {
			pChainAddrs := sk.P()
			for _, pChainAddr := range pChainAddrs {
				addrInfo, err := getPChainAddrInfo(clients.p, network, pChainAddr, "stored", keyName)
				if err != nil {
					return nil, err
				}
				addrInfos = append(addrInfos, addrInfo)
			}
		}
		if _, ok := clients.x[network]; ok {
			xChainAddrs := sk.X()
			for _, xChainAddr := range xChainAddrs {
				addrInfo, err := getXChainAddrInfo(clients.x, network, xChainAddr, "stored", keyName)
				if err != nil {
					return nil, err
				}
				addrInfos = append(addrInfos, addrInfo)
			}
		}
	}
	return addrInfos, nil
}

func getLedgerIndicesInfo(
	pClients map[models.Network]*platformvm.Client,
	ledgerIndices []uint32,
	networks []models.Network,
) ([]addressInfo, error) {
	ledgerDevice, err := ledger.NewLedger()
	if err != nil {
		return nil, err
	}
	addresses, err := ledgerDevice.GetAddresses(ledgerIndices)
	if err != nil {
		return nil, err
	}
	if len(addresses) != len(ledgerIndices) {
		return nil, fmt.Errorf("derived addresses length %d differs from expected %d", len(addresses), len(ledgerIndices))
	}
	addrInfos := []addressInfo{}
	for i, index := range ledgerIndices {
		addr := addresses[i]
		ledgerAddrInfos, err := getLedgerIndexInfo(pClients, index, networks, addr)
		if err != nil {
			return []addressInfo{}, err
		}
		addrInfos = append(addrInfos, ledgerAddrInfos...)
	}
	return addrInfos, nil
}

func getLedgerIndexInfo(
	pClients map[models.Network]*platformvm.Client,
	index uint32,
	networks []models.Network,
	addr ids.ShortID,
) ([]addressInfo, error) {
	addrInfos := []addressInfo{}
	for _, network := range networks {
		pChainAddr, err := address.Format("P", key.GetHRP(network.ID()), addr[:])
		if err != nil {
			return nil, err
		}
		addrInfo, err := getPChainAddrInfo(
			pClients,
			network,
			pChainAddr,
			"ledger",
			fmt.Sprintf("index %d", index),
		)
		if err != nil {
			return nil, err
		}
		addrInfos = append(addrInfos, addrInfo)
	}
	return addrInfos, nil
}

func getPChainAddrInfo(
	pClients map[models.Network]*platformvm.Client,
	network models.Network,
	pChainAddr string,
	kind string,
	name string,
) (addressInfo, error) {
	balance, err := getPChainBalanceStr(pClients[network], pChainAddr)
	if err != nil {
		// just ignore local network errors
		if network != models.Local {
			return addressInfo{}, err
		}
	}
	return addressInfo{
		kind:    kind,
		name:    name,
		chain:   "P-Chain",
		token:   "LUX",
		address: pChainAddr,
		balance: balance,
		network: network.Name(),
	}, nil
}

func getXChainAddrInfo(
	xClients map[models.Network]*exchangevm.Client,
	network models.Network,
	xChainAddr string,
	kind string,
	name string,
) (addressInfo, error) {
	balance, err := getXChainBalanceStr(xClients[network], xChainAddr)
	if err != nil {
		// just ignore local network errors
		if network != models.Local {
			return addressInfo{}, err
		}
	}
	return addressInfo{
		kind:    kind,
		name:    name,
		chain:   "X-Chain",
		token:   "LUX",
		address: xChainAddr,
		balance: balance,
		network: network.Name(),
	}, nil
}

func getEvmBasedChainAddrInfo(
	chainName string,
	chainToken string,
	cClient *ethclient.Client,
	cGethClient *ethclient.Client,
	network models.Network,
	cChainAddr string,
	kind string,
	name string,
) ([]addressInfo, error) {
	addressInfos := []addressInfo{}
	if showNativeToken {
		cChainBalance, err := getCChainBalanceStr(cClient, cChainAddr)
		if err != nil {
			// just ignore local network errors
			if network.Kind() != models.Local {
				return nil, err
			}
		}
		taggedChainToken := chainToken
		if taggedChainToken != "LUX" {
			taggedChainToken = fmt.Sprintf("%s (Native)", taggedChainToken)
		}
		info := addressInfo{
			kind:    kind,
			name:    name,
			chain:   chainName,
			token:   taggedChainToken,
			address: cChainAddr,
			balance: cChainBalance,
			network: network.Name(),
		}
		addressInfos = append(addressInfos, info)
	}
	if cGethClient != nil {
		for _, tokenAddress := range tokenAddresses {
			token, err := erc20.NewGGToken(common.HexToAddress(tokenAddress), cGethClient)
			if err != nil {
				return addressInfos, err
			}

			// Ignore contract address access errors as those may depend on network
			tokenSymbol, err := token.Symbol(nil)
			if err != nil {
				continue
			}

			// Get the raw balance for the given token.
			balance, err := token.BalanceOf(nil, common.HexToAddress(cChainAddr))
			if err != nil {
				return addressInfos, err
			}

			// Get the decimal count for the token to format the balance.
			// Note: decimals() is not officially part of the IERC20 interface, but is a common extension.
			decimals, err := token.Decimals(nil)
			if err != nil {
				return addressInfos, err
			}

			// Format the balance to a human-readable string.
			var formattedBalance string
			if useGwei {
				formattedBalance = fmt.Sprintf("%d", balance)
			} else {
				formattedBalance = utils.FormatAmount(balance, decimals)
			}

			info := addressInfo{
				kind:    kind,
				name:    name,
				chain:   chainName,
				token:   fmt.Sprintf("%s (%s.)", tokenSymbol, tokenAddress[:6]),
				address: cChainAddr,
				balance: formattedBalance,
				network: network.Name(),
			}
			addressInfos = append(addressInfos, info)
		}
	}
	return addressInfos, nil
}

func printAddrInfos(addrInfos []addressInfo) {
	_ = []string{"Kind", "Name", "Subnet", "Address", "Token", "Balance", "Network"}
	table := tablewriter.NewWriter(os.Stdout)
	// table.SetHeader(header)
	// table.SetRowLine(true)
	// table.SetAutoMergeCellsByColumnIndex([]int{0, 1, 2})
	for _, addrInfo := range addrInfos {
		table.Append([]string{
			addrInfo.kind,
			addrInfo.name,
			addrInfo.chain,
			addrInfo.address,
			addrInfo.token,
			addrInfo.balance,
			addrInfo.network,
		})
	}
	table.Render()
}

func getCChainBalanceStr(cClient *ethclient.Client, addrStr string) (string, error) {
	addr := common.HexToAddress(addrStr)
	ctx, cancel := utils.GetAPIContext()
	balance, err := cClient.BalanceAt(ctx, addr, nil)
	cancel()
	if err != nil {
		return "", err
	}
	return formatCChainBalance(balance)
}

func formatCChainBalance(balance *big.Int) (string, error) {
	if useGwei {
		return fmt.Sprintf("%d", balance), nil
	}

	result := evm.ConvertToNanoLux(balance)
	if result.Cmp(big.NewInt(0)) == 0 {
		return "0", nil
	}
	balanceStr := ""
	if useNanoLux {
		balanceStr = fmt.Sprintf("%9d", result.Uint64())
	} else {
		balanceStr = fmt.Sprintf("%.9f", float64(result.Uint64())/float64(units.Lux))
	}
	return balanceStr, nil
}

func getPChainBalanceStr(pClient *platformvm.Client, addr string) (string, error) {
	pID, err := address.ParseToID(addr)
	if err != nil {
		return "", err
	}
	ctx, cancel := utils.GetAPIContext()
	resp, err := pClient.GetBalance(ctx, []ids.ShortID{pID})
	cancel()
	if err != nil {
		return "", err
	}
	if resp.Balance == 0 {
		return "0", nil
	}
	balanceStr := ""
	if useNanoLux {
		balanceStr = fmt.Sprintf("%9d", resp.Balance)
	} else {
		balanceStr = fmt.Sprintf("%.9f", float64(resp.Balance)/float64(units.Lux))
	}
	return balanceStr, nil
}

func getXChainBalanceStr(xClient *exchangevm.Client, addr string) (string, error) {
	xID, err := address.ParseToID(addr)
	if err != nil {
		return "", err
	}
	ctx, cancel := utils.GetAPIContext()
	defer cancel()
	asset, err := xClient.GetAssetDescription(ctx, "LUX")
	if err != nil {
		return "", err
	}
	ctx, cancel = utils.GetAPILargeContext()
	defer cancel()
	resp, err := xClient.GetBalance(ctx, xID, asset.AssetID.String(), false)
	if err != nil {
		return "", err
	}
	if resp.Balance == 0 {
		return "0", nil
	}
	balanceStr := ""
	if useNanoLux {
		balanceStr = fmt.Sprintf("%9d", resp.Balance)
	} else {
		balanceStr = fmt.Sprintf("%.9f", float64(resp.Balance)/float64(units.Lux))
	}
	return balanceStr, nil
}
