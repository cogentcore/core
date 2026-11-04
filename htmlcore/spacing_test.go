// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
)

// TestBrowserStyleSpacing tests that the spacing of CSS in browser style
// is not scaled by the Spacing setting.
func TestBrowserStyleSpacing(t *testing.T) {
	b := core.NewBody()
	ctx := NewContext()
	ctx.BrowserStyle = true
	ReadHTMLString(ctx, b, `<div style="padding: 10px 20px; margin-left: 30px; gap: 4px; display: flex">x</div>`)
	var div *core.Frame
	b.RenderImage(func() {
		core.AppearanceSettings.Spacing = 30
		b.Scene.StyleTree()
		b.Scene.NeedsLayout()
	}, func() {
		b.WalkDown(func(n tree.Node) bool {
			if fr, ok := n.(*core.Frame); ok && fr.Property("tag") == "div" {
				div = fr
			}
			return tree.Continue
		})
		core.AppearanceSettings.Spacing = 100
	})
	if assert.NotNil(t, div) {
		dp := div.Styles.UnitContext.Dots(units.UnitDp)
		s := &div.Styles
		assert.InDelta(t, 10*dp, s.Padding.Top.Dots, 0.1)
		assert.InDelta(t, 20*dp, s.Padding.Left.Dots, 0.1)
		assert.InDelta(t, 30*dp, s.Margin.Left.Dots, 0.1)
		assert.InDelta(t, 4*dp, s.Gap.X.Dots, 0.1)
	}
}
