// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
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

// TestTableShortRows tests that a row of a table with fewer cells than the
// table has columns leaves the rest of its row empty (as the header of the
// parties of the major debates table of Wikipedia), so that the cells of
// the next row are in the next row, also below cells with row spans.
func TestTableShortRows(t *testing.T) {
	widgetsByID(t, `<table><tr><th rowspan="3" id="date">Date</th><th colspan="3">Participants</th></tr>`+
		`<tr><th id="p1">A</th><th id="p2">B</th></tr><tr><th id="s1">a</th><th id="s2">b</th></tr>`+
		`<tr><td>1</td><td id="d1">x</td><td>y</td><td>z</td></tr></table>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			cell := func(id string) image.Point { return ws[id].Geom.Cell }
			assert.Equal(t, image.Pt(1, 1), cell("p1"))
			assert.Equal(t, image.Pt(2, 1), cell("p2"))
			assert.Equal(t, image.Pt(1, 2), cell("s1"), "in its own row, beside the row span")
			assert.Equal(t, image.Pt(1, 3), cell("d1"))
		})
}

// TestWrapRowInBlock tests that a flex row that wraps, in a block in a flex
// item (as the social icons in the footer of github.com), is as wide as its
// content on one line when there is room (it was a column of one icon).
func TestWrapRowInBlock(t *testing.T) {
	icon := `<li><a href="#" style="display: inline-flex; width: 24px; height: 24px"></a></li>`
	widgetsByID(t, `<div style="display: flex; justify-content: space-between; width: 800px"><nav>legal links</nav>`+
		`<div style="display: flex; flex-wrap: wrap; gap: 32px"><nav><ul id="icons" style="display: flex; flex-wrap: wrap; gap: 16px; list-style: none; margin: 0; padding: 0">`+
		strings.Repeat(icon, 5)+`</ul></nav><button>English</button></div></div>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			ic := ws["icons"].Geom.Size.Actual.Total
			assert.Greater(t, ic.X, 5*24*dp, "the icons in one row")
			assert.Less(t, ic.Y, 30*dp)
		})
}

// TestTableSpanFixedWidth tests that a cell that spans columns, with a fixed
// width, in a table narrower than its content (as the party column of a table
// of donations on Wikipedia) shrinks with its columns, as in Chrome: its row
// was wider than the others, which made the page wider than its window.
func TestTableSpanFixedWidth(t *testing.T) {
	notes := strings.Repeat("Notes about the donations of the party. ", 20)
	widgetsByID(t, `<div style="width: 500px"><table id="table"><tr><th id="party" colspan="2" style="width:150px">Party</th><th id="amount">Amount</th><th id="notes">Notes</th></tr>`+
		`<tr><td style="width: 2px; background-color: blue"></td><td id="name">New Zealand National Party</td><td>$1</td><td id="note">`+notes+`</td></tr></table></div>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			right := func(id string) float32 {
				g := ws[id].Geom
				return g.Pos.Total.X + g.Size.Actual.Total.X
			}
			assert.InDelta(t, right("name"), right("party"), 1, "the party cell over its columns")
			assert.InDelta(t, right("note"), right("notes"), 1)
			assert.LessOrEqual(t, ws["table"].Geom.Size.Actual.Total.X, 500*dp+2)
		})
}

// TestMaxWidthNarrower tests that a block with a max-width that is wider
// than its parent, and long text in it, is as wide as its parent (as the
// page container of Wikipedia, which was its max-width, wider than the
// window, after the window was widened).
func TestMaxWidthNarrower(t *testing.T) {
	text := strings.Repeat("The text of the footer of the page. ", 40)
	widgetsByID(t, `<div id="outer" style="width: 800px"><div id="container" style="max-width: 1000px; margin: 0 auto; padding: 0 24px; box-sizing: border-box">`+
		`<div><footer><ul><li id="text">`+text+`</li></ul></footer></div></div></div>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			assert.InDelta(t, 800*dp, ws["container"].Geom.Size.Actual.Total.X, 1)
			assert.LessOrEqual(t, ws["text"].Geom.Size.Actual.Total.X, 800*dp)
		})
	// in a grid with areas, with the text in an area over both columns
	widgetsByID(t, `<style>.inner { display: grid; column-gap: 24px; grid-template: min-content 1fr min-content / 196px minmax(0, 1fr); `+
		`grid-template-areas: "notice notice" "start content" "footer footer" }</style>`+
		`<div id="outer" style="width: 800px"><div id="container" style="max-width: 1000px; margin: 0 auto; padding: 0 24px; box-sizing: border-box">`+
		`<div class="inner"><div style="grid-area: notice">notice</div><div style="grid-area: start">start</div><main style="grid-area: content">content</main>`+
		`<div style="grid-area: footer"><footer><ul><li id="text">`+text+`</li></ul></footer></div></div></div></div>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			assert.InDelta(t, 800*dp, ws["container"].Geom.Size.Actual.Total.X, 2)
			assert.LessOrEqual(t, ws["text"].Geom.Size.Actual.Total.X, 800*dp)
		})
}

// TestScrollContentWidth tests that a box that scrolls in width, in a flex
// row, is as wide as its content when there is room (as the scrolling list
// of the chapters of the video of github.com, which was not shown), and
// narrower, scrolling its content, when there is not.
func TestScrollContentWidth(t *testing.T) {
	item := `<li style="flex-shrink: 0; display: flex"><button style="padding: 8px 16px"><span style="display: block">Review changes</span></button></li>`
	page := func(width string) string {
		return `<div style="display: flex; justify-content: center; gap: 8px; width: ` + width + `">` +
			`<div id="chapters" style="min-width: 0"><div id="viewport" style="overflow-x: auto"><ul id="list" style="gap: 12px; width: max-content; margin: 0 auto; padding: 0; list-style: none; display: flex">` +
			item + item + item + `</ul></div></div><button>play</button></div>`
	}
	widgetsByID(t, page("1000px"), func(ws map[string]*core.WidgetBase, dp float32) {
		list := ws["list"].Geom.Size.Actual.Total.X
		assert.Greater(t, list, 300*dp)
		assert.InDelta(t, list, ws["chapters"].Geom.Size.Actual.Total.X, 1, "as wide as its content")
	})
	widgetsByID(t, page("200px"), func(ws map[string]*core.WidgetBase, dp float32) {
		assert.Less(t, ws["chapters"].Geom.Size.Actual.Total.X, 200*dp, "narrowed to fit")
		assert.True(t, core.AsFrame(ws["viewport"].This).HasScroll[math32.X], "its content scrolls")
	})
	// a button that hides its overflow, in a flex item, is as wide as its
	// label (as the chapter buttons of github.com, which were empty pills)
	widgetsByID(t, `<div style="display: flex; width: 600px"><button id="button" style="padding: 8px 16px; overflow: hidden"><span id="label" style="display: block">Review changes</span></button></div>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			assert.Greater(t, ws["button"].Geom.Size.Actual.Total.X, ws["label"].Geom.Size.Actual.Total.X+30*dp)
		})
}

