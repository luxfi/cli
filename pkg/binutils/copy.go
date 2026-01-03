// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package binutils

import (
	"io"
	"os"

	"github.com/luxfi/cli/pkg/constants"
)

// CopyFile copies a file from src to dest.
func CopyFile(src, dest string) error {
	in, err := os.Open(src) //nolint:gosec // G304: Copying from known source
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.Create(dest) //nolint:gosec // G304: Copying to known destination
	if err != nil {
		return err
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	if err = out.Sync(); err != nil {
		return err
	}
	if err = out.Chmod(constants.DefaultPerms755); err != nil {
		return err
	}
	return nil
}
