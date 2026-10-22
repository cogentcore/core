// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"strconv"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
)

// renderCounter is a widget that records the size it has each time it renders.
type renderCounter struct {
	WidgetBase

	// sizes has the actual total size of the widget at each render.
	sizes []math32.Vector2
}

func (rc *renderCounter) Init() {
	rc.WidgetBase.Init()
	rc.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(10))
		s.Grow.Set(1, 1)
	})
}

func (rc *renderCounter) Render() {
	rc.sizes = append(rc.sizes, rc.Geom.Size.Actual.Total)
	rc.WidgetBase.Render()
}

// newOverflowToolbarBody returns a body of the given width in dp with a widget
// that counts its renders above a toolbar that has n buttons, which do not
// all fit in that width.
func newOverflowToolbarBody(width float32, n int) (*Body, *Toolbar, *renderCounter) {
	b := NewBody()
	b.Styler(func(s *styles.Style) {
		s.Min.X.Dp(width)
		s.Max.X.Dp(width)
		s.Min.Y.Dp(200)
	})
	rc := tree.New[renderCounter](b)
	tb := NewToolbar(b)
	tb.Maker(func(p *tree.Plan) {
		for i := range n {
			tree.AddAt(p, "button-"+strconv.Itoa(i), func(w *Button) {
				w.SetText("Button " + strconv.Itoa(i))
			})
		}
	})
	return b, tb, rc
}

// overflowNames returns the names of the items in the overflow menu.
func overflowNames(tb *Toolbar) []string {
	var names []string
	for _, it := range tb.overflowItems {
		names = append(names, it.Name)
	}
	return names
}

// childNames returns the names of the children of the toolbar.
func childNames(tb *Toolbar) []string {
	var names []string
	for _, k := range tb.Children {
		names = append(names, k.AsTree().Name)
	}
	return names
}

func TestToolbarOverflow(t *testing.T) {
	b, tb, _ := newOverflowToolbarBody(300, 10)
	b.AssertRender(t, "toolbar/overflow", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		ch := childNames(tb)
		ov := overflowNames(tb)
		assert.Greater(t, len(ov), 0)
		assert.Greater(t, len(ch), 1)
		assert.Equal(t, "overflow-menu", ch[len(ch)-1])
		// all of the items are either in the toolbar or in the overflow menu, in order
		all := append(ch[:len(ch)-1:len(ch)-1], ov...)
		for i, nm := range all {
			assert.Equal(t, "button-"+strconv.Itoa(i), nm)
		}
		assert.Len(t, all, 10)
		assert.True(t, tb.overflowButton.IsVisible())

		// the overflow menu has the overflow items
		m := newMenuScene(tb.overflowButton.Menu, image.Point{})
		var mitems []string
		for _, k := range m.Children {
			if bt, ok := k.(*Button); ok {
				mitems = append(mitems, bt.Text)
			}
		}
		for i, nm := range ov {
			n, _ := strconv.Atoi(nm[len("button-"):])
			assert.Equal(t, "Button "+strconv.Itoa(n), mitems[i])
		}
	})
}

func TestToolbarOverflowMenuOpen(t *testing.T) {
	b, tb, _ := newOverflowToolbarBody(300, 10)
	b.AssertRender(t, "toolbar/overflow-menu-open", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		tb.overflowButton.Send(events.Click)
	})
}

func TestToolbarNoOverflow(t *testing.T) {
	b, tb, _ := newOverflowToolbarBody(800, 3)
	b.AssertRender(t, "toolbar/no-overflow", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		assert.Len(t, tb.overflowItems, 0)
		assert.Equal(t, 4, tb.NumChildren())
		assert.False(t, tb.overflowButton.IsVisible())
	})
}

// TestToolbarOverflowLayoutNoRender tests that a layout of a scene with
// an overflowing toolbar does not render anything, and that widgets are
// only ever rendered with their final size.
func TestToolbarOverflowLayoutNoRender(t *testing.T) {
	b, tb, rc := newOverflowToolbarBody(300, 10)
	b.AssertRender(t, "toolbar/overflow-layout", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		final := rc.Geom.Size.Actual.Total
		nov := len(tb.overflowItems)
		assert.Greater(t, nov, 0)
		for _, sz := range rc.sizes {
			assert.Equal(t, final, sz, "render at a transitional size")
		}
		n := len(rc.sizes)
		for range 5 {
			b.Scene.LayoutScene()
		}
		assert.Equal(t, n, len(rc.sizes), "render during layout")
		assert.Equal(t, final, rc.Geom.Size.Actual.Total)
		assert.Equal(t, nov, len(tb.overflowItems))
	})
}

// TestToolbarOverflowResize tests that resizing a scene with an overflowing
// toolbar puts the right items in the overflow menu, and that no widget is
// rendered with a transitional size during the resizing.
func TestToolbarOverflowResize(t *testing.T) {
	b, tb, rc := newOverflowToolbarBody(300, 10)
	b.Styler(func(s *styles.Style) {
		s.Min.X.Dp(100)
		s.Max.X.Zero()
	})
	b.AssertRender(t, "toolbar/overflow-resize", func() {
		rw := b.Scene.RenderWindow()
		ht := b.Scene.SceneGeom.Size.Y
		b.AsyncLock()
		finals := map[math32.Vector2]bool{rc.Geom.Size.Actual.Total: true} // initial size
		b.AsyncUnlock()
		novs := map[int]int{}
		// end at the initial width, so that the window image is all current
		wd0 := b.Scene.SceneGeom.Size.X
		for _, wd := range []int{1200, 200, 600, 1200, 320, 900, 320, wd0} {
			b.AsyncLock()
			b.Scene.Resize(math32.Geom2DInt{Size: image.Pt(wd, ht)})
			b.AsyncUnlock()
			for range 6 {
				rw.waitRenderDone() // or the render is skipped
				rw.renderWindow()
			}
			b.AsyncLock()
			finals[rc.Geom.Size.Actual.Total] = true
			assert.Equal(t, float32(wd), b.Scene.Geom.Size.Actual.Total.X)
			ch := childNames(tb)
			ov := overflowNames(tb)
			all := append(ch[:len(ch)-1:len(ch)-1], ov...)
			assert.Len(t, all, 10)
			for i, nm := range all {
				assert.Equal(t, "button-"+strconv.Itoa(i), nm)
			}
			assert.Equal(t, len(ov) > 0, tb.overflowButton.IsVisible())
			if n, ok := novs[wd]; ok {
				assert.Equal(t, n, len(ov), "different overflow for same width")
			}
			novs[wd] = len(ov)
			// the toolbar items fit in the toolbar
			end := tb.Geom.Pos.Content.X + tb.Geom.Size.Actual.Content.X
			for _, k := range tb.Children {
				kb := AsWidget(k)
				if !kb.IsVisible() {
					continue
				}
				assert.LessOrEqual(t, kb.Geom.Pos.Total.X+kb.Geom.Size.Actual.Total.X, end+1, kb.Name)
			}
			b.AsyncUnlock()
		}
		assert.Equal(t, 0, novs[1200])
		assert.Greater(t, novs[200], novs[320])
		assert.Greater(t, novs[320], novs[600])
		assert.Greater(t, novs[600], novs[900])
		for _, sz := range rc.sizes {
			assert.True(t, finals[sz], "render at a transitional size %v", sz)
		}
	})
}
