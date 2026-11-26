// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"os"
	osexec "os/exec"
	"path/filepath"
	"testing"

	"cogentcore.org/core/cmd/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGoSymlinkApp tests that the executable of an app is made a script
// that runs a symbolic link to the one installed by go install, for
// AppGoSymlink, with a module and a Go bin directory of the test.
func TestGoSymlinkApp(t *testing.T) {
	tmp := t.TempDir()
	mod := filepath.Join(tmp, "zzapp")
	require.NoError(t, os.MkdirAll(mod, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(mod, "go.mod"), []byte("module example.com/zzapp\n\ngo 1.21\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(mod, "main.go"), []byte("package main\n\nimport (\n\t\"fmt\"\n\t\"os\"\n)\n\nfunc main() {\n\texe, _ := os.Executable()\n\tfmt.Print(exe, \" \", os.Args[1])\n}\n"), 0o644))
	bin := filepath.Join(tmp, "bin")
	t.Setenv("GOBIN", bin)
	t.Setenv("GOWORK", "off")
	t.Chdir(mod)

	c := &config.Config{Name: "ZZ App"}
	app := filepath.Join(tmp, "ZZ App.app")
	exe := appExecutable(c, app)
	require.NoError(t, os.MkdirAll(filepath.Dir(exe), 0o755))
	require.NoError(t, os.WriteFile(exe, []byte("packed"), 0o755))

	// a packed executable is removed too, so that cp does not write over it
	// in place, which macOS kills as having an invalid signature
	require.NoError(t, removeExecutable(exe))
	assert.NoFileExists(t, exe)
	require.NoError(t, removeExecutable(exe)) // none is fine

	require.NoError(t, goSymlinkApp(c, app))
	link := filepath.Join(filepath.Dir(exe), "zzapp")
	target, err := os.Readlink(link)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(bin, "zzapp"), target)
	assert.FileExists(t, target, "installed by go install")
	// it runs the installed one, in the bundle, with the arguments
	out, err := osexec.Command(exe, "arg").CombinedOutput()
	require.NoError(t, err, string(out))
	assert.Equal(t, link+" arg", string(out))

	require.NoError(t, goSymlinkApp(c, app)) // again
	require.NoError(t, removeExecutable(exe))
	assert.NoFileExists(t, exe)
	assert.FileExists(t, target, "the installed one stays")
}
