// Copyright (C) 2022-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package networkcmd

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// archiveDirectory creates a tar.gz archive of the source directory
func archiveDirectory(src string, dst string) error {
	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Create the destination file
	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create archive file: %w", err)
	}
	defer out.Close()

	// Create gzip writer
	gw := gzip.NewWriter(out)
	defer gw.Close()

	// Create tar writer
	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Walk the source directory
	return filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create tar header
		header, err := tar.FileInfoHeader(fi, file)
		if err != nil {
			return err
		}

		// Update header name to be relative to source directory
		relPath, err := filepath.Rel(src, file)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}
		header.Name = relPath

		// Windows compatibility for path separators
		header.Name = strings.ReplaceAll(header.Name, "\\", "/")

		// Write header
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// If not a regular file, return
		if !fi.Mode().IsRegular() {
			return nil
		}

		// Copy file data
		f, err := os.Open(file)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := io.Copy(tw, f); err != nil {
			return err
		}

		return nil
	})
}

// extractArchive extracts a tar.gz archive to the destination directory
func extractArchive(src string, dst string) error {
	// Open the archive file
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open archive file: %w", err)
	}
	defer in.Close()

	// Create gzip reader
	gr, err := gzip.NewReader(in)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gr.Close()

	// Create tar reader
	tr := tar.NewReader(gr)

	// Iterate through files
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Construct target path
		target := filepath.Join(dst, header.Name)

		// Validate path to prevent Zip Slip vulnerability
		destAbs, err := filepath.Abs(dst)
		if err != nil {
			return err
		}
		targetAbs, err := filepath.Abs(target)
		if err != nil {
			return err
		}
		if !strings.HasPrefix(targetAbs, destAbs+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path in archive: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			// Create parent directory if needed
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}

			// Create file
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			// Copy contents
			// Use a limit reader to prevent decompression bombs if needed, but for now just copy
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}

	return nil
}
