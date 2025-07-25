// Copyright (C) 2022, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/cli/pkg/models"
	luxlog "github.com/luxfi/log"
	"github.com/stretchr/testify/require"
	"golang.org/x/mod/semver"
)

var _ VersionMapper = &testMapper{}

type testContext struct {
	// expected mapping of binaries to their versions
	expected map[string]string
	// fake versions set for the evm binaries, faking github
	sourceEVM string
	// fake versions set for the node binaries, faking github
	sourceLux string
	// should the test fail
	shouldFail bool
	// name of the test
	name string
}

// testMapper is used to bypass github,
// to test the `GetVersionMapping` function
// We want to make sure that given a set of
// versions mocking the structure of github releases API,
// `GetVersionMapping` is able to correctly evaluate
// the set of compatible versions for each test.
type testMapper struct {
	app            *application.Lux
	currentContext *testContext
	srv            *httptest.Server
	t              *testing.T
}

func newTestMapper(t *testing.T) *testMapper {
	app := &application.Lux{
		Downloader: application.NewDownloader(),
		Log:        luxlog.NewNoOpLogger(),
	}
	return &testMapper{
		app,
		nil,
		nil,
		t,
	}
}

// implement VersionMapper
func (*testMapper) GetEligibleVersions(sorted []string, _ string, _ *application.Lux) ([]string, error) {
	// tests were written with the assumption that the first version is always in progress
	return sorted[1:], nil
}

// implement VersionMapper
func (m *testMapper) GetLatestLuxByProtoVersion(_ *application.Lux, rpcVersion int, _ string) (string, error) {
	cBytes := []byte(m.currentContext.sourceLux)

	var compat models.LuxCompatiblity
	if err := json.Unmarshal(cBytes, &compat); err != nil {
		return "", err
	}
	vers := compat[strconv.Itoa(rpcVersion)]
	if len(vers) == 0 {
		return "", errors.New("test zero length versions")
	}
	if len(vers) > 1 {
		semver.Sort(vers)
	}
	// take the last
	return vers[len(vers)-1], nil
}

// implement VersionMapper
// We just set a currentContext for a duration of a single test,
// so that when the faked github URL is called,
// it knows what faked versions to return
func (m *testMapper) getVersionMapping(tc *testContext) (map[string]string, error) {
	binaryToVersion = nil
	// allows to know which test is currently running
	m.currentContext = tc
	return GetVersionMapping(m)
}

// implement VersionMapper
func (m *testMapper) GetApp() *application.Lux {
	return m.app
}

// GetCompatURL fakes a github endpoint for
// evm release
// implement VersionMapper
func (m *testMapper) GetCompatURL(vmType models.VMType) string {
	switch vmType {
	case models.EVM:
		return m.srv.URL + "/evm"
	default:
		m.t.Fatalf("unexpected vmType: %T", vmType)
	}
	return ""
}

// GetLuxURL fakes a github endpoint for
// node releases
// implement VersionMapper
func (m *testMapper) GetLuxURL() string {
	return m.srv.URL + "/lux"
}

// This is the server function which the local
// httptest.NewServer() will serve for tests.
// Therefore, the tests hit this endpoint,
// and get a faked list of versions (simulating github)
func (m *testMapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	var err error
	// return the correct faked versions based on the URL
	// which is being requested, returning the faked
	// versions for each binary release endpoint
	switch r.URL.Path {
	case "/evm":
		_, err = w.Write([]byte(m.currentContext.sourceEVM))
	case "/lux":
		_, err = w.Write([]byte(m.currentContext.sourceLux))
	default:
		m.t.Fatalf("Unexpected path URL for test server: %s\n", r.URL.Path)
	}
	if err != nil {
		m.t.Fatal(err)
	}
}

