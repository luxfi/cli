// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package ux

import (
	"os"

	"github.com/olekukonko/tablewriter"
)

// TableCompatWrapper provides backward compatibility for tablewriter v0.0.5 API
// on top of tablewriter v0.0.5+ (maintaining compatibility)
type TableCompatWrapper struct {
	*tablewriter.Table
	headers []string
}

// NewCompatTable creates a new table with v0.0.5-like API
func NewCompatTable() *TableCompatWrapper {
	return &TableCompatWrapper{
		Table: tablewriter.NewWriter(os.Stdout),
	}
}

// SetHeader sets the headers using the old API
func (t *TableCompatWrapper) SetHeader(headers []string) {
	t.headers = headers
	t.Table.SetHeader(headers)
}

// SetRowLine enables/disables row lines
func (t *TableCompatWrapper) SetRowLine(enable bool) {
	t.Table.SetRowLine(enable)
}

// SetAutoMergeCells enables/disables cell merging
func (t *TableCompatWrapper) SetAutoMergeCells(enable bool) {
	t.Table.SetAutoMergeCells(enable)
}

// SetAlignment sets the alignment (compatibility constant)
const ALIGN_LEFT = tablewriter.ALIGN_LEFT

// SetAlignment sets the alignment for rows
func (t *TableCompatWrapper) SetAlignment(align int) {
	t.Table.SetAlignment(align)
}

// AppendCompat adds a row using string slice (old API)
func (t *TableCompatWrapper) AppendCompat(row []string) {
	t.Table.Append(row)
}

// CreateCompatTable creates a table with v0.0.5-like API
func CreateCompatTable() *TableCompatWrapper {
	return NewCompatTable()
}
