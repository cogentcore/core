// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xyz_test

import (
	"image"
	"image/color"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/gpu"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/tree"
	. "cogentcore.org/core/xyz"
	"cogentcore.org/core/xyz/examples/assets"
	_ "cogentcore.org/core/xyz/io/obj"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newRenderScene returns a new offscreen scene of a small size, with an
// ambient light, and a camera looking at the origin from z = 10.
func newRenderScene(t *testing.T) *Scene {
	gpu.SkipIfNoGPU(t)
	gp, dev, err := gpu.NoDisplayGPU()
	require.NoError(t, err)
	sc := NewScene().SetSize(image.Pt(160, 120))
	sc.Background = colors.Uniform(color.Black)
	sc.ConfigOffscreen(gp, dev)
	NewAmbient(sc, "ambient", 1, DirectSun)
	t.Cleanup(sc.Destroy)
	return sc
}

// render rebuilds and renders the scene, returning the image.
func render(t *testing.T, sc *Scene) *image.NRGBA {
	sc.Rebuild()
	img := sc.RenderGrabImage()
	require.NotNil(t, img)
	return img
}

// pixel returns the color at the given position.
func pixel(img image.Image, x, y int) color.RGBA {
	return color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)
}

// dominant asserts that the given channel (0 = R, 1 = G, 2 = B) of the
// color is clearly larger than the others.
func dominant(t *testing.T, c color.RGBA, ch int, msg string) {
	t.Helper()
	v := []uint8{c.R, c.G, c.B}
	for i, o := range v {
		if i != ch {
			assert.Greater(t, int(v[ch]), int(o)+60, "%s: %v", msg, c)
		}
	}
}

// TestRenderSolids tests rendering solids, and finding the nodes under a
// point after rendering.
func TestRenderSolids(t *testing.T) {
	sc := newRenderScene(t)
	box := NewBox(sc, "box", 1, 1, 1)
	sphere := NewSphere(sc, "sphere", 0.6, 16)
	var red, blue *Solid
	tree.AddChild(sc, func(n *Solid) {
		red = n
		n.SetName("red")
		n.SetMesh(box).SetColor(colors.Red).SetReflective(0).SetPos(-1.5, 0, 0)
	})
	tree.AddChild(sc, func(g *Group) {
		g.SetName("group")
		g.SetPos(1.5, 0, 0)
		tree.AddChild(g, func(n *Solid) {
			blue = n
			n.SetName("blue")
			n.SetMesh(sphere).SetColor(colors.Blue).SetReflective(0)
		})
	})
	img := render(t, sc)
	dominant(t, pixel(img, 50, 60), 0, "red box on the left")
	dominant(t, pixel(img, 110, 60), 2, "blue sphere on the right")
	assert.Equal(t, color.RGBA{0, 0, 0, 255}, pixel(img, 80, 5), "background")

	under := NodesUnderPoint(sc, image.Pt(50, 60))
	assert.Contains(t, under, Node(red))
	assert.NotContains(t, under, Node(blue))
	under = NodesUnderPoint(sc, image.Pt(110, 60))
	assert.Contains(t, under, Node(blue))
	assert.Empty(t, NodesUnderPoint(sc, image.Pt(80, 5)))

	// a ray from the camera through the center of the red box hits it
	ray := red.RayPick(image.Pt(50, 60))
	_, hit := ray.IntersectBox(math32.Box3{Min: math32.Vec3(-0.5, -0.5, -0.5), Max: math32.Vec3(0.5, 0.5, 0.5)})
	assert.True(t, hit, "ray %v", ray)
	grp := sc.ChildByName("group", 0).(*Group)
	// a world ray from the camera to the blue sphere
	wray := math32.NewRay(math32.Vec3(0, 0, 10), math32.Vec3(1.5, 0, -10).Normal())
	sps := grp.RaySolidIntersections(*wray)
	require.Len(t, sps, 1)
	assert.Equal(t, blue, sps[0].Solid)
	assert.InDelta(t, 0.6, sps[0].Point.Z, 0.1, "hits the front of its bounding box")
	assert.Empty(t, grp.RaySolidIntersections(*math32.NewRay(math32.Vec3(0, 0, 10), math32.Vec3(0, 1, 0))))

	assert.Len(t, sc.SolidsIntersectingPoint(image.Pt(50, 60)), 2)
	assert.NoError(t, sc.Validate())
	assert.Contains(t, sc.MeshList(), "box")
	assert.True(t, red.IsVisible())
	assert.False(t, red.IsTransparent())
}

