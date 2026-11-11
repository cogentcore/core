// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"testing"
	"time"

	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tabNames returns the names of the tab buttons and of the tab frames of
// the given tabs, which must be the same.
func tabNames(t *testing.T, ts *Tabs) []string {
	t.Helper()
	var names []string
	fr := ts.getFrame()
	for i := range ts.NumTabs() {
		_, tab := ts.TabAtIndex(i)
		require.Equal(t, tab.Name, fr.Child(i).AsTree().Name, "the button and the frame of tab %d", i)
		names = append(names, tab.Name)
	}
	return names
}

// selectedTab returns the name of the selected tab of the given tabs, and
// whether only its button is selected.
func selectedTab(ts *Tabs) (string, bool) {
	w, idx := ts.CurrentTab()
	if w == nil {
		return "", false
	}
	n := 0
	for i := range ts.NumTabs() {
		_, tab := ts.TabAtIndex(i)
		if tab.StateIs(states.Selected) {
			n++
			if i != idx {
				return w.AsTree().Name, false
			}
		}
	}
	return w.AsTree().Name, n == 1
}

func newDragTabs(b *Body, names ...string) *Tabs {
	ts := NewTabs(b).SetType(FunctionalTabs)
	for _, n := range names {
		fr, _ := ts.NewTab(n)
		NewText(fr).SetText("content of " + n)
	}
	return ts
}

func TestTabsMoveTabIndex(t *testing.T) {
	b := NewBody()
	ts := newDragTabs(b, "A", "B", "C", "D")
	var moves [][2]int
	ts.MoveTabFunc = func(from int, to *Tabs, at int) {
		assert.Same(t, ts, to)
		moves = append(moves, [2]int{from, at})
	}
	ts.SelectTabIndex(1)
	assert.True(t, ts.MoveTabIndex(0, 2))
	assert.Equal(t, []string{"B", "C", "A", "D"}, tabNames(t, ts))
	sel, one := selectedTab(ts)
	assert.Equal(t, "B", sel, "the selected tab stays selected")
	assert.True(t, one)
	assert.True(t, ts.MoveTabIndex(0, 3))
	assert.Equal(t, []string{"C", "A", "D", "B"}, tabNames(t, ts))
	sel, _ = selectedTab(ts)
	assert.Equal(t, "B", sel)
	assert.True(t, ts.MoveTabIndex(3, 1))
	assert.Equal(t, []string{"C", "B", "A", "D"}, tabNames(t, ts))
	sel, _ = selectedTab(ts)
	assert.Equal(t, "B", sel)
	assert.False(t, ts.MoveTabIndex(0, 4))
	assert.Equal(t, [][2]int{{0, 2}, {0, 3}, {3, 1}}, moves)
}

// TestTabsDragReorder tests that a tab of [FunctionalTabs] that is dragged
// with the mouse onto another tab moves to its place.
func TestTabsDragReorder(t *testing.T) {
	prev := TimingSettings.DragStartTime
	TimingSettings.DragStartTime = 0
	defer func() { TimingSettings.DragStartTime = prev }()
	b := NewBody()
	ts := newDragTabs(b, "First", "Second", "Third")
	center := func(i int) image.Point {
		_, tab := ts.TabAtIndex(i)
		bb := tab.Geom.TotalBBox
		return bb.Min.Add(bb.Size().Div(2))
	}
	b.RenderImage(func() {
		b.AsyncLock()
		from, to := center(0), center(2)
		b.AsyncUnlock()
		ev := b.Scene.RenderWindow().SystemWindow.Events()
		ev.MouseButton(events.MouseDown, events.Left, from, 0)
		time.Sleep(20 * time.Millisecond)
		for i := 1; i <= 10; i++ {
			ev.MouseMove(from.Add(to.Sub(from).Mul(i).Div(10)))
			time.Sleep(10 * time.Millisecond)
		}
		ev.MouseButton(events.MouseUp, events.Left, to, 0)
	}, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		assert.Equal(t, []string{"Second", "Third", "First"}, tabNames(t, ts))
		sel, one := selectedTab(ts)
		assert.Equal(t, "First", sel)
		assert.True(t, one)
	})
}

