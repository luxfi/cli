// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package ux

import (
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

// TableCompatWrapper provides backward compatibility for tablewriter v0.0.5 API
// on top of tablewriter v1.0.9
type TableCompatWrapper struct {
	*tablewriter.Table
	headers []string
}

// NewCompatTable creates a new table with v0.0.5-like API
func NewCompatTable() *TableCompatWrapper {
	return &TableCompatWrapper{
		Table: tablewriter.NewTable(os.Stdout,
			tablewriter.WithRendition(tw.Rendition{
				Borders: tw.Border{Top: tw.On, Bottom: tw.On, Left: tw.On, Right: tw.On},
			}),
		),
	}
}

// SetHeader sets the headers using the old API
func (t *TableCompatWrapper) SetHeader(headers []string) {
	t.headers = headers
	// Convert to interface{} slice for v1.0.9 API
	headerInterface := make([]interface{}, len(headers))
	for i, h := range headers {
		headerInterface[i] = h
	}
	t.Header(headerInterface...)
}

// SetRowLine enables/disables row lines
func (t *TableCompatWrapper) SetRowLine(enable bool) {
	state := tw.Off
	if enable {
		state = tw.On
	}
	t.Options(tablewriter.WithRendition(tw.Rendition{
		Settings: tw.Settings{
			Separators: tw.Separators{
				BetweenRows: state,
			},
		},
	}))
}

// SetAutoMergeCells enables/disables cell merging
func (t *TableCompatWrapper) SetAutoMergeCells(enable bool) {
	mode := tw.MergeNone
	if enable {
		mode = tw.MergeVertical
	}
	t.Options(tablewriter.WithRowMergeMode(mode))
}

// SetAlignment sets the alignment (compatibility constant)
const ALIGN_LEFT = 0

// SetAlignment sets the alignment for rows
func (t *TableCompatWrapper) SetAlignment(align int) {
	// Map old constants to new tw.Align type
	var alignment tw.Align
	switch align {
	case ALIGN_LEFT:
		alignment = tw.AlignLeft
	default:
		alignment = tw.AlignLeft
	}
	t.Options(tablewriter.WithRowAlignment(alignment))
}

// AppendCompat adds a row using string slice (old API)
func (t *TableCompatWrapper) AppendCompat(row []string) {
	// Convert to interface{} for v1.0.9 API
	rowInterface := make([]interface{}, len(row))
	for i, r := range row {
		rowInterface[i] = r
	}
	t.Append(rowInterface...)
}

// CreateCompatTable creates a table with v0.0.5-like API
func CreateCompatTable() *TableCompatWrapper {
	return NewCompatTable()
}