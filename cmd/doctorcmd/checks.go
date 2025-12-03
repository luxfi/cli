// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package doctorcmd

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/constants"
	"github.com/luxfi/cli/pkg/ux"
	"golang.org/x/mod/semver"
)

// CheckStatus represents the result of a single check
type CheckStatus int

const (
	StatusOK CheckStatus = iota
	StatusWarn
	StatusError
)

// CheckResult holds the outcome of a single check
type CheckResult struct {
	Name          string
	Status        CheckStatus
	Message       string
	FixSuggestion string
	CanAutoFix    bool
	AutoFix       func() error
}

// Doctor performs environment checks
type Doctor struct {
	app     *application.Lux
	fixMode bool
	results []CheckResult
	output  io.Writer
}

// Version requirements
const (
	MinGoVersion     = "1.21.0"
	MinDockerVersion = "20.0.0"
	MinDiskSpaceGB   = 50 // GB for state storage
)

// NewDoctor creates a new Doctor instance
func NewDoctor(app *application.Lux, fixMode bool) *Doctor {
	return &Doctor{
		app:     app,
		fixMode: fixMode,
		results: make([]CheckResult, 0),
		output:  os.Stdout,
	}
}

// printToUser prints a message to the user (handles nil logger gracefully)
func (d *Doctor) printToUser(msg string, args ...interface{}) {
	formattedMsg := fmt.Sprintf(msg, args...)
	if ux.Logger != nil {
		ux.Logger.PrintToUser("%s", formattedMsg)
	} else {
		fmt.Fprintln(d.output, formattedMsg)
	}
}

// Run executes all checks and reports results
func (d *Doctor) Run() error {
	d.printToUser("Lux CLI Doctor")
	d.printToUser("==============")
	d.printToUser("")

	// Run all checks
	d.checkGoVersion()
	d.checkDockerAvailability()
	d.checkLuxNodeBinary()
	d.checkNetworkConnectivity()
	d.checkDiskSpace()
	d.checkCLIDirectories()

	// Print summary
	d.printResults()

	// Attempt fixes if requested
	if d.fixMode {
		return d.attemptFixes()
	}

	// Return error if any critical issues
	for _, r := range d.results {
		if r.Status == StatusError {
			return fmt.Errorf("environment check failed: see above for details")
		}
	}

	return nil
}

// checkGoVersion verifies Go installation and version
func (d *Doctor) checkGoVersion() {
	result := CheckResult{
		Name: "Go Version",
	}

	goPath, err := exec.LookPath("go")
	if err != nil {
		result.Status = StatusError
		result.Message = "Go not found in PATH"
		result.FixSuggestion = "Install Go from https://go.dev/dl/ (minimum version " + MinGoVersion + ")"
		d.results = append(d.results, result)
		return
	}

	cmd := exec.Command(goPath, "version")
	output, err := cmd.Output()
	if err != nil {
		result.Status = StatusError
		result.Message = "Failed to get Go version: " + err.Error()
		result.FixSuggestion = "Ensure Go is properly installed"
		d.results = append(d.results, result)
		return
	}

	versionStr := string(output)
	parts := strings.Fields(versionStr)
	if len(parts) < 3 {
		result.Status = StatusError
		result.Message = "Unable to parse Go version"
		d.results = append(d.results, result)
		return
	}

	version := strings.TrimPrefix(parts[2], "go")
	semVersion := "v" + version
	minSemVersion := "v" + MinGoVersion

	if semver.Compare(semVersion, minSemVersion) < 0 {
		result.Status = StatusWarn
		result.Message = fmt.Sprintf("Go %s installed, minimum recommended is %s", version, MinGoVersion)
		result.FixSuggestion = "Upgrade Go from https://go.dev/dl/"
	} else {
		result.Status = StatusOK
		result.Message = fmt.Sprintf("Go %s (path: %s)", version, goPath)
	}

	d.results = append(d.results, result)
}

