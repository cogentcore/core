// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fileinfo

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMimeStrings tests the parts of mime type strings, and the
// categories of mime types.
func TestMimeStrings(t *testing.T) {
	m := "Text/Plain; charset=utf-8"
	assert.Equal(t, "text/plain", MimeNoChar(m))
	assert.Equal(t, "Text", MimeTop(m))
	assert.Equal(t, "plain", MimeSub(m), "without the charset")
	assert.Equal(t, "", MimeTop("plain"))
	assert.Equal(t, "", MimeSub("plain"))

	assert.Equal(t, Code, CategoryFromMime(MimeString(Go)))
	cats := map[string]Categories{
		"":                      UnknownCategory,
		"image/x-unknown":       Image,
		"audio/x-unknown":       Audio,
		"video/x-unknown":       Video,
		"font/x-unknown":        Font,
		"model/x-unknown":       Model,
		"text/x-unknown":        Text,
		"application/x-unknown": UnknownCategory,
		"unknown":               UnknownCategory,
	}
	for mime, cat := range cats {
		assert.Equal(t, cat, CategoryFromMime(mime), mime)
	}
}

// TestKnown tests the known file types, their mime types, categories,
// and matching.
func TestKnown(t *testing.T) {
	assert.NotEmpty(t, MimeString(Go))
	assert.Equal(t, "", MimeString(Known(-1)))
	assert.Equal(t, Code, Go.Cat())
	assert.Equal(t, UnknownCategory, Unknown.Cat())
	assert.Equal(t, UnknownCategory, Known(-1).Cat())
	assert.Equal(t, Go, MimeKnown(MimeString(Go)))
	assert.Equal(t, Unknown, MimeKnown("application/x-unknown"))
	assert.Equal(t, Go, ExtKnown(".go"))
	assert.Equal(t, Unknown, ExtKnown(".unknownext"))
	assert.Equal(t, Go, MimeFromKnown(Go).Known)
	assert.Equal(t, MimeType{}, MimeFromKnown(Known(-1)))

	kn, err := KnownByName("Go")
	assert.NoError(t, err)
	assert.Equal(t, Go, kn)
	_, err = KnownByName("NotAType")
	assert.Error(t, err)

	assert.True(t, IsMatch(Any, Unknown))
	assert.True(t, IsMatch(AnyKnown, Go))
	assert.False(t, IsMatch(AnyKnown, Unknown))
	assert.True(t, IsMatch(Go, Go))
	assert.True(t, IsMatch(AnyCode, Go))
	assert.False(t, IsMatch(AnyImage, Go))
	assert.False(t, IsMatch(Python, Go))
	for _, any := range []Known{AnyFolder, AnyArchive, AnyBackup, AnyDoc, AnySheet, AnyData, AnyText, AnyImage, AnyModel, AnyAudio, AnyVideo, AnyFont, AnyExe, AnyBin} {
		assert.False(t, IsMatch(any, Go), any.String())
	}
	assert.True(t, IsMatchList(nil, Go))
	assert.True(t, IsMatchList([]Known{AnyImage, AnyCode}, Go))
	assert.False(t, IsMatchList([]Known{AnyImage}, Go))
}

// TestMimeFromFile tests getting mime types from file names and contents,
// for files outside of the current directory.
func TestMimeFromFile(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	pngNoExt := filepath.Join(dir, "picture")
	require.NoError(t, os.WriteFile(pngNoExt, buf.Bytes(), 0644))
	mtyp, _, err := MimeFromFile(pngNoExt)
	require.NoError(t, err)
	assert.Equal(t, "image/png", mtyp, "from the content of a file in another directory")
	assert.Equal(t, PNG, KnownFromFile(pngNoExt))

	mtyp, _, err = MimeFromFile(filepath.Join(dir, "backup~"))
	require.NoError(t, err)
	assert.Equal(t, MimeString(Trash), mtyp)
	mf := filepath.Join(dir, "Makefile")
	require.NoError(t, os.WriteFile(mf, []byte("all:\n"), 0644))
	mtyp, _, err = MimeFromFile(mf)
	require.NoError(t, err)
	assert.Equal(t, MimeString(Makefile), mtyp)
	_, _, err = MimeFromFile(filepath.Join(dir, "mystery.zzz"))
	assert.Error(t, err)
	assert.Equal(t, Unknown, KnownFromFile(filepath.Join(dir, "mystery.zzz")))
}

// TestFileInfoTypes tests file infos of types, from file systems, and
// symlinks.
func TestFileInfoTypes(t *testing.T) {
	fi := NewFileInfoType(Go)
	assert.Equal(t, Go, fi.Known)
	assert.Equal(t, Code, fi.Cat)
	assert.Equal(t, "_fake.go", fi.Name)
	assert.Contains(t, fi.Kind, "Go")

	fsys := fstest.MapFS{"d/a.go": {Data: []byte("package a\n")}}
	var ffi FileInfo
	require.NoError(t, ffi.InitFileFS(fsys, "d/a.go"))
	assert.Equal(t, "a.go", ffi.Name)
	assert.Equal(t, Go, ffi.Known)
	assert.Error(t, ffi.InitFileFS(fsys, "d/missing.go"))
	assert.Equal(t, Go, ffi.Known, "from the name")

	dir := t.TempDir()
	tgt := filepath.Join(dir, "a.go")
	require.NoError(t, os.WriteFile(tgt, []byte("package a\n"), 0755))
	lnk := filepath.Join(dir, "link.go")
	require.NoError(t, os.Symlink(tgt, lnk))
	efi, err := NewFileInfo(tgt)
	require.NoError(t, err)
	assert.True(t, efi.IsExec())
	assert.False(t, efi.IsSymlink(), "stat follows links")

	efi.Path = lnk
	assert.NoError(t, efi.Delete())
	_, err = os.Lstat(lnk)
	assert.True(t, os.IsNotExist(err))
}
