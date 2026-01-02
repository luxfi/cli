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

// PrintToUser prints msg directly on the screen, but also to log file
func (ul *UserLog) PrintToUser(msg string, args ...interface{}) {
	formattedMsg := fmt.Sprintf(msg, args...)
	_, _ = fmt.Fprintln(ul.writer, formattedMsg)
	ul.log.Info(formattedMsg)
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
