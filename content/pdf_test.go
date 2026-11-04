// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package content

import (
	"strings"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/paginate"
	"cogentcore.org/core/text/rich"
	"github.com/stretchr/testify/assert"
)

// TestPagePDF tests that making a PDF of a page leaves the page as it was:
// the frame of the page is not sized for the PDF afterwards, which made
// it scroll in the frame that scrolls it (a second vertical scrollbar).
func TestPagePDF(t *testing.T) {
	b := core.NewBody()
	b.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(1000), units.Dp(700))
		s.Max = s.Min
	})
	ct := NewContent(b).SetSource(exampleContent)
	ct.Open("button")
	b.RenderImage(func() { // the snackbars of the PDF are shown at varying times
		b.AsyncLock()
		defer b.AsyncUnlock()
		pf := ct.pageFrame()
		nfinal := len(pf.Stylers.Final)
		scrolls := func() []string {
			var s []string
			ct.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
				if fr := core.AsFrame(cw); fr != nil && fr.HasScroll[math32.Y] {
					s = append(s, cwb.Name)
				}
				return true
			})
			return s
		}
		before := scrolls()
		assert.NoError(t, ct.PagePDF(t.TempDir()))
		ct.Scene.LayoutRenderScene()
		assert.Equal(t, nfinal, len(pf.Stylers.Final), "no stylers are left on the page frame")
		assert.Equal(t, before, scrolls(), "the same frames scroll")
	})
}

// TestTextHTMLTag tests that the texts of headings in the browser style of
// htmlcore, which are in the widgets of their elements, have the tags of
// those elements, which the PDF uses for its styles of headings.
func TestTextHTMLTag(t *testing.T) {
	b := core.NewBody()
	assert.NoError(t, htmlcore.ReadMDString(htmlcore.NewContext(), b, "# Title\n\n## Sub `code` and [a link](https://example.com)\n\nText"))
	b.StyleTree()
	tags := map[string]string{}
	b.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
		if tx, ok := cw.(*core.Text); ok {
			tags[tx.Text] = tx.HTMLTag()
			if tx.HTMLTag() == "h1" {
				paginate.APAHeaders(tx)
				assert.Equal(t, rich.Bold, tx.Styles.Font.Weight)
			}
		}
		return true
	})
	assert.Equal(t, "h1", tags["Title"])
	assert.Equal(t, "p", tags["Text"])
	found := false
	for text, tag := range tags {
		if strings.HasPrefix(text, "Sub") {
			found = true
			assert.Equal(t, "h2", tag, text)
		}
	}
	assert.True(t, found, "the text of the h2: %v", tags)
}
