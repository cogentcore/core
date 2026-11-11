// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"archive/zip"
	"go/parser"
	"go/token"
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

func TestWindowsInstallerTemplate(t *testing.T) {
	// The installer is valid Go, with and without files of FFmpeg.
	for _, files := range [][]string{nil, {"avutil-61.dll", "FFmpeg-LICENSE.txt"}} {
		var b strings.Builder
		require.NoError(t, windowsInstallerTmpl.Execute(&b, &windowsInstallerData{Name: "My App", Desc: "an app", Files: files}))
		src := b.String()
		_, err := parser.ParseFile(token.NewFileSet(), "installer.go", src, 0)
		require.NoError(t, err, src)
		for _, f := range files {
			assert.Contains(t, src, `files.ReadFile("files/`+f+`")`)
			assert.Contains(t, src, `dir+"\\`+f+`"`)
		}
		assert.Equal(t, files != nil, strings.Contains(src, "//go:embed files"))
	}
}

func TestFFmpegWindowsFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("lists the packages of an app")
	}
	// The Go environment stays that of the user while HOME (and so the
	// cache directory of FFmpeg) is a temporary directory.
	for _, name := range []string{"GOPATH", "GOMODCACHE", "GOCACHE", "GOENV"} {
		out, err := exec.Command("go", "env", name).Output()
		require.NoError(t, err)
		t.Setenv(name, strings.TrimSpace(string(out)))
	}
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(tmp, ".cache"))
	t.Setenv("LocalAppData", filepath.Join(tmp, "AppData"))
	t.Chdir(filepath.Join("..", "examples", "video"))

	// The video example uses FFmpeg, whose DLLs are not installed yet.
	_, err := ffmpegWindowsFiles()
	assert.ErrorContains(t, err, "core setup -video")

	dir := ffmpegDir()
	for _, f := range []string{"bin/avutil-61.dll", "bin/avcodec-63.dll", "bin/ffmpeg.exe", "LICENSE.txt"} {
		require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(dir, f)), 0777))
		require.NoError(t, os.WriteFile(filepath.Join(dir, f), []byte(f), 0666))
	}
	files, err := ffmpegWindowsFiles()
	require.NoError(t, err)
	var names []string
	for _, f := range files {
		names = append(names, filepath.Base(f))
	}
	assert.ElementsMatch(t, []string{"avutil-61.dll", "avcodec-63.dll", "LICENSE.txt"}, names, "the DLLs and license, not the programs")

	// built without video, it does not use FFmpeg
	t.Setenv("GOFLAGS", "-tags=novideo")
	files, err = ffmpegWindowsFiles()
	require.NoError(t, err)
	assert.Empty(t, files)
}
