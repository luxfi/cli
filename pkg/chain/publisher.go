// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package chain provides chain deployment and management utilities.
package chain

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/luxfi/cli/pkg/ux"
	"github.com/luxfi/constants"
)

// Publisher defines the interface for publishing subnet and VM configurations to a git repository.
type Publisher interface {
	// Publish commits and pushes subnet and VM YAML configurations to the repository.
	Publish(r *git.Repository, subnetName, vmName string, subnetYAML []byte, vmYAML []byte) error
	// GetRepo returns the git repository, cloning it if necessary.
	GetRepo() (*git.Repository, error)
}

type publisherImpl struct {
	alias    string
	repoURL  string
	repoPath string
}

var _ Publisher = &publisherImpl{}

// NewPublisher creates a new Publisher instance for the given repository.
func NewPublisher(repoDir, repoURL, alias string) Publisher {
	repoPath := filepath.Join(repoDir, alias)
	return &publisherImpl{
		alias:    alias,
		repoURL:  repoURL,
		repoPath: repoPath,
	}
}

// GetRepo returns the git repository, opening it if it exists locally or cloning it otherwise.
func (p *publisherImpl) GetRepo() (repo *git.Repository, err error) {
	// path exists
	if _, err := os.Stat(p.repoPath); err == nil {
		return git.PlainOpen(p.repoPath)
	}
	return git.PlainClone(p.repoPath, false, &git.CloneOptions{
		URL:      p.repoURL,
		Progress: os.Stdout,
	})
}

// Publish writes the subnet and VM YAML files to the repository,
// commits the changes, and pushes to the remote.
func (p *publisherImpl) Publish(
	repo *git.Repository,
	subnetName, vmName string,
	subnetYAML []byte,
	vmYAML []byte,
) error {
	wt, err := repo.Worktree()
	if err != nil {
		return err
	}
	// Determine the correct path based on repo structure
	subnetPath := getSubnetPath(p.repoPath, subnetName)
	if err := os.MkdirAll(filepath.Dir(subnetPath), constants.DefaultPerms755); err != nil {
		return err
	}
	vmPath := filepath.Join(p.repoPath, constants.VMDir, vmName+constants.YAMLSuffix)
	if err := os.MkdirAll(filepath.Dir(vmPath), constants.DefaultPerms755); err != nil {
		return err
	}
	if err := os.WriteFile(subnetPath, subnetYAML, constants.DefaultPerms755); err != nil {
		return err
	}

	if err := os.WriteFile(vmPath, vmYAML, constants.DefaultPerms755); err != nil {
		return err
	}

	ux.Logger.PrintToUser("Adding resources to local git repo...")

	if _, err := wt.Add("subnets"); err != nil {
		return err
	}

	if _, err := wt.Add("vms"); err != nil {
		return err
	}

	ux.Logger.PrintToUser("Committing resources to local git repo...")
	now := time.Now()
	commitStr := fmt.Sprintf("commit-%s", now.String())

	// use the global git config to try identifying the author
	conf, err := config.LoadConfig(config.GlobalScope)
	authorName := conf.Author.Name
	authorEmail := conf.Author.Email
	if err != nil || authorName == "" || authorEmail == "" { // a commit must have both
		authorName = constants.GitRepoCommitName
		authorEmail = constants.GitRepoCommitEmail
	}

	commit, err := wt.Commit(commitStr, &git.CommitOptions{
		Author: &object.Signature{
			Name:  authorName,
			Email: authorEmail,
			When:  now,
		},
	})
	if err != nil {
		return err
	}

	if _, err := repo.CommitObject(commit); err != nil {
		return err
	}

	ux.Logger.PrintToUser("Pushing to remote...")
	return repo.Push(&git.PushOptions{})
}

// getSubnetPath determines the correct path for the subnet file based on repository structure
func getSubnetPath(repoPath, subnetName string) string {
	// Check if the repository has a custom structure
	customPath := filepath.Join(repoPath, "subnets", subnetName+constants.YAMLSuffix)
	if _, err := os.Stat(filepath.Dir(customPath)); err == nil {
		return customPath
	}

	// Check for legacy structure
	legacyPath := filepath.Join(repoPath, "subnet", subnetName+constants.YAMLSuffix)
	if _, err := os.Stat(filepath.Dir(legacyPath)); err == nil {
		return legacyPath
	}

	// Default to the standard structure
	return filepath.Join(repoPath, constants.ChainsDir, subnetName+constants.YAMLSuffix)
}