// TestRenderTransparent tests that a transparent solid shows what is
// behind it.
func TestRenderTransparent(t *testing.T) {
	sc := newRenderScene(t)
	box := NewBox(sc, "box", 1, 1, 1)
	tree.AddChild(sc, func(n *Solid) {
		n.SetMesh(box).SetColor(colors.Red).SetReflective(0).SetScale(4, 4, 1).SetPos(0, 0, -2)
	})
	var front *Solid
	tree.AddChild(sc, func(n *Solid) {
		front = n
		n.SetMesh(box).SetColor(color.RGBA{0, 0, 128, 128}).SetReflective(0)
	})
	img := render(t, sc)
	assert.True(t, front.IsTransparent())
	c := pixel(img, 80, 60)
	assert.Greater(t, int(c.R), 60, "the red behind shows through: %v", c)
	assert.Greater(t, int(c.B), 60, "the blue in front shows: %v", c)
}

// TestRenderLightsAndCameras tests directional, point, and spot lights,
// and saving and restoring cameras.
func TestRenderLightsAndCameras(t *testing.T) {
	sc := newRenderScene(t)
	plane := NewPlane(sc, "plane", 4, 4)
	var pl *Solid
	tree.AddChild(sc, func(n *Solid) {
		pl = n
		n.SetMesh(plane).SetColor(colors.White).SetReflective(0).SetAxisRotation(1, 0, 0, 90) // facing +z
	})
	brightness := func() int {
		c := pixel(render(t, sc), 80, 60)
		return int(c.R) + int(c.G) + int(c.B)
	}
	sc.Lights.Reset()
	NewAmbient(sc, "ambient", 0.2, DirectSun)
	amb := brightness()

	NewDirectional(sc, "dir", 1, DirectSun).Pos.Set(0, 0, 1)
	assert.Greater(t, brightness(), amb+30, "directional light in front")

	sc.Lights.Reset()
	NewAmbient(sc, "ambient", 0.2, DirectSun)
	pt := NewPoint(sc, "point", 1, DirectSun)
	pt.Pos.Set(0, 0, 2)
	assert.Greater(t, brightness(), amb+30, "point light")

	sc.Lights.Reset()
	NewAmbient(sc, "ambient", 0.2, DirectSun)
	sp := NewSpot(sc, "spot", 1, DirectSun)
	sp.Pose.Pos.Set(0, 0, 3)
	sp.LookAtOrigin()
	assertNear(t, math32.Vec3(0, 0, -1), sp.ViewDir(), "spot points at the origin")
	assert.Greater(t, brightness(), amb+30, "spot light")
	sp.LookAt(math32.Vec3(0, 0, 10), math32.Vec3(0, 1, 0)) // away
	assertNear(t, math32.Vec3(0, 0, 1), sp.ViewDir(), "spot points away")

	sc.SaveCamera("front")
	sc.Camera.Pose.Pos.Set(0, 0, -10)
	sc.Camera.LookAtOrigin()
	require.NoError(t, sc.SetCamera("front"))
	assertNear(t, math32.Vec3(0, 0, 10), sc.Camera.Pose.Pos, "restored camera")
	assert.Error(t, sc.SetCamera("none"))
	_ = pl
}

// assertNear asserts that two vectors are near each other.
func assertNear(t *testing.T, want, got math32.Vector3, msg string) {
	t.Helper()
	assert.InDelta(t, 0, want.Sub(got).Length(), 1e-3, "%s: want %v, got %v", msg, want, got)
}

