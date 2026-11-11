// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/tree"
)

// toolbarDrag is the data of the drag of an item of a [Toolbar] with a
// [Toolbar.MoveFunc].
type toolbarDrag struct {
	toolbar *Toolbar
	item    Widget
}

// dragMovesDefault returns that a drag of an item of the toolbar moves it
// (see [dragMover]).
func (tb *Toolbar) dragMovesDefault() bool { return tb.MoveFunc != nil }

// itemIndex returns the index of the given item among the items of the
// toolbar, or -1 if it is not one of them (as the overflow menu).
func (tb *Toolbar) itemIndex(w Widget) int {
	if w.AsTree().Parent != tb.This || w.AsTree().Name == "overflow-menu" {
		return -1
	}
	return w.AsTree().IndexInParent()
}

// initItemDrag makes the given new item of the toolbar draggable to other
// places in it, and a target of drops of its other items, if the toolbar
// has a [Toolbar.MoveFunc].
func (tb *Toolbar) initItemDrag(n tree.Node) {
	w, ok := n.(Widget)
	if !ok || n.AsTree().Name == "overflow-menu" {
		return
	}
	wb := w.AsWidget()
	wb.FinalStyler(func(s *styles.Style) {
		if tb.MoveFunc != nil {
			s.SetAbilities(true, abilities.Draggable, abilities.Droppable)
		}
	})
	dragged := func() *toolbarDrag {
		td, ok := tb.Scene.Events.dragData.(*toolbarDrag)
		if !ok || td.toolbar != tb {
			return nil
		}
		return td
	}
	wb.On(events.DragStart, func(e events.Event) {
		if tb.MoveFunc == nil || tb.itemIndex(w) < 0 {
			return
		}
		e.SetHandled()
		tb.Scene.Events.DragStart(w, &toolbarDrag{toolbar: tb, item: w}, e)
	})
	wb.On(events.DragEnter, func(e events.Event) {
		if dragged() != nil {
			wb.SetState(true, states.DragHovered)
			wb.NeedsRender()
			e.SetHandled()
		}
	})
	wb.On(events.DragLeave, func(e events.Event) {
		wb.SetState(false, states.DragHovered)
		wb.NeedsRender()
	})
	wb.On(events.Drop, func(e events.Event) {
		td, ok := e.(*events.DragDrop).Data.(*toolbarDrag)
		if !ok || td.toolbar != tb || tb.MoveFunc == nil {
			return
		}
		e.SetHandled()
		from, to := tb.itemIndex(td.item), tb.itemIndex(w)
		if from < 0 || to < 0 || from == to {
			return
		}
		tb.MoveFunc(from, to)
		tb.Update()
	})
}
