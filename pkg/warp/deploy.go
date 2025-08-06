// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package warp

import (
	_ "embed"
	"math/big"
	"os"
	"path/filepath"

	"github.com/luxfi/cli/pkg/contract"
	"github.com/luxfi/cli/pkg/utils"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/crypto"
)

type WarpFeeInfo struct {
	FeeTokenAddress crypto.Address
	Amount          *big.Int
}

type TokenRemoteSettings struct {
	WarpRegistryAddress   crypto.Address
	WarpManager           crypto.Address
	TokenHomeBlockchainID [32]byte
	TokenHomeAddress      crypto.Address
	TokenHomeDecimals     uint8
}

func RegisterRemote(
	rpcURL string,
	privateKey string,
	remoteAddress crypto.Address,
) error {
	ux.Logger.PrintToUser("Registering remote contract with home contract")
	feeInfo := WarpFeeInfo{
		Amount: big.NewInt(0),
	}
	_, _, err := contract.TxToMethod(
		rpcURL,
		false,
		crypto.Address{},
		privateKey,
		remoteAddress,
		nil,
		"register remote with home",
		nil,
		"registerWithHome((address, uint256))",
		feeInfo,
	)
	return err
}

func DeployERC20Remote(
	srcDir string,
	rpcURL string,
	privateKey string,
	warpRegistryAddress crypto.Address,
	warpManagerAddress crypto.Address,
	tokenHomeBlockchainID [32]byte,
	tokenHomeAddress crypto.Address,
	tokenHomeDecimals uint8,
	tokenRemoteName string,
	tokenRemoteSymbol string,
	tokenRemoteDecimals uint8,
) (crypto.Address, error) {
	binPath := filepath.Join(srcDir, "contracts/out/ERC20TokenRemote.sol/ERC20TokenRemote.bin")
	binBytes, err := os.ReadFile(binPath)
	if err != nil {
		return crypto.Address{}, err
	}
	tokenRemoteSettings := TokenRemoteSettings{
		WarpRegistryAddress:   warpRegistryAddress,
		WarpManager:           warpManagerAddress,
		TokenHomeBlockchainID: tokenHomeBlockchainID,
		TokenHomeAddress:      tokenHomeAddress,
		TokenHomeDecimals:     tokenHomeDecimals,
	}
	return contract.DeployContract(
		rpcURL,
		privateKey,
		binBytes,
		"((address, address, bytes32, address, uint8), string, string, uint8)",
		tokenRemoteSettings,
		tokenRemoteName,
		tokenRemoteSymbol,
		tokenRemoteDecimals,
	)
}

func DeployNativeRemote(
	srcDir string,
	rpcURL string,
	privateKey string,
	warpRegistryAddress crypto.Address,
	warpManagerAddress crypto.Address,
	tokenHomeBlockchainID [32]byte,
	tokenHomeAddress crypto.Address,
	tokenHomeDecimals uint8,
	nativeAssetSymbol string,
	initialReserveImbalance *big.Int,
	burnedFeesReportingRewardPercentage *big.Int,
) (crypto.Address, error) {
	binPath := filepath.Join(srcDir, "contracts/out/NativeTokenRemote.sol/NativeTokenRemote.bin")
	binBytes, err := os.ReadFile(binPath)
	if err != nil {
		return crypto.Address{}, err
	}
	tokenRemoteSettings := TokenRemoteSettings{
		WarpRegistryAddress:   warpRegistryAddress,
		WarpManager:           warpManagerAddress,
		TokenHomeBlockchainID: tokenHomeBlockchainID,
		TokenHomeAddress:      tokenHomeAddress,
		TokenHomeDecimals:     tokenHomeDecimals,
	}
	return contract.DeployContract(
		rpcURL,
		privateKey,
		binBytes,
		"((address, address, bytes32, address, uint8), string, uint256, uint256)",
		tokenRemoteSettings,
		nativeAssetSymbol,
		initialReserveImbalance,
		burnedFeesReportingRewardPercentage,
	)
}

func DeployERC20Home(
	srcDir string,
	rpcURL string,
	privateKey string,
	warpRegistryAddress crypto.Address,
	warpManagerAddress crypto.Address,
	erc20TokenAddress crypto.Address,
	erc20TokenDecimals uint8,
) (crypto.Address, error) {
	binPath := filepath.Join(srcDir, "contracts/out/ERC20TokenHome.sol/ERC20TokenHome.bin")
	binBytes, err := os.ReadFile(binPath)
	if err != nil {
		return crypto.Address{}, err
	}
	return contract.DeployContract(
		rpcURL,
		privateKey,
		binBytes,
		"(address, address, address, uint8)",
		warpRegistryAddress,
		warpManagerAddress,
		erc20TokenAddress,
		erc20TokenDecimals,
	)
}

func DeployNativeHome(
	srcDir string,
	rpcURL string,
	privateKey string,
	warpRegistryAddress crypto.Address,
	warpManagerAddress crypto.Address,
	wrappedNativeTokenAddress crypto.Address,
) (crypto.Address, error) {
	binPath := filepath.Join(srcDir, "contracts/out/NativeTokenHome.sol/NativeTokenHome.bin")
	binBytes, err := os.ReadFile(binPath)
	if err != nil {
		return crypto.Address{}, err
	}
	return contract.DeployContract(
		rpcURL,
		privateKey,
		binBytes,
		"(address, address, address)",
		warpRegistryAddress,
		warpManagerAddress,
		wrappedNativeTokenAddress,
	)
}

func DeployWrappedNativeToken(
	srcDir string,
	rpcURL string,
	privateKey string,
	tokenSymbol string,
) (crypto.Address, error) {
	binPath := filepath.Join(utils.ExpandHome(srcDir), "contracts/out/WrappedNativeToken.sol/WrappedNativeToken.bin")
	binBytes, err := os.ReadFile(binPath)
	if err != nil {
		return crypto.Address{}, err
	}
	return contract.DeployContract(
		rpcURL,
		privateKey,
		binBytes,
		"(string)",
		tokenSymbol,
	)
}
