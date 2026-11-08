// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"image"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
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

// TestBoxShadowColor tests that a CSS box shadow has its color (as the
// line under the toolbar of Wikipedia, a shadow of 1px without blur, which
// was half as opaque as in core, so very faint).
func TestBoxShadowColor(t *testing.T) {
	bd := core.NewBody()
	require.NoError(t, ReadHTMLString(NewContext(), bd, `<div style="padding: 20px"><div id="bar" style="width: 100px; height: 20px; box-shadow: 0 4px 0 0 rgb(200, 0, 0)"></div></div>`))
	var bar *core.WidgetBase
	bd.WalkDown(func(k tree.Node) bool {
		if wb := core.AsWidget(k); wb != nil && wb.Property("id") == "bar" {
			bar = wb
		}
		return tree.Continue
	})
	var bb image.Rectangle
	img := bd.RenderImage(func() {}, func() { bb = bar.Geom.TotalBBox })
	r, g, b, _ := img.At(bb.Min.X+10, bb.Max.Y+1).RGBA()
	assert.Equal(t, [3]uint32{200, 0, 0}, [3]uint32{r >> 8, g >> 8, b >> 8}, "below the box, in its color")
}

// TestPseudoBox tests ::before and ::after that are boxes without text (as
// the line under the title of Wikipedia, an absolute ::after of 1px at the
// bottom of the title bar, and icons with masks): they are rendered with
// their background, position, size, and mask.
func TestPseudoBox(t *testing.T) {
	icon := `url('data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 20 20"><rect width="20" height="20" fill="%23000"/></svg>')`
	bd := core.NewBody()
	require.NoError(t, ReadHTMLString(NewContext(), bd, `<style>.bar { position: relative; width: 200px; height: 40px } .bar::after { content: ""; position: absolute; left: 0; right: 0; bottom: 0; height: 2px; background-color: rgb(162, 169, 177) }`+
		`.icon::before { content: ""; display: inline-block; width: 20px; height: 20px; background-color: rgb(0, 0, 200); mask-image: `+icon+` } .text::after { content: "text" }</style>`+
		`<div class="bar" id="bar">Title</div><p class="icon" id="icon">x</p><p class="text">plain </p>`))
	var bar, ic *core.WidgetBase
	texts := 0
	bd.WalkDown(func(k tree.Node) bool {
		if wb := core.AsWidget(k); wb != nil {
			switch wb.Property("id") {
			case "bar":
				bar = wb
			case "icon":
				ic = wb
			}
		}
		if tx, ok := k.(*core.Text); ok && tx.Text == "plain text" {
			texts++
		}
		return tree.Continue
	})
	require.NotNil(t, bar)
	require.NotNil(t, ic)
	var bb, ib image.Rectangle
	img := bd.RenderImage(func() {}, func() { bb, ib = bar.Geom.TotalBBox, ic.Geom.TotalBBox })
	r, g, b, _ := img.At(bb.Min.X+50, bb.Max.Y-1).RGBA()
	assert.Equal(t, [3]uint32{162, 169, 177}, [3]uint32{r >> 8, g >> 8, b >> 8}, "the line at the bottom")
	r, g, b, _ = img.At(ib.Min.X+5, ib.Min.Y+(ib.Dy()/2)).RGBA()
	assert.Equal(t, [3]uint32{0, 0, 200}, [3]uint32{r >> 8, g >> 8, b >> 8}, "the icon")
	assert.Equal(t, 1, texts, "the text of a pseudo-element with text is in the text of its element")
}

