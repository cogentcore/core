// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"image"
	"log/slog"
	"slices"

	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/tree"
)

// tabMimeType is the MIME type of the data of a tab that is dragged out of
// its window by the operating system (see [Tab.osDragData]), which other
// apps ignore, and which a window of this app drops into its tabs.
const tabMimeType = "application/x-cogentcore-tab"

// MoveTabIndex moves the tab at the given index to the given index (the
// one that it then has), returning whether it was successful. The
// selected tab stays selected.
func (ts *Tabs) MoveTabIndex(from, to int) bool {
	ts.mu.Lock()
	fr := ts.getFrame()
	tb := ts.getTabs()
	n := len(fr.Children)
	if from < 0 || from >= n || to < 0 || to >= n {
		ts.mu.Unlock()
		slog.Error("Tabs: index out of range for number of tabs", "from", from, "to", to, "numTabs", n)
		return false
	}
	if from == to {
		ts.mu.Unlock()
		return true
	}
	moveChild(fr, from, to)
	moveChild(tb, from, to)
	switch top := fr.StackTop; {
	case top == from:
		fr.StackTop = to
	case from < top && to >= top:
		fr.StackTop--
	case from > top && to <= top:
		fr.StackTop++
	}
	ts.mu.Unlock()
	ts.NeedsLayout()
	if ts.MoveTabFunc != nil {
		ts.MoveTabFunc(from, ts, to)
	}
	return true
}

// moveChild moves the child of the given node at the given index to the
// other given index, without deleting it.
func moveChild(n tree.Node, from, to int) {
	nb := n.AsTree()
	k := nb.Children[from]
	nb.Children = slices.Delete(nb.Children, from, from+1)
	nb.Children = slices.Insert(nb.Children, to, k)
}

// MoveTabTo moves the tab at the given index (its content and its button)
// into the given other tabs, at the given index there, where it is
// selected, returning the index that it has there, or -1 if it was not
// moved. The tab next to it is selected in these tabs, as when it is
// deleted, but it is not closed: [Tabs.CloseTabFunc] is not called, and
// [Tabs.MoveTabFunc] of these tabs and [Tabs.MoveInFunc] of the other
// tabs are. The tabs must be in the same window (or one not yet shown).
func (ts *Tabs) MoveTabTo(idx int, other *Tabs, at int) int {
	if other == ts || idx < 0 || idx >= ts.NumTabs() {
		return -1
	}
	at = min(max(at, 0), other.NumTabs())
	return other.attachTab(ts.detachTab(idx, other, at), at)
}

// detachedTab is a tab that has been removed from its tabs, to be added to
// other tabs (see [Tabs.detachTab]).
type detachedTab struct {
	frame                *Frame
	label, text, tooltip string
	icon                 icons.Icon
}

// detachTab removes the tab at the given index, which moves to the given
// index in the given other tabs, keeping its content for them, selects the
// tab next to it, as when it is deleted, and calls [Tabs.MoveTabFunc].
func (ts *Tabs) detachTab(idx int, other *Tabs, at int) *detachedTab {
	frame, tab := ts.TabAtIndex(idx)
	if frame == nil {
		return nil
	}
	dt := &detachedTab{frame: frame, label: tab.Name, text: tab.Text, tooltip: tab.Tooltip, icon: tab.Icon}
	if f := ts.Scene.Events.focus; f != nil && (tree.Node(f) == frame || isParentOf(frame, f)) {
		ts.Scene.Events.focusClear() // (not to send events to it here)
	}
	ts.mu.Lock()
	fr := ts.getFrame()
	nidx := -1
	if len(fr.Children) > 1 {
		switch {
		case fr.StackTop == idx:
			nidx = max(idx-1, 0)
		case fr.StackTop > idx:
			nidx = fr.StackTop - 1
		default:
			nidx = fr.StackTop
		}
	}
	ts.getTabs().DeleteChildAt(idx)
	fr.Children = slices.Delete(fr.Children, idx, idx+1)
	fr.StackTop = -1 // (selected again below)
	ts.mu.Unlock()
	if nidx >= 0 {
		ts.SelectTabIndex(nidx)
	}
	ts.NeedsLayout()
	if ts.MoveTabFunc != nil {
		ts.MoveTabFunc(idx, other, at)
	}
	return dt
}

