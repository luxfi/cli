// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package networkcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type rpcRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error"`
	ID      int             `json:"id"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func callRPC(ctx context.Context, rpcURL string, req *rpcRequest, result interface{}) error {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", rpcURL, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var rpcResp rpcResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return err
	}

	if rpcResp.Error != nil {
		return fmt.Errorf("RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return json.Unmarshal(rpcResp.Result, result)
}

func getCurrentBlock(ctx context.Context, rpcURL string) (uint64, error) {
	req := &rpcRequest{
		JSONRPC: "2.0",
		Method:  "eth_blockNumber",
		Params:  []interface{}{},
		ID:      1,
	}

	var result string
	if err := callRPC(ctx, rpcURL, req, &result); err != nil {
		return 0, err
	}

	var blockNum uint64
	_, err := fmt.Sscanf(result, "0x%x", &blockNum)
	return blockNum, err
}

func getBlocks(ctx context.Context, rpcURL string, start, end uint64) ([]interface{}, error) {
	blocks := make([]interface{}, 0, end-start+1)

	// Fetch blocks one by one using standard eth_getBlockByNumber
	for blockNum := start; blockNum <= end; blockNum++ {
		req := &rpcRequest{
			JSONRPC: "2.0",
			Method:  "eth_getBlockByNumber",
			Params:  []interface{}{fmt.Sprintf("0x%x", blockNum), true}, // true = include transactions
			ID:      1,
		}

		var block interface{}
		if err := callRPC(ctx, rpcURL, req, &block); err != nil {
			return nil, fmt.Errorf("failed to get block %d: %w", blockNum, err)
		}
		blocks = append(blocks, block)
	}

	return blocks, nil
}

func importBlocks(ctx context.Context, rpcURL string, blocks []interface{}) (int, error) {
	req := &rpcRequest{
		JSONRPC: "2.0",
		Method:  "migrate_importBlocks",
		Params:  []interface{}{blocks},
		ID:      1,
	}

	var count int
	if err := callRPC(ctx, rpcURL, req, &count); err != nil {
		return 0, err
	}
	return count, nil
}
