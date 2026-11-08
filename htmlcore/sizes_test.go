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
	"github.com/stretchr/testify/require"
)

// widgetsByID reads the given HTML and returns the widgets of its elements
// with ids, by id, after it is rendered.
func widgetsByID(t *testing.T, src string, fun func(ws map[string]*core.WidgetBase, dp float32)) {
	bd := core.NewBody()
	require.NoError(t, ReadHTMLString(NewContext(), bd, src))
	ws := map[string]*core.WidgetBase{}
	bd.WalkDown(func(k tree.Node) bool {
		if wb := core.AsWidget(k); wb != nil {
			if id, ok := wb.Property("id").(string); ok {
				ws[id] = wb
			}
		}
		return tree.Continue
	})
	bd.RenderImage(func() {}, func() {
		fun(ws, bd.Styles.UnitContext.Dots(units.UnitDp))
	})
}

// TestTableContentWidth tests that a table (with the automatic layout) is
// at least as wide as its cells need to be, over its width (as the
// infoboxes of Wikipedia with photos wider than them), and that one with
// the fixed layout keeps its width.
func TestTableContentWidth(t *testing.T) {
	widgetsByID(t, `<table id="auto" style="width: 100px"><tr><td><div style="width: 150px; height: 10px"></div></td><td><div style="width: 150px; height: 10px"></div></td></tr></table>`+
		`<table id="narrow" style="width: 100px"><tr><td>a</td></tr></table>`+
		`<table id="fixed" style="width: 100px; table-layout: fixed"><tr><td><div style="width: 150px; height: 10px"></div></td></tr></table>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			assert.GreaterOrEqual(t, ws["auto"].Geom.Size.Actual.Total.X, 300*dp, "as wide as its cells")
			assert.InDelta(t, 100*dp, ws["narrow"].Geom.Size.Actual.Total.X, 8, "its width (with its spacing), for narrower cells")
			assert.InDelta(t, 100*dp, ws["fixed"].Geom.Size.Actual.Total.X, 8, "the fixed layout keeps its width")
		})
}

// TestWidthMaxWidth tests a fixed width with a max-width: a percentage
// max-width (as max-width: 100%) does not make the box fill its parent
// (as the percentage bars of Wikipedia did), and a smaller fixed one
// applies.
func TestWidthMaxWidth(t *testing.T) {
	widgetsByID(t, `<div style="width: 400px"><div id="bar" style="width: 100px; max-width: 100%; height: 10px; background: green"></div>`+
		`<div id="smaller" style="width: 100px; max-width: 60px; height: 10px; background: green"></div></div>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			assert.InDelta(t, 100*dp, ws["bar"].Geom.Size.Actual.Total.X, 2)
			assert.InDelta(t, 60*dp, ws["smaller"].Geom.Size.Actual.Total.X, 2)
		})
}

// TestVisibilityHidden tests that an element with visibility: hidden (as
// the closed menus of Wikipedia, which are over the page) and the elements
// in it are not rendered and get no events (not even a box that scrolls),
// and keep their space.
func TestVisibilityHidden(t *testing.T) {
	widgetsByID(t, `<div id="menu" style="visibility: hidden; overflow: auto; height: 50px"><p id="item"><a href="/a">Recent changes</a></p></div><p id="after">after</p>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			for _, id := range []string{"menu", "item"} {
				assert.Zero(t, ws[id].Styles.Opacity, id)
				assert.Zero(t, ws[id].Styles.Abilities, id)
			}
			assert.Greater(t, ws["after"].Geom.Pos.Total.Y, ws["menu"].Geom.Pos.Total.Y+40*dp, "it keeps its space")
		})
}
