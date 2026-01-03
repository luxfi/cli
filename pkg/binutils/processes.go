// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package binutils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/binpaths"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/ux"
	luxlog "github.com/luxfi/log"
	"github.com/luxfi/log/level"
	"github.com/luxfi/netrunner/client"
	"github.com/luxfi/netrunner/server"
	"github.com/luxfi/node/utils/perms"
	"github.com/shirou/gopsutil/process"
	"go.uber.org/zap"
)

// ErrGRPCTimeout is a common error message if the gRPC server can't be reached
var ErrGRPCTimeout = errors.New("timed out trying to contact backend controller, it is most probably not running")

// ProcessChecker is responsible for checking if the gRPC server is running
type ProcessChecker interface {
	// IsServerProcessRunning returns true if the gRPC server is running,
	// or false if not
	IsServerProcessRunning(app *application.Lux) (bool, error)
}

type realProcessRunner struct{}

// NewProcessChecker creates a new process checker which can respond if the server is running
func NewProcessChecker() ProcessChecker {
	return &realProcessRunner{}
}

// GRPCClientOp holds options for gRPC client operations.
type GRPCClientOp struct {
	avoidRPCVersionCheck bool
	endpoint             string // Custom endpoint, overrides default
}

// GRPCClientOpOption is a function that modifies GRPCClientOp.
type GRPCClientOpOption func(*GRPCClientOp)

func (op *GRPCClientOp) applyOpts(opts []GRPCClientOpOption) {
	for _, opt := range opts {
		opt(op)
	}
}

// WithAvoidRPCVersionCheck sets whether to skip RPC version checking.
func WithAvoidRPCVersionCheck(avoidRPCVersionCheck bool) GRPCClientOpOption {
	return func(op *GRPCClientOp) {
		op.avoidRPCVersionCheck = avoidRPCVersionCheck
	}
}

// WithEndpoint sets a custom gRPC endpoint for the client
func WithEndpoint(endpoint string) GRPCClientOpOption {
	return func(op *GRPCClientOp) {
		op.endpoint = endpoint
	}
}

// WithNetworkType configures the client to use the gRPC port for a specific network type
func WithNetworkType(networkType string) GRPCClientOpOption {
	return func(op *GRPCClientOp) {
		ports := GetGRPCPorts(networkType)
		op.endpoint = fmt.Sprintf(":%d", ports.Server)
	}
}

// NewGRPCClient hides away the details (params) of creating a gRPC server connection
func NewGRPCClient(opts ...GRPCClientOpOption) (client.Client, error) {
	op := GRPCClientOp{}
	op.applyOpts(opts)
	logLevel, err := luxlog.ToLevel(gRPCClientLogLevel)
	if err != nil {
		return nil, err
	}
	logFactory := luxlog.NewFactoryWithConfig(luxlog.Config{
		DisplayLevel: logLevel,
		LogLevel:     level.Fatal,
	})
	log, err := logFactory.Make("grpc-client")
	if err != nil {
		return nil, err
	}
	// Adapt the logger to the interface expected by netrunner
	adaptedLog := NewLoggerAdapter(log)

	// Use custom endpoint if provided, otherwise default
	endpoint := gRPCServerEndpoint
	if op.endpoint != "" {
		endpoint = op.endpoint
	}

	client, err := client.New(client.Config{
		Endpoint:    endpoint,
		DialTimeout: gRPCDialTimeout,
	}, adaptedLog)
	if errors.Is(err, context.DeadlineExceeded) {
		err = ErrGRPCTimeout
	}
	if client != nil && !op.avoidRPCVersionCheck {
		ctx := GetAsyncContext()
		rpcVersion, err := client.RPCVersion(ctx)
		if err != nil {
			return nil, err
		}
		// obtained using server API
		serverVersion := rpcVersion.Version
		// obtained from ANR source code
		clientVersion := server.RPCVersion
		if serverVersion != clientVersion {
			return nil, fmt.Errorf("trying to connect to a backend controller that uses a different RPC version (%d) than the CLI client (%d). Use 'network stop' to stop the controller and then restart the operation",
				serverVersion,
				clientVersion)
		}
	}
	return client, err
}

// NewGRPCServer creates a gRPC server with default ports (for backward compatibility)
func NewGRPCServer(snapshotsDir string) (server.Server, error) {
	return NewGRPCServerForNetwork(snapshotsDir, "mainnet")
}

// NewGRPCServerForNetwork creates a gRPC server with network-specific ports
func NewGRPCServerForNetwork(snapshotsDir, networkType string) (server.Server, error) {
	logFactory := luxlog.NewFactoryWithConfig(luxlog.Config{
		DisplayLevel: level.Info,
		LogLevel:     level.Fatal,
	})
	log, err := logFactory.Make("grpc-server")
	if err != nil {
		return nil, err
	}
	// Adapt the logger to the interface expected by netrunner
	adaptedLog := NewLoggerAdapter(log)

	// Get network-specific ports
	ports := GetGRPCPorts(networkType)

	return server.New(server.Config{
		Port:                fmt.Sprintf(":%d", ports.Server),
		GwPort:              fmt.Sprintf(":%d", ports.Gateway),
		DialTimeout:         gRPCDialTimeout,
		SnapshotsDir:        snapshotsDir,
		RedirectNodesOutput: true,
	}, adaptedLog)
}

