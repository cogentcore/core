// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"os"
	osexec "os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"cogentcore.org/core/cmd/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGoSymlinkApp tests that the executable of an app is made a script
// that runs a copy of the one installed by go install, in the app, signed
// with the ID of the app, for AppGoSymlink, and copies it again when it
// changes, with a module and a Go bin directory of the test.
func TestGoSymlinkApp(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("the app is signed with codesign, on macOS")
	}
	tmp := t.TempDir()
	mod := filepath.Join(tmp, "zzapp")
	require.NoError(t, os.MkdirAll(mod, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(mod, "go.mod"), []byte("module example.com/zzapp\n\ngo 1.21\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(mod, "main.go"), []byte("package main\n\nimport (\n\t\"fmt\"\n\t\"os\"\n)\n\nfunc main() {\n\texe, _ := os.Executable()\n\tfmt.Print(exe, \" \", os.Args[1])\n}\n"), 0o644))
	bin := filepath.Join(tmp, "bin")
	t.Setenv("GOBIN", bin)
	t.Setenv("GOWORK", "off")
	t.Chdir(mod)

	c := &config.Config{Name: "ZZ App", ID: "com.example.zzapp"}
	app := filepath.Join(tmp, "ZZ App.app")
	exe := appExecutable(c, app)
	require.NoError(t, os.MkdirAll(filepath.Dir(exe), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(app, "Contents", "Info.plist"), []byte(`<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><dict><key>CFBundleExecutable</key><string>ZZ App.app</string><key>CFBundleIdentifier</key><string>com.example.zzapp</string></dict></plist>`), 0o644))
	require.NoError(t, os.WriteFile(exe, []byte("packed"), 0o755))

	// a packed executable is removed too, so that cp does not write over it
	// in place, which macOS kills as having an invalid signature
	require.NoError(t, removeExecutable(exe))
	assert.NoFileExists(t, exe)
	require.NoError(t, removeExecutable(exe)) // none is fine

	require.NoError(t, goSymlinkApp(c, app))
	cp := filepath.Join(filepath.Dir(exe), "zzapp")
	target := filepath.Join(bin, "zzapp")
	assert.FileExists(t, target, "installed by go install")
	fi, err := os.Lstat(cp)
	require.NoError(t, err)
	assert.True(t, fi.Mode().IsRegular(), "a copy, not a link")
	out, err := osexec.Command("codesign", "-dv", cp).CombinedOutput()
	require.NoError(t, err, string(out))
	assert.Contains(t, string(out), "Identifier=com.example.zzapp")

	// it runs the copy, in the app, with the arguments
	out, err = osexec.Command(exe, "arg").CombinedOutput()
	require.NoError(t, err, string(out))
	assert.Equal(t, cp+" arg", string(out))

	// a newer one installed by go install is copied when the app starts
	before := time.Now().Add(-time.Hour)
	require.NoError(t, os.Chtimes(cp, before, before))
	_, err = osexec.Command(exe, "arg").CombinedOutput()
	require.NoError(t, err)
	fi, err = os.Stat(cp)
	require.NoError(t, err)
	assert.True(t, fi.ModTime().After(before.Add(time.Minute)), "copied again")

	require.NoError(t, goSymlinkApp(c, app)) // again
	require.NoError(t, removeExecutable(exe))
	assert.NoFileExists(t, exe)
	assert.FileExists(t, target, "the installed one stays")
}

// TestParseSignIdentity tests choosing the identity for signing apps.
func TestParseSignIdentity(t *testing.T) {
	out := `  1) 0123456789ABCDEF0123456789ABCDEF01234567 "Developer ID Application: Someone (TEAM1)"
  2) 89ABCDEF0123456789ABCDEF0123456789ABCDEF "Apple Development: Someone (TEAM2)"
     2 valid identities found
`
	assert.Equal(t, "89ABCDEF0123456789ABCDEF0123456789ABCDEF", parseSignIdentity(out))
	assert.Equal(t, "-", parseSignIdentity("     0 valid identities found\n"), "ad hoc")
}
