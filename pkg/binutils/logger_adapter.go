// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package binutils

import (
	"github.com/luxfi/log"
)

// NewLoggerAdapter returns the logger directly since luxfi/log.Logger already implements the interface
func NewLoggerAdapter(logger log.Logger) log.Logger {
	return logger
}