// checkDockerAvailability verifies Docker installation
func (d *Doctor) checkDockerAvailability() {
	result := CheckResult{
		Name: "Docker",
	}

	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		result.Status = StatusWarn
		result.Message = "Docker not found in PATH"
		result.FixSuggestion = "Install Docker from https://docs.docker.com/get-docker/ (optional)"
		d.results = append(d.results, result)
		return
	}

	cmd := exec.Command(dockerPath, "info")
	_, err = cmd.Output()
	if err != nil {
		result.Status = StatusWarn
		result.Message = "Docker installed but daemon not running"
		result.FixSuggestion = "Start Docker daemon"
		result.CanAutoFix = true
		result.AutoFix = func() error {
			if runtime.GOOS == "darwin" {
				return exec.Command("open", "-a", "Docker").Run()
			}
			return exec.Command("systemctl", "start", "docker").Run()
		}
		d.results = append(d.results, result)
		return
	}

	cmd = exec.Command(dockerPath, "version", "--format", "{{.Server.Version}}")
	output, err := cmd.Output()
	if err != nil {
		result.Status = StatusOK
		result.Message = fmt.Sprintf("Docker available (path: %s)", dockerPath)
	} else {
		version := strings.TrimSpace(string(output))
		result.Status = StatusOK
		result.Message = fmt.Sprintf("Docker %s (path: %s)", version, dockerPath)
	}

	d.results = append(d.results, result)
}

// checkLuxNodeBinary verifies luxd binary availability
func (d *Doctor) checkLuxNodeBinary() {
	result := CheckResult{
		Name: "Lux Node",
	}

	locations := []string{"luxd"}
	home, err := os.UserHomeDir()
	if err == nil {
		locations = append(locations,
			filepath.Join(home, ".lux", "bin", "luxd"),
			filepath.Join(home, "go", "bin", "luxd"),
		)
	}

	var foundPath string
	for _, loc := range locations {
		path, err := exec.LookPath(loc)
		if err == nil {
			foundPath = path
			break
		}
		if _, err := os.Stat(loc); err == nil {
			foundPath = loc
			break
		}
	}

	if foundPath == "" {
		result.Status = StatusWarn
		result.Message = "luxd binary not found"
		result.FixSuggestion = "Install luxd: go install github.com/luxfi/node/cmd/luxd@latest"
		d.results = append(d.results, result)
		return
	}

	cmd := exec.Command(foundPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		result.Status = StatusOK
		result.Message = fmt.Sprintf("luxd found (path: %s)", foundPath)
	} else {
		version := strings.TrimSpace(string(output))
		if strings.Contains(version, "version") {
			parts := strings.Fields(version)
			for i, p := range parts {
				if p == "version" && i+1 < len(parts) {
					version = parts[i+1]
					break
				}
			}
		}
		result.Status = StatusOK
		result.Message = fmt.Sprintf("luxd %s (path: %s)", version, foundPath)
	}

	d.results = append(d.results, result)
}

// checkNetworkConnectivity verifies connectivity to Lux endpoints
func (d *Doctor) checkNetworkConnectivity() {
	endpoints := []struct {
		name string
		url  string
	}{
		{"Mainnet API", constants.MainnetAPIEndpoint},
		{"Testnet API", constants.TestnetAPIEndpoint},
	}

	for _, ep := range endpoints {
		result := CheckResult{Name: ep.name}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, ep.url+"/ext/health", nil)
		if err != nil {
			result.Status = StatusError
			result.Message = "Failed to create request: " + err.Error()
			d.results = append(d.results, result)
			continue
		}

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				result.Status = StatusWarn
				result.Message = "Connection timeout"
			} else {
				result.Status = StatusWarn
				result.Message = "Connection failed: " + err.Error()
			}
			result.FixSuggestion = "Check network connectivity and firewall settings"
			d.results = append(d.results, result)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			result.Status = StatusOK
			result.Message = fmt.Sprintf("Connected (%s)", ep.url)
		} else {
			result.Status = StatusWarn
			result.Message = fmt.Sprintf("Unexpected status %d from %s", resp.StatusCode, ep.url)
		}

		d.results = append(d.results, result)
	}
}

// checkDiskSpace verifies available disk space
func (d *Doctor) checkDiskSpace() {
	result := CheckResult{Name: "Disk Space"}

	home, err := os.UserHomeDir()
	if err != nil {
		result.Status = StatusWarn
		result.Message = "Unable to determine home directory"
		d.results = append(d.results, result)
		return
	}

	luxDir := filepath.Join(home, constants.BaseDirName)
	checkPath := luxDir
	if _, err := os.Stat(luxDir); os.IsNotExist(err) {
		checkPath = home
	}

	var stat syscall.Statfs_t
	if err := syscall.Statfs(checkPath, &stat); err != nil {
		result.Status = StatusWarn
		result.Message = "Unable to check disk space: " + err.Error()
		d.results = append(d.results, result)
		return
	}

	availableGB := float64(stat.Bavail*uint64(stat.Bsize)) / (1024 * 1024 * 1024)
	totalGB := float64(stat.Blocks*uint64(stat.Bsize)) / (1024 * 1024 * 1024)

	if availableGB < MinDiskSpaceGB {
		result.Status = StatusWarn
		result.Message = fmt.Sprintf("%.1f GB available (%.1f GB total), recommended minimum is %d GB",
			availableGB, totalGB, MinDiskSpaceGB)
		result.FixSuggestion = "Free up disk space or use external storage for node data"
	} else {
		result.Status = StatusOK
		result.Message = fmt.Sprintf("%.1f GB available (%.1f GB total)", availableGB, totalGB)
	}

	d.results = append(d.results, result)
}