// TestWidthMaxContent tests a width of max-content (as the track of the
// ticker of activity of github.com, a grid of items in one row in a box
// that hides its overflow): it is as wide as its content on one line,
// over the width of its parent, and its items are not wrapped.
func TestWidthMaxContent(t *testing.T) {
	item := `<li style="display: grid; overflow-wrap: anywhere; border-right: 1px solid gray"><a href="#" style="padding: 24px">repo forked from someone-somewhere</a></li>`
	widgetsByID(t, `<div style="width: 400px; overflow: hidden"><ul id="track" style="width: max-content; display: grid; grid-auto-flow: column; margin: 0; padding: 0; list-style: none">`+
		strings.Replace(item, "<li ", `<li id="first" `, 1)+item+item+`</ul></div>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			first := ws["first"].Geom.Size.Actual.Total
			assert.Greater(t, first.X, 200*dp, "an item on one line")
			assert.Less(t, first.Y, 80*dp)
			assert.Greater(t, ws["track"].Geom.Size.Actual.Total.X, 3*200*dp, "over the width of its parent")
		})
}

// TestTransformZeroScale tests that an element that a transform scales to
// nothing is not shown (as the progress bar over the label of the current
// chapter button of github.com, before the video plays), and that a later
// transform that does not undoes that.
func TestTransformZeroScale(t *testing.T) {
	widgetsByID(t, `<style>.bar { transform: scaleX(0) } .bar.on { transform: scaleX(1) }</style>`+
		`<div id="off" class="bar">a</div><div id="on" class="bar on">b</div><div id="both" style="transform: translateY(-50%) scale(0, 1)">c</div>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			assert.Equal(t, float32(0), ws["off"].Styles.Opacity)
			assert.Equal(t, float32(1), ws["on"].Styles.Opacity)
			assert.Equal(t, float32(0), ws["both"].Styles.Opacity)
		})
}

// TestGridStack tests a grid whose children are all in its first cell (as
// the counters of github.com: a hidden copy of the value under it, which
// keeps its width): they are over each other, not in rows.
func TestGridStack(t *testing.T) {
	widgetsByID(t, `<div style="font-size: 40px"><span id="value" style="display: inline-grid">`+
		`<span style="grid-area: 1 / 1; opacity: 0">225M+</span><span style="grid-area: 1 / 1">225M+</span></span></div>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			assert.Less(t, ws["value"].Geom.Size.Actual.Total.Y, 60*dp, "one line high")
			assert.Greater(t, ws["value"].Geom.Size.Actual.Total.X, 80*dp, "with its value shown")
		})
	// the copy that is not shown is last (as the names of the repositories
	// of github.com, which were hidden)
	widgetsByID(t, `<p id="name" style="display: grid"><span id="shown" style="grid-area: 1 / 1">hyprwm / Hyprland</span>`+
		`<span id="copy" style="grid-area: 1 / 1; opacity: 0">hyprwm / Hyprland</span></p>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			assert.Equal(t, 0, core.AsFrame(ws["name"].This).StackTop, "the copy that is shown")
		})
}

