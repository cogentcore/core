// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"strings"
	"testing"

	"cogentcore.org/core/styles"

	"github.com/stretchr/testify/assert"
)

func TestToHTML(t *testing.T) {
	b := NewBody()
	NewButton(b).SetText("Hello!")
	h, err := ToHTML(b)
	assert.NoError(t, err)
	fmt.Println(string(h))
}

func TestToHTMLTable(t *testing.T) {
	b := NewBody()
	tb := NewFrame(b)
	tb.SetProperty("tag", "table")
	tb.Styler(func(s *styles.Style) {
		s.Display = styles.Grid
		s.Columns = 2
	})
	for i, tag := range []string{"th", "th", "td", "td"} {
		NewText(tb).SetText(fmt.Sprint(i)).SetProperty("tag", tag)
	}
	h := GenerateHTMLCore(tb)
	assert.Equal(t, 2, strings.Count(h, "<tr>"))
	assert.Equal(t, 2, strings.Count(h, "</tr>"))
	assert.True(t, strings.HasPrefix(h, "<table><tr><th "), h)
}

func TestToHTMLProperties(t *testing.T) {
	b := NewBody()
	fr := NewFrame(b)
	fr.SetProperty("class", "my-class")
	NewText(fr).SetText("caption: <p>text</p>").SetProperty("tag", "p")
	NewText(fr).SetText("heading").SetProperty("tag", "h2")
	NewText(fr).SetText("custom").SetProperty("html", "<pre>custom</pre>")
	h := GenerateHTMLCore(fr)
	assert.Contains(t, h, `class="my-class"`)
	assert.Contains(t, h, ">caption: text</p>")
	assert.Contains(t, h, `<h2 id="text-1"`)
	assert.Contains(t, h, "<pre>custom</pre>")
	assert.NotContains(t, h, `id="text-0"`) // automatic names are not ids
	assert.NotContains(t, h, ">custom</p>")
}
