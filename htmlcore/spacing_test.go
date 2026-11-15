// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBrowserStyleSpacing tests that the spacing of CSS in browser style
// is not scaled by the Spacing setting.
func TestBrowserStyleSpacing(t *testing.T) {
	b := core.NewBody()
	ctx := NewContext()
	ReadHTMLString(ctx, b, `<div style="padding: 10px 20px; margin-left: 30px; gap: 4px; display: flex">x</div>`)
	var div *core.Frame
	core.RenderImage(b, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		core.AppearanceSettings.Spacing = 30
		b.Scene.StyleTree()
		b.Scene.NeedsLayout()
	}, core.Locked(b, func() {
		b.WalkDown(func(n tree.Node) bool {
			if fr, ok := n.(*core.Frame); ok && fr.Property("tag") == "div" {
				div = fr
			}
			return tree.Continue
		})
		core.AppearanceSettings.Spacing = 100
	}))
	if assert.NotNil(t, div) {
		dp := div.Styles.UnitContext.Dots(units.UnitDp)
		s := &div.Styles
		assert.InDelta(t, 10*dp, s.Padding.Top.Dots, 0.1)
		assert.InDelta(t, 20*dp, s.Padding.Left.Dots, 0.1)
		assert.InDelta(t, 30*dp, s.Margin.Left.Dots, 0.1)
		assert.InDelta(t, 4*dp, s.Gap.X.Dots, 0.1)
	}
}

// TestPageSpacing tests that the layout of a page does not depend on the
// Spacing setting: the spacing of its images, svgs and form controls, as
// that of its frames and text (only that of the window around it does).
func TestPageSpacing(t *testing.T) {
	const page = `<html><head><style>
.flex { display: flex; gap: 10px; padding: 8px }
.grid { display: grid; grid-template-columns: 1fr 200px; column-gap: 20px; row-gap: 5px }
img { margin: 6px; padding: 4px; width: 40px; height: 30px }
button { padding: 5px 9px; margin: 3px }
input { padding: 4px; margin: 2px }
span.p { padding: 0 12px; margin: 0 7px }
</style></head><body>
<div id="d1" class="flex"><div id="a" style="padding:10px">a</div><div id="b">b</div><img id="im" src="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="></div>
<div id="g" class="grid"><div id="g1">one</div><div id="g2">two</div><div id="g3">three</div></div>
<p id="pp">text <span id="sp" class="p">pad</span> more <button id="bt">Btn</button> <input id="in" value="x"></p>
<ul id="ul"><li id="li1">item</li><li id="li2">item2</li></ul>
<table id="tb"><tr><td id="td1" style="padding:6px">c1</td><td id="td2">c2</td></tr></table>
<svg id="sv" width="20" height="20" style="margin:5px"><rect width="20" height="20"/></svg>
<div id="end">end</div>
</body></html>`
	layout := func(spacing float32) map[string]string {
		old := core.AppearanceSettings.Spacing
		core.AppearanceSettings.Spacing = spacing
		defer func() { core.AppearanceSettings.Spacing = old }()
		b := core.NewBody()
		b.Scene.Styler(func(s *styles.Style) { s.Padding.Zero() }) // (that of the window)
		require.NoError(t, ReadHTMLString(NewContext(), b, page))
		var m map[string]string
		core.RenderImage(b, func() {}, core.Locked(b, func() { m = widgetStates(b) }))
		return m
	}
	assert.Equal(t, layout(100), layout(40))
}