// TestAspectRatio tests the aspect-ratio property: the height of a box
// follows its width (as the visuals of the platform features of
// github.com, which were as high as their content), unless its height
// is set, and its content can make it higher.
func TestAspectRatio(t *testing.T) {
	widgetsByID(t, `<div style="width: 400px"><div id="box" style="aspect-ratio: 16 / 15; position: relative"><div id="child" style="height: 100%"></div></div>`+
		`<div id="set" style="aspect-ratio: 2; height: 50px"></div><div id="tall" style="aspect-ratio: 4"><div style="height: 300px"></div></div>`+
		`<div id="border" style="aspect-ratio: 1; box-sizing: border-box; padding: 20px; width: 100px"></div></div>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			box := ws["box"].Geom.Size.Actual.Total
			assert.InDelta(t, 400*dp, box.X, 1)
			assert.InDelta(t, 375*dp, box.Y, 2, "from its width")
			assert.InDelta(t, 375*dp, ws["child"].Geom.Size.Actual.Total.Y, 2, "a percentage of that height")
			assert.InDelta(t, 50*dp, ws["set"].Geom.Size.Actual.Total.Y, 1, "its height")
			assert.GreaterOrEqual(t, ws["tall"].Geom.Size.Actual.Total.Y, 300*dp, "its content")
			b := ws["border"].Geom.Size.Actual.Total
			assert.InDelta(t, b.X, b.Y, 1, "of its border box")
		})
}

// TestContainerUnits tests the container query units (cqw, cqh), of the
// size of the nearest container (container-type: size or inline-size),
// as the screenshots of the platform features of github.com, whose width
// is a min() of calc()s of the width and height of their container,
// which were empty boxes.
func TestContainerUnits(t *testing.T) {
	widgetsByID(t, `<div style="width: 400px"><div id="container" style="container-type: size; aspect-ratio: 16 / 15; position: relative">`+
		`<div style="display: flex; padding: 10px"><div id="half" style="width: 50cqw; height: 10px"></div></div>`+
		`<div id="content" style="inline-size: min(calc(100cqw - 32px), calc((100cqh - 64px) * 984 / 840)); aspect-ratio: 984 / 840"></div></div>`+
		`<div style="container-type: inline-size; width: 200px"><div id="inline" style="width: 50cqw; height: 50cqh"></div></div></div>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			assert.InDelta(t, 375*dp, ws["container"].Geom.Size.Actual.Total.Y, 2)
			assert.InDelta(t, 200*dp, ws["half"].Geom.Size.Actual.Total.X, 2, "50% of the width of its container, not of its parent")
			// min(400 - 32, (375 - 64) * 984 / 840 = 364.3)
			content := ws["content"].Geom.Size.Actual.Total
			assert.InDelta(t, 364.3*dp, content.X, 3, "of the height of its container")
			assert.InDelta(t, 364.3*840/984*dp, content.Y, 3)
			assert.InDelta(t, 100*dp, ws["inline"].Geom.Size.Actual.Total.X, 2)
		})
}

// TestSVGHeightAspect tests that an svg with a height and no width is as
// wide as the aspect ratio of its view box makes it (as the logos of
// github.com, which were as wide as their container).
func TestSVGHeightAspect(t *testing.T) {
	widgetsByID(t, `<div style="display: flex; width: 800px"><svg id="logo" viewBox="0 0 400 94" style="height: 32px"><rect width="400" height="94"/></svg></div>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			sz := ws["logo"].Geom.Size.Actual.Content
			assert.InDelta(t, 32*dp, sz.Y, 1)
			assert.InDelta(t, 32*400/94.0*dp, sz.X, 2)
		})
}

// TestVisibilityInherit tests that visibility: inherit overrides a
// visibility of hidden (as the repository panels of github.com, which a
// script shows with visibility: inherit, over the hidden of their class).
func TestVisibilityInherit(t *testing.T) {
	widgetsByID(t, `<style>.p { visibility: hidden }</style><div><div id="shown" class="p" style="visibility: inherit">a</div><div id="hidden" class="p">b</div></div>`+
		`<div style="visibility: hidden"><div id="inh" style="visibility: inherit">c</div></div>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			assert.Equal(t, float32(1), ws["shown"].Styles.Opacity)
			assert.Equal(t, float32(0), ws["hidden"].Styles.Opacity)
			assert.Equal(t, float32(0), ws["inh"].Styles.Opacity, "the hidden of its parent")
		})
}

