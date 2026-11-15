// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDialogMoveResize tests moving and resizing a dialog in its window by
// dragging its move and resize handles, which keep sliding when the mouse
// leaves the dialog.
func TestDialogMoveResize(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(500)) })
	var st *Stage
	var pos0, pos1 image.Point
	var size0 image.Point
	handle := func(name string) image.Point {
		b.AsyncLock()
		defer b.AsyncUnlock()
		h := st.Scene.Parts.ChildByName(name)
		require.NotNil(t, h, name)
		bb := h.(Widget).AsWidget().Geom.TotalBBox
		return bb.Min.Add(bb.Max).Div(2).Add(st.Scene.SceneGeom.Pos) // the center
	}
	RenderImage(b, Locked(b, func() { // (the image is not compared: the assertions check the geometry)
		d := NewBody("Dialog")
		NewText(d).SetText("Move and resize me")
		st = d.NewDialog(b).Run()
	}), Locked(b, func() {
		pos0 = st.Scene.SceneGeom.Pos
		size0 = st.Scene.SceneGeom.Size
	}), func() {
		from := handle("move")
		DragMouse(b, from, from.Add(image.Pt(-40, -30)))
	}, Locked(b, func() {
		pos1 = st.Scene.SceneGeom.Pos
		assert.Less(t, pos1.X, pos0.X, "moved left")
		assert.Less(t, pos1.Y, pos0.Y, "moved up")
	}), func() {
		from := handle("resize")
		DragMouse(b, from, from.Add(image.Pt(30, 20)))
	}, Locked(b, func() {
		assert.Greater(t, st.Scene.SceneGeom.Size.X, size0.X, "wider")
		assert.Greater(t, st.Scene.SceneGeom.Size.Y, size0.Y, "taller")
	}), func() {
		from := handle("move")
		DragMouse(b, from, from.Add(image.Pt(-100, -60))) // beyond the left edge, quickly
	}, Locked(b, func() {
		pos := st.Scene.SceneGeom.Pos
		assert.Equal(t, 0, pos.X, "kept in the window")
		assert.Less(t, pos.Y, pos1.Y, "moved up")
		assert.GreaterOrEqual(t, pos.Y, 0, "kept in the window")
	}))
}

// TestTopScene tests getting the scene on top of a window: the window,
// a dialog in it, and a menu over the dialog.
func TestTopScene(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(300)) })
	RenderImage(b, Locked(b, func() {
		assert.Equal(t, b.Scene, TopScene(b))
		d := NewBody("Dialog")
		bt := NewButton(d).SetText("Menu")
		bt.SetMenu(func(m *Scene, _ image.Point) { NewButton(m).SetText("Item") })
		d.AddBottomBar(func(bar *Frame) { d.AddOK(bar) })
		d.RunDialog(b)
	}), Locked(b, func() {
		top := TopScene(b)
		require.NotNil(t, top)
		assert.Equal(t, "Dialog", top.Body.Title)
		bt := ButtonByText(top, "Menu")
		require.NotNil(t, bt)
		bt.Send(events.Click)
	}), Locked(b, func() {
		top := TopScene(b)
		require.NotNil(t, ButtonByText(top, "Item"), "the menu")
		assert.Nil(t, ButtonByText(top, "Menu"))
	}))
}