// attachTab adds the given detached tab at the given index, where it is
// selected, calls [Tabs.MoveInFunc], and returns the index.
func (ts *Tabs) attachTab(dt *detachedTab, at int) int {
	if dt == nil {
		return -1
	}
	ts.mu.Lock()
	fr := ts.getFrame()
	at = min(max(at, 0), len(fr.Children))
	// its name is unique in these tabs, whose tabs are found by name
	label := dt.label
	for n := 2; fr.ChildByName(label) != nil; n++ {
		label = fmt.Sprintf("%s-%d", dt.label, n)
	}
	dt.label = label
	dt.frame.SetName(label)
	fr.AddChild(dt.frame) // (in the scene of these tabs)
	moveChild(fr, len(fr.Children)-1, at)
	nt := ts.insertTabButtonAt(dt.label, at)
	if dt.text != "" { // (as shown, which may have changed from its label)
		nt.SetText(dt.text)
	}
	nt.SetIcon(dt.icon)
	nt.SetTooltip(dt.tooltip)
	if len(fr.Children) > 1 && fr.StackTop >= at {
		fr.StackTop++ // (it is after the new tab)
	}
	ts.mu.Unlock()
	ts.SelectTabIndex(at)
	ts.Update()
	if ts.MoveInFunc != nil {
		ts.MoveInFunc(at)
	}
	return at
}

// transferTab moves the tab of the given button of the given tabs to the
// given index in the other given tabs, which may be in another window,
// from the event loop of the given widget. Each window only has its own
// lock, so the tab is removed in the event loop of the window of its tabs,
// and then added in that of the other tabs (see [WidgetBase.runInWindow]).
// A window whose last tab is dragged into another window is closed.
func transferTab(from *WidgetBase, src *Tabs, tab *Tab, dst *Tabs, at int) {
	src.runInWindow(from, func() {
		idx := tab.IndexInParent()
		if idx < 0 || tab.tabs() != src {
			return
		}
		srw, drw := src.Scene.RenderWindow(), dst.Scene.RenderWindow()
		if srw == drw {
			src.MoveTabTo(idx, dst, at)
			return
		}
		dt := src.detachTab(idx, dst, at)
		dst.runInWindow(src.AsWidget(), func() { dst.attachTab(dt, at) })
		if src.NumTabs() == 0 && src.DragToWindow != nil {
			src.Scene.Close()
		}
	})
}

// isParentOf returns whether the given node is in the given parent node.
func isParentOf(parent tree.Node, n tree.Node) bool {
	for p := n.AsTree().Parent; p != nil; p = p.AsTree().Parent {
		if p == parent {
			return true
		}
	}
	return false
}

// tabDrag is the data of the drag of a tab.
type tabDrag struct {
	// tabs are the tabs of the dragged tab.
	tabs *Tabs

	// tab is the button of the dragged tab.
	tab *Tab
}

// dropAt moves the dragged tab to the given index in the given tabs, into
// which it was dropped: in its own tabs, it moves there (the index that it
// then has), and from other tabs, it is inserted there, also from those
// of another window, as a drag of the operating system.
func (td *tabDrag) dropAt(ts *Tabs, at int) {
	if td.tabs == ts {
		if from := td.tab.IndexInParent(); from >= 0 {
			ts.MoveTabIndex(from, min(at, ts.NumTabs()-1))
		}
		return
	}
	if ts.Type.effective(ts) == FunctionalTabs {
		transferTab(ts.AsWidget(), td.tabs, td.tab, ts, at)
	}
}

// dropIndex returns the index in the tabs at which a tab dropped at the
// given position in the window goes: before the first tab whose middle it
// is before, in the row of tabs that it is in, or that is in a later row
// (as the tabs wrap), or at the end.
func (ts *Tabs) dropIndex(pos image.Point) int {
	n := ts.NumTabs()
	tb := ts.getTabs()
	for i := range n {
		bb := AsWidget(tb.Child(i)).Geom.TotalBBox
		if pos.Y < bb.Min.Y || pos.Y < bb.Max.Y && pos.X < (bb.Min.X+bb.Max.X)/2 {
			return i
		}
	}
	return n
}

// initStripDrop makes the given list of tabs a target of drops of tabs,
// for [FunctionalTabs]: one dropped between tabs goes there, and one
// dropped after them goes to the end (see [Tabs.dropIndex]).
func (ts *Tabs) initStripDrop(w *Frame) {
	w.On(events.DragEnter, func(e events.Event) { e.SetHandled() })
	w.On(events.DragLeave, func(e events.Event) { e.SetHandled() })
	w.On(events.Drop, func(e events.Event) {
		td, ok := e.(*events.DragDrop).Data.(*tabDrag)
		if !ok {
			return
		}
		e.SetHandled()
		td.dropAt(ts, ts.dropIndex(e.Pos()))
	})
	w.FinalStyler(func(s *styles.Style) {
		s.SetAbilities(ts.Type.effective(ts) == FunctionalTabs, abilities.Droppable)
	})
}