// TestBackgroundImage tests background images (as the browser icons of
// MDN and the chevrons of go.dev): an image placed at an offset (as in a
// sprite) and repeated on one axis, and its url() in a linked style sheet,
// which is relative to the style sheet.
func TestBackgroundImage(t *testing.T) {
	// a 4x4 image: red on its top half, blue on its bottom half
	img := `<svg xmlns="http://www.w3.org/2000/svg" width="4" height="4"><rect width="4" height="2" fill="red"/><rect y="2" width="4" height="2" fill="blue"/></svg>`
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "css"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "css", "icon.svg"), []byte(img), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "css", "style.css"), []byte(`#sprite { background: url(icon.svg) 0 -2px repeat-x }`), 0644))
	ctx := NewContext()
	ctx.PageURL = "file://" + dir + "/page.html"
	ctx.GetURL = func(u string) (*http.Response, error) {
		pu, _ := url.Parse(u)
		b, err := os.ReadFile(pu.Path)
		if err != nil {
			return &http.Response{StatusCode: 404, Status: "404", Body: io.NopCloser(strings.NewReader(""))}, nil
		}
		return &http.Response{StatusCode: 200, Status: "200 OK", Request: &http.Request{URL: pu}, Body: io.NopCloser(strings.NewReader(string(b)))}, nil
	}
	bd := core.NewBody()
	require.NoError(t, ReadHTMLString(ctx, bd, `<link rel="stylesheet" href="css/style.css"><div id="sprite" style="width: 40px; height: 20px"></div>`))
	var sp *core.WidgetBase
	bd.WalkDown(func(k tree.Node) bool {
		if wb := core.AsWidget(k); wb != nil && wb.Property("id") == "sprite" {
			sp = wb
		}
		return tree.Continue
	})
	require.NotNil(t, sp)
	var bb image.Rectangle
	rendered := bd.RenderImage(func() {}, func() { bb = sp.Geom.TotalBBox })
	at := func(x, y int) [3]uint32 {
		r, g, b, _ := rendered.At(bb.Min.X+x, bb.Min.Y+y).RGBA()
		return [3]uint32{r >> 8, g >> 8, b >> 8}
	}
	dp := int(sp.Styles.UnitContext.Dots(units.UnitDp) + 0.5)
	assert.Equal(t, [3]uint32{0, 0, 255}, at(1, 0), "the bottom half of the image, moved up by the offset")
	assert.Equal(t, [3]uint32{0, 0, 255}, at(30*dp, 0), "repeated on x")
	assert.Equal(t, [3]uint32{255, 255, 255}, at(1, 10*dp), "not repeated on y")
}

// TestTableFixedLayout tests a table with the fixed layout and col widths
// (as the diffs of Wikipedia): the cells are in their columns (the
// colgroup is not a cell), with the widths of the columns, and the long
// lines in them do not make them wider; the marker text of an empty cell,
// from its ::before, is shown.
func TestTableFixedLayout(t *testing.T) {
	long := strings.Repeat("word ", 200)
	widgetsByID(t, `<style>.m::before { content: attr(data-marker) }</style><div style="width: 500px"><table id="t" style="table-layout: fixed; width: 100%"><colgroup><col style="width: 2%"> <col style="width: 48%"> <col style="width: 2%"> <col style="width: 48%"></colgroup>`+
		`<tr><td colspan="2" id="ot">old</td><td colspan="2" id="nt">new</td></tr>`+
		`<tr><td class="m" id="m" data-marker="−"></td><td id="a">`+long+`</td><td class="m" data-marker="+"></td><td id="b">b</td></tr></table></div>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			tw := ws["t"].Geom.Size.Actual.Total.X
			assert.InDelta(t, 500*dp, tw, float64(4*dp), "the width of the table")
			assert.InDelta(t, 0.48*tw, ws["a"].Geom.Size.Actual.Total.X, float64(8*dp), "48% (the long line wraps), less the spacing")
			assert.Less(t, ws["m"].Geom.Size.Actual.Total.X, 0.05*tw, "2%")
			assert.InDelta(t, ws["nt"].Geom.Pos.Total.X, ws["m"].Geom.Pos.Total.X+0.5*tw, float64(6*dp), "the titles span two columns each")
			text := ""
			ws["m"].WalkDown(func(k tree.Node) bool {
				if tx, ok := k.(*core.Text); ok {
					text += tx.Text
				}
				return tree.Continue
			})
			assert.Contains(t, text, "−", "the marker of the empty cell")
		})
}

// TestUniqueNames tests that the names of the children of each widget are
// unique, as tree updates require (the inspector crashed with a duplicate
// name of the fillers of a grid, and an id can be on more than one
// element of a page).
func TestUniqueNames(t *testing.T) {
	bd := core.NewBody()
	require.NoError(t, ReadHTMLString(NewContext(), bd, `<div style="display: grid; grid-template-areas: 'a a b' 'c . .'"><div style="grid-area: a">a</div><div style="grid-area: c">c</div></div>`+
		`<p id="dup">one</p><p id="dup">two</p><style>.e::before { content: "x" }</style><div><span class="e"></span><span class="e"></span></div>`))
	bd.WalkDown(func(k tree.Node) bool {
		names := map[string]bool{}
		for _, c := range k.AsTree().Children {
			name := c.AsTree().Name
			assert.False(t, names[name], "duplicate name %q in %s", name, k.AsTree().Path())
			names[name] = true
		}
		return tree.Continue
	})
}