// IsServerProcessRunning returns true if the gRPC server is running,
// or false if not
func (*realProcessRunner) IsServerProcessRunning(app *application.Lux) (bool, error) {
	pid, err := GetServerPID(app)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return false, err
		}
		return false, nil
	}

	// get OS process list
	procs, err := process.Processes()
	if err != nil {
		return false, err
	}

	p32 := int32(pid) //nolint:gosec // G115: PID values are within int32 range
	// iterate all processes...
	for _, p := range procs {
		if p.Pid == p32 {
			return true, nil
		}
	}
	return false, nil
}

type runFile struct {
	Pid                int    `json:"pid"`
	GRPCserverFileName string `json:"gRPCserverFileName"`
	NetworkType        string `json:"networkType,omitempty"` // "mainnet", "testnet", "local"
	GRPCPort           int    `json:"grpcPort,omitempty"`
	GatewayPort        int    `json:"gatewayPort,omitempty"`
}

// GetBackendLogFile returns the path to the backend log file.
func GetBackendLogFile(app *application.Lux) (string, error) {
	var rf runFile
	serverRunFilePath := app.GetRunFile()
	run, err := os.ReadFile(serverRunFilePath) //nolint:gosec // G304: Reading from app's data directory
	if err != nil {
		return "", fmt.Errorf("failed reading process info file at %s: %w", serverRunFilePath, err)
	}
	if err := json.Unmarshal(run, &rf); err != nil {
		return "", fmt.Errorf("failed unmarshalling server run file at %s: %w", serverRunFilePath, err)
	}

	return rf.GRPCserverFileName, nil
}

// GetServerPID returns the PID of the running server process.
func GetServerPID(app *application.Lux) (int, error) {
	var rf runFile
	serverRunFilePath := app.GetRunFile()
	run, err := os.ReadFile(serverRunFilePath) //nolint:gosec // G304: Reading from app's data directory
	if err != nil {
		return 0, fmt.Errorf("failed reading process info file at %s: %w", serverRunFilePath, err)
	}
	if err := json.Unmarshal(run, &rf); err != nil {
		return 0, fmt.Errorf("failed unmarshalling server run file at %s: %w", serverRunFilePath, err)
	}

	if rf.Pid == 0 {
		return 0, fmt.Errorf("failed reading pid from info file at %s: %w", serverRunFilePath, err)
	}
	return rf.Pid, nil
}

// StartServerProcess starts the gRPC server as a reentrant process of this binary
// for the default network type (mainnet).
// Deprecated: Use StartServerProcessForNetwork instead.
func StartServerProcess(app *application.Lux) error {
	return StartServerProcessForNetwork(app, "mainnet")
}

// StartServerProcessForNetwork starts a network-specific gRPC server using the
// external netrunner binary. Each network type (mainnet, testnet, local) gets its
// own server on a dedicated port. This allows running multiple networks simultaneously.
func StartServerProcessForNetwork(app *application.Lux, networkType string) error {
	// Get netrunner binary path, download if necessary
	netrunnerPath := binpaths.GetNetrunnerPath()
	if !binpaths.Exists(netrunnerPath) {
		var err error
		netrunnerPath, err = EnsureNetrunnerBinary(app, "latest")
		if err != nil {
			return fmt.Errorf("failed to get netrunner binary: %w", err)
		}
	}

	// Get network-specific ports
	ports := GetGRPCPorts(networkType)

	// Create output directory for logs
	outputDirPrefix := path.Join(app.GetRunDir(), "server", networkType)
	if err := os.MkdirAll(outputDirPrefix, 0o750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create timestamped directory
	timestamp := fmt.Sprintf("%d", os.Getpid())
	outputDir := filepath.Join(outputDirPrefix, timestamp)
	if err := os.MkdirAll(outputDir, 0o750); err != nil {
		return fmt.Errorf("failed to create timestamped output directory: %w", err)
	}

	outputFile, err := os.Create(path.Join(outputDir, "netrunner-server.log")) //nolint:gosec // G304: Creating log file in app's directory
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}

	// Build command args for netrunner server
	args := []string{
		"server",
		"--port", fmt.Sprintf(":%d", ports.Server),
		"--grpc-gateway-port", fmt.Sprintf(":%d", ports.Gateway),
		"--log-dir", outputDir,
		"--snapshots-dir", app.GetSnapshotsDir(),
	}

	cmd := exec.Command(netrunnerPath, args...) //nolint:gosec // G204: Running our netrunner binary
	cmd.Env = append(os.Environ(), fmt.Sprintf("LUX_NETWORK_TYPE=%s", networkType))
	cmd.Stdout = outputFile
	cmd.Stderr = outputFile

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start netrunner server: %w", err)
	}

	ux.Logger.PrintToUser("Backend controller (%s) started, pid: %d, grpc: %d, output: %s",
		networkType, cmd.Process.Pid, ports.Server, outputFile.Name())

	rf := runFile{
		Pid:                cmd.Process.Pid,
		GRPCserverFileName: outputFile.Name(),
		NetworkType:        networkType,
		GRPCPort:           ports.Server,
		GatewayPort:        ports.Gateway,
	}

	rfBytes, err := json.Marshal(&rf)
	if err != nil {
		return err
	}

	// Use network-specific run file
	runFilePath := app.GetRunFileForNetwork(networkType)
	if err := os.WriteFile(runFilePath, rfBytes, perms.ReadWrite); err != nil {
		app.Log.Warn("could not write gRPC process info to file", zap.Error(err))
	}
	return nil
}