// TestTabsMoveTabTo tests that a tab moves into other tabs (as into those
// of a new window, for [Tabs.DragToWindow]), with its content, which is
// then in their scene and selected, and that the tab next to it is
// selected in its tabs.
func TestTabsMoveTabTo(t *testing.T) {
	b := NewBody()
	ts := newDragTabs(b, "A", "B", "C")
	ts.SelectTabIndex(1)
	b2 := NewBody()
	other := newDragTabs(b2, "X", "Y")
	frame, _ := ts.TabAtIndex(1)
	moved := false
	ts.MoveTabFunc = func(from int, to *Tabs, at int) {
		moved = true
		assert.Equal(t, 1, from)
		assert.Same(t, other, to)
		assert.Equal(t, 1, at)
	}
	assert.Equal(t, 1, ts.MoveTabTo(1, other, 1))
	assert.True(t, moved)
	assert.Equal(t, []string{"A", "C"}, tabNames(t, ts))
	assert.Equal(t, []string{"X", "B", "Y"}, tabNames(t, other))
	sel, one := selectedTab(ts)
	assert.Equal(t, "A", sel)
	assert.True(t, one)
	sel, one = selectedTab(other)
	assert.Equal(t, "B", sel)
	assert.True(t, one)
	assert.Same(t, b2.Scene, frame.Scene, "the content is in the scene of the other tabs")
	assert.Same(t, b2.Scene, AsWidget(frame.Child(0)).Scene)
}

// TestTabsMoveTabSameName tests that a tab moved into tabs that have a tab
// of the same name gets a unique name, and that clicking on either selects
// it, not the other one.
func TestTabsMoveTabSameName(t *testing.T) {
	b := NewBody()
	ts := newDragTabs(b, "A", "B")
	b2 := NewBody()
	other := newDragTabs(b2, "B", "X")
	assert.Equal(t, 2, ts.MoveTabTo(1, other, 2))
	assert.Equal(t, []string{"B", "X", "B-2"}, tabNames(t, other))
	_, tab := other.TabAtIndex(2)
	assert.Equal(t, "B", tab.Text, "its text is kept")
	b2.AssertRender(t, "tabs/move-same-name", func() {
		other.SelectTabIndex(0)
		tab.Send(events.Click)
		sel, one := selectedTab(other)
		assert.Equal(t, "B-2", sel)
		assert.True(t, one)
		_, first := other.TabAtIndex(0)
		first.Send(events.Click)
		sel, _ = selectedTab(other)
		assert.Equal(t, "B", sel)
	})
}

// TestTabsOSDragData tests that a tab is dragged by the operating system
// out of its window only when its tabs have [Tabs.DragToWindow].
func TestTabsOSDragData(t *testing.T) {
	b := NewBody()
	ts := newDragTabs(b, "A", "B")
	_, tab := ts.TabAtIndex(0)
	assert.Nil(t, tab.osDragData())
	ts.DragToWindow = func(b *Body) *Tabs { return NewTabs(b).SetType(FunctionalTabs) }
	md := tab.osDragData()
	require.Len(t, md, 1)
	assert.Equal(t, tabMimeType, md[0].Type)
	assert.Equal(t, "A", string(md[0].Data))
}

// TestTabsDroppedNowhere tests that a tab dropped where nothing accepted it
// (as onto the desktop) moves into a new window, initialized by
// [Tabs.DragToWindow], and that the last tab does not (its window moves).
func TestTabsDroppedNowhere(t *testing.T) {
	b := NewBody("one")
	ts := newDragTabs(b, "A", "B")
	var nt *Tabs
	ts.DragToWindow = func(nb *Body) *Tabs {
		nt = NewTabs(nb).SetType(FunctionalTabs)
		return nt
	}
	b.runAndShowNewWindow()
	b.AsyncLock()
	_, tab := ts.TabAtIndex(0)
	tab.SetText("Shown") // (as a web page sets its title)
	end := events.NewMouse(events.MouseUp, events.Left, image.Pt(-50, -50), 0)
	tab.droppedNowhere(end, image.Pt(300, 300))
	require.NotNil(t, nt)
	assert.Equal(t, []string{"B"}, tabNames(t, ts))
	assert.Equal(t, []string{"A"}, tabNames(t, nt))
	assert.NotNil(t, nt.DragToWindow, "the new tabs can be dragged too")
	assert.Equal(t, "Shown", nt.Scene.Body.Title, "titled as the tab is shown")
	_, ntab := nt.TabAtIndex(0)
	assert.Equal(t, "Shown", ntab.Text)
	nw := nt.Scene.RenderWindow()
	assert.NotNil(t, nw, "in a new window")
	assert.NotSame(t, b.Scene.RenderWindow(), nw)
	// the last tab does not move into another window
	nt = nil
	_, tab = ts.TabAtIndex(0)
	tab.droppedNowhere(end, image.Pt(300, 300))
	assert.Nil(t, nt)
	assert.Equal(t, []string{"B"}, tabNames(t, ts))
	b.Close()
	b.AsyncUnlock()
}

