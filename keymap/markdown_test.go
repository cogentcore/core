// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package keymap

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMarkdown(t *testing.T) {
	str := AvailableMaps.MarkdownDoc()
	golden, err := os.ReadFile(filepath.Join("testdata", "keymaps-golden.md"))
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join("testdata", "keymaps.md"), []byte(str), 0666)
	assert.NoError(t, err)
	assert.Equal(t, string(golden), str)
}
