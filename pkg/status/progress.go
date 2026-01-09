// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package status

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/schollz/progressbar/v3"
)

// ProgressTracker handles progress reporting and UX
type ProgressTracker struct {
	writer         io.Writer
	isTTY          bool
	spinnerChars   []string
	spinnerIndex   int
	spinnerMutex   sync.Mutex
	lastLineLength int
	startTime      time.Time
	mu             sync.Mutex
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker(writer io.Writer) *ProgressTracker {
	isTTY := isTerminal(writer)

	return &ProgressTracker{
		writer:       writer,
		isTTY:        isTTY,
		spinnerChars: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		startTime:    time.Now(),
	}
}

// isTerminal checks if the writer is a terminal
func isTerminal(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		return isTerminalFile(f)
	}
	return false
}

// isTerminalFile checks if a file is a terminal
func isTerminalFile(f *os.File) bool {
	return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
}

// StartStep begins a new step with optional message
func (pt *ProgressTracker) StartStep(stepName string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if pt.isTTY {
		pt.clearLine()
		fmt.Fprintf(pt.writer, "%s %s...", pt.getSpinner(), stepName)
		pt.lastLineLength = len(stepName) + 5 // spinner + "..."
	} else {
		fmt.Fprintf(pt.writer, "%s...\n", stepName)
	}
}

// UpdateStep updates the current step progress
func (pt *ProgressTracker) UpdateStep(message string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if pt.isTTY {
		pt.clearLine()
		fmt.Fprintf(pt.writer, "%s %s", pt.getSpinner(), message)
		pt.lastLineLength = len(message) + 2 // spinner + space
	}
}

// CompleteStep marks a step as completed
func (pt *ProgressTracker) CompleteStep(stepName string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if pt.isTTY {
		pt.clearLine()
		fmt.Fprintf(pt.writer, "✓ %s (%.1fs)\n", stepName, time.Since(pt.startTime).Seconds())
		pt.startTime = time.Now()
	} else {
		fmt.Fprintf(pt.writer, "✓ %s (%.1fs)\n", stepName, time.Since(pt.startTime).Seconds())
		pt.startTime = time.Now()
	}
}

// FailStep marks a step as failed
func (pt *ProgressTracker) FailStep(stepName string, err error) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if pt.isTTY {
		pt.clearLine()
		fmt.Fprintf(pt.writer, "✗ %s: %v\n", stepName, err)
	} else {
		fmt.Fprintf(pt.writer, "✗ %s: %v\n", stepName, err)
	}
}

// getSpinner gets the current spinner character and advances it
func (pt *ProgressTracker) getSpinner() string {
	pt.spinnerMutex.Lock()
	defer pt.spinnerMutex.Unlock()

	char := pt.spinnerChars[pt.spinnerIndex]
	pt.spinnerIndex = (pt.spinnerIndex + 1) % len(pt.spinnerChars)
	return char
}

// clearLine clears the current line
func (pt *ProgressTracker) clearLine() {
	if pt.lastLineLength > 0 {
		// Move cursor to beginning of line and clear
		fmt.Fprint(pt.writer, "\r")
		// Clear the line
		fmt.Fprint(pt.writer, strings.Repeat(" ", pt.lastLineLength))
		fmt.Fprint(pt.writer, "\r")
	}
}

// CreateProgressBar creates a progress bar for a specific task
func (pt *ProgressTracker) CreateProgressBar(task string, total int) *progressbar.ProgressBar {
	if !pt.isTTY {
		return nil
	}

	bar := progressbar.NewOptions(
		total,
		progressbar.OptionSetWriter(pt.writer),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetWidth(15),
		progressbar.OptionSetDescription(fmt.Sprintf("[[cyan]]%s[[reset]]", task)),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)

	return bar
}

// PrintInfo prints an informational message
func (pt *ProgressTracker) PrintInfo(message string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if pt.isTTY {
		pt.clearLine()
		fmt.Fprintf(pt.writer, "ℹ %s\n", message)
	} else {
		fmt.Fprintf(pt.writer, "ℹ %s\n", message)
	}
}

// PrintWarning prints a warning message
func (pt *ProgressTracker) PrintWarning(message string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if pt.isTTY {
		pt.clearLine()
		fmt.Fprintf(pt.writer, "⚠ %s\n", message)
	} else {
		fmt.Fprintf(pt.writer, "⚠ %s\n", message)
	}
}

// PrintSuccess prints a success message
func (pt *ProgressTracker) PrintSuccess(message string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if pt.isTTY {
		pt.clearLine()
		fmt.Fprintf(pt.writer, "✓ %s\n", message)
	} else {
		fmt.Fprintf(pt.writer, "✓ %s\n", message)
	}
}

// PrintError prints an error message
func (pt *ProgressTracker) PrintError(message string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if pt.isTTY {
		pt.clearLine()
		fmt.Fprintf(pt.writer, "✗ %s\n", message)
	} else {
		fmt.Fprintf(pt.writer, "✗ %s\n", message)
	}
}

// Summary prints a summary of the operation
func (pt *ProgressTracker) Summary(duration time.Duration, networks int, nodes int, chains int) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if pt.isTTY {
		pt.clearLine()
	}

	fmt.Fprintf(pt.writer, "\n📊 Status Summary:\n")
	fmt.Fprintf(pt.writer, "   Networks: %d | Nodes: %d | Chains: %d\n", networks, nodes, chains)
	fmt.Fprintf(pt.writer, "   Duration: %.2fs\n", duration.Seconds())
}
