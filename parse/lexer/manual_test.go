// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lexer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMarkupPathsAsLinks(t *testing.T) {
	flds := []string{
		"./path/file.go",
		"/absolute/path/file.go",
		"../relative/path/file.go",
		"file.go",
	}

	orig, link := MarkupPathsAsLinks(flds, 3)

	expectedOrig := []byte("./path/file.go")
	expectedLink := []byte(`<a href="file:///./path/file.go">./path/file.go</a>`)

	assert.Equal(t, expectedOrig, orig)
	assert.Equal(t, expectedLink, link)
}
