// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"os"
	"path/filepath"
	"testing"

	"cogentcore.org/core/cmd/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyFiles(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	write := func(name, data string) {
		require.NoError(t, os.MkdirAll(filepath.Dir(name), 0777))
		require.NoError(t, os.WriteFile(name, []byte(data), 0666))
	}
	write("deer.mp4", "video")
	write("media/a.png", "a")
	write("media/sub/b.png", "b")
	write("clips/x.webm", "x")
	write("clips/y.txt", "y")

	c := &config.Config{}
	c.Build.Output = filepath.Join("bin", "web")
	c.Web.Files = []string{"deer.mp4", "media", "clips/*.webm"}
	require.NoError(t, copyFiles(c))
	read := func(name string) string {
		b, err := os.ReadFile(filepath.Join("bin", "web", name))
		require.NoError(t, err, name)
		return string(b)
	}
	assert.Equal(t, "video", read("deer.mp4"))
	assert.Equal(t, "a", read("media/a.png"))
	assert.Equal(t, "b", read("media/sub/b.png"))
	assert.Equal(t, "x", read("clips/x.webm"))
	assert.NoFileExists(t, filepath.Join("bin", "web", "clips", "y.txt"))

	// errors: no match, and paths outside of the app
	c.Web.Files = []string{"missing.mp4"}
	assert.Error(t, copyFiles(c))
	c.Web.Files = []string{"../*"}
	assert.Error(t, copyFiles(c))
}
