// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"cogentcore.org/core/cmd/config"
)

// copyFiles copies the files and directories of [config.Web.Files] (paths
// and glob patterns relative to the directory of the app, the current
// directory) into the website, at the same relative paths.
func copyFiles(c *config.Config) error {
	for _, pat := range c.Web.Files {
		paths, err := filepath.Glob(pat)
		if err != nil {
			return fmt.Errorf("web files: %w", err)
		}
		if len(paths) == 0 {
			return fmt.Errorf("web files: no files match %q", pat)
		}
		for _, p := range paths {
			if filepath.IsAbs(p) || !filepath.IsLocal(p) {
				return fmt.Errorf("web files: %q is not within the directory of the app", p)
			}
			if err := copyTree(p, filepath.Join(c.Build.Output, p)); err != nil {
				return fmt.Errorf("web files: %w", err)
			}
		}
	}
	return nil
}

// copyTree copies the file or directory src to dst, replacing the files
// that are there.
func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		to := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(to, 0777)
		}
		return copyFile(path, to)
	})
}

// copyFile copies the file src to dst, making the directory of dst.
func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0777); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
