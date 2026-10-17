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
