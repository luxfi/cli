package binutils

import (
	"github.com/luxfi/log"
)

// NewLoggerAdapter returns the logger directly since luxfi/log.Logger already implements the interface
func NewLoggerAdapter(logger log.Logger) log.Logger {
	return logger
}