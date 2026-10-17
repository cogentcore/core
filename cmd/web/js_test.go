// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyExternalEmbeds(t *testing.T) {
	// GoScript writes the external embedded files beside the output of each
	// package, here a package and a nested one.
	output := t.TempDir()
	files := map[string]string{
		"@goscript/example.com/app/@embed/0123-a.png":     "a",
		"@goscript/example.com/app/sub/@embed/4567-b.jpg": "b",
		"@goscript/example.com/app/main.gs.ts":            "code",
	}
	for p, data := range files {
		fp := filepath.Join(output, filepath.FromSlash(p))
		require.NoError(t, os.MkdirAll(filepath.Dir(fp), 0777))
		require.NoError(t, os.WriteFile(fp, []byte(data), 0666))
	}

	// They are all copied into one directory beside app.gs.js, replacing
	// those of an earlier build.
	odir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(odir, "@embed"), 0777))
	require.NoError(t, os.WriteFile(filepath.Join(odir, "@embed", "old.png"), nil, 0666))
	require.NoError(t, copyExternalEmbeds(output, odir))
	assert.Equal(t, []string{"@embed/0123-a.png", "@embed/4567-b.jpg"}, externalEmbedFiles(odir))
	b, err := os.ReadFile(filepath.Join(odir, "@embed", "4567-b.jpg"))
	require.NoError(t, err)
	assert.Equal(t, "b", string(b))
}

func TestJSExternalEmbedEnv(t *testing.T) {
	env := jsExternalEmbedEnv()
	assert.Equal(t, "8192", env["GOSCRIPT_EXTERNAL_EMBED_MIN_SIZE"])
	assert.Contains(t, env["GOSCRIPT_EXTERNAL_EMBED"], "*.png,")
	assert.NotContains(t, env["GOSCRIPT_EXTERNAL_EMBED"], "*.md")
}

func TestExternalContentFiles(t *testing.T) {
	// A content image that the app loads from the @embed directory, where
	// GoScript names it by the digest of its contents.
	content, odir := t.TempDir(), t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(content, "media"), 0777))
	require.NoError(t, os.WriteFile(filepath.Join(content, "media", "a b.png"), []byte("png"), 0666))
	require.NoError(t, os.WriteFile(filepath.Join(content, "media", "small.png"), []byte("small"), 0666))
	require.NoError(t, os.MkdirAll(filepath.Join(odir, "@embed"), 0777))
	name := "8f8cbb7dcf46e0bc-a b.png" // sha256("png")[:16]
	require.NoError(t, os.WriteFile(filepath.Join(odir, "@embed", name), []byte("png"), 0666))

	// The pages show it from there, and the inline image from its copy.
	files := externalContentFiles(content, odir)
	assert.Equal(t, map[string]string{"media/a b.png": "@embed/8f8cbb7dcf46e0bc-a%20b.png"}, files)
	h := rebaseURLs(`<img src="media/a b.png"><img src="media/small.png">`, "../", "page", nil, files)
	assert.Equal(t, `<img src="../@embed/8f8cbb7dcf46e0bc-a%20b.png"><img src="../media/small.png">`, h)
	require.NoError(t, copyContentFiles(content, odir, files))
	assert.NoFileExists(t, filepath.Join(odir, "media", "a b.png"))
	assert.FileExists(t, filepath.Join(odir, "media", "small.png"))
}
