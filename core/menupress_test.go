// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"testing"
	"time"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// menuTest is a body with a button with a menu, for testing menus with
// mouse events sent to its window.
type menuTest struct {
	t       *testing.T
	b       *Body
	bt      *Button
	clicked []string
}

// newMenuTest returns a new [menuTest], with a button with a menu made by
// the given function, which can add items with [menuTest.item].
func newMenuTest(t *testing.T, menu func(mt *menuTest, m *Scene)) *menuTest {
	mt := &menuTest{t: t, b: NewBody()}
	mt.b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(400), units.Dp(300)) })
	mt.bt = NewButton(mt.b).SetText("Menu")
	mt.bt.SetMenu(func(m *Scene, pos image.Point) { menu(mt, m) })
	return mt
}

// item adds an item with the given text to the given menu, which records
// its clicks, and has the given submenu, if it is not nil.
func (mt *menuTest) item(m *Scene, text string, sub func(m *Scene)) {
	b := NewButton(m).SetText(text)
	if sub != nil {
		b.SetMenu(func(m *Scene, pos image.Point) { sub(m) })
		return
	}
	b.OnClick(func(e events.Event) { mt.clicked = append(mt.clicked, text) })
}

// menus returns the texts of the first items of the open menus,
// from the bottom one.
func (mt *menuTest) menus() []string {
	mt.b.AsyncLock()
	defer mt.b.AsyncUnlock()
	var texts []string
	ps := mt.b.Scene.Stage.popups
	for i := range ps.stack.Len() {
		st := ps.stack.ValueByIndex(i)
		if st.Type == MenuStage && st.Scene.HasChildren() {
			texts = append(texts, AsButton(st.Scene.Child(0)).Text)
		}
	}
	return texts
}

// center returns the center of the button with the given text, in the
// window: the menu button, or an item in the top menu with it.
func (mt *menuTest) center(text string) image.Point {
	mt.b.AsyncLock()
	defer mt.b.AsyncUnlock()
	var w *WidgetBase
	if text == mt.bt.Text {
		w = mt.bt.AsWidget()
	} else {
		ps := mt.b.Scene.Stage.popups
		for i := ps.stack.Len() - 1; i >= 0 && w == nil; i-- {
			for _, kid := range ps.stack.ValueByIndex(i).Scene.Children {
				if b := AsButton(kid); b != nil && b.Text == text {
					w = b.AsWidget()
					break
				}
			}
		}
	}
	require.NotNil(mt.t, w, text)
	bb := w.winBBox()
	return bb.Min.Add(bb.Max).Div(2)
}

