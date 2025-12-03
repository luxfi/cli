// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package networkcmd

import (
	"testing"

	"github.com/luxfi/netrunner/server"
	"github.com/stretchr/testify/require"
)

func Test_isNotBootstrappedError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "not bootstrapped error",
			err:      server.ErrNotBootstrapped,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNotBootstrappedError(tt.err)
			require.Equal(t, tt.expected, result)
		})
	}
}
