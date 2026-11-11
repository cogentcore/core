// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"archive/zip"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBtbnAssetName(t *testing.T) {
	// (the names of the assets of the release of BtbN)
	assert.Equal(t, "ffmpeg-n9.0-latest-linux64-lgpl-shared-9.0.tar.xz", btbnAssetName("linux", "amd64"))
	assert.Equal(t, "ffmpeg-n9.0-latest-linuxarm64-lgpl-shared-9.0.tar.xz", btbnAssetName("linux", "arm64"))
	assert.Equal(t, "ffmpeg-n9.0-latest-win64-lgpl-shared-9.0.zip", btbnAssetName("windows", "amd64"))
	assert.Equal(t, "ffmpeg-n9.0-latest-winarm64-lgpl-shared-9.0.zip", btbnAssetName("windows", "arm64"))
	assert.Equal(t, "", btbnAssetName("darwin", "arm64"), "Homebrew on macOS")
	assert.Equal(t, "", btbnAssetName("freebsd", "amd64"))
	assert.Equal(t, "9", ffmpegMajor())
}

// fakeFFmpegTree makes the files of a build of BtbN in the given top
// directory under dir.
func fakeFFmpegTree(t *testing.T, dir, top, lib string) {
	for _, f := range []string{"lib/pkgconfig/libavutil.pc", "include/libavutil/avutil.h", "lib/" + lib} {
		p := filepath.Join(dir, top, f)
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0777))
		require.NoError(t, os.WriteFile(p, []byte(f), 0666))
	}
}

func TestInstallFFmpegArchive(t *testing.T) {
	if _, err := exec.LookPath("tar"); err != nil {
		t.Skip("no tar")
	}
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "cache", "ffmpeg")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "old"), 0777)) // a previous version

	// a tar.xz of Linux
	top := "ffmpeg-n9.0-latest-linux64-lgpl-shared-9.0"
	src := filepath.Join(tmp, "src")
	fakeFFmpegTree(t, src, top, "libavutil.so.61")
	archive := filepath.Join(tmp, top+".tar.xz")
	if out, err := exec.Command("tar", "-cJf", archive, "-C", src, top).CombinedOutput(); err != nil {
		t.Skipf("tar cannot make an xz archive here: %s", out)
	}
	require.NoError(t, installFFmpegArchive(archive, dir, "linux stamp"))
	assert.FileExists(t, filepath.Join(dir, "lib", "pkgconfig", "libavutil.pc"))
	assert.FileExists(t, filepath.Join(dir, "lib", "libavutil.so.61"))
	assert.NoDirExists(t, filepath.Join(dir, "old"), "replaced")
	b, err := os.ReadFile(filepath.Join(dir, "core-build.txt"))
	require.NoError(t, err)
	assert.Equal(t, "linux stamp", strings.TrimSpace(string(b)))

	// a zip of Windows
	top = "ffmpeg-n9.0-latest-win64-lgpl-shared-9.0"
	zipFile := filepath.Join(tmp, top+".zip")
	f, err := os.Create(zipFile)
	require.NoError(t, err)
	zw := zip.NewWriter(f)
	for _, name := range []string{"lib/pkgconfig/libavutil.pc", "bin/avutil-61.dll"} {
		w, err := zw.Create(top + "/" + name)
		require.NoError(t, err)
		w.Write([]byte(name))
	}
	require.NoError(t, zw.Close())
	require.NoError(t, f.Close())
	if err := installFFmpegArchive(zipFile, dir, "windows stamp"); err != nil {
		t.Skipf("tar cannot extract a zip here: %v", err)
	}
	assert.FileExists(t, filepath.Join(dir, "bin", "avutil-61.dll"))
	assert.NoFileExists(t, filepath.Join(dir, "lib", "libavutil.so.61"), "replaced")

	// an archive without pkg-config files is not installed
	bad := filepath.Join(tmp, "bad")
	require.NoError(t, os.MkdirAll(filepath.Join(bad, "x", "lib"), 0777))
	badArchive := filepath.Join(tmp, "x.tar.xz")
	require.NoError(t, exec.Command("tar", "-cJf", badArchive, "-C", bad, "x").Run())
	assert.Error(t, installFFmpegArchive(badArchive, dir, "bad"))
	assert.FileExists(t, filepath.Join(dir, "bin", "avutil-61.dll"), "kept")
}

func TestSetFFmpegEnv(t *testing.T) {
	// the cache directory of the user is in the temporary directory
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(tmp, ".cache"))
	t.Setenv("LocalAppData", filepath.Join(tmp, "AppData"))
	t.Setenv("PKG_CONFIG_PATH", "/old")
	t.Setenv("CGO_LDFLAGS", "")
	t.Setenv("PATH", os.Getenv("PATH"))
	dir := ffmpegDir()
	require.NotEmpty(t, dir)
	require.True(t, strings.HasPrefix(dir, tmp), dir)
	assert.False(t, ffmpegInstalled())

	fakeFFmpegTree(t, filepath.Dir(dir), filepath.Base(dir), "libavutil.so.61")
	require.True(t, ffmpegInstalled())
	SetFFmpegEnv()
	pc := filepath.Join(dir, "lib", "pkgconfig")
	assert.Equal(t, pc+string(os.PathListSeparator)+"/old", os.Getenv("PKG_CONFIG_PATH"))
	switch runtime.GOOS {
	case "linux":
		assert.Equal(t, "-Wl,-rpath,"+filepath.Join(dir, "lib"), os.Getenv("CGO_LDFLAGS"))
	case "windows":
		assert.True(t, strings.HasPrefix(os.Getenv("PATH"), filepath.Join(dir, "bin")))
	}
}
