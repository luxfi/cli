// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package models

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/luxfi/cli/v2/pkg/constants"
)

// timedFunction executes a function and returns its result with error
func timedFunction(f func() (any, error), actionMsg string, timeout ...time.Duration) (any, error) {
	fmt.Printf("  %s...", actionMsg)
	start := time.Now()
	result, err := f()
	elapsed := time.Since(start)
	if err != nil {
		fmt.Printf(" failed (%v)\n", elapsed)
		return nil, err
	}
	fmt.Printf(" done (%v)\n", elapsed)
	return result, nil
}

// timedFunctionWithRetry executes a function with retry logic
func timedFunctionWithRetry[T any](f func() (T, error), actionMsg string, timeout time.Duration, numRetries int, sleepBetweenRetries time.Duration) (T, error) {
	var result T
	var err error
	for i := 0; i <= numRetries; i++ {
		result, err = f()
		if err == nil {
			return result, nil
		}
		if i < numRetries {
			time.Sleep(sleepBetweenRetries)
		}
	}
	return result, err
}

// randomString generates a random string of the given length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// isE2E checks if running in E2E test mode
func isE2E() bool {
	return os.Getenv("RUN_E2E") != ""
}

// e2eConvertIP converts IP for E2E testing
func e2eConvertIP(ip string) string {
	if os.Getenv("RUN_E2E") != "" {
		return constants.E2EDockerLoopbackHost
	}
	return ip
}