// TestPseudoElementMarked tests that the widget of the box of a
// pseudo-element is marked as one (PseudoElement), as for an app that
// gives the elements their clicks, and that no selector matches it.
func TestPseudoElementMarked(t *testing.T) {
	widgetsByID(t, `<style>.s::before { content: ""; display: inline-block; width: 10px; height: 10px; background: red } [title] { background: blue }</style><div id="s" class="s">text</div>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			var kind string
			var box *core.WidgetBase
			ws["s"].WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
				if k := PseudoElement(NodeOf(cw)); k != "" {
					kind, box = k, cwb
				}
				return true
			})
			assert.Equal(t, "before", kind)
			if assert.NotNil(t, box) {
				assert.Equal(t, color.RGBA{255, 0, 0, 255}, colors.AsRGBA(colors.ToUniform(box.Styles.Background)))
			}
			assert.Equal(t, "", PseudoElement(NodeOf(ws["s"].This.(core.Widget))))
		})
}

// TestCalcPercentLength tests a width of a calc() with a percentage and a
// length (as calc(100% - 256px) of the grids of github.com, which was taken
// to be 100%), and a max-width of a min() of them, which are computed with
// the width of the parent in the layout.
func TestCalcPercentLength(t *testing.T) {
	widgetsByID(t, `<div style="width: 800px"><div id="calc" style="width: calc(100% - 200px); height: 10px"></div></div>`+
		`<div style="width: 400px"><div id="min" style="max-width: min(100%, 600px)">`+strings.Repeat("text that is long ", 20)+`</div></div>`,
		func(ws map[string]*core.WidgetBase, dp float32) {
			assert.InDelta(t, 600*dp, ws["calc"].Geom.Size.Actual.Total.X, 2)
			assert.LessOrEqual(t, ws["min"].Geom.Size.Actual.Total.X, 400*dp+1)
		})
}

// TestGridPercentTrack tests a grid column of a percentage (as the 21.5%
// navigation of pkg.go.dev), with padding, beside a column of wide content:
// the column is that percentage of the grid, which the content beside it
// does not narrow, and the padding is within it (with border-box).
func TestGridPercentTrack(t *testing.T) {
	src := `<div id="grid" style="display: grid; grid-template: min-content / 20% minmax(0, auto); grid-template-areas: 'nav article'; width: 500px">` +
		`<nav id="nav" style="grid-area: nav; padding-left: 24px; box-sizing: border-box"><a href="#">Index</a></nav>` +
		`<article style="grid-area: article; overflow: auto"><pre>` + strings.Repeat("wide content ", 30) + `</pre></article></div>`
	widgetsByID(t, src, func(ws map[string]*core.WidgetBase, dp float32) {
		assert.InDelta(t, 100*dp, ws["nav"].Geom.Size.Actual.Total.X, 2, "20% of the grid")
	})
}

// TestGridJustifyItemsStacked tests the items of a grid with justify-items:
// center that are in the same cell (as the logo of Google, and a hidden
// doodle over it): the one that is shown is as wide as its content, and
// centered in the cell.
func TestGridJustifyItemsStacked(t *testing.T) {
	src := `<div id="grid" style="display: grid; justify-items: center; width: 600px">` +
		`<div id="logo" style="grid-area: 1 / 1"><div style="width: 200px; height: 50px; background: blue"></div></div>` +
		`<div style="grid-area: 1 / 1; opacity: 0; visibility: hidden"><div style="width: 100px; height: 50px"></div></div></div>`
	widgetsByID(t, src, func(ws map[string]*core.WidgetBase, dp float32) {
		g, logo := ws["grid"].Geom.TotalBBox, ws["logo"].Geom.TotalBBox
		assert.InDelta(t, 200*dp, float32(logo.Dx()), 2, "as wide as its content")
		assert.InDelta(t, float32(g.Min.X+g.Max.X)/2, float32(logo.Min.X+logo.Max.X)/2, 2, "centered")
	})
}

// TestFlexOrder tests the order property of flex items (as the images of
// the stories of npr.org, before their titles by an order of -1).
func TestFlexOrder(t *testing.T) {
	src := `<div style="display: flex; flex-direction: column; width: 300px"><div id="title" style="height: 20px">Title</div>` +
		`<div id="image" style="order: -1; height: 50px; background: blue"></div></div>`
	widgetsByID(t, src, func(ws map[string]*core.WidgetBase, dp float32) {
		assert.Less(t, ws["image"].Geom.TotalBBox.Min.Y, ws["title"].Geom.TotalBBox.Min.Y, "the image is first")
	})
}

// TestGridRowSpanLastRow tests a grid of auto rows and a last row of 1fr,
// with an item that spans all of its rows (1 / -1) beside the others, all
// of which span columns (as the stories of npr.org, with their image
// beside their topic and title): the rows of the others stay together,
// and the extra height of the image is in the last row; the image is in
// its columns.
func TestGridRowSpanLastRow(t *testing.T) {
	src := `<div id="card" style="display: grid; grid-template-columns: repeat(12, 1fr); grid-template-rows: repeat(2, auto) 1fr; width: 600px">` +
		`<div id="topic" style="grid-column: 1 / span 8; height: 20px">Congress</div>` +
		`<div id="title" style="grid-column: 1 / span 8; height: 40px">Title</div>` +
		`<div id="image" style="grid-area: 1 / 9 / -1 / span 4; height: 200px; background: blue"></div></div>`
	widgetsByID(t, src, func(ws map[string]*core.WidgetBase, dp float32) {
		topic, title, img, card := ws["topic"].Geom.TotalBBox, ws["title"].Geom.TotalBBox, ws["image"].Geom.TotalBBox, ws["card"].Geom.TotalBBox
		assert.InDelta(t, float32(topic.Max.Y), float32(title.Min.Y), 2, "the title is below the topic")
		assert.Greater(t, img.Min.X, card.Min.X+int(390*dp), "the image is in its columns")
		assert.Equal(t, topic.Min.Y, img.Min.Y, "beside the topic")
	})
}

// TestPercentHeightChain tests heights of 100% in a box of a specified
// height, down to a text field (as the search box of duckduckgo.com): they
// are of its height, and it is not higher, where the text field was as
// high as the window.
func TestPercentHeightChain(t *testing.T) {
	src := `<form><div id="box" style="height: 48px; position: relative"><div style="height: 100%; display: flex; align-items: center">` +
		`<div style="height: 100%; display: flex; align-items: center"><input id="in" style="height: 100%; box-sizing: border-box; display: block" value="cogent core"></div></div></div></form>`
	widgetsByID(t, src, func(ws map[string]*core.WidgetBase, dp float32) {
		assert.InDelta(t, 48*dp, float32(ws["box"].Geom.TotalBBox.Dy()), 2, "the box is as high as specified")
		assert.InDelta(t, 48*dp, float32(ws["in"].Geom.TotalBBox.Dy()), 2, "the text field is as high as it")
	})
}

// TestBlockFlexFlowFloats tests a block with a flex-flow of column wrap
// (for a flex display of narrow screens, as the body of the postings of
// craigslist.org), which only applies to flex containers: its floats are
// laid out, side by side, below its centered body of 100% up to a
// max-width (which is not narrowed to its content).
func TestBlockFlexFlowFloats(t *testing.T) {
	src := `<style>figure, section, div { margin: 0; padding: 0 }
		.body { width: 100%; max-width: 600px; margin: 0 auto; box-sizing: border-box }
		.userbody { display: block; flex-flow: column wrap }
		.iw { width: 100%; max-width: 300px; float: left }
		.attrs { float: right; width: 200px }
		#text { clear: left }</style>
		<div style="width: 800px"><section class="body" id="body"><section class="userbody"><figure class="iw" id="iw"><div style="height: 100px">photo</div></figure>
		<div class="attrs" id="attrs"><p>bicycle type</p></div><section id="text">Heybike Explore</section></section></section></div>`
	widgetsByID(t, src, func(ws map[string]*core.WidgetBase, dp float32) {
		body, iw, attrs, tx := ws["body"].Geom.TotalBBox, ws["iw"].Geom.TotalBBox, ws["attrs"].Geom.TotalBBox, ws["text"].Geom.TotalBBox
		assert.InDelta(t, 600*dp, float32(body.Dx()), 2, "the body is its max-width")
		assert.InDelta(t, 300*dp, float32(iw.Dx()), 2, "the figure is its max-width")
		assert.Equal(t, iw.Min.Y, attrs.Min.Y, "the floats are side by side")
		assert.Greater(t, attrs.Min.X, iw.Max.X, "at the right")
		assert.GreaterOrEqual(t, tx.Min.Y, iw.Max.Y-1, "the text clears the figure")
	})
}

// TestWebkitTransform tests a -webkit-transform, which is the transform
// (as with which the slider of the photos of craigslist.org moves its
// slides, which were all on each other).
func TestWebkitTransform(t *testing.T) {
	widgetsByID(t, `<div id="d" style="-webkit-transform: translate(50px, 0)">slide</div>`, func(ws map[string]*core.WidgetBase, dp float32) {
		assert.InDelta(t, 50, ws["d"].Styles.Translate.X.Value, 0.1)
	})
}

// TestFloatOfBlockOfFloats tests a float whose content is a block of
// floats (as the breadcrumbs of the header of craigslist.org, a floating
// nav of a list of floating crumbs): it is as wide as its floats, side by
// side, so that the header has one line.
func TestFloatOfBlockOfFloats(t *testing.T) {
	src := `<style>* { margin: 0; padding: 0 } ul { list-style: none }
.global-header { background: #eee; height: 1.925em; line-height: 1.925em }
.header-logo { display: block; float: left; margin-top: 3px }
.breadcrumbs-container { float: left; font-size: 0.9375em; min-height: 1.3em; position: relative }
.breadcrumbs { margin-top: 4px; columns: 1 }
.crumb { float: left; line-height: 1.5em; margin: 0 5px 0 0 }
@media (orientation: landscape) { .breadcrumbs { columns: 2; margin-top: 1px } .breadcrumbs .section { break-before: column } }
@media only screen and (min-width: 840px) { .breadcrumbs { columns: 1; margin-top: 4px } }
.userlinks { float: right; font-size: 0.9em }
.user-actions { float: right; margin-left: 5px; margin-top: 5px }
.user-actions li { float: left; line-height: 1.4em; margin-right: 5px }</style>
<div style="width: 1000px"><header class="global-header" id="h"><a class="header-logo" href="/">CL</a>
<nav class="breadcrumbs-container" id="nav"><ul class="breadcrumbs" id="ul">
<li class="crumb" id="c1"><p><a href="/a">boulder</a> <span>&gt;</span></p></li>
<li class="crumb section" id="c2"><p><a href="/b">for sale by owner</a> <span>&gt;</span></p></li>
<li class="crumb" id="c3"><p><a href="/c">bicycles</a></p></li></ul></nav>
<div class="userlinks" id="ul2"><ul class="user-actions"><li><a href="/p">post</a></li><li><a href="/acc">account</a></li></ul></div></header>
<p id="after">reply</p></div>`
	ctx := NewContext()
	ctx.MediaWidth = 1290
	bd := core.NewBody()
	ReadHTMLString(ctx, bd, src)
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
		// laid out again, as a page is when its scripts change it
		bd.AsyncLock()
		bd.Scene.LayoutScene()
		bd.AsyncUnlock()
		nav, c1, c3, h := ws["nav"].Geom.TotalBBox, ws["c1"].Geom.TotalBBox, ws["c3"].Geom.TotalBBox, ws["h"].Geom.TotalBBox
		assert.Equal(t, c1.Min.Y, c3.Min.Y, "the crumbs are on one line")
		assert.GreaterOrEqual(t, nav.Max.X, c3.Max.X-1, "the nav is as wide as its crumbs")
		assert.Less(t, h.Dy(), 45, "the header is one line")
	})
}

// TestImageNaturalSizeDp tests that an image with no size is its size in
// pixels in CSS px (dp), not in dots, so that it is as large as in
// browsers on a screen with more dots per dp (as the photos of the postings
// of craigslist.org, which were half their size on retina screens).
func TestImageNaturalSizeDp(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 100, 50))))
	src := `<div style="width: 400px"><img id="i" src="data:image/png;base64,` + base64.StdEncoding.EncodeToString(buf.Bytes()) + `"></div>`
	// at a zoom of 2, as for a screen of two dots per dp
	ctx := NewContext()
	ctx.Zoom = 2
	bd := core.NewBody()
	require.NoError(t, ReadHTMLString(ctx, bd, src))
	var img *core.Image
	bd.WalkDown(func(k tree.Node) bool {
		if im, ok := k.(*core.Image); ok {
			img = im
		}
		return tree.Continue
	})
	require.NotNil(t, img)
	bd.RenderImage(func() {}, func() {
		dp := img.Styles.UnitContext.Dots(units.UnitDp)
		assert.Greater(t, dp, float32(1.5))
		assert.InDelta(t, 100*dp, img.Geom.Size.Actual.Content.X, 2, "100 pixels in dp")
	})
}

// firstElement returns the first element of the given tag in the given HTML.
func firstElement(t *testing.T, src, tag string) *html.Node {
	doc, err := html.Parse(strings.NewReader(src))
	require.NoError(t, err)
	var found *html.Node
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if found != nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == tag {
			found = n
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	require.NotNil(t, found)
	return found
}

func TestImageSrcset(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		zoom    float32
		url     string
		density float32
	}{
		{"src is 1x", `<img src="a.png" srcset="b.png 2x">`, 0, "a.png", 1},
		{"src is 1x, 2x screen", `<img src="a.png" srcset="b.png 2x">`, 2, "b.png", 2},
		{"srcset only", `<img srcset="s.png 1x, l.png 2x">`, 0, "s.png", 1},
		{"no descriptor", `<img srcset="s.png">`, 0, "s.png", 1},
		{"commas in url", `<img srcset="a.jpg?w=1,2 1x, b.jpg 2x">`, 0, "a.jpg?w=1,2", 1},
		{"widths and sizes", `<img src="f.jpg" srcset="w300.jpg 300w, w600.jpg 600w, w1200.jpg 1200w" sizes="(max-width: 500px) 100vw, 400px">`, 0, "w600.jpg", 1.5},
		{"picture", `<picture><source media="(max-width: 412px)" srcset="small.jpg"><source type="image/avif" srcset="x.avif"><source srcset="big.jpg 1x"><img src="fb.jpg"></picture>`, 0, "big.jpg", 1},
		{"picture fallback", `<picture><source media="(max-width: 412px)" srcset="small.jpg"><img src="fb.jpg"></picture>`, 0, "fb.jpg", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewContext()
			ctx.MediaWidth = 1280
			ctx.Zoom = tt.zoom
			url, density := ctx.imageSource(firstElement(t, tt.src, "img"))
			assert.Equal(t, tt.url, url)
			assert.InDelta(t, tt.density, density, 0.001)
		})
	}
}

func TestImageSrcsetDensity(t *testing.T) {
	// an image of only a srcset of 2x is shown at half its size in pixels
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 100, 50))))
	src := `<div style="width: 400px"><img srcset="data:image/png;base64,` + base64.StdEncoding.EncodeToString(buf.Bytes()) + ` 2x"></div>`
	bd := core.NewBody()
	require.NoError(t, ReadHTMLString(NewContext(), bd, src))
	var img *core.Image
	bd.WalkDown(func(k tree.Node) bool {
		if im, ok := k.(*core.Image); ok {
			img = im
		}
		return tree.Continue
	})
	require.NotNil(t, img, "an img of only a srcset is shown")
	bd.RenderImage(func() {}, func() {
		dp := img.Styles.UnitContext.Dots(units.UnitDp)
		assert.InDelta(t, 50*dp, img.Geom.Size.Actual.Content.X, 2, "100 pixels of 2x in dp")
	})
}

func TestZeroAreaClip(t *testing.T) {
	for _, tt := range []struct {
		prop, val string
		zero      bool
	}{
		{"clip", "rect(0 0 0 0)", true},
		{"clip", "rect(1px, 1px, 1px, 1px)", true},
		{"clip", "rect(0px 10px 10px 0px)", false},
		{"clip", "rect(0 auto auto 0)", false},
		{"clip", "auto", false},
		{"clip-path", "inset(50%)", true},
		{"clip-path", "inset(100% 0 0 0)", true},
		{"clip-path", "inset(10%)", false},
		{"clip-path", "circle(40%)", false},
	} {
		assert.Equal(t, tt.zero, zeroAreaClip(tt.prop, tt.val), tt.prop+": "+tt.val)
	}
}

// TestClippedOutHidden tests that an element clipped to nothing (as the
// visually hidden skip links of go.dev) is not rendered and gets no
// events, and is shown again when the clip is unset (as on its focus).
func TestClippedOutHidden(t *testing.T) {
	src := `<style>.skip { position: absolute; clip: rect(0 0 0 0) } .shown { clip: unset }</style>
<div><div id="hidden" class="skip">Skip</div></div><div><div id="shown" class="skip shown">Shown</div></div>`
	widgetsByID(t, src, func(ws map[string]*core.WidgetBase, dp float32) {
		require.NotNil(t, ws["hidden"])
		require.NotNil(t, ws["shown"])
		assert.Equal(t, float32(0), ws["hidden"].Styles.Opacity, "clipped to nothing")
		assert.Equal(t, abilities.Abilities(0), ws["hidden"].Styles.Abilities, "no events")
		assert.Equal(t, float32(1), ws["shown"].Styles.Opacity, "the clip unset")
	})
}

// TestGridLonghandsAndColumnStart tests the grid of the articles of
// nytimes.com/wirecutter: a 12 column grid (of repeat(var(--grid-cols),
// 1fr)) in the middle column of 1fr 1024px 1fr, whose children span
// columns by grid-column-end (span var(), span 8), one starting at column
// 10 by grid-column-start, in the order of their order. Their columns were
// not read, so they were all in one row, and the space of the empty third
// column of the outer grid went to the first.
func TestGridLonghandsAndColumnStart(t *testing.T) {
	src := `<div style="width: 1280px"><style>
.outer { display: grid; grid-template-columns: 1fr 1024px 1fr }
.outer > * { grid-column: 2/2 }
.inner { --grid-cols: 12; column-gap: 20px; display: grid; grid-template-columns: repeat(var(--grid-cols), 1fr); grid-column-start: 2 }
.inner > * { grid-column-end: span var(--grid-cols) }
.s3 { grid-column-end: span 3 } .c10 { grid-column-start: 10 } .s8 { grid-column-end: span 8 }
</style><div class="outer"><div id="inner" class="inner">
<div id="head">The Best Ebike for Commuters</div>
<div id="side" class="s3 c10" style="order: 3">sidebar</div>
<div id="intro" class="s8" style="order: 2">Live at the top of a steep hill? No problem.</div>
<div id="wide" style="order: 4">wide</div>
</div></div></div>`
	ctx := NewContext()
	ctx.MediaWidth = 1280
	bd := core.NewBody()
	require.NoError(t, ReadHTMLString(ctx, bd, src))
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
		dp := bd.Styles.UnitContext.Dots(units.UnitDp)
		box := func(id string) image.Rectangle { return ws[id].Geom.TotalBBox }
		inner := box("inner")
		assert.InDelta(t, 1024*dp, float32(inner.Dx()), float64(3*dp), "the middle column")
		assert.InDelta(t, float32(box("head").Dx()), float32(inner.Dx()), 2, "the head spans 12 columns")
		col := (1024 - 11*20) / 12.0 // the width of a column, in dp
		assert.InDelta(t, float32(8*col+7*20)*dp, float32(box("intro").Dx()), float64(3*dp), "the intro spans 8 columns")
		assert.Equal(t, box("intro").Min.Y, box("side").Min.Y, "the sidebar is beside the intro")
		assert.InDelta(t, float32(inner.Min.X)+float32(9*(col+20))*dp, float32(box("side").Min.X), float64(3*dp), "the sidebar starts at column 10")
		assert.Greater(t, box("wide").Min.Y, box("intro").Min.Y, "below them")
	})
}

// TestImageMaxWidthPercentCaps tests that an image with a width attribute
// and a max-width of 100% (the common responsive images) is no wider than
// its container, with the height of its aspect at that width, as the
// photos of the picks of nytimes.com/wirecutter (width="399" in boxes of
// 200px of aspect-ratio 1/1), which overflowed onto their titles.
func TestImageMaxWidthPercentCaps(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 399, 266))))
	src := `<style>img { display: block; height: auto; max-width: 100% } .box { aspect-ratio: 1/1; width: 200px }</style>
<div style="display: flex; gap: 20px; align-items: center; width: 800px"><div class="box" id="box"><a href="#x"><img id="im" src="data:image/png;base64,` + base64.StdEncoding.EncodeToString(buf.Bytes()) + `" width="399" height="266"></a></div><div id="txt">Our pick for most commuters</div></div>`
	widgetsByID(t, src, func(ws map[string]*core.WidgetBase, dp float32) {
		box, im, txt := ws["box"], ws["im"], ws["txt"]
		require.NotNil(t, im)
		assert.InDelta(t, 200*dp, im.Geom.Size.Actual.Total.X, 2, "as wide as the box")
		assert.InDelta(t, 200*dp*266/399, im.Geom.Size.Actual.Total.Y, 5, "of its aspect at that width")
		assert.InDelta(t, 200*dp, box.Geom.Size.Actual.Total.Y, 2, "the box is of its aspect ratio")
		assert.GreaterOrEqual(t, txt.Geom.TotalBBox.Min.X, box.Geom.TotalBBox.Max.X+int(20*dp)-1, "the text after the box")
	})
}

// TestNegativeMarginOverlapShown tests that a block that overlaps the one
// before it by a negative margin does not paint the background of its
// parent over it, as in the comments of news.ycombinator.com, whose
// headers (with a margin-bottom of -10px, followed by a br) were cut off.
func TestNegativeMarginOverlapShown(t *testing.T) {
	src := `<div style="background: #f6f6ef; font-size: 10pt"><div id="head" style="margin-top: 2px; margin-bottom: -10px"><span style="font-size: 8pt">rzk 4 hours ago | next</span></div><br>
<div style="font-size: 9pt">It seems to be an A/B testing experiment</div></div>`
	bd := core.NewBody()
	require.NoError(t, ReadHTMLString(NewContext(), bd, src))
	var head *core.WidgetBase
	bd.WalkDown(func(k tree.Node) bool {
		if wb := core.AsWidget(k); wb != nil && head == nil {
			if id, ok := wb.Property("id").(string); ok && id == "head" {
				head = wb
			}
		}
		return tree.Continue
	})
	var bb image.Rectangle
	img := bd.RenderImage(func() {}, func() {
		require.NotNil(t, head)
		bb = head.Geom.ContentBBox
	})
	// the text is dark in the lower half of the box of the header
	dark := 0
	for y := (bb.Min.Y + bb.Max.Y) / 2; y < bb.Max.Y; y++ {
		for x := bb.Min.X; x < bb.Max.X; x++ {
			if r, _, _, _ := img.At(x, y).RGBA(); r < 0xc000 {
				dark++
			}
		}
	}
	assert.Greater(t, dark, 20, "the lower half of the header is shown")
}

// TestPseudoPaddingBox tests that a ::before or ::after with only padding
// (and content "") is a box, as the boxes of the aspect ratios of the
// images of edition.cnn.com (of padding-bottom: 56.25%), in which the
// image is absolute, which were of no height, so the images were slits.
func TestPseudoPaddingBox(t *testing.T) {
	src := `<style>.media { display: flex; flex-direction: column; position: relative; overflow: hidden; width: 320px }
.media::after { content: ""; padding-bottom: 56.25%; display: block }
.media img { position: absolute; top: 0; left: 0; width: 100%; height: 100% }</style>
<div class="media" id="media"><picture><img src="x.png"></picture></div>`
	widgetsByID(t, src, func(ws map[string]*core.WidgetBase, dp float32) {
		require.NotNil(t, ws["media"])
		assert.InDelta(t, 180*dp, ws["media"].Geom.Size.Actual.Total.Y, 2, "of its ::after, of 56.25% of its width")
	})
}

// TestFlexBasisZero tests flex items of a basis of 0 (as flex: 3 1 0),
// which overrides their width of 100%, as the columns of edition.cnn.com
// (in a row that wraps), which were each in a row of its own: they are in
// one row, as wide as their shares of its grow.
func TestFlexBasisZero(t *testing.T) {
	src := `<style>.row { display: flex; flex-wrap: wrap; justify-content: space-between; column-gap: 24px; width: 1024px }
.row > div { width: 100% } .row > div:first-child { flex: 3 1 0 } .row > div:nth-child(2) { flex: 4 1 0 } .row > div:nth-child(3) { flex: 3 1 0 }</style>
<div class="row"><div id="a">Gunman who killed at least 6 in Kraków mass shooting has died in hospital</div><div id="b">Trump says deportation policy was too tough, and other 60 Minutes interview takeaways</div><div id="c">Catch up on today's global news</div></div>`
	widgetsByID(t, src, func(ws map[string]*core.WidgetBase, dp float32) {
		a, b, c := ws["a"], ws["b"], ws["c"]
		require.NotNil(t, c)
		w := (1024 - 48) * dp / 10
		// (they are up to 3% narrower than in Chrome)
		assert.InEpsilon(t, 3*w, a.Geom.Size.Actual.Total.X, 0.04)
		assert.InEpsilon(t, 4*w, b.Geom.Size.Actual.Total.X, 0.04)
		assert.InEpsilon(t, 3*w, c.Geom.Size.Actual.Total.X, 0.04)
		assert.Equal(t, a.Geom.TotalBBox.Min.Y, c.Geom.TotalBBox.Min.Y, "in one row")
	})
}

// TestPictureSourcesHidden tests that the source elements of a picture
// have no box, as in browsers: they were a line of text, as high as the
// pictures of edition.cnn.com (of absolute images), which clipped them.
func TestPictureSourcesHidden(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 160, 90))))
	u := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
	src := `<div style="width: 320px"><picture id="pic" style="display: block"><source media="(min-width: 1280px)" srcset="` + u + ` 1x" type="image/png"><source media="(max-width: 479px)" srcset="` + u + ` 1x"><img id="im" src="` + u + `" style="display: block"></picture></div>`
	widgetsByID(t, src, func(ws map[string]*core.WidgetBase, dp float32) {
		pic, im := ws["pic"], ws["im"]
		require.NotNil(t, im)
		assert.InDelta(t, im.Geom.Size.Actual.Total.Y, pic.Geom.Size.Actual.Total.Y, 1, "only the image")
	})
}

