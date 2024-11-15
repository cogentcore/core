// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package content

import (
	"embed"
	"testing"

	"cogentcore.org/core/base/fsx"
	"github.com/stretchr/testify/assert"
)

//go:embed examples/basic/content
var content embed.FS

func TestNewPage(t *testing.T) {
	fsys := fsx.Sub(content, "examples/basic/content")
	pg, err := NewPage(fsys, "index.md")
	assert.NoError(t, err)
	assert.Equal(t, *pg, Page{
		FS:         fsys,
		Filename:   "index.md",
		Name:       "Home",
		Authors:    []string{"Cogent Core", "Go Gopher"},
		Categories: []string{"General"},
	})
}
