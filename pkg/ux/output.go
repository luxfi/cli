// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package ux

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/message"

	luxlog "github.com/luxfi/log"
	"github.com/luxfi/netrunner/rpcpb"
	"github.com/olekukonko/tablewriter"
)

var Logger *UserLog

type UserLog struct {
	log    luxlog.Logger
	writer io.Writer
}

func NewUserLog(log luxlog.Logger, userwriter io.Writer) {
	if Logger == nil {
		Logger = &UserLog{
			log:    log,
			writer: userwriter,
		}
	}
}

// PrintToUser prints msg directly to stdout (command output)
// Does NOT log to avoid duplication - logs should go to stderr separately
func (ul *UserLog) PrintToUser(msg string, args ...interface{}) {
	formattedMsg := fmt.Sprintf(msg, args...)
	_, _ = fmt.Fprintln(ul.writer, formattedMsg)
}

// Info logs an info message
func (ul *UserLog) Info(msg string, args ...interface{}) {
	formattedMsg := fmt.Sprintf(msg, args...)
	ul.log.Info(formattedMsg)
}

// PrintLineSeparator prints a line separator
func (ul *UserLog) PrintLineSeparator(msg ...string) {
	separator := "=========================================="
	if len(msg) > 0 && msg[0] != "" {
		separator = msg[0]
	}
	_, _ = fmt.Fprintln(ul.writer, separator)
	ul.log.Info(separator)
}

// Error logs an error message
func (ul *UserLog) Error(msg string, args ...interface{}) {
	formattedMsg := fmt.Sprintf(msg, args...)
	ul.log.Error(formattedMsg)
}

// RedXToUser prints a red X error message to the user
func (ul *UserLog) RedXToUser(msg string, args ...interface{}) {
	formattedMsg := fmt.Sprintf("✗ %s", fmt.Sprintf(msg, args...))
	_, _ = fmt.Fprintln(ul.writer, formattedMsg)
	ul.log.Error(formattedMsg)
}

// GreenCheckmarkToUser prints a green checkmark success message to the user
func (ul *UserLog) GreenCheckmarkToUser(msg string, args ...interface{}) {
	formattedMsg := fmt.Sprintf("✓ %s", fmt.Sprintf(msg, args...))
	_, _ = fmt.Fprintln(ul.writer, formattedMsg)
	ul.log.Info(formattedMsg)
}

// PrintError prints a visible error message with ERROR prefix to the user
func (ul *UserLog) PrintError(msg string, args ...interface{}) {
	formattedMsg := fmt.Sprintf(msg, args...)
	errorMsg := fmt.Sprintf("\nERROR: %s\n", formattedMsg)
	_, _ = fmt.Fprintln(ul.writer, errorMsg)
	ul.log.Error(formattedMsg)
}

// PrintWait does some dot printing to entertain the user
func PrintWait(cancel chan struct{}) {
	for {
		select {
		case <-time.After(1 * time.Second):
			fmt.Print(".")
		case <-cancel:
			return
		}
	}
}

// StepTracker tracks progress of multi-step operations with elapsed time
type StepTracker struct {
	stepStart    time.Time
	warnAfter    time.Duration
	warningShown bool
	stepName     string
	ul           *UserLog
}

// NewStepTracker creates a tracker that warns if a step takes longer than warnAfter
func NewStepTracker(ul *UserLog, warnAfter time.Duration) *StepTracker {
	return &StepTracker{
		ul:        ul,
		warnAfter: warnAfter,
	}
}

// Start begins tracking a new step
func (st *StepTracker) Start(stepName string) {
	st.stepStart = time.Now()
	st.stepName = stepName
	st.warningShown = false
	st.ul.PrintToUser("%s...", stepName)
}

// Elapsed returns the elapsed time for the current step
func (st *StepTracker) Elapsed() time.Duration {
	return time.Since(st.stepStart)
}

// CheckWarn prints a warning if the step has taken longer than the threshold
// Returns true if warning was printed
func (st *StepTracker) CheckWarn() bool {
	if st.warningShown {
		return false
	}
	elapsed := st.Elapsed()
	if elapsed > st.warnAfter {
		st.ul.PrintToUser("Warning: %s taking longer than expected (%.1fs)...", st.stepName, elapsed.Seconds())
		st.warningShown = true
		return true
	}
	return false
}

// Complete marks the step as done with success
func (st *StepTracker) Complete(suffix string) {
	elapsed := st.Elapsed()
	if suffix != "" {
		st.ul.GreenCheckmarkToUser("%s (%.1fs) - %s", st.stepName, elapsed.Seconds(), suffix)
	} else {
		st.ul.GreenCheckmarkToUser("%s (%.1fs)", st.stepName, elapsed.Seconds())
	}
}

// CompleteSuccess is shorthand for Complete with "Success" suffix
func (st *StepTracker) CompleteSuccess() {
	st.Complete("Success")
}

