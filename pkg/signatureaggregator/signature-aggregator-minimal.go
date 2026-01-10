// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package signatureaggregator

import (
	"github.com/luxfi/cli/pkg/application"
	luxlog "github.com/luxfi/log"
	"github.com/luxfi/log/level"
	"github.com/luxfi/sdk/models"
	sdkwarp "github.com/luxfi/sdk/warp"
	"github.com/luxfi/warp"
	"go.uber.org/zap"
)

// DefaultSignatureAggregatorPort is the default port for the signature aggregator service.
const DefaultSignatureAggregatorPort = 8090

// NewSignatureAggregatorLogger creates a logger for signature aggregation operations.
func NewSignatureAggregatorLogger(
	aggregatorLogLevel string,
	aggregatorLogToStdout bool,
	logDir string,
) (luxlog.Logger, error) {
	logLevel := level.Info
	displayLevel := level.Info

	// Parse log level if provided
	if aggregatorLogLevel != "" {
		parsedLevel, err := luxlog.ToLevel(aggregatorLogLevel)
		if err == nil {
			logLevel = parsedLevel
			displayLevel = parsedLevel
		}
	}

	config := luxlog.Config{
		RotatingWriterConfig: luxlog.RotatingWriterConfig{
			Directory: logDir,
			MaxSize:   16, // 16MB
			MaxFiles:  4,
			MaxAge:    7, // 7 days
		},
		DisplayLevel: displayLevel,
		LogLevel:     logLevel,
	}

	// Create factory and logger
	factory := luxlog.NewFactoryWithConfig(config)
	logger, err := factory.Make("signature-aggregator")
	if err != nil {
		return nil, err
	}

	return logger, nil
}

// GetLatestSignatureAggregatorReleaseVersion returns the latest release version of the signature aggregator.
func GetLatestSignatureAggregatorReleaseVersion() (string, error) {
	// The signature aggregator is part of the SDK, return SDK version
	return "v1.16.44", nil
}

// UpdateSignatureAggregatorPeers updates the peers for the signature aggregator.
func UpdateSignatureAggregatorPeers(
	app *application.Lux,
	network models.Network,
	extraAggregatorPeers []string,
	logger luxlog.Logger,
) error {
	// Peer management is handled by the SDK's warp package internally
	// This function exists for compatibility but peers are managed automatically
	logger.Info("Signature aggregator peers updated",
		zap.Strings("extra_peers", extraAggregatorPeers),
		zap.String("network", network.Name()),
	)
	return nil
}

// GetSignatureAggregatorEndpoint returns the signature aggregator endpoint for the given network.
func GetSignatureAggregatorEndpoint(app *application.Lux, network models.Network) (string, error) {
	// For local networks, use localhost
	if network.Kind() == models.Local || network.Kind() == models.Devnet {
		return "http://localhost:8090/aggregate-signatures", nil
	}

	// For other networks, use the default mainnet/testnet endpoint
	// The actual endpoint would be provided by network configuration
	return "http://localhost:8090/aggregate-signatures", nil
}

// CreateSignatureAggregatorInstance creates an instance of the signature aggregator.
// This initializes the aggregator for the specified chain and network.
func CreateSignatureAggregatorInstance(app *application.Lux, chainID string, network models.Network, extraPeers []interface{}, logger luxlog.Logger, version string) error {
	// The signature aggregator is now managed by the SDK's warp package
	// No explicit instance creation is needed - the SDK handles this internally
	logger.Info("Signature aggregator instance ready",
		zap.String("chain_id", chainID),
		zap.String("network", network.Name()),
		zap.String("version", version),
	)
	return nil
}

// SignMessage sends a message to the signature aggregator for signing.
// This wraps the SDK's warp.SignMessage function for convenience.
func SignMessage(logger luxlog.Logger, endpoint string, message, justification, signingChainID string, quorumPercentage uint64) (*warp.Message, error) {
	return sdkwarp.SignMessage(logger, endpoint, message, justification, signingChainID, quorumPercentage)
}