// initDrag makes the tab draggable, and a target of drops of tabs, for
// [FunctionalTabs].
func (tb *Tab) initDrag() {
	tb.FinalStyler(func(s *styles.Style) {
		s.SetAbilities(tb.Type.effective(tb) == FunctionalTabs, abilities.Draggable, abilities.Droppable)
	})
	tb.On(events.DragStart, func(e events.Event) {
		ts := tb.tabs()
		if ts == nil || !tb.AbilityIs(abilities.Draggable) {
			return
		}
		e.SetHandled()
		tb.Scene.Events.DragStart(tb.This.(Widget), &tabDrag{tabs: ts, tab: tb}, e)
	})
	tb.On(events.DragEnter, func(e events.Event) {
		if draggingTab(tb.Scene) {
			tb.SetState(true, states.DragHovered)
			tb.NeedsRender()
			e.SetHandled()
		}
	})
	tb.On(events.DragLeave, func(e events.Event) {
		tb.SetState(false, states.DragHovered)
		tb.NeedsRender()
		e.SetHandled()
	})
	tb.On(events.Drop, func(e events.Event) {
		td, ok := e.(*events.DragDrop).Data.(*tabDrag)
		ts := tb.tabs()
		if !ok || ts == nil {
			return
		}
		e.SetHandled()
		td.dropAt(ts, tb.IndexInParent())
	})
}

// draggingTab returns whether a tab is being dragged over the given scene:
// within its window, or from another window of the app, by the operating
// system.
func draggingTab(sc *Scene) bool {
	if _, ok := sc.Events.dragData.(*tabDrag); ok {
		return true
	}
	osDrag.Lock()
	defer osDrag.Unlock()
	_, ok := osDrag.data.(*tabDrag)
	return ok
}

// dragMovesDefault returns that a drag of a tab moves it (see [dragMover]).
func (tb *Tab) dragMovesDefault() bool { return true }

// osDragData returns the data of a drag of a tab of tabs with
// [Tabs.DragToWindow] that leaves its window, which the operating system
// then drags (see [osDragger]), to be dropped into the tabs of another
// window of the app, or into a new window where nothing accepts it (see
// [Tab.droppedNowhere]), and nil for other tabs.
func (tb *Tab) osDragData() mimedata.Mimes {
	ts := tb.tabs()
	if ts == nil || ts.DragToWindow == nil || ts.Type.effective(ts) != FunctionalTabs {
		return nil
	}
	return mimedata.Mimes{{Type: tabMimeType, Data: []byte(tb.Name)}}
}

// droppedNowhere moves the tab into a new window when it was dragged out of
// its window by the operating system and dropped where nothing accepted it
// (as onto the desktop), at the given position on the screen, with its tab
// under the drop, as in web browsers (see [osDropper]). The window of a last
// tab moves there instead.
func (tb *Tab) droppedNowhere(e events.Event, screen image.Point) {
	ts := tb.tabs()
	if ts == nil || ts.DragToWindow == nil {
		return
	}
	src := ts.Scene.RenderWindow()
	idx := tb.IndexInParent()
	if src == nil || src.SystemWindow == nil || idx < 0 {
		return
	}
	dpr := screenPixelRatio(src.SystemWindow)
	if ts.NumTabs() == 1 {
		// the place of the press in the window goes to the drop
		src.SystemWindow.SetPos(screen.Sub(scalePoint(e.StartPos(), 1/dpr)), nil)
		return
	}
	// the place of the press in the tab, which goes to the drop, as in the
	// new window (initialized as this one) the tabs start where they do here
	in := e.StartPos().Sub(tb.Geom.TotalBBox.Min)
	start := ts.getTabs().Geom.TotalBBox.Min
	// (moved into the new tabs before their window runs)
	// (titled as the tab is shown, not by its name, which is unique, as
	// "New tab-4" for a tab of a web browser that was new)
	title := tb.Text
	if title == "" {
		title = tb.Name
	}
	bd := NewBody(title)
	nt := ts.DragToWindow(bd)
	if nt == nil {
		return
	}
	if nt.DragToWindow == nil { // (so that it closes when its tab is dragged away)
		nt.DragToWindow = ts.DragToWindow
	}
	if ts.MoveTabTo(idx, nt, nt.NumTabs()) < 0 {
		return
	}
	bd.RunWindow()
	if win := bd.Scene.RenderWindow(); win != nil && win.SystemWindow != nil {
		// of the size of the window of the tabs, as in web browsers
		pos := screen.Sub(scalePoint(start.Add(in), 1/dpr))
		win.SystemWindow.SetGeometry(false, pos, src.SystemWindow.Size(), nil)
		win.SystemWindow.Raise()
	}
}

// dropInScene moves the dragged tab, dropped by the operating system where
// nothing in the given scene took it (as onto the content of a page), into
// the first tabs of the scene with [Tabs.DragToWindow], at their end, if
// it is in another window, as in web browsers.
func (td *tabDrag) dropInScene(sc *Scene) {
	if td.tabs.Scene.RenderWindow() == sc.RenderWindow() {
		return // (back in its own window)
	}
	var dst *Tabs
	sc.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
		if ts, ok := cw.(*Tabs); ok && dst == nil && ts.DragToWindow != nil && ts.Type.effective(ts) == FunctionalTabs {
			dst = ts
			return tree.Break
		}
		return dst == nil
	})
	if dst != nil {
		td.dropAt(dst, dst.NumTabs())
	}
}
