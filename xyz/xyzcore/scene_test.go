// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xyzcore

import (
	"image"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/xyz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestEditor returns a new scene editor in a new body, with a box.
// Note that the offscreen test driver has no GPU composer, so the scene
// is not rendered in the window: these tests are of the widget and its
// selection and navigation, not of its image.
func newTestEditor() (*core.Body, *SceneEditor, *xyz.Solid) {
	b := core.NewBody()
	sc := xyz.NewScene()
	xyz.NewAmbient(sc, "ambient", 1, xyz.DirectSun)
	box := xyz.NewBox(sc, "box", 1, 1, 1)
	sld := xyz.NewSolid(sc).SetMesh(box).SetColor(colors.Red)
	sld.SetName("solid")
	se := NewSceneEditorForScene(sc, b)
	return b, se, sld
}

// TestSceneEditor tests a scene editor, which has a toolbar and a scene.
func TestSceneEditor(t *testing.T) {
	b, se, _ := newTestEditor()
	core.RenderImage(b, core.Locked(b, func() {
		sw := se.SceneWidget()
		require.NotNil(t, sw)
		assert.Equal(t, sw.XYZ, se.SceneXYZ())
		assert.Equal(t, sw.XYZ, sw.SceneXYZ())
		var tb *core.Toolbar
		se.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
			if t, ok := cw.(*core.Toolbar); ok {
				tb = t
			}
			return tb == nil
		})
		require.NotNil(t, tb, "a toolbar")
		assert.Greater(t, tb.NumChildren(), 3)
	}))
}

// TestSceneSelection tests selecting nodes of the scene, with a
// selection box and with manipulation points.
func TestSceneSelection(t *testing.T) {
	b, se, sld := newTestEditor()
	core.RenderImage(b, core.Locked(b, func() {
		sw := se.SceneWidget()
		sw.XYZ.Update()
		xyz.UpdateWorldMatrix(sw.XYZ.This)
		sw.XYZ.UpdateMeshBBox()

		sw.SelectionMode = NotSelectable
		sw.SetSelected(sld)
		assert.Nil(t, sw.CurrentSelected, "not selectable")

		sw.SelectionMode = SelectionBox
		sw.SetSelected(sld)
		assert.Equal(t, xyz.Node(sld), sw.CurrentSelected)
		assert.NotNil(t, sw.XYZ.ChildByName(SelectedBoxName, 0), "a selection box")

		sw.SetSelected(nil)
		assert.Nil(t, sw.CurrentSelected)
		assert.Nil(t, sw.XYZ.ChildByName(SelectedBoxName, 0))

		sw.SelectionMode = Manipulable
		sw.SetSelected(sld)
		mb := sw.XYZ.ChildByName(ManipBoxName, 0)
		require.NotNil(t, mb, "a manipulation box")
		var mp *ManipPoint
		mb.AsTree().WalkDown(func(n tree.Node) bool {
			if p, ok := n.(*ManipPoint); ok && mp == nil {
				mp = p
			}
			return true
		})
		require.NotNil(t, mp, "manipulation points")
		sw.SetSelected(mp)
		assert.Equal(t, mp, sw.CurrentManipPoint)
		assert.Equal(t, xyz.Node(sld), sw.CurrentSelected, "the node stays selected")
	}))
}

// TestSceneKeys tests navigating the camera of the scene with keys.
func TestSceneKeys(t *testing.T) {
	b, se, _ := newTestEditor()
	core.RenderImage(b, core.Locked(b, func() {
		sw := se.SceneWidget()
		pos := sw.XYZ.Camera.Pose.Pos
		core.SendKey(sw, 0, key.CodeLeftArrow)
		assert.NotEqual(t, pos, sw.XYZ.Camera.Pose.Pos, "orbits")
		pos = sw.XYZ.Camera.Pose.Pos
		sw.HandleEvent(events.NewScroll(image.Point{}, math32.Vec2(0, 1), 0))
		assert.NotEqual(t, pos, sw.XYZ.Camera.Pose.Pos, "zooms")
	}))
}
