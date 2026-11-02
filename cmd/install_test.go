// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"cogentcore.org/core/cmd/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGoSymlinkApp tests that the executable of an app is made a symbolic
// link to the one installed by go install, for AppGoSymlink, with a module
// and a Go bin directory of the test.
func TestGoSymlinkApp(t *testing.T) {
	tmp := t.TempDir()
	mod := filepath.Join(tmp, "zzapp")
	require.NoError(t, os.MkdirAll(mod, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(mod, "go.mod"), []byte("module example.com/zzapp\n\ngo 1.21\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(mod, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))
	bin := filepath.Join(tmp, "bin")
	t.Setenv("GOBIN", bin)
	t.Setenv("GOWORK", "off")
	t.Chdir(mod)

	c := &config.Config{Name: "ZZ App"}
	app := filepath.Join(tmp, "ZZ App.app")
	exe := appExecutable(c, app)
	require.NoError(t, os.MkdirAll(filepath.Dir(exe), 0o755))
	require.NoError(t, os.WriteFile(exe, []byte("packed"), 0o755))

	require.NoError(t, removeSymlink(exe))
	assert.FileExists(t, exe, "not a symbolic link")

	require.NoError(t, goSymlinkApp(c, app))
	target, err := os.Readlink(exe)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(bin, "zzapp"), target)
	assert.FileExists(t, target, "installed by go install")

	require.NoError(t, goSymlinkApp(c, app)) // again
	require.NoError(t, removeSymlink(exe))
	assert.NoFileExists(t, exe)
	assert.FileExists(t, target, "the installed one stays")
}
