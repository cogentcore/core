// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core_test

import (
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// selectPage is an htmlcore page of the fidelity tests (in
// htmlcore/testdata/fidelity) in a window of the given size, for testing
// the selection of text across texts by sliding the mouse.
type selectPage struct {
	t *testing.T
	b *core.Body
}

func newSelectPage(t *testing.T, name string, width, height float32) *selectPage {
	src, err := os.ReadFile(filepath.Join("..", "htmlcore", "testdata", "fidelity", name+".html"))
	require.NoError(t, err)
	b := core.NewBody()
	b.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(width), units.Dp(height))
		s.Max = s.Min
	})
	require.NoError(t, htmlcore.ReadHTMLString(htmlcore.NewContext(), b, string(src)))
	return &selectPage{t: t, b: b}
}

// text returns the text that has the given substring in it.
func (sp *selectPage) text(sub string) *core.Text {
	var found *core.Text
	sp.b.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
		if tx, ok := cw.(*core.Text); ok && found == nil && strings.Contains(string(tx.TextRunes()), sub) {
			found = tx
		}
		return found == nil
	})
	require.NotNil(sp.t, found, "a text with %q", sub)
	return found
}

// anyText returns a text of the page.
func (sp *selectPage) anyText() *core.Text { return sp.text("") }

// at returns the position of the middle of the first rune of the given
// substring of the text that has it, plus the given offset in runes.
func (sp *selectPage) at(sub string, off int) image.Point {
	tx := sp.text(sub)
	i := strings.Index(string(tx.TextRunes()), sub)
	return core.TextRunePoint(tx, len([]rune(string(tx.TextRunes())[:i]))+off)
}

// drag presses the mouse at the first of the given points (each computed
// with the lock of the window), moves it through the others, and releases
// it at the last one, and returns the text that is selected then.
func (sp *selectPage) drag(points ...func() image.Point) string {
	return sp.dragCheck(nil, points...)
}

// jump presses the mouse at the given point, moves it a little to start
// the slide, and then jumps to the other point in a single move, skipping
// over the texts in between, as a fast mouse does, and returns the text
// that is selected then.
func (sp *selectPage) jump(from, to func() image.Point) string {
	return sp.dragSteps(1, nil, from, func() image.Point { return from().Add(image.Pt(6, 0)) }, to)
}

// dragCheck is [selectPage.drag] that also calls the given function, if
// any, with the lock of the window after the drag.
func (sp *selectPage) dragCheck(check func(), points ...func() image.Point) string {
	return sp.dragSteps(5, check, points...)
}

// dragSteps is [selectPage.dragCheck] that moves the mouse in the given
// number of steps between the points.
func (sp *selectPage) dragSteps(steps int, check func(), points ...func() image.Point) string {
	sel := ""
	ev := func() *events.Source { return sp.b.Scene.RenderWindow().SystemWindow.Events() }
	pts := make([]image.Point, len(points))
	core.RenderImage(sp.b, core.Locked(sp.b, func() {
		for i, p := range points {
			pts[i] = p()
		}
	}), func() {
		ev().MouseMove(pts[0]) // (the mouse is there first, as in a real window)
		time.Sleep(15 * time.Millisecond)
		ev().MouseButton(events.MouseDown, events.Left, pts[0], 0)
		// a slide starts after its time (as in [core.DragMouse])
		time.Sleep(max(core.TimingSettings.DragStartTime, core.TimingSettings.SlideStartTime) + 20*time.Millisecond)
		// in steps, as a real mouse moves (the first move starts the slide)
		for i, p := range pts[1:] {
			from := pts[i]
			for j := 1; j <= steps; j++ {
				ev().MouseMove(from.Add(p.Sub(from).Mul(j).Div(steps)))
				time.Sleep(10 * time.Millisecond)
			}
		}
		ev().MouseButton(events.MouseUp, events.Left, pts[len(pts)-1], 0)
		time.Sleep(15 * time.Millisecond)
	}, core.Locked(sp.b, func() {
		sel = core.TextSelected(sp.anyText()) // the selection of the scene
		if check != nil {
			check()
		}
	}))
	return sel
}