// wait waits for the open menus to have the given first items.
func (mt *menuTest) wait(want ...string) {
	for range 100 {
		if got := mt.menus(); assert.ObjectsAreEqual(want, got) || (len(want) == 0 && len(got) == 0) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	assert.Equal(mt.t, want, mt.menus())
}

// TestMenuPressDrag tests opening a menu by pressing the mouse button on
// its button, dragging to an item, and releasing it on the item, which
// clicks it; and that a click on the button leaves the menu open, and
// releasing it elsewhere after dragging closes it.
func TestMenuPressDrag(t *testing.T) {
	mt := newMenuTest(t, func(mt *menuTest, m *Scene) {
		mt.item(m, "A", nil)
		mt.item(m, "B", nil)
	})
	imagex.UpdateTestImagesFor(t) // the image is not compared
	mt.b.AssertRender(t, "menu-press-drag", func() {
		src := mt.b.Scene.RenderWindow().SystemWindow.Events()
		btc := mt.center("Menu")
		src.MouseButton(events.MouseDown, events.Left, btc, 0)
		mt.wait("A") // open while pressed
		bc := mt.center("B")
		src.MouseMove(btc.Add(image.Pt(0, 2)))
		src.MouseMove(bc)
		src.MouseButton(events.MouseUp, events.Left, bc, 0)
		mt.wait()
		assert.Equal(t, []string{"B"}, mt.clicked)

		// a click leaves it open, and an item can then be clicked
		src.MouseButton(events.MouseDown, events.Left, btc, 0)
		src.MouseButton(events.MouseUp, events.Left, btc, 0)
		mt.wait("A")
		time.Sleep(100 * time.Millisecond)
		mt.wait("A")
		ac := mt.center("A")
		src.MouseMove(ac)
		src.MouseButton(events.MouseDown, events.Left, ac, 0)
		src.MouseButton(events.MouseUp, events.Left, ac, 0)
		mt.wait()
		assert.Equal(t, []string{"B", "A"}, mt.clicked)

		// releasing it elsewhere closes it
		src.MouseButton(events.MouseDown, events.Left, btc, 0)
		mt.wait("A")
		off := image.Pt(btc.X+300, btc.Y+300)
		src.MouseMove(off)
		src.MouseButton(events.MouseUp, events.Left, off, 0)
		mt.wait()
		assert.Equal(t, []string{"B", "A"}, mt.clicked)
	})
}

// TestSubmenuParentClick tests that clicking on an item of a menu with an
// open submenu clicks it: the submenu of another item opens in place of
// the open one, and another item is clicked, closing all of the menus.
func TestSubmenuParentClick(t *testing.T) {
	mt := newMenuTest(t, func(mt *menuTest, m *Scene) {
		mt.item(m, "Sub1", func(m *Scene) { mt.item(m, "X", nil) })
		mt.item(m, "Sub2", func(m *Scene) { mt.item(m, "Y", nil) })
		mt.item(m, "Plain", nil)
	})
	imagex.UpdateTestImagesFor(t) // the image is not compared
	mt.b.AssertRender(t, "submenu-parent-click", func() {
		src := mt.b.Scene.RenderWindow().SystemWindow.Events()
		click := func(text string) {
			p := mt.center(text)
			src.MouseMove(p)
			src.MouseButton(events.MouseDown, events.Left, p, 0)
			src.MouseButton(events.MouseUp, events.Left, p, 0)
		}
		click("Menu")
		mt.wait("Sub1")
		click("Sub1")
		mt.wait("Sub1", "X")
		click("Sub2")
		mt.wait("Sub1", "Y")
		click("Sub1")
		mt.wait("Sub1", "X")
		click("Plain")
		mt.wait()
		assert.Equal(t, []string{"Plain"}, mt.clicked)
	})
}

// TestMenuPressDragSubmenu tests dragging through submenus with the mouse
// button pressed on the menu button: dragging over an item with a submenu
// opens it, in place of another, and releasing over an item of a submenu,
// or of the menu after dragging back into it, clicks it.
func TestMenuPressDragSubmenu(t *testing.T) {
	mt := newMenuTest(t, func(mt *menuTest, m *Scene) {
		mt.item(m, "Sub1", func(m *Scene) { mt.item(m, "X", nil) })
		mt.item(m, "Sub2", func(m *Scene) { mt.item(m, "Y", nil) })
		mt.item(m, "Plain", nil)
	})
	imagex.UpdateTestImagesFor(t) // the image is not compared
	mt.b.AssertRender(t, "menu-press-drag-submenu", func() {
		src := mt.b.Scene.RenderWindow().SystemWindow.Events()
		drag := func(text string) image.Point {
			p := mt.center(text)
			src.MouseMove(p)
			return p
		}
		btc := mt.center("Menu")
		src.MouseButton(events.MouseDown, events.Left, btc, 0)
		mt.wait("Sub1")
		drag("Sub1")
		mt.wait("Sub1", "X")
		drag("Sub1") // stays open
		mt.wait("Sub1", "X")
		drag("Sub2")
		mt.wait("Sub1", "Y")
		p := drag("Y")
		src.MouseButton(events.MouseUp, events.Left, p, 0)
		mt.wait()
		assert.Equal(t, []string{"Y"}, mt.clicked)

		src.MouseButton(events.MouseDown, events.Left, btc, 0)
		mt.wait("Sub1")
		drag("Sub2")
		mt.wait("Sub1", "Y")
		p = drag("Plain") // closes the submenu
		mt.wait("Sub1")
		src.MouseButton(events.MouseUp, events.Left, p, 0)
		mt.wait()
		assert.Equal(t, []string{"Y", "Plain"}, mt.clicked)
	})
}

// TestMenuPressDragTooltip tests dragging into a submenu with a tooltip
// shown on top of the menus (which crashed).
func TestMenuPressDragTooltip(t *testing.T) {
	mt := newMenuTest(t, func(mt *menuTest, m *Scene) {
		mt.item(m, "Sub1", func(m *Scene) { mt.item(m, "X", nil) })
		mt.item(m, "Plain", nil)
	})
	imagex.UpdateTestImagesFor(t) // the image is not compared
	mt.b.AssertRender(t, "menu-press-drag-tooltip", func() {
		src := mt.b.Scene.RenderWindow().SystemWindow.Events()
		btc := mt.center("Menu")
		src.MouseButton(events.MouseDown, events.Left, btc, 0)
		mt.wait("Sub1")
		mt.b.AsyncLock()
		tip := NewScene("tip")
		NewText(tip).SetText("A tooltip")
		NewPopupStage(TooltipStage, tip, mt.bt).Run()
		mt.b.AsyncUnlock()
		p := mt.center("Sub1")
		src.MouseMove(p)
		mt.wait("Sub1", "X")
		p = mt.center("X")
		src.MouseMove(p)
		src.MouseButton(events.MouseUp, events.Left, p, 0)
		mt.wait()
		assert.Equal(t, []string{"X"}, mt.clicked)
	})
}

// TestMenuItemContextMenu tests the context menu of an item of a menu,
// which is opened on top of the menu by right-clicking the item, and whose
// items can be clicked, such as to edit a script of the commands.
func TestMenuItemContextMenu(t *testing.T) {
	mt := newMenuTest(t, func(mt *menuTest, m *Scene) {
		mt.item(m, "A", nil)
		b := NewButton(m).SetText("B")
		b.AddContextMenu(func(m *Scene, pos image.Point) { mt.item(m, "Edit B", nil) })
	})
	imagex.UpdateTestImagesFor(t) // the image is not compared
	mt.b.AssertRender(t, "menu-item-context-menu", func() {
		src := mt.b.Scene.RenderWindow().SystemWindow.Events()
		click := func(text string, but events.Buttons) {
			p := mt.center(text)
			src.MouseMove(p)
			src.MouseButton(events.MouseDown, but, p, 0)
			src.MouseButton(events.MouseUp, but, p, 0)
		}
		click("Menu", events.Left)
		mt.wait("A")
		click("B", events.Right)
		mt.wait("A", "Edit B")
		click("Edit B", events.Left)
		mt.wait()
		assert.Equal(t, []string{"Edit B"}, mt.clicked)
	})
}