// TestAbsoluteOverInFlow tests that an absolute image (of no z-index) is
// rendered over the widgets in the flow of its positioned ancestor after
// the one that it is in, as in CSS: the images of edition.cnn.com, in a
// picture before the ::after that gives their box its height, were
// painted over by it.
func TestAbsoluteOverInFlow(t *testing.T) {
	var buf bytes.Buffer
	red := image.NewRGBA(image.Rect(0, 0, 160, 90))
	draw.Draw(red, red.Bounds(), image.NewUniform(color.RGBA{255, 0, 0, 255}), image.Point{}, draw.Src)
	require.NoError(t, png.Encode(&buf, red))
	u := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
	src := `<style>.media { display: flex; flex-direction: column; position: relative; width: 320px }
.media::after { content: ""; padding-bottom: 56.25%; display: block }
.media img { position: absolute; top: 0; left: 0; width: 100%; height: 100% }</style>
<div class="media"><picture><img src="` + u + `"></picture></div>`
	bd := core.NewBody()
	require.NoError(t, ReadHTMLString(NewContext(), bd, src))
	img := bd.RenderImage()
	n := 0
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			if r, g, _, _ := img.At(x, y).RGBA(); r > 0xf000 && g < 0x1000 {
				n++
			}
		}
	}
	dp := bd.Styles.UnitContext.Dots(units.UnitDp)
	assert.InEpsilon(t, 320*180*dp*dp, float32(n), 0.05, "the image fills its box")
}