// Failed marks the step as failed with an error
func (st *StepTracker) Failed(reason string) {
	elapsed := st.Elapsed()
	st.ul.RedXToUser("%s (%.1fs) - FAILED: %s", st.stepName, elapsed.Seconds(), reason)
}

// PrintTableEndpoints prints the endpoints coming from the healthy call
func PrintTableEndpoints(clusterInfo *rpcpb.ClusterInfo) {
	table := tablewriter.NewWriter(os.Stdout)
	// Note: SetHeader is not available in v1.0.9, use Append for header row

	nodeInfos := map[string]*rpcpb.NodeInfo{}
	for _, nodeInfo := range clusterInfo.NodeInfos {
		nodeInfos[nodeInfo.Name] = nodeInfo
	}
	for _, nodeName := range clusterInfo.NodeNames {
		nodeInfo := nodeInfos[nodeName]
		for blockchainID, chainInfo := range clusterInfo.CustomChains {
			_ = table.Append([]string{nodeInfo.Name, chainInfo.GetChainName(), fmt.Sprintf("%s/ext/bc/%s/rpc", nodeInfo.GetUri(), blockchainID), fmt.Sprintf("%s/ext/bc/%s/rpc", nodeInfo.GetUri(), chainInfo.GetChainName())})
		}
	}
	_ = table.Render()
}

// DefaultTable creates a default table with the given title and headers
func DefaultTable(title string, headers []string) *tablewriter.Table {
	table := tablewriter.NewWriter(os.Stdout)
	// Note: v1.0.9 API doesn't have SetCaption, SetBorder, SetAutoWrapText, SetAlignment
	// These would need to be set via Options during creation or not at all
	return table
}

func ConvertToStringWithThousandSeparator(input uint64) string {
	p := message.NewPrinter(language.English)
	s := p.Sprintf("%d", input)
	return strings.ReplaceAll(s, ",", "_")
}

// NativeChainInfo holds info for pretty-printing a native chain
type NativeChainInfo struct {
	Letter string // P, C, X, Q, A, B, T, Z, G, K, D
	Name   string // Platform, Contract, Exchange, etc.
	Type   string // RPC endpoint type
	Path   string // URL path suffix
}

// GetNativeChains returns all native chain definitions for RPC display
func GetNativeChains() []NativeChainInfo {
	return []NativeChainInfo{
		{Letter: "P", Name: "Platform", Type: "RPC", Path: "/ext/bc/P"},
		{Letter: "C", Name: "Contract (EVM)", Type: "RPC", Path: "/ext/bc/C/rpc"},
		{Letter: "C", Name: "Contract (EVM)", Type: "WS", Path: "/ext/bc/C/ws"},
		{Letter: "X", Name: "Exchange (DAG)", Type: "RPC", Path: "/ext/bc/X"},
		{Letter: "Q", Name: "Quantum", Type: "RPC", Path: "/ext/bc/Q/rpc"},
		{Letter: "A", Name: "AI", Type: "RPC", Path: "/ext/bc/A/rpc"},
		{Letter: "B", Name: "Bridge", Type: "RPC", Path: "/ext/bc/B/rpc"},
		{Letter: "T", Name: "Threshold", Type: "RPC", Path: "/ext/bc/T/rpc"},
		{Letter: "Z", Name: "Zero-knowledge", Type: "RPC", Path: "/ext/bc/Z/rpc"},
		{Letter: "G", Name: "Graph", Type: "RPC", Path: "/ext/bc/G/rpc"},
		{Letter: "K", Name: "KMS", Type: "RPC", Path: "/ext/bc/K/rpc"},
		{Letter: "D", Name: "DEX", Type: "RPC", Path: "/ext/bc/D/rpc"},
	}
}

// PrintNativeChainEndpoints prints all native chain RPC endpoints in a formatted table
func PrintNativeChainEndpoints(baseURL string, portBase int, includeUtility bool) {
	Logger.PrintToUser("\n╔══════════════════════════════════════════════════════════════════════╗")
	Logger.PrintToUser("║                        LUX CHAIN ENDPOINTS                           ║")
	Logger.PrintToUser("╠══════════════════════════════════════════════════════════════════════╣")
	Logger.PrintToUser("║ Chain   │ Name              │ Type │ Endpoint                        ║")
	Logger.PrintToUser("╠═════════╪═══════════════════╪══════╪═════════════════════════════════╣")

	chains := GetNativeChains()
	for _, c := range chains {
		var url string
		if baseURL != "" {
			url = baseURL + c.Path
		} else {
			protocol := "http"
			if c.Type == "WS" {
				protocol = "ws"
			}
			url = fmt.Sprintf("%s://localhost:%d%s", protocol, portBase, c.Path)
		}
		Logger.PrintToUser("║ %-7s │ %-17s │ %-4s │ %-31s ║", c.Letter+"-Chain", c.Name, c.Type, url)
	}

	if includeUtility {
		Logger.PrintToUser("╠═════════╪═══════════════════╪══════╪═════════════════════════════════╣")
		Logger.PrintToUser("║ UTILITY │ Health            │ HTTP │ http://localhost:%d/ext/health  ║", portBase)
		Logger.PrintToUser("║ UTILITY │ Info              │ HTTP │ http://localhost:%d/ext/info    ║", portBase)
		Logger.PrintToUser("║ UTILITY │ Admin             │ HTTP │ http://localhost:%d/ext/admin   ║", portBase)
	}
	Logger.PrintToUser("╚══════════════════════════════════════════════════════════════════════╝")
}

