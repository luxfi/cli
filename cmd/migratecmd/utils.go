package migratecmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/luxfi/cli/pkg/ux"
)

// RPCRequest represents a JSON-RPC request
type RPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

// RPCResponse represents a JSON-RPC response
type RPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
	Error   *RPCError       `json:"error"`
	ID      int             `json:"id"`
}

// RPCError represents a JSON-RPC error
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// runMigration performs migration via RPC to source and destination nodes
func runMigration(sourceRPC, destRPC string, chainID int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// Discover RPC endpoints from netrunner if not provided
	if sourceRPC == "" || destRPC == "" {
		ux.Logger.PrintToUser("🔍 Discovering RPC endpoints from netrunner...")
		discoveredSource, discoveredDest, err := discoverRPCEndpoints(ctx)
		if err != nil {
			ux.Logger.PrintToUser("⚠️  Could not auto-discover endpoints: %v", err)
			ux.Logger.PrintToUser("💡 Use --source-rpc and --dest-rpc flags to specify manually")
			return err
		}
		if sourceRPC == "" {
			sourceRPC = discoveredSource
		}
		if destRPC == "" {
			destRPC = discoveredDest
		}
		ux.Logger.PrintToUser("  Discovered source: %s", sourceRPC)
		ux.Logger.PrintToUser("  Discovered dest: %s", destRPC)
	}

	ux.Logger.PrintToUser("🚀 Starting RPC-based migration...")
	ux.Logger.PrintToUser("  Source RPC: %s", sourceRPC)
	ux.Logger.PrintToUser("  Destination RPC: %s", destRPC)
	ux.Logger.PrintToUser("  Chain ID: %d", chainID)
	ux.Logger.PrintToUser("")

	// Step 1: Get chain info from source
	ux.Logger.PrintToUser("📊 Step 1: Getting chain information from source...")
	currentBlock, err := getCurrentBlock(ctx, sourceRPC)
	if err != nil {
		return fmt.Errorf("failed to get current block: %w", err)
	}
	ux.Logger.PrintToUser("  Current block: %d", currentBlock)
	ux.Logger.PrintToUser("")

	// Step 2: Export blocks from source
	ux.Logger.PrintToUser("📤 Step 2: Exporting blocks from source...")
	ux.Logger.PrintToUser("  This will use admin_exportChain RPC method")
	ux.Logger.PrintToUser("  Blocks: 0 to %d", currentBlock)
	ux.Logger.PrintToUser("")

	// Step 3: Stream blocks via RPC
	// Note: Traditional admin_exportChain exports to a file on the NODE's filesystem
	// For true RPC streaming, we'd need a new eth_streamBlocks method
	// For now, we'll use the existing admin_exportChain and document the limitation

	ux.Logger.PrintToUser("⚠️  Current Limitation:")
	ux.Logger.PrintToUser("  admin_exportChain writes to node's local filesystem")
	ux.Logger.PrintToUser("  For remote export, need to implement eth_streamBlocks")
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("💡 Recommended Approach:")
	ux.Logger.PrintToUser("  1. Deploy source EVM node with netrunner (readonly DB)")
	ux.Logger.PrintToUser("  2. Call admin_exportChain RPC (exports to node's /tmp/)")
	ux.Logger.PrintToUser("  3. Fetch exported file via HTTP or rsync from node")
	ux.Logger.PrintToUser("  4. Deploy destination C-Chain with netrunner")
	ux.Logger.PrintToUser("  5. Upload file to destination node")
	ux.Logger.PrintToUser("  6. Call admin_importChain RPC on destination")
	ux.Logger.PrintToUser("")
	ux.Logger.PrintToUser("🔧 Next Steps for Full RPC Control:")
	ux.Logger.PrintToUser("  - Add eth_streamBlocks RPC method to EVM")
	ux.Logger.PrintToUser("  - Streams blocks as JSON over RPC")
	ux.Logger.PrintToUser("  - Add eth_importBlocks RPC method")
	ux.Logger.PrintToUser("  - Accepts block stream and imports")

	_ = ctx
	return nil
}

// getCurrentBlock gets the current block number via RPC
func getCurrentBlock(ctx context.Context, rpcURL string) (uint64, error) {
	req := &RPCRequest{
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

// callRPC makes a JSON-RPC call
func callRPC(ctx context.Context, rpcURL string, req *RPCRequest, result interface{}) error {
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

	var rpcResp RPCResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return err
	}

	if rpcResp.Error != nil {
		return fmt.Errorf("RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return json.Unmarshal(rpcResp.Result, result)
}

// discoverRPCEndpoints discovers RPC endpoints from netrunner
// Internal RPC uses port 9630 (not 9650)
func discoverRPCEndpoints(ctx context.Context) (source, dest string, err error) {
	// TODO: Query netrunner for running nodes and their RPC endpoints
	// netrunner should expose API to list nodes with their ports
	// For now, use default internal ports

	// Internal RPC port is 9630 (not 9650!)
	source = "http://127.0.0.1:9630/ext/bc/C/rpc"  // Internal, not public
	dest = "http://127.0.0.1:9640/ext/bc/C/rpc"    // Next node in fleet

	ux.Logger.PrintToUser("⚠️  Using default internal ports (9630, 9640)")
	ux.Logger.PrintToUser("💡 TODO: Query netrunner API for actual node endpoints")

	return source, dest, nil
}

// Placeholder functions to fix later
func createPChainGenesis(outputDir string, numValidators int) error {
	return fmt.Errorf("createPChainGenesis not implemented")
}

func createNodeConfig(outputDir string, nodeCount int) error {
	return fmt.Errorf("createNodeConfig not implemented")
}

// Other migrate functions can be added here as needed

func generateNodeConfigs(outputDir string, nodeCount int) error {
	return fmt.Errorf("generateNodeConfigs not implemented")
}

func startBootstrapNodes(outputDir string, nodeCount int, detached bool) error {
	// Start bootstrap nodes with optional detached mode
	if detached {
		ux.Logger.PrintToUser("Starting nodes in detached mode...")
	}
	return fmt.Errorf("startBootstrapNodes not implemented")
}

func validateNetwork(endpoint string) error {
	return fmt.Errorf("validateNetwork not implemented")
}
