package mcpcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	zapmcp "github.com/zap-proto/mcp"
	"github.com/zap-proto/zip"
)

// Height is a chain's last accepted block height.
type Height struct {
	Height uint64 `json:"height"`
}

// getHeight answers the last accepted height.
func getHeight(context.Context, *struct{}) (*Height, error) { return &Height{Height: 7}, nil }

// TestRelay_ReachesTheDoorOverZAP is the whole claim of `lux mcp`: an agent
// speaking MCP on stdio reaches a node's tools with no HTTP anywhere between
// them. The node here is a zip app with a door on a unix socket — frames, no
// listener, no request, no status code — and the tool it offers is a typed op
// nobody registered as a tool.
func TestRelay_ReachesTheDoorOverZAP(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "node.sock")

	app := zip.New(zip.Config{
		AppName:               "platform",
		DisableStartupMessage: true,
		MCP:                   zip.MCPConfig{Addr: sock},
	})
	zip.Get(app, "/v1/platform/height", getHeight)
	if _, err := zip.Serve(app, filepath.Join(t.TempDir(), "app.sock")); err != nil {
		t.Fatalf("serve: %v", err)
	}
	defer func() { _ = app.Shutdown() }()

	// The door comes up on its own address, beside the app's.
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := zapmcp.Dial(zip.NetworkOf(sock), sock).Do(&zapmcp.Frame{
			Kind: zapmcp.Request, ID: "0", Method: "tools/list",
		}); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the MCP door never came up")
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Now drive the relay exactly as an MCP client would: newline-delimited
	// JSON-RPC in, newline-delimited JSON-RPC out.
	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"get_platform_height","arguments":{}}}
`)
	var out bytes.Buffer
	if err := relay(zapmcp.Dial(zip.NetworkOf(sock), sock), in, &out); err != nil {
		t.Fatalf("relay: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d answers, want 2:\n%s", len(lines), out.String())
	}

	// The tool list is the registry: the op is there because it was registered,
	// not because anything here named it.
	if !strings.Contains(lines[0], "get_platform_height") {
		t.Errorf("tools/list does not offer the registered op:\n%s", lines[0])
	}
	// And calling it runs the handler and brings the value back.
	if !strings.Contains(lines[1], "7") {
		t.Errorf("tools/call did not carry the handler's answer:\n%s", lines[1])
	}
	// Both answers correlate to the ids the client sent, verbatim.
	for i, want := range []string{`"id":1`, `"id":2`} {
		var got map[string]json.RawMessage
		if err := json.Unmarshal([]byte(lines[i]), &got); err != nil {
			t.Fatalf("answer %d is not JSON-RPC: %v", i, err)
		}
		if string(got["id"]) != want[len(`"id":`):] {
			t.Errorf("answer %d has id %s, want %s", i, got["id"], want)
		}
	}
}

// TestRelay_ADeadDoorIsAnAnswer pins that a client is never left holding an id
// nothing will answer. A relay that dropped the exchange would hang the agent.
func TestRelay_ADeadDoorIsAnAnswer(t *testing.T) {
	nowhere := filepath.Join(t.TempDir(), "absent.sock")
	var out bytes.Buffer
	err := relay(
		zapmcp.Dial(zip.NetworkOf(nowhere), nowhere),
		strings.NewReader(`{"jsonrpc":"2.0","id":9,"method":"tools/list"}`+"\n"),
		&out,
	)
	if err != nil {
		t.Fatalf("relay should carry on after a failed exchange: %v", err)
	}
	if !strings.Contains(out.String(), `"error"`) || !strings.Contains(out.String(), `"id":9`) {
		t.Fatalf("a failed exchange did not answer the client:\n%s", out.String())
	}
}
