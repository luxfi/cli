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
	"syscall"

	"github.com/docker/docker/pkg/reexec"
	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/ux"
	luxlog "github.com/luxfi/log"
	"github.com/luxfi/log/level"
	"github.com/luxfi/netrunner/client"
	"github.com/luxfi/netrunner/server"
	"github.com/luxfi/netrunner/utils"
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

type GRPCClientOp struct {
	avoidRPCVersionCheck bool
}

type GRPCClientOpOption func(*GRPCClientOp)

func (op *GRPCClientOp) applyOpts(opts []GRPCClientOpOption) {
	for _, opt := range opts {
		opt(op)
	}
}

func WithAvoidRPCVersionCheck(avoidRPCVersionCheck bool) GRPCClientOpOption {
	return func(op *GRPCClientOp) {
		op.avoidRPCVersionCheck = avoidRPCVersionCheck
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
	client, err := client.New(client.Config{
		Endpoint:    gRPCServerEndpoint,
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

// NewGRPCClient hides away the details (params) of creating a gRPC server
func NewGRPCServer(snapshotsDir string) (server.Server, error) {
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
	return server.New(server.Config{
		Port:                gRPCServerEndpoint,
		GwPort:              gRPCGatewayEndpoint,
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

	p32 := int32(pid)
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
}

func GetBackendLogFile(app *application.Lux) (string, error) {
	var rf runFile
	serverRunFilePath := app.GetRunFile()
	run, err := os.ReadFile(serverRunFilePath)
	if err != nil {
		return "", fmt.Errorf("failed reading process info file at %s: %w", serverRunFilePath, err)
	}
	if err := json.Unmarshal(run, &rf); err != nil {
		return "", fmt.Errorf("failed unmarshalling server run file at %s: %w", serverRunFilePath, err)
	}

	return rf.GRPCserverFileName, nil
}

func GetServerPID(app *application.Lux) (int, error) {
	var rf runFile
	serverRunFilePath := app.GetRunFile()
	run, err := os.ReadFile(serverRunFilePath)
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
// it just executes `cli backend start`
func StartServerProcess(app *application.Lux) error {
	thisBin := reexec.Self()

	args := []string{constants.BackendCmd}
	cmd := exec.Command(thisBin, args...)
	// Inherit environment variables from the parent process
	// This is important for passing DISABLE_MIGRATION_DETECTION and other env vars to the backend
	cmd.Env = os.Environ()

	outputDirPrefix := path.Join(app.GetRunDir(), "server")
	outputDir, err := utils.MkDirWithTimestamp(outputDirPrefix)
	if err != nil {
		return err
	}

	outputFile, err := os.Create(path.Join(outputDir, "cli-backend.log"))
	if err != nil {
		return err
	}
	// Direct output to dedicated backend log file for easier debugging
	// This keeps backend logs separate from main application logs
	cmd.Stdout = outputFile
	cmd.Stderr = outputFile

	if err := cmd.Start(); err != nil {
		return err
	}

	ux.Logger.PrintToUser("Backend controller started, pid: %d, output at: %s", cmd.Process.Pid, outputFile.Name())

	rf := runFile{
		Pid:                cmd.Process.Pid,
		GRPCserverFileName: outputFile.Name(),
	}

	rfBytes, err := json.Marshal(&rf)
	if err != nil {
		return err
	}

	if err := os.WriteFile(app.GetRunFile(), rfBytes, perms.ReadWrite); err != nil {
		app.Log.Warn("could not write gRPC process info to file", zap.Error(err))
	}
	return nil
}

// GetAsyncContext returns a timeout context with the cancel function suppressed
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

func KillgRPCServerProcess(app *application.Lux) error {
	cli, err := NewGRPCClient(WithAvoidRPCVersionCheck(true))
	if err != nil {
		return err
	}
	defer cli.Close()
	ctx := GetAsyncContext()
	_, err = cli.Stop(ctx)
	if err != nil {
		if server.IsServerError(err, server.ErrNotBootstrapped) {
			app.Log.Debug("No local network running")
		} else {
			app.Log.Debug("failed stopping local network", zap.Error(err))
		}
	}

	pid, err := GetServerPID(app)
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

	serverRunFilePath := app.GetRunFile()
	if err := os.Remove(serverRunFilePath); err != nil {
		return fmt.Errorf("failed removing run file %s: %w", serverRunFilePath, err)
	}
	return nil
}

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
