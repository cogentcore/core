// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGoDocWikilink(t *testing.T) {
	h := GoDocWikilink("doc", "cogentcore.org/core")
	url, label := h("doc:htmlcore")
	assert.Equal(t, "https://pkg.go.dev/cogentcore.org/core/htmlcore", url)
	assert.Equal(t, "htmlcore", label)
	url, label = h("doc:core.Button")
	assert.Equal(t, "https://pkg.go.dev/cogentcore.org/core/core#Button", url)
	assert.Equal(t, "core.Button", label)
	url, label = h("doc:core.Button.Text")
	assert.Equal(t, "https://pkg.go.dev/cogentcore.org/core/core#Button.Text", url)
	assert.Equal(t, "core.Button.Text", label)
	url, label = h("core.Button")
	assert.Equal(t, "", url)
	assert.Equal(t, "", label)

	h = GoDocWikilink("go", "github.com/some/package")
	url, label = h("go:my.Symbol")
	assert.Equal(t, "https://pkg.go.dev/github.com/some/package/my#Symbol", url)
	assert.Equal(t, "my.Symbol", label)
	url, label = h("go my.Symbol")
	assert.Equal(t, "", url)
	assert.Equal(t, "", label)
}
