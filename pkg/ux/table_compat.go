// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package ux

import (
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

// Alignment constants for backward compatibility
var (
	ALIGN_LEFT   = tw.AlignLeft
	ALIGN_CENTER = tw.AlignCenter
	ALIGN_RIGHT  = tw.AlignRight
)

// TableCompatWrapper provides backward compatibility for tablewriter v0.0.5 API
// on top of tablewriter v1.0.9+
type TableCompatWrapper struct {
	*tablewriter.Table
	headers   []string
	alignment tw.Align
}

// NewCompatTable creates a new table with v0.0.5-like API
func NewCompatTable() *TableCompatWrapper {
	return &TableCompatWrapper{
		Table:     tablewriter.NewTable(os.Stdout),
		alignment: tw.AlignLeft,
	}
}

// SetHeader sets the headers using the old API
func (t *TableCompatWrapper) SetHeader(headers []string) {
	t.headers = headers
	// Convert []string to []any for the new API
	anyHeaders := make([]any, len(headers))
	for i, h := range headers {
		anyHeaders[i] = h
	}
	t.Table.Header(anyHeaders...)
}

// SetRowLine is a no-op in v1.0.9 (row lines controlled via renderer settings)
func (t *TableCompatWrapper) SetRowLine(enable bool) {
	// Row lines are now controlled via renderer configuration
	// This is a no-op for compatibility
}

// SetAutoMergeCells is a no-op in v1.0.9 (merge mode controlled via config)
func (t *TableCompatWrapper) SetAutoMergeCells(enable bool) {
	// Cell merging is now controlled via config.Row.Formatting.MergeMode
	// This is a no-op for compatibility
}

// SetAlignment sets the alignment for rows
func (t *TableCompatWrapper) SetAlignment(align tw.Align) {
	t.alignment = align
	t.Table.Configure(func(config *tablewriter.Config) {
		config.Row.Alignment.Global = t.alignment
	})
}

// AppendCompat adds a row using string slice (old API)
func (t *TableCompatWrapper) AppendCompat(row []string) {
	_ = t.Table.Append(row)
}

// CreateCompatTable creates a table with v0.0.5-like API
func CreateCompatTable() *TableCompatWrapper {
	return NewCompatTable()
}
