// Copyright (C) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package localnet

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/luxfi/sdk/models"
	"github.com/luxfi/sdk/network"
	"github.com/luxfi/sdk/publicarchive"
	"github.com/luxfi/node/utils/logging"

	"go.uber.org/zap"
)

// Downloads luxd database into the given [nodeNames]
// To be used on [testnet] only, after creating the nodes, but previously starting them.
func DownloadLuxdDB(
	clusterNetwork models.Network,
	rootDir string,
	nodeNames []string,
	log logging.Logger,
	printFunc func(msg string, args ...interface{}),
) error {
	// only for testnet
	if clusterNetwork.Kind() != models.Testnet {
		return nil
	}
	testnet := &network.Network{
		Name: "testnet",
		Type: network.NetworkTypeTestnet,
	}
	printFunc("Downloading public archive for network %s", clusterNetwork.Name())
	publicArcDownloader, err := publicarchive.NewDownloader(testnet, logging.NewLogger("public-archive-downloader", logging.NewWrappedCore(logging.Off, os.Stdout, logging.JSON.ConsoleEncoder()))) // off as we run inside of the spinner
	if err != nil {
		return fmt.Errorf("failed to create public archive downloader for network %s: %w", clusterNetwork.Name(), err)
	}

	if err := publicArcDownloader.Download(); err != nil {
		return fmt.Errorf("failed to download public archive: %w", err)
	}
	defer publicArcDownloader.CleanUp()
	if path, err := publicArcDownloader.GetFilePath(); err != nil {
		return fmt.Errorf("failed to get downloaded file path: %w", err)
	} else {
		log.Info("public archive downloaded into", zap.String("path", path))
	}

	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	var firstErr error

	for _, nodeName := range nodeNames {
		target := filepath.Join(rootDir, nodeName, "db")
		log.Info("unpacking public archive into", zap.String("target", target))

		// Skip if target already exists
		if _, err := os.Stat(target); err == nil {
			log.Info("data folder already exists. Skipping...", zap.String("target", target))
			continue
		}
		wg.Add(1)
		go func(target string) {
			defer wg.Done()

			if err := publicArcDownloader.UnpackTo(target); err != nil {
				// Capture the first error encountered
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("failed to unpack public archive: %w", err)
					_ = cleanUpClusterNodeData(rootDir, nodeNames)
				}
				mu.Unlock()
			}
		}(target)
	}
	wg.Wait()

	if firstErr != nil {
		return firstErr
	}
	printFunc("Public archive unpacked to: %s", rootDir)
	return nil
}

func cleanUpClusterNodeData(rootDir string, nodesNames []string) error {
	for _, nodeName := range nodesNames {
		if err := os.RemoveAll(filepath.Join(rootDir, nodeName, "db")); err != nil {
			return err
		}
	}
	return nil
}