// checkCLIDirectories verifies CLI directories exist and are writable
func (d *Doctor) checkCLIDirectories() {
	result := CheckResult{Name: "CLI Directories"}

	home, err := os.UserHomeDir()
	if err != nil {
		result.Status = StatusError
		result.Message = "Unable to determine home directory"
		d.results = append(d.results, result)
		return
	}

	baseDir := filepath.Join(home, constants.BaseDirName)
	info, err := os.Stat(baseDir)
	if os.IsNotExist(err) {
		result.Status = StatusWarn
		result.Message = fmt.Sprintf("CLI directory not found: %s", baseDir)
		result.FixSuggestion = "Run any lux command to initialize directories"
		result.CanAutoFix = true
		result.AutoFix = func() error {
			return os.MkdirAll(baseDir, 0755)
		}
		d.results = append(d.results, result)
		return
	}

	if err != nil {
		result.Status = StatusError
		result.Message = "Error checking CLI directory: " + err.Error()
		d.results = append(d.results, result)
		return
	}

	if !info.IsDir() {
		result.Status = StatusError
		result.Message = fmt.Sprintf("%s exists but is not a directory", baseDir)
		d.results = append(d.results, result)
		return
	}

	testFile := filepath.Join(baseDir, ".doctor_test_"+strconv.FormatInt(time.Now().UnixNano(), 10))
	f, err := os.Create(testFile)
	if err != nil {
		result.Status = StatusError
		result.Message = fmt.Sprintf("CLI directory not writable: %s", baseDir)
		result.FixSuggestion = "Check directory permissions"
		d.results = append(d.results, result)
		return
	}
	f.Close()
	os.Remove(testFile)

	result.Status = StatusOK
	result.Message = fmt.Sprintf("CLI directory: %s", baseDir)
	d.results = append(d.results, result)
}

// printResults displays all check results with color coding
func (d *Doctor) printResults() {
	d.printToUser("")

	okCount, warnCount, errorCount := 0, 0, 0

	for _, r := range d.results {
		var statusIcon, statusColor string
		switch r.Status {
		case StatusOK:
			statusIcon = "[OK]"
			statusColor = "\033[32m"
			okCount++
		case StatusWarn:
			statusIcon = "[WARN]"
			statusColor = "\033[33m"
			warnCount++
		case StatusError:
			statusIcon = "[ERROR]"
			statusColor = "\033[31m"
			errorCount++
		}
		resetColor := "\033[0m"

		d.printToUser("%s%s%s %s: %s", statusColor, statusIcon, resetColor, r.Name, r.Message)

		if r.FixSuggestion != "" && r.Status != StatusOK {
			d.printToUser("      Fix: %s", r.FixSuggestion)
		}
	}

	d.printToUser("")
	d.printToUser("Summary: %d OK, %d warnings, %d errors", okCount, warnCount, errorCount)

	if warnCount > 0 || errorCount > 0 {
		canFix := 0
		for _, r := range d.results {
			if r.CanAutoFix && r.Status != StatusOK {
				canFix++
			}
		}
		if canFix > 0 {
			d.printToUser("")
			d.printToUser("Run 'lux doctor --fix' to attempt automatic fixes for %d issue(s)", canFix)
		}
	}
}

// attemptFixes tries to automatically fix issues that support it
func (d *Doctor) attemptFixes() error {
	d.printToUser("")
	d.printToUser("Attempting automatic fixes...")
	d.printToUser("")

	fixedCount, failedCount := 0, 0

	for _, r := range d.results {
		if r.Status == StatusOK || !r.CanAutoFix {
			continue
		}

		d.printToUser("Fixing: %s", r.Name)
		if err := r.AutoFix(); err != nil {
			d.printToUser("  Failed: %s", err.Error())
			failedCount++
		} else {
			d.printToUser("  Fixed")
			fixedCount++
		}
	}

	d.printToUser("")
	d.printToUser("Fixed %d issue(s), %d failed", fixedCount, failedCount)

	if failedCount > 0 {
		return fmt.Errorf("%d fix(es) failed", failedCount)
	}

	return nil
}
