// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

// fakeTool writes an empty executable file for the given tool in dir.
func fakeTool(t *testing.T, dir, tool string) {
	if runtime.GOOS == "windows" {
		tool += ".exe"
	}
	assert.NoError(t, os.WriteFile(filepath.Join(dir, tool), nil, 0755))
}

func TestEnsureTool(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	// A tool in one of its install directories is found there, without
	// installing it, and the directory is added to the PATH.
	dir := t.TempDir()
	fakeTool(t, dir, "gstoolfound")
	installed := false
	err := ensureTool("gstoolfound", []string{dir}, func() error { installed = true; return nil })
	assert.NoError(t, err)
	assert.False(t, installed)
	assert.Contains(t, filepath.SplitList(os.Getenv("PATH")), dir)

	// A tool that is not found is installed, and then found.
	dir = t.TempDir()
	err = ensureTool("gstoolnew", []string{dir}, func() error {
		fakeTool(t, dir, "gstoolnew")
		return nil
	})
	assert.NoError(t, err)
	assert.Contains(t, filepath.SplitList(os.Getenv("PATH")), dir)

	// A tool that is still missing after installing is an error.
	err = ensureTool("gstoolmissing", []string{t.TempDir()}, func() error { return nil })
	assert.Error(t, err)
}