// TestRenderTexture tests a solid with a texture, set by name.
func TestRenderTexture(t *testing.T) {
	sc := newRenderScene(t)
	NewTextureFileFS(assets.Content, sc, "wood", "wood.png")
	assert.Contains(t, sc.TextureList(), "wood")
	plane := NewPlane(sc, "plane", 4, 4)
	var pl *Solid
	tree.AddChild(sc, func(n *Solid) {
		pl = n
		n.SetMesh(plane).SetColor(colors.White).SetReflective(0).SetAxisRotation(1, 0, 0, 90).SetTextureName("wood")
	})
	// the wood texture is light gray, so it is darker than plain white
	textured := pixel(render(t, sc), 80, 60)
	assert.NotNil(t, pl.Material.Texture)
	pl.Material.NoTexture()
	assert.Nil(t, pl.Material.Texture)
	plain := pixel(sc.RenderGrabImage(), 80, 60)
	assert.Greater(t, int(plain.R), int(textured.R)+15, "textured %v, plain %v", textured, plain)
	assert.Error(t, pl.Material.SetTextureName(sc, "none"))
	assert.Contains(t, pl.Material.String(), "Color")
}

// TestRenderObjects tests opening objects into the library and adding
// them from it, and the alternative frame.
func TestRenderObjects(t *testing.T) {
	sc := newRenderScene(t)
	lib, err := sc.OpenToLibrary("testdata/tri.obj", "")
	require.NoError(t, err)
	assert.Equal(t, "tri", lib.Name)
	gp, err := sc.AddFromLibrary("tri", sc)
	require.NoError(t, err)
	assert.NotNil(t, gp)
	_, err = sc.AddFromLibrary("none", sc)
	assert.Error(t, err)

	img := render(t, sc)
	dominant(t, pixel(img, 80, 60), 0, "the red quad of the obj")

	gp2 := NewGroup(sc)
	require.NoError(t, sc.OpenObj("testdata/tri.obj", gp2))
	assert.Equal(t, 1, gp2.NumChildren())
	assert.Error(t, sc.OpenObj("testdata/none.obj", gp2))

	sc.UseAltFrame(image.Pt(40, 30))
	sc.Rebuild()
	alt := sc.RenderGrabImage()
	require.NotNil(t, alt)
	assert.Equal(t, image.Rect(0, 0, 40, 30), alt.Bounds())
	sc.UseMainFrame()
	assert.Equal(t, image.Rect(0, 0, 160, 120), render(t, sc).Bounds())
}

// TestNavKeys tests navigating the camera with the keyboard.
func TestNavKeys(t *testing.T) {
	sc := NewScene()
	sc.Camera.Defaults()
	sc.Camera.UpdateMatrix()
	pos := sc.Camera.Pose.Pos
	for _, k := range []struct {
		code key.Codes
		mods key.Modifiers
	}{
		{key.CodeUpArrow, 0}, {key.CodeDownArrow, 0}, {key.CodeLeftArrow, 0}, {key.CodeRightArrow, 0},
	} {
		e := events.NewKey(events.KeyChord, 0, k.code, k.mods)
		sc.KeyChordEvent(e)
		assert.True(t, e.IsHandled(), "%v", k.code)
	}
	assert.NotEqual(t, pos, sc.Camera.Pose.Pos, "orbited")
	var mods key.Modifiers
	mods.SetFlag(true, key.Shift)
	e := events.NewKey(events.KeyChord, 0, key.CodeUpArrow, mods)
	sc.KeyChordEvent(e)
	assert.True(t, e.IsHandled(), "shift pans")

	sc.NoNav = true
	e = events.NewKey(events.KeyChord, 0, key.CodeUpArrow, 0)
	sc.KeyChordEvent(e)
	assert.False(t, e.IsHandled(), "no navigation")
}
