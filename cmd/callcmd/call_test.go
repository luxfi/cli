package callcmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zap-proto/zip"
)

// up waits for the listener. zip.Serve starts it on a goroutine and returns, so
// a test that dialled immediately would be racing the socket into existence.
func up(t *testing.T, sock string) string {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(sock); err == nil {
			return sock
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s never came up", sock)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// Height is a chain's last accepted block height.
type Height struct {
	Height uint64 `json:"height"`
}

// Stake asks what one network has staked.
type Stake struct {
	NetID string `json:"netID" validate:"required"`
}

// Staked is the answer.
type Staked struct {
	NetID  string `json:"netID"`
	Staked uint64 `json:"staked"`
}

// getHeight answers the last accepted height.
func getHeight(context.Context, *struct{}) (*Height, error) { return &Height{Height: 7}, nil }

// getStake answers one network's stake.
func getStake(_ context.Context, in *Stake) (*Staked, error) {
	return &Staked{NetID: in.NetID, Staked: uint64(len(in.NetID))}, nil
}

func serve(t *testing.T) string {
	t.Helper()
	sock := filepath.Join(t.TempDir(), "node.sock")
	app := zip.New(zip.Config{AppName: "platform", DisableStartupMessage: true})
	zip.Get(app, "/v1/platform/height", getHeight)
	zip.Post(app, "/v1/platform/stake", getStake)
	if _, err := zip.Serve(app, sock); err != nil {
		t.Fatalf("serve: %v", err)
	}
	t.Cleanup(func() { _ = app.Shutdown() })
	return up(t, sock)
}

func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	c := NewCmd()
	c.SetOut(&out)
	c.SetErr(&out)
	c.SetArgs(args)
	err := c.Execute()
	return out.String(), err
}

// TestCall_RunsAnOpNobodyWroteDown is the claim: this binary contains no list
// of the node's operations, and running one works anyway. The command, its
// flags and its help are read off the document the node derives from its own
// registry, so an op registered on the node is a command here with nothing
// rebuilt.
func TestCall_RunsAnOpNobodyWroteDown(t *testing.T) {
	sock := serve(t)

	out, err := run(t, "--at", sock)
	if err != nil {
		t.Fatalf("listing: %v\n%s", err, out)
	}
	if !strings.Contains(out, "platform") {
		t.Fatalf("the node's service is not listed:\n%s", out)
	}

	out, err = run(t, "--at", sock, "platform", "height-get")
	if err != nil {
		t.Fatalf("running an op: %v\n%s", err, out)
	}
	if !strings.Contains(out, "7") {
		t.Fatalf("the handler's answer did not come back:\n%s", out)
	}
}

// TestCall_AnInputFieldIsAFlag pins the other half: the flags of an operation
// are the fields of its input, read off the same document. Nothing here binds
// them.
func TestCall_AnInputFieldIsAFlag(t *testing.T) {
	sock := serve(t)

	out, err := run(t, "--at", sock, "platform", "stake-create", "--net-id", "8675309")
	if err != nil {
		t.Fatalf("running with a flag: %v\n%s", err, out)
	}
	if !strings.Contains(out, "8675309") {
		t.Fatalf("the flag did not reach the handler:\n%s", out)
	}
	if !strings.Contains(out, "7") { // len("8675309")
		t.Fatalf("the handler's answer did not come back:\n%s", out)
	}
}

// TestCall_SaysSoWhenTheNodePublishesNothing is the state the fleet is in
// today: a node serving no typed ops has a document with no operations in it,
// and a CLI that printed an empty list would look like a CLI that was broken.
func TestCall_SaysSoWhenTheNodePublishesNothing(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "bare.sock")
	app := zip.New(zip.Config{AppName: "bare", DisableStartupMessage: true})
	if _, err := zip.Serve(app, sock); err != nil {
		t.Fatalf("serve: %v", err)
	}
	t.Cleanup(func() { _ = app.Shutdown() })
	up(t, sock)

	out, err := run(t, "--at", sock)
	if err == nil {
		t.Fatalf("a node with no ops should say so, got:\n%s", out)
	}
	if !strings.Contains(err.Error(), "no operations") {
		t.Fatalf("unhelpful refusal: %v", err)
	}
}