// GetAsyncContext returns a timeout context with the cancel function suppressed
// For local networks, this uses a short timeout (15s) since operations should complete quickly
func GetAsyncContext() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), constants.RequestTimeout)
	// don't call since "start" is async
	// and the top-level context here "ctx" is passed
	// to all underlying function calls
	// just set the timeout to halt "Start" async ops
	// when the deadline is reached
	_ = cancel

	return ctx
}

// GetDeployContext returns a timeout context for chain deployment operations.
// For local networks, deployment should complete in <30s:
//   - Blockchain creation: ~5-10s (P-chain tx)
//   - Chain health: ~5-10s (node sync)
//
// If deployment takes longer, something is wrong and we fail fast.
func GetDeployContext() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	_ = cancel
	return ctx
}

// KillgRPCServerProcess kills the default (mainnet) gRPC server.
// Deprecated: Use KillgRPCServerProcessForNetwork instead.
func KillgRPCServerProcess(app *application.Lux) error {
	return KillgRPCServerProcessForNetwork(app, "mainnet")
}

// KillgRPCServerProcessForNetwork kills a network-specific gRPC server.
func KillgRPCServerProcessForNetwork(app *application.Lux, networkType string) error {
	cli, err := NewGRPCClient(WithAvoidRPCVersionCheck(true), WithNetworkType(networkType))
	if err != nil {
		return err
	}
	defer func() { _ = cli.Close() }()
	ctx := GetAsyncContext()
	_, err = cli.Stop(ctx)
	if err != nil {
		if server.IsServerError(err, server.ErrNotBootstrapped) {
			app.Log.Debug("No local network running")
		} else {
			app.Log.Debug("failed stopping local network", zap.Error(err))
		}
	}

	pid, err := GetServerPIDForNetwork(app, networkType)
	if err != nil {
		return fmt.Errorf("failed getting PID from run file: %w", err)
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("could not find process with pid %d: %w", pid, err)
	}
	if err := proc.Signal(os.Interrupt); err != nil {
		return fmt.Errorf("failed killing process with pid %d: %w", pid, err)
	}

	serverRunFilePath := app.GetRunFileForNetwork(networkType)
	if err := os.Remove(serverRunFilePath); err != nil {
		return fmt.Errorf("failed removing run file %s: %w", serverRunFilePath, err)
	}
	return nil
}

// GetServerPIDForNetwork returns the server PID for a specific network type.
func GetServerPIDForNetwork(app *application.Lux, networkType string) (int, error) {
	var rf runFile
	serverRunFilePath := app.GetRunFileForNetwork(networkType)
	run, err := os.ReadFile(serverRunFilePath) //nolint:gosec // G304: Reading from app's data directory
	if err != nil {
		return 0, fmt.Errorf("failed reading process info file at %s: %w", serverRunFilePath, err)
	}
	if err := json.Unmarshal(run, &rf); err != nil {
		return 0, fmt.Errorf("failed unmarshalling server run file at %s: %w", serverRunFilePath, err)
	}

	if rf.Pid == 0 {
		return 0, fmt.Errorf("failed reading pid from info file at %s: %w", serverRunFilePath, err)
	}
	return rf.Pid, nil
}

// IsServerProcessRunningForNetwork checks if a network-specific gRPC server is running.
func IsServerProcessRunningForNetwork(app *application.Lux, networkType string) (bool, error) {
	pid, err := GetServerPIDForNetwork(app, networkType)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || strings.Contains(err.Error(), "no such file") {
			return false, nil
		}
		return false, err
	}

	// get OS process list
	procs, err := process.Processes()
	if err != nil {
		return false, err
	}

	p32 := int32(pid) //nolint:gosec // G115: PID values are within int32 range
	for _, p := range procs {
		if p.Pid == p32 {
			return true, nil
		}
	}
	return false, nil
}

// WatchServerProcess monitors the server process for signals or errors.
func WatchServerProcess(serverCancel context.CancelFunc, errc chan error, logger luxlog.Logger) {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-sigc:
		logger.Warn("signal received; closing server", "signal", sig.String())
		serverCancel()
		err := <-errc
		logger.Warn("closed server", "error", err)
	case err := <-errc:
		logger.Warn("server closed", "error", err)
		serverCancel()
	}
}
