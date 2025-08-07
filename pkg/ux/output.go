// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
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
	fmt.Fprintln(ul.writer, formattedMsg)
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
	fmt.Fprintln(ul.writer, separator)
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
	fmt.Fprintln(ul.writer, formattedMsg)
	ul.log.Error(formattedMsg)
}

// GreenCheckmarkToUser prints a green checkmark success message to the user
func (ul *UserLog) GreenCheckmarkToUser(msg string, args ...interface{}) {
	formattedMsg := fmt.Sprintf("✓ %s", fmt.Sprintf(msg, args...))
	fmt.Fprintln(ul.writer, formattedMsg)
	ul.log.Info(formattedMsg)
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

// PrintTableEndpoints prints the endpoints coming from the healthy call
func PrintTableEndpoints(clusterInfo *rpcpb.ClusterInfo) {
	table := tablewriter.NewWriter(os.Stdout)
	header := []string{"node", "VM", "URL", "ALIAS_URL"}
	table.Append(header)

	nodeInfos := map[string]*rpcpb.NodeInfo{}
	for _, nodeInfo := range clusterInfo.NodeInfos {
		nodeInfos[nodeInfo.Name] = nodeInfo
	}
	for _, nodeName := range clusterInfo.NodeNames {
		nodeInfo := nodeInfos[nodeName]
		for blockchainID, chainInfo := range clusterInfo.CustomChains {
			table.Append([]string{nodeInfo.Name, chainInfo.ChainName, fmt.Sprintf("%s/ext/bc/%s/rpc", nodeInfo.GetUri(), blockchainID), fmt.Sprintf("%s/ext/bc/%s/rpc", nodeInfo.GetUri(), chainInfo.ChainName)})
		}
	}
	table.Render()
}

// DefaultTable creates a default table with the given title and headers
func DefaultTable(title string, headers []string) *tablewriter.Table {
	table := tablewriter.NewWriter(os.Stdout)
	if title != "" {
		// Table title is set using caption in some versions
		table.SetCaption(true, title)
	}
	if headers != nil && len(headers) > 0 {
		table.SetHeader(headers)
	}
	table.SetBorder(true)
	table.SetAutoWrapText(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	return table
}

func ConvertToStringWithThousandSeparator(input uint64) string {
	p := message.NewPrinter(language.English)
	s := p.Sprintf("%d", input)
	return strings.ReplaceAll(s, ",", "_")
}
