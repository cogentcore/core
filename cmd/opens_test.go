// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"cogentcore.org/core/base/fileinfo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenTypes(t *testing.T) {
	mts, err := openTypes([]string{"Code", "Markdown", "PlainText"})
	require.NoError(t, err)
	mimes := func(mts []fileinfo.MimeType) []string {
		var ms []string
		for _, mt := range mts {
			ms = append(ms, mt.Mime)
		}
		return ms
	}
	ms := mimes(mts)
	assert.Contains(t, ms, "text/x-gosrc")
	assert.Contains(t, ms, "text/x-python")
	assert.Contains(t, ms, "text/markdown")
	assert.Contains(t, ms, "text/plain")
	assert.NotContains(t, ms, "text/x-tex")
	assert.NotContains(t, ms, "application/msword")
	assert.Len(t, slices.Compact(slices.Sorted(slices.Values(ms))), len(ms), "no duplicates")

	_, err = openTypes([]string{"Code", "NotAType"})
	assert.Error(t, err)
	none, err := openTypes(nil)
	require.NoError(t, err)
	assert.Empty(t, none)

	dts := macDocTypes(mts)
	byName := map[string]macDocType{}
	for _, dt := range dts {
		byName[dt.Name] = dt
	}
	assert.Equal(t, "public.python-script", byName["Python"].UTI, "the type of macOS")
	assert.Empty(t, byName["Python"].Extensions, "not .pyc")
	assert.Equal(t, "", byName["Go"].UTI)
	assert.Equal(t, []string{"go", "mod", "work", "goal"}, byName["Go"].Extensions, "matched by extensions")
	for _, dt := range dts {
		assert.NotContains(t, dt.Extensions, "jar")
	}
	assert.True(t, strings.HasPrefix(mimeTypeList(mts), "text/"))
	assert.True(t, strings.HasSuffix(mimeTypeList(mts), ";"))
}

func TestInfoPlistDocTypes(t *testing.T) {
	mts, err := openTypes([]string{"Code", "Markdown"})
	require.NoError(t, err)
	var b strings.Builder
	require.NoError(t, infoPlistTmpl.Execute(&b, &infoPlistData{Name: "Test", Executable: "MacOS/Test", Identifier: "org.test.app", DocTypes: macDocTypes(mts)}))
	pl := b.String()
	assert.Contains(t, pl, "<key>CFBundleDocumentTypes</key>")
	assert.Contains(t, pl, "<string>net.daringfireball.markdown</string>")
	assert.Contains(t, pl, "<string>goal</string>")
	assert.Equal(t, len(mts), strings.Count(pl, "<string>Alternate</string>"))

	b.Reset()
	require.NoError(t, infoPlistTmpl.Execute(&b, &infoPlistData{Name: "Test"}))
	assert.NotContains(t, b.String(), "CFBundleDocumentTypes", "none without Opens")

	if runtime.GOOS != "darwin" {
		return
	}
	file := filepath.Join(t.TempDir(), "Info.plist")
	require.NoError(t, os.WriteFile(file, []byte(pl), 0o644))
	out, err := exec.Command("plutil", "-lint", file).CombinedOutput()
	assert.NoError(t, err, string(out))
}

func TestDesktopFileMimeTypes(t *testing.T) {
	mts, err := openTypes([]string{"Markdown", "PlainText"})
	require.NoError(t, err)
	var b strings.Builder
	require.NoError(t, desktopFileTmpl.Execute(&b, &desktopFileData{Name: "Test", Desc: "A test", Exec: "test", MimeTypes: mimeTypeList(mts)}))
	assert.Contains(t, b.String(), "\nExec=test %F\n")
	assert.Contains(t, b.String(), "\nMimeType=text/markdown;text/plain;\n")

	b.Reset()
	require.NoError(t, desktopFileTmpl.Execute(&b, &desktopFileData{Name: "Test", Desc: "A test", Exec: "test"}))
	assert.Contains(t, b.String(), "\nExec=test\n")
	assert.NotContains(t, b.String(), "MimeType")
}
