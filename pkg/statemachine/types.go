// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package statemachine

// StateType represents a state in the state machine
type StateType int

const (
	// StateUnknown is the unknown state
	StateUnknown StateType = iota
	// StateInit is the initial state
	StateInit
	// StateInProgress is when operation is in progress
	StateInProgress
	// StateComplete is when operation is complete
	StateComplete
)