// TestTabsTransferWindows tests that a tab dropped into the tabs of another
// window (as by a drag of the operating system, or of a new window onto
// them) moves into them, in the event loops of the windows, and that its
// window closes when it was its last tab.
func TestTabsTransferWindows(t *testing.T) {
	drag := func(b *Body) *Tabs { return NewTabs(b).SetType(FunctionalTabs) }
	b1 := NewBody("one")
	src := newDragTabs(b1, "A")
	src.DragToWindow = drag
	b2 := NewBody("two")
	dst := newDragTabs(b2, "X", "Y")
	dst.DragToWindow = drag
	movedIn := -1
	dst.MoveInFunc = func(index int) { movedIn = index }
	b1.runAndShowNewWindow()
	b2.runAndShowNewWindow()
	w1 := b1.Scene.RenderWindow()
	require.NotSame(t, w1, b2.Scene.RenderWindow())
	b2.AsyncLock()
	_, tab := src.TabAtIndex(0)
	tabsDrop := &tabDrag{tabs: src, tab: tab}
	// (a drop between X and Y, in the event loop of the second window)
	tabsDrop.dropAt(dst, 1)
	// (a drop onto the content of the window is not a second move)
	tabsDrop.dropInScene(b1.Scene)
	b2.AsyncUnlock()
	assert.Eventually(t, func() bool {
		b2.AsyncLock()
		defer b2.AsyncUnlock()
		return dst.NumTabs() == 3
	}, 5*time.Second, 10*time.Millisecond)
	b2.AsyncLock()
	assert.Equal(t, []string{"X", "A", "Y"}, tabNames(t, dst))
	assert.Equal(t, 1, movedIn)
	frame, _ := dst.TabAtIndex(1)
	assert.Same(t, b2.Scene, frame.Scene)
	sel, _ := selectedTab(dst)
	assert.Equal(t, "A", sel)
	b2.AsyncUnlock()
	assert.Eventually(t, func() bool { return w1.isClosed() }, 5*time.Second, 10*time.Millisecond, "the window of its last tab is closed")
	b2.AsyncLock()
	b2.Close()
	b2.AsyncUnlock()
}

// TestTabsDropIndex tests the index at which a tab dropped onto the list of
// tabs goes.
func TestTabsDropIndex(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) { s.Min.X.Dp(500) }) // (one row of tabs)
	ts := newDragTabs(b, "A", "B", "C")
	b.RenderImage(func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		_, tab := ts.TabAtIndex(1)
		bb := tab.Geom.TotalBBox
		mid := bb.Min.Add(bb.Size().Div(2))
		assert.Equal(t, 1, ts.dropIndex(mid.Sub(image.Pt(2, 0))))
		assert.Equal(t, 2, ts.dropIndex(mid.Add(image.Pt(2, 0))))
		assert.Equal(t, 3, ts.dropIndex(image.Pt(ts.getTabs().Geom.TotalBBox.Max.X-1, mid.Y)))
	})
}

// TestTabsDropInScene tests that a tab of another window dropped by the
// operating system onto a window where nothing takes it (as onto the
// content of a page) moves to the end of its tabs.
func TestTabsDropInScene(t *testing.T) {
	drag := func(b *Body) *Tabs { return NewTabs(b).SetType(FunctionalTabs) }
	b1 := NewBody("one")
	src := newDragTabs(b1, "A", "B")
	src.DragToWindow = drag
	b2 := NewBody("two")
	NewText(b2).SetText("content")
	dst := newDragTabs(b2, "X")
	dst.DragToWindow = drag
	b1.runAndShowNewWindow()
	b2.runAndShowNewWindow()
	b2.AsyncLock()
	_, tab := src.TabAtIndex(1)
	(&tabDrag{tabs: src, tab: tab}).dropInScene(b2.Scene)
	b2.AsyncUnlock()
	assert.Eventually(t, func() bool {
		b2.AsyncLock()
		defer b2.AsyncUnlock()
		return dst.NumTabs() == 2
	}, 5*time.Second, 10*time.Millisecond)
	b2.AsyncLock()
	assert.Equal(t, []string{"X", "B"}, tabNames(t, dst))
	b2.Close()
	b2.AsyncUnlock()
	b1.AsyncLock()
	assert.Equal(t, []string{"A"}, tabNames(t, src), "its window stays, with another tab")
	b1.Close()
	b1.AsyncUnlock()
}
