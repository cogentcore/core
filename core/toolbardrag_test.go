// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"slices"
	"testing"
	"time"

	"cogentcore.org/core/events"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
)

// TestToolbarMoveFunc tests that an item of a toolbar with a MoveFunc that
// is dragged onto another one moves to its place, with the items made
// from the moved data, and that the items of a toolbar without one are
// not draggable.
func TestToolbarMoveFunc(t *testing.T) {
	prev := TimingSettings.DragStartTime
	TimingSettings.DragStartTime = 0
	defer func() { TimingSettings.DragStartTime = prev }()
	names := []string{"One", "Two", "Three"}
	b := NewBody()
	tb := NewToolbar(b)
	tb.Maker(func(p *tree.Plan) {
		for _, n := range names {
			tree.AddAt(p, n, func(w *Button) { w.SetText(n) })
		}
	})
	var moves [][2]int
	tb.MoveFunc = func(from, to int) {
		moves = append(moves, [2]int{from, to})
		n := names[from]
		names = slices.Delete(names, from, from+1)
		names = slices.Insert(names, to, n)
	}
	texts := func() []string {
		var ts []string
		for _, k := range tb.Children {
			if bt, ok := k.(*Button); ok && bt.Name != "overflow-menu" {
				ts = append(ts, bt.Text)
			}
		}
		return ts
	}
	center := func(i int) image.Point {
		bb := AsWidget(tb.Child(i)).Geom.TotalBBox
		return bb.Min.Add(bb.Size().Div(2))
	}
	b.RenderImage(func() {
		b.AsyncLock()
		from, to := center(0), center(2)
		assert.True(t, AsWidget(tb.Child(0)).AbilityIs(abilities.Draggable))
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
		assert.Equal(t, [][2]int{{0, 2}}, moves)
		assert.Equal(t, []string{"Two", "Three", "One"}, texts())
		tb.MoveFunc = nil
		tb.Restyle()
	}, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		assert.False(t, AsWidget(tb.Child(0)).AbilityIs(abilities.Draggable), "not without a MoveFunc")
	})
}
