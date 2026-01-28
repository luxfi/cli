// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utils

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/luxfi/evm/ethclient"
	"github.com/luxfi/evm/rpc"
)

// EVMClient wraps the native Lux EVM client
type EVMClient struct {
	client    ethclient.Client
	rpcClient *rpc.Client
	timeout   time.Duration
}

// NewEVMClientWithTimeout creates an EVM client with a custom timeout
func NewEVMClientWithTimeout(url string, timeout time.Duration) (*EVMClient, error) {
	// Create native Lux EVM client
	client, err := ethclient.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to dial EVM RPC: %w", err)
	}

	return &EVMClient{
		client:    client,
		rpcClient: client.Client(),
		timeout:   timeout,
	}, nil
}

// BlockNumber gets the current block number
func (c *EVMClient) BlockNumber(ctx context.Context) (uint64, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.BlockNumber(ctx)
}

// ChainID gets the chain ID
func (c *EVMClient) ChainID(ctx context.Context) (*big.Int, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return c.client.ChainID(ctx)
}

// Syncing checks if the node is syncing
func (c *EVMClient) Syncing(ctx context.Context) (interface{}, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	var result interface{}
	err := c.rpcClient.CallContext(ctx, &result, "eth_syncing")
	if err != nil {
		return false, err
	}
	return result, nil
}

// ClientVersion gets the client version
func (c *EVMClient) ClientVersion(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	var result string
	err := c.rpcClient.CallContext(ctx, &result, "web3_clientVersion")
	if err != nil {
		return "", err
	}
	return result, nil
}

// Close closes the client connection
func (c *EVMClient) Close() {
	if c.client != nil {
		c.client.Close()
	}
}
