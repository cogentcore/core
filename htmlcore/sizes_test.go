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
	"cogentcore.org/core/math32"
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
}