// TestGetVersionMapping tests that mapping the binaries
// to versions function (`GetVersionMapping`) returns
// the expected values.
// For the test to be meaningful, we start a httptest HTTP
// server locally, which then returns fake versions for each request
// (sourceEVM, sourceLux) which then
// the mapping code in `GetVersionMapping` is expected
// to correctly evaluate for the global `binaryToVersion` map,
// used by the tests to know which version to use for which test.
func TestGetVersionMapping(t *testing.T) {
	require := require.New(t)
	m := newTestMapper(t)
	// start local test HTTP server
	srv := httptest.NewServer(m)
	defer srv.Close()
	m.srv = srv

	testContexts := []*testContext{
		{
			// This test contains a combination
			// of versions which will be used
			// by `GetVersionMapping` to evaluate versions.
			// The function should be able to correctly
			// evaluate compatible versions, hence
			// `shouldFail` is false
			name:       "latest evm match latest lux",
			shouldFail: false,
			expected: map[string]string{
				SoloEVMKey1:      "v0.4.2",
				SoloEVMKey2:      "v0.4.1",
				SoloLuxKey:           "v1.9.1",
				OnlyLuxKey:           OnlyLuxValue,
				MultiLux1Key:         "v1.9.3",
				MultiLux2Key:         "v1.9.2",
				MultiLuxEVMKey: "v0.4.3",
				LatestEVM2LuxKey:     "v0.4.3",
				LatestLux2EVMKey:     "v1.9.3",
			},
			sourceEVM: `{
						"rpcChainVMProtocolVersion": {
							"v0.4.4": 19,
							"v0.4.3": 19,
							"v0.4.2": 18,
							"v0.4.1": 18,
							"v0.4.0": 17
						}
				  }`,
			sourceLux: `{
						"19": [
							"v1.9.2",
							"v1.9.3"
						],
						"18": [
							"v1.9.1"
						],
						"17": [
							"v1.9.0"
						]
				  }`,
		},
		{
			// This test does the same, but a different
			// constellation of versions
			name:       ">0 major version",
			shouldFail: false,
			expected: map[string]string{
				SoloEVMKey1:      "v0.9.9",
				SoloEVMKey2:      "v0.9.8",
				SoloLuxKey:           "v2.3.4",
				OnlyLuxKey:           OnlyLuxValue,
				MultiLux1Key:         "v2.3.4",
				MultiLux2Key:         "v2.3.3",
				MultiLuxEVMKey: "v0.9.9",
				LatestEVM2LuxKey:     "v0.9.9",
				LatestLux2EVMKey:     "v2.3.4",
			},
			sourceEVM: `{
					"rpcChainVMProtocolVersion": {
						"v1.0.0": 100,
						"v0.9.9": 99,
						"v0.9.8": 99,
						"v0.4.2": 18,
						"v0.4.1": 18,
						"v0.4.0": 17
					}
			  }`,
			sourceLux: `{
					"99": [
						"v2.3.4",
						"v2.3.3"
					],
					"18": [
						"v1.9.1"
					],
					"17": [
						"v1.9.0"
					]
			  }`,
		},
		{
			// This test does the same, but a different
			// constellation of versions
			name:       "subsecuent evm versions are older",
			shouldFail: false,
			expected: map[string]string{
				SoloEVMKey1:      "v0.4.2",
				SoloEVMKey2:      "v0.4.1",
				SoloLuxKey:           "v2.1.1",
				OnlyLuxKey:           OnlyLuxValue,
				MultiLux1Key:         "v2.1.1",
				MultiLux2Key:         "v2.1.0",
				MultiLuxEVMKey: "v0.4.2",
				LatestEVM2LuxKey:     "v0.9.9",
				LatestLux2EVMKey:     "v4.3.2",
			},
			sourceEVM: `{
					"rpcChainVMProtocolVersion": {
						"v1.0.0": 100,
						"v0.9.9": 99,
						"v0.5.3": 88,
						"v0.5.2": 87,
						"v0.5.1": 86,
						"v0.4.2": 77,
						"v0.4.1": 77,
						"v0.4.0": 17
					}
			  }`,
			sourceLux: `{
					"99": [
						"v4.3.2"
					],
					"88": [
						"v2.3.4"
					],
					"87": [
						"v2.2.2"
					],
					"86": [
						"v2.2.1"
					],
					"77": [
						"v2.1.1",
						"v2.1.0"
					],
					"18": [
						"v1.9.1"
					],
					"17": [
						"v1.9.0"
					]
			  }`,
		},
		{
			// this test should fail, simulating that
			// the APIs would return empty releases for some reason
			name:        "all-empty responses",
			shouldFail:  true,
			expected:    map[string]string{},
			sourceEVM:   `{}`,
			sourceLux: `{}`,
		},
		{
			// this test should fail, simulating that
			// the APIs would return empty releases for some reason
			// but only got sourceEVM versions
			name:        "only evm",
			shouldFail:  true,
			expected:    map[string]string{},
			sourceLux: `{}`,
			sourceEVM: `{
					"rpcChainVMProtocolVersion": {
						"v1.0.0": 100,
						"v0.9.9": 99,
						"v0.9.8": 99,
						"v0.4.2": 18,
						"v0.4.1": 18,
						"v0.4.0": 17
					}
			  }`,
		},
		{
			// this test should fail, simulating that
			// the APIs would return empty releases for some reason
			// but only got sourceLux versions
			name:       "only lux",
			shouldFail: true,
			expected:   map[string]string{},
			sourceEVM:  `{}`,
			sourceLux: `{
					"99": [
						"v2.3.4",
						"v2.3.3"
					],
					"18": [
						"v1.9.1"
					],
					"17": [
						"v1.9.0"
					]
			  }`,
		},
	}

	for i, tc := range testContexts {
		t.Run(tc.name, func(tt *testing.T) {
			// run the function, but use the testMapper,
			// so that we can set the currentContext
			mapping, err := m.getVersionMapping(tc)
			if tc.shouldFail {
				require.Error(err)
				return
			}
			require.NoError(err)
			// make sure the mapping returned from `GetVersionMapping`
			// matches the expected one
			require.Equal(tc.expected, mapping, fmt.Sprintf("iteration: %d", i))
		})
	}
}
