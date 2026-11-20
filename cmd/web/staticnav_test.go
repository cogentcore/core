// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"testing"

	"cogentcore.org/core/content/bcontent"
	"github.com/stretchr/testify/assert"
)

// TestStaticNavPages tests that the static navigation lists the pages,
// as "Pages", when they have no categories (all in "Other"), and the
// categories otherwise.
func TestStaticNavPages(t *testing.T) {
	prp := &bcontent.PreRenderPage{}
	pages := []*bcontent.NavPage{{Name: "Home", URL: ""}, {Name: "Cogent Lists", URL: "docs/lists"}}
	flat := &bcontent.Site{Categories: []*bcontent.Category{{Name: "Other", Pages: pages}}}
	nav := staticNavHTML(flat, prp, "../")
	assert.Contains(t, nav, "<b>Pages</b>")
	assert.NotContains(t, nav, "Categories")
	assert.NotContains(t, nav, "Other")
	assert.Contains(t, nav, `>Cogent Lists</a>`)

	cats := &bcontent.Site{Categories: []*bcontent.Category{{Name: "Widgets", Pages: pages[1:]}, {Name: "Other", Pages: pages[:1]}}}
	nav = staticNavHTML(cats, prp, "../")
	assert.Contains(t, nav, "<b>Categories</b>")
	assert.Contains(t, nav, "Widgets")
	assert.Contains(t, nav, "Other")
}