// TestFlexWrapPercentWidth tests that items of width: 100% in a row that
// wraps are each on a line of their own, as in CSS, as the containers of
// the stacks of edition.cnn.com, which were all side by side.
func TestFlexWrapPercentWidth(t *testing.T) {
	src := `<div style="display: flex; flex-wrap: wrap; justify-content: space-between; width: 300px">
<div id="a" style="width: 100%">Nvidia CEO admits affair</div><div id="b" style="width: 100%">US and Iran are negotiating</div></div>`
	widgetsByID(t, src, func(ws map[string]*core.WidgetBase, dp float32) {
		a, b := ws["a"], ws["b"]
		require.NotNil(t, b)
		assert.InDelta(t, 300*dp, a.Geom.Size.Actual.Total.X, 2)
		assert.InDelta(t, 300*dp, b.Geom.Size.Actual.Total.X, 2)
		assert.Greater(t, b.Geom.TotalBBox.Min.Y, a.Geom.TotalBBox.Min.Y, "on the next line")
	})
}

// TestGridAutoFit tests the tracks of repeat(auto-fit, minmax(25%, 1fr)),
// as many as fit and as there are items, as the 3 cards of a grid of
// edition.cnn.com, which were in one column.
func TestGridAutoFit(t *testing.T) {
	src := `<div style="width: 561px"><div style="display: grid; gap: 32px 24px; grid-template-columns: repeat(auto-fit, minmax(25%, 1fr))">
<div id="a">Hegseth impeachment drama</div><div id="b">Blanche: The Constitution</div><div id="c">Congress returns</div></div></div>`
	widgetsByID(t, src, func(ws map[string]*core.WidgetBase, dp float32) {
		a, b, c := ws["a"], ws["b"], ws["c"]
		require.NotNil(t, c)
		assert.Equal(t, a.Geom.TotalBBox.Min.Y, c.Geom.TotalBBox.Min.Y, "in one row")
		assert.InDelta(t, (561-48)*dp/3, b.Geom.Size.Actual.Total.X, 4)
	})
}

// TestContainerQuery tests the conditions of @container rules, of the
// width of the named container: as the image boxes of the spotlight cards
// of edition.cnn.com, of aspect-ratio 3/2 in a container of 537px (width <=
// 799px), not the 2/3 of a later rule (width <= 382px), which all applied.
func TestContainerQuery(t *testing.T) {
	src := `<style>.card { container: spotlight / inline-size; width: 537px }
@container spotlight (width <= 799px) { .box { aspect-ratio: 3/2 } }
@container spotlight (width <= 382px) { .box { aspect-ratio: 2/3 } }</style>
<div class="card"><div class="box" id="box"></div></div>`
	widgetsByID(t, src, func(ws map[string]*core.WidgetBase, dp float32) {
		require.NotNil(t, ws["box"])
		assert.InDelta(t, 537*dp*2/3, ws["box"].Geom.Size.Actual.Total.Y, 3, "of aspect 3/2")
	})
}
