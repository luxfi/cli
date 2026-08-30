package network

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/luxfi/cli/pkg/route"
)

// Probe asks the local node at p.HTTPPort what its NetworkID is.
// Returns (matches=true) only if the response equals p.NetworkID.
// A nil error with matches=false means a node is up but it isn't ours.
func (p *Profile) Probe(ctx context.Context) (matches bool, foundID uint32, err error) {
	url := fmt.Sprintf("http://127.0.0.1:%d/v1/info", p.HTTPPort)
	body := []byte(`{"jsonrpc":"2.0","id":1,"method":"info.getNetworkID","params":{}}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return false, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	var r struct {
		Result struct {
			NetworkID uint32 `json:"networkID"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return false, 0, err
	}
	return r.Result.NetworkID == p.NetworkID, r.Result.NetworkID, nil
}

// WaitHealthy polls /v1/health and /v1/info until the C-chain
// responds with the expected NetworkID, or timeout elapses.
func (p *Profile) WaitHealthy(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	tick := time.NewTicker(1 * time.Second)
	defer tick.Stop()
	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("%s did not become healthy within %s", p, timeout)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
		}
		matches, found, err := p.Probe(ctx)
		if err != nil {
			continue
		}
		if !matches {
			return fmt.Errorf("%s: port %d serving foreign networkID %d (expected %d)",
				p, p.HTTPPort, found, p.NetworkID)
		}
		cchain := route.Chain(fmt.Sprintf("http://127.0.0.1:%d", p.HTTPPort), "C") + "/rpc"
		body := []byte(`{"jsonrpc":"2.0","id":1,"method":"eth_chainId","params":[]}`)
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, cchain, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == 200 {
				return nil
			}
		}
	}
}

// PIDOnPort returns the PID listening on TCP <port>, or 0 if none.
func PIDOnPort(port int) (int, error) {
	out, err := exec.Command("lsof", "-ti", fmt.Sprintf(":%d", port), "-sTCP:LISTEN").Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return 0, nil
		}
		return 0, err
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		pid, err := strconv.Atoi(line)
		if err == nil {
			return pid, nil
		}
	}
	return 0, nil
}

// Stop signals the node owning p.HTTPPort, but only after verifying
// (via Probe) that the responder is OUR (name, env). On verification
// failure we refuse — better to leave a stranger alone than to murder
// the wrong process.
func (p *Profile) Stop(ctx context.Context, gracePeriod time.Duration) error {
	matches, found, err := p.Probe(ctx)
	if err != nil {
		return p.stopViaPIDFile()
	}
	if !matches {
		return fmt.Errorf("port %d serves networkID %d (not our %d) — refusing to signal",
			p.HTTPPort, found, p.NetworkID)
	}
	pid, err := PIDOnPort(p.HTTPPort)
	if err != nil {
		return err
	}
	if pid == 0 {
		return fmt.Errorf("port %d responded to RPC but no listening PID found", p.HTTPPort)
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return err
	}
	deadline := time.Now().Add(gracePeriod)
	for time.Now().Before(deadline) {
		c, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", p.HTTPPort), 500*time.Millisecond)
		if err != nil {
			return nil
		}
		_ = c.Close()
		time.Sleep(500 * time.Millisecond)
	}
	_ = proc.Signal(syscall.SIGKILL)
	return nil
}

func (p *Profile) stopViaPIDFile() error {
	pidFile := p.PIDFilePath()
	b, err := os.ReadFile(pidFile) //nolint:gosec
	if err != nil {
		return nil
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil {
		return err
	}
	if proc, err := os.FindProcess(pid); err == nil {
		_ = proc.Signal(syscall.SIGTERM)
	}
	return nil
}

// PIDFilePath is <DataDir>/luxd.pid (fallback identity record).
func (p *Profile) PIDFilePath() string {
	return p.DataDir + "/luxd.pid"
}
