// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package signatureaggregator

import (
	"errors"

	"github.com/luxfi/cli/pkg/application"
	"github.com/luxfi/sdk/models"
	"github.com/luxfi/node/utils/logging"
)

// Minimal stub implementation until warp packages are available

func NewSignatureAggregatorLogger(
	aggregatorLogLevel string,
	aggregatorLogToStdout bool,
	logDir string,
) (logging.Logger, error) {
	return nil, errors.New("signature aggregator functionality temporarily disabled")
}

func GetLatestSignatureAggregatorReleaseVersion() (string, error) {
	return "", errors.New("signature aggregator functionality temporarily disabled")
}

func UpdateSignatureAggregatorPeers(
	app *application.Lux,
	network models.Network,
	extraAggregatorPeers []string,
	logger logging.Logger,
) error {
	return errors.New("signature aggregator functionality temporarily disabled")
}

func GetSignatureAggregatorEndpoint(app *application.Lux, network models.Network) (string, error) {
	// Return a default endpoint for now
	return "http://localhost:8090/aggregate-signatures", nil
}