// PrintCompactChainEndpoints prints chain endpoints in a compact format
func PrintCompactChainEndpoints(portBase int) {
	Logger.PrintToUser("\n📡 Native Chain RPC Endpoints:")
	Logger.PrintToUser("  ┌─────────────────────────────────────────────────────────────────┐")
	Logger.PrintToUser("  │ P-Chain (Platform):     http://localhost:%d/ext/bc/P            │", portBase)
	Logger.PrintToUser("  │ C-Chain (EVM) RPC:      http://localhost:%d/ext/bc/C/rpc        │", portBase)
	Logger.PrintToUser("  │ C-Chain (EVM) WS:       ws://localhost:%d/ext/bc/C/ws           │", portBase)
	Logger.PrintToUser("  │ X-Chain (Exchange):     http://localhost:%d/ext/bc/X            │", portBase)
	Logger.PrintToUser("  │ Q-Chain (Quantum):      http://localhost:%d/ext/bc/Q/rpc        │", portBase)
	Logger.PrintToUser("  │ A-Chain (AI):           http://localhost:%d/ext/bc/A/rpc        │", portBase)
	Logger.PrintToUser("  │ B-Chain (Bridge):       http://localhost:%d/ext/bc/B/rpc        │", portBase)
	Logger.PrintToUser("  │ T-Chain (Threshold):    http://localhost:%d/ext/bc/T/rpc        │", portBase)
	Logger.PrintToUser("  │ Z-Chain (ZK):           http://localhost:%d/ext/bc/Z/rpc        │", portBase)
	Logger.PrintToUser("  │ G-Chain (Graph):        http://localhost:%d/ext/bc/G/rpc        │", portBase)
	Logger.PrintToUser("  │ K-Chain (KMS):          http://localhost:%d/ext/bc/K/rpc        │", portBase)
	Logger.PrintToUser("  │ D-Chain (DEX):          http://localhost:%d/ext/bc/D/rpc        │", portBase)
	Logger.PrintToUser("  └─────────────────────────────────────────────────────────────────┘")
	Logger.PrintToUser("\n🔧 Utility Endpoints:")
	Logger.PrintToUser("  Health:  http://localhost:%d/ext/health", portBase)
	Logger.PrintToUser("  Info:    http://localhost:%d/ext/info", portBase)
	Logger.PrintToUser("  Admin:   http://localhost:%d/ext/admin", portBase)
}

// ValidatorKeyInfo holds derived key info for a validator
type ValidatorKeyInfo struct {
	Index       int
	NodeID      string
	PChainAddr  string
	XChainAddr  string
	CChainAddr  string // Ethereum-style 0x address
	BLSPubKey   string // Hex-encoded BLS public key
}

// PrintValidatorKeys prints validator key information in a formatted table
func PrintValidatorKeys(validators []ValidatorKeyInfo, networkHRP string) {
	if len(validators) == 0 {
		return
	}

	Logger.PrintToUser("\n🔑 Validator Keys (derived from LUX_MNEMONIC):")
	Logger.PrintToUser("  ╔═══════╤════════════════════════════════════════════════════════════════╗")
	Logger.PrintToUser("  ║  #    │ Validator Details                                              ║")
	Logger.PrintToUser("  ╠═══════╪════════════════════════════════════════════════════════════════╣")

	for _, v := range validators {
		Logger.PrintToUser("  ║  %d    │ NodeID:  %s", v.Index, v.NodeID)
		Logger.PrintToUser("  ║       │ P-Chain: %s", v.PChainAddr)
		Logger.PrintToUser("  ║       │ X-Chain: %s", v.XChainAddr)
		Logger.PrintToUser("  ║       │ C-Chain: %s", v.CChainAddr)
		if v.Index < len(validators)-1 {
			Logger.PrintToUser("  ╟───────┼────────────────────────────────────────────────────────────────╢")
		}
	}
	Logger.PrintToUser("  ╚═══════╧════════════════════════════════════════════════════════════════╝")
}

// PrintValidatorKeysCompact prints validator keys in a compact single-line format
func PrintValidatorKeysCompact(validators []ValidatorKeyInfo) {
	if len(validators) == 0 {
		return
	}

	Logger.PrintToUser("\n🔑 Validator Keys (from LUX_MNEMONIC):")
	for _, v := range validators {
		Logger.PrintToUser("  [%d] %s | C: %s", v.Index, v.NodeID, v.CChainAddr)
	}
}
