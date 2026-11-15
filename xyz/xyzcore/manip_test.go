// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xyzcore

import (
	"image"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/gpu"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/xyz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// offscreenXYZ renders the 3D scene of the given scene widget offscreen,
// with a GPU, as the offscreen test driver has no GPU composer, so that
// its nodes have their scene bounding boxes for selecting them.
func offscreenXYZ(t *testing.T, sw *Scene) {
	t.Helper()
	gp, dev, err := gpu.NoDisplayGPU()
	require.NoError(t, err)
	xy := sw.XYZ
	xy.SetSize(sw.frameBBox().Size())
	xy.ConfigOffscreen(gp, dev)
	t.Cleanup(xy.Destroy)
	xy.Rebuild()
	require.NotNil(t, xy.RenderGrabImage())
}

// sceneEvent returns a mouse event of the given type at the given point
// of the 3D scene of the given scene widget.
func sceneEvent(sw *Scene, typ events.Types, pt, prev image.Point, mods key.Modifiers) events.Event {
	off := sw.frameBBox().Min
	e := events.NewMouseMove(events.Left, pt.Add(off), prev.Add(off), mods)
	e.Typ = typ
	return e
}

// TestSceneSelectAndManipulate tests selecting a node by clicking on it
// in the scene, and moving, scaling, and rotating it by sliding its
// manipulation points, and sliding the scene to orbit the camera.
func TestSceneSelectAndManipulate(t *testing.T) {
	gpu.SkipIfNoGPU(t)
	b, se, sld := newTestEditor()
	core.RenderImage(b, core.Locked(b, func() {
		sw := se.SceneWidget()
		offscreenXYZ(t, sw)
		bb := sld.SceneBBox
		require.False(t, bb.Empty())
		ctr := bb.Min.Add(bb.Max).Div(2)

		sw.SelectionMode = Manipulable
		sw.HandleEvent(sceneEvent(sw, events.MouseDown, ctr, ctr, 0))
		assert.Equal(t, xyz.Node(sld), sw.CurrentSelected, "clicked on")
		sw.HandleEvent(sceneEvent(sw, events.MouseDown, image.Pt(1, 1), image.Pt(1, 1), 0))
		assert.Nil(t, sw.CurrentSelected, "clicked on nothing")

		sw.SetSelected(sld)
		var mp *ManipPoint
		mb := sw.XYZ.ChildByName(ManipBoxName, 0)
		require.NotNil(t, mb)
		mb.AsTree().WalkDown(func(n tree.Node) bool {
			if p, ok := n.(*ManipPoint); ok && mp == nil {
				mp = p
			}
			return true
		})
		require.NotNil(t, mp)
		sw.SetSelected(mp)
		require.Equal(t, mp, sw.CurrentManipPoint)

		slide := func(ms key.Modifiers) {
			sw.HandleEvent(sceneEvent(sw, events.SlideMove, image.Pt(60, 55), image.Pt(50, 50), ms))
		}
		pos := sld.Pose.Pos
		slide(0)
		assert.NotEqual(t, pos, sld.Pose.Pos, "moved")
		scale := sld.Pose.Scale
		slide(mods(key.Control))
		assert.NotEqual(t, scale, sld.Pose.Scale, "scaled")
		quat := sld.Pose.Quat
		slide(mods(key.Alt))
		assert.NotEqual(t, quat, sld.Pose.Quat, "rotated")

		// with the camera looking down the other axes
		for _, eye := range []math32.Vector3{math32.Vec3(10, 0, 0), math32.Vec3(0, 10, 0.01)} {
			sw.XYZ.Camera.Pose.Pos = eye
			sw.XYZ.Camera.LookAtOrigin()
			pos = sld.Pose.Pos
			slide(0)
			assert.NotEqual(t, pos, sld.Pose.Pos, "moved from %v", eye)
			slide(mods(key.Alt))
		}

		sw.SetSelected(nil)
		cam := sw.XYZ.Camera.Pose.Pos
		slide(0)
		assert.NotEqual(t, cam, sw.XYZ.Camera.Pose.Pos, "orbits without a selection")
	}))
}

// mods returns the modifiers of the given keys.
func mods(ks ...key.Modifiers) key.Modifiers {
	var m key.Modifiers
	for _, k := range ks {
		m.SetFlag(true, k)
	}
	return m
}

// TestMeshButton tests the button of a mesh name value.
func TestMeshButton(t *testing.T) {
	b := core.NewBody()
	mb := NewMeshButton(b)
	core.RenderImage(b, core.Locked(b, func() {
		assert.Equal(t, "Select mesh", mb.Text)
		assert.Equal(t, &mb.MeshName, mb.WidgetValue())
		mb.SetMeshName("box")
		mb.Update()
		assert.Equal(t, "box", mb.Text)
	}))
}

// TestSceneSelectCycle tests that clicking where several nodes are cycles
// through selecting them.
func TestSceneSelectCycle(t *testing.T) {
	gpu.SkipIfNoGPU(t)
	b, se, sld := newTestEditor()
	back := xyz.NewSolid(sld.Scene).SetMesh(sld.Mesh).SetPos(0, 0, -2)
	back.SetName("back")
	far := xyz.NewSolid(sld.Scene).SetMesh(sld.Mesh).SetPos(3, 3, 0)
	far.SetName("far")
	core.RenderImage(b, core.Locked(b, func() {
		sw := se.SceneWidget()
		offscreenXYZ(t, sw)
		bb := sld.SceneBBox
		ctr := bb.Min.Add(bb.Max).Div(2)
		click := func() xyz.Node {
			sw.HandleEvent(sceneEvent(sw, events.DoubleClick, ctr, ctr, 0))
			return sw.CurrentSelected
		}
		sw.SelectionMode = Selectable
		first := click()
		require.NotNil(t, first)
		second := click()
		assert.NotEqual(t, first, second, "the next one")
		assert.Equal(t, first, click(), "back to the first")
		sw.SetSelected(far) // not under the point
		assert.Equal(t, first, click(), "the first one")
	}))
}
