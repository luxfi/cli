// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package subnetcmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMutuallyExclusive(t *testing.T) {
	assert := assert.New(t)
	type test struct {
		flagA       bool
		flagB       bool
		flagC       bool
		expectError bool
	}

	tests := []test{
		{
			flagA:       false,
			flagB:       false,
			flagC:       false,
			expectError: false,
		},
		{
			flagA:       true,
			flagB:       false,
			flagC:       false,
			expectError: false,
		},
		{
			flagA:       false,
			flagB:       true,
			flagC:       false,
			expectError: false,
		},
		{
			flagA:       false,
			flagB:       false,
			flagC:       true,
			expectError: false,
		},
		{
			flagA:       true,
			flagB:       false,
			flagC:       true,
			expectError: true,
		},
		{
			flagA:       false,
			flagB:       true,
			flagC:       true,
			expectError: true,
		},
		{
			flagA:       true,
			flagB:       true,
			flagC:       false,
			expectError: true,
		},
		{
			flagA:       true,
			flagB:       true,
			flagC:       true,
			expectError: true,
		},
	}

	for _, tt := range tests {
		err := checkMutuallyExclusive(tt.flagA, tt.flagB, tt.flagC)
		if tt.expectError {
			assert.Error(err, tt)
			assert.ErrorIs(err, errMutuallyExlusive)
		} else {
			assert.NoError(err)
		}
	}
}