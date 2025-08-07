// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package comparator

import (
	"errors"
	"fmt"
)

// Comparison type constants
const (
	LessThanEq = "Less Than Or Eq"
	MoreThanEq = "More Than Or Eq"
	MoreThan   = "More Than"
)

// Comparator struct for value comparisons
type Comparator struct {
	Label string // Label that identifies reference value
	Type  string // Less Than Eq, More Than Eq, or More Than
	Value uint64 // Value to Compare To
}

// Validate checks if the given value satisfies the comparator constraint
func (c *Comparator) Validate(val uint64) error {
	switch c.Type {
	case LessThanEq:
		if val > c.Value {
			return fmt.Errorf("%s must be less than or equal to %d", c.Label, c.Value)
		}
	case MoreThanEq:
		if val < c.Value {
			return fmt.Errorf("%s must be more than or equal to %d", c.Label, c.Value)
		}
	case MoreThan:
		if val <= c.Value {
			return fmt.Errorf("%s must be more than %d", c.Label, c.Value)
		}
	default:
		return errors.New("invalid comparator type")
	}
	return nil
}