// TestTextSelectParagraphs tests selecting text across the paragraphs of
// a page, as in a browser: down from the middle of one paragraph to the
// middle of another, up, and down and back up past the start (which must
// not leave anything selected where the mouse was before), within a
// paragraph that wraps onto several lines, and to the end of a line by
// going past it to the right.
func TestTextSelectParagraphs(t *testing.T) {
	const p1 = "A single short paragraph."
	const p2 = "A second paragraph that is long enough to wrap onto more than one line at the width of the page, so that the line height and wrapping of the text can be compared with each other, which is the point of this test."
	const p3 = "A third paragraph."
	from := func(s, sub string) string { return s[strings.Index(s, sub):] }
	to := func(s, sub string) string { return s[:strings.Index(s, sub)] }
	between := func(s, a, b string) string { return s[strings.Index(s, a):strings.Index(s, b)] }

	t.Run("down", func(t *testing.T) {
		sp := newSelectPage(t, "01-paragraphs", 400, 400)
		at := func(sub string) func() image.Point { return func() image.Point { return sp.at(sub, 0) } }
		assert.Equal(t, from(p2, "second")+"\n"+to(p3, "third"), sp.drag(at("second"), at("compared"), at("third")))
	})
	t.Run("up", func(t *testing.T) {
		sp := newSelectPage(t, "01-paragraphs", 400, 400)
		at := func(sub string) func() image.Point { return func() image.Point { return sp.at(sub, 0) } }
		assert.Equal(t, from(p1, "short")+"\n"+to(p2, "wrapping"), sp.drag(at("wrapping"), at("single"), at("short")))
	})
	t.Run("down and back up past the start", func(t *testing.T) {
		sp := newSelectPage(t, "01-paragraphs", 400, 400)
		at := func(sub string) func() image.Point { return func() image.Point { return sp.at(sub, 0) } }
		// it starts in a later line of the wrapped paragraph, goes down
		// into the next paragraph, and up past the start into its first line
		sel := sp.dragCheck(func() {
			// nothing stays selected where the mouse was before
			assert.Empty(t, core.TextOwnSelection(sp.text("third")))
			assert.Empty(t, core.TextOwnSelection(sp.text("single")))
		}, at("compared"), at("third"), at("paragraph."), at("single"), at("long"))
		assert.Equal(t, between(p2, "long", "compared"), sel)
	})
	t.Run("up and back down past the start", func(t *testing.T) {
		sp := newSelectPage(t, "01-paragraphs", 400, 400)
		at := func(sub string) func() image.Point { return func() image.Point { return sp.at(sub, 0) } }
		sel := sp.dragCheck(func() {
			assert.Empty(t, core.TextOwnSelection(sp.text("single")), "nothing stays selected above")
		}, at("wrapping"), at("short"), at("second"), at("third"), func() image.Point { return sp.at("third", 6) })
		assert.Equal(t, from(p2, "wrapping")+"\n"+to(p3, "paragraph."), sel)
	})
	t.Run("a jump over a paragraph", func(t *testing.T) {
		sp := newSelectPage(t, "01-paragraphs", 400, 400)
		at := func(sub string) func() image.Point { return func() image.Point { return sp.at(sub, 0) } }
		// the second paragraph, which the mouse never was over, is selected
		assert.Equal(t, from(p1, "single")+"\n"+p2+"\n"+to(p3, "third"), sp.jump(at("single"), at("third")))
	})
	t.Run("past the end of a line", func(t *testing.T) {
		sp := newSelectPage(t, "01-paragraphs", 400, 400)
		at := func(sub string) func() image.Point { return func() image.Point { return sp.at(sub, 0) } }
		right := func() image.Point { return image.Pt(390, sp.at("short", 0).Y) } // (in the window)
		assert.Equal(t, from(p1, "single"), sp.drag(at("single"), right))
	})
	t.Run("into the space between paragraphs", func(t *testing.T) {
		sp := newSelectPage(t, "01-paragraphs", 400, 400)
		at := func(sub string) func() image.Point { return func() image.Point { return sp.at(sub, 0) } }
		// just below the first paragraph, at its right: its end
		below := func() image.Point {
			p, q := sp.at("short", 0), sp.at("second", 0)
			return image.Pt(390, (p.Y+q.Y)/2-4)
		}
		assert.Equal(t, from(p1, "single"), sp.drag(at("single"), below))
	})
}

// TestTextSelectColumns tests selecting text across the columns of a
// multi-column list, which is in the order of the document, as in a
// browser: down a column and into the next, and back.
func TestTextSelectColumns(t *testing.T) {
	names := []string{"alpha", "beta", "gamma", "delta", "epsilon", "zeta", "eta", "theta", "iota", "kappa", "lambda", "mu"}
	t.Run("forward", func(t *testing.T) {
		sp := newSelectPage(t, "28-columns", 500, 400)
		// beta is in the first column; theta is not
		assert.Equal(t, strings.Join(names[1:7], "\n")+"\nth", sp.drag(func() image.Point { return sp.at("beta", 0) }, func() image.Point { return sp.at("theta", 2) }))
	})
	t.Run("backward", func(t *testing.T) {
		sp := newSelectPage(t, "28-columns", 500, 400)
		assert.Equal(t, "ta\n"+strings.Join(names[6:9], "\n")+"\nkap", sp.drag(func() image.Point { return sp.at("kappa", 3) }, func() image.Point { return sp.at("zeta", 2) }))
	})
}

// TestTextSelectLayout tests selecting text from the main content of a
// page into its side panel, which is before it in the document, as in a
// browser: from the side panel item to the start in the main content.
func TestTextSelectLayout(t *testing.T) {
	sp := newSelectPage(t, "21-page-layout", 700, 400)
	assert.Equal(t, "Details\nSummary\nMain ", sp.drag(func() image.Point { return sp.at("content", 0) }, func() image.Point { return sp.at("Details", 0) }))
}
