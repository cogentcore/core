// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package content

import (
	"strings"
	"testing"
	"testing/fstest"

	"cogentcore.org/core/core"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWideWidgets tests that the paragraphs of a page, also in lists, are at
// most maxTextWidth wide, and that code and the widget of an element handler
// (as for a simulation) can be wider.
func TestWideWidgets(t *testing.T) {
	text := strings.Repeat("A long paragraph of text, which wraps. ", 40)
	src := fstest.MapFS{
		// a simulation is on its own line, as in the simulation pages of
		// compcogneuro, which do not close its element (the rest is in it)
		"home.md": {Data: []byte("+++\nURL = \"\"\n+++\n\n" + text + "\n\n- " + text + "\n\n```go\nfmt.Println(\"" + strings.Repeat("wide code ", 30) + "\")\n```\n\n<wide-sim>\n\n<div>\n\nNot shown.\n\n</div>\n")},
	}
	b := core.NewBody()
	b.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(1800), units.Dp(800))
		s.Max = s.Min
	})
	ct := NewContent(b)
	var sim *core.Frame
	ct.Context.ElementHandlers["wide-sim"] = func(ctx *htmlcore.Context) bool {
		sim = core.NewFrame(ctx.BlockParent)
		sim.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 0)
			s.Min.Y.Dp(100)
		})
		return true
	}
	ct.SetSource(src)
	core.AssertRender(t, b, "wide-widgets", func() {
		ct.AsyncLock()
		defer ct.AsyncUnlock()
		require.NotNil(t, sim)
		uc := &ct.Styles.UnitContext
		maxw := uc.Dp(maxTextWidth*96) + 1 // in dots, rounded up
		var para, code float32
		paras := 0
		ct.rightFrame.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
			if tx, ok := cw.(*core.Text); ok && strings.HasPrefix(tx.Text, "A long paragraph") {
				para = max(para, cwb.Geom.Size.Actual.Total.X)
				paras++
			}
			if cwb.Property("paginate-block") == true {
				code = cwb.Geom.Size.Actual.Total.X
			}
			return true
		})
		assert.Equal(t, 2, paras)
		assert.LessOrEqual(t, para, maxw, "the paragraphs")
		assert.Greater(t, code, maxw+100, "the code, wider than the text")
		assert.Greater(t, sim.Geom.Size.Actual.Total.X, maxw+100, "the simulation, wider than the text")
	})
}
