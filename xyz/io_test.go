// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xyz_test

import (
	"bytes"
	"image/color"
	"io"
	"os"
	"testing"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/tree"
	. "cogentcore.org/core/xyz"
	_ "cogentcore.org/core/xyz/io/obj"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOpenRead tests opening and reading objects and scenes from files and
// readers, without a GPU.
func TestOpenRead(t *testing.T) {
	sc := NewScene()
	dec, err := DecodeFile("testdata/tri.obj")
	require.NoError(t, err)
	assert.NotNil(t, dec)
	_, err = DecodeFile("testdata/tri.png")
	assert.Error(t, err, "no decoder")
	_, err = DecodeFile("testdata/none.obj")
	assert.Error(t, err)

	gp, err := sc.OpenNewObj("testdata/tri.obj", sc)
	require.NoError(t, err)
	assert.Equal(t, "tri.obj", gp.Name)
	assert.Equal(t, 1, gp.NumChildren(), "the quad")
	_, err = sc.OpenNewObj("testdata/none.obj", sc)
	assert.Error(t, err)

	n := sc.NumChildren()
	require.NoError(t, sc.OpenScene("testdata/tri.obj"))
	assert.Equal(t, n+1, sc.NumChildren(), "a new group")
	assert.Error(t, sc.OpenScene("testdata/none.obj"))

	readers := func() []io.Reader {
		obj, err := os.ReadFile("testdata/tri.obj")
		require.NoError(t, err)
		mtl, err := os.ReadFile("testdata/tri.mtl")
		require.NoError(t, err)
		return []io.Reader{bytes.NewReader(obj), bytes.NewReader(mtl)}
	}
	rg := NewGroup(sc)
	require.NoError(t, sc.ReadObj("embedded.obj", readers(), rg))
	assert.Equal(t, 1, rg.NumChildren())
	assert.Error(t, sc.ReadObj("embedded.xyz", readers(), rg))
	assert.Error(t, sc.ReadObj("embedded.obj", nil, rg), "no readers")

	n = sc.NumChildren()
	require.NoError(t, sc.ReadScene("embedded.obj", readers(), nil))
	assert.Equal(t, n+1, sc.NumChildren())
	assert.Error(t, sc.ReadScene("embedded.xyz", readers(), nil))
	assert.Error(t, sc.ReadScene("embedded.obj", nil, nil))

	_, err = sc.OpenToLibrary("testdata/none.obj", "")
	assert.Error(t, err)
}

// TestLineBox tests the meshes and solids of a box drawn with lines.
func TestLineBox(t *testing.T) {
	sc := NewScene()
	bbox := math32.Box3{Min: math32.Vec3(-1, -2, -3), Max: math32.Vec3(1, 2, 3)}
	front, side := NewLineBoxMeshes(sc, "box", bbox, 0.1)
	assert.Equal(t, "box-front", front.Name)
	assert.Equal(t, "box-side", side.Name)
	assert.Contains(t, sc.MeshList(), "box-front")
	nv, ni, _ := front.MeshSize()
	assert.Positive(t, nv)
	assert.Positive(t, ni)

	var g *Group
	tree.AddChild(sc, func(n *Group) {
		g = n
		InitLineBox(n, "lbox", bbox, 0.1, color.RGBA{255, 0, 0, 255}, Active)
	})
	sc.Update()
	assert.Equal(t, math32.Vec3(0, 0, 0), g.Pose.Pos, "at the center")
	require.Equal(t, 4, g.NumChildren(), "back, left, right, and front")
	for _, k := range g.Children {
		sld := k.(*Solid)
		assert.Equal(t, uint8(255), sld.Material.Color.R)
	}
	assert.Equal(t, float32(-3), g.Child(0).(*Solid).Pose.Pos.Z, "the back")
}

// TestNodeSetters tests the setters of solids, groups, and nodes, lights
// in view coordinates, and tracking a light.
func TestNodeSetters(t *testing.T) {
	sc := NewScene()
	sc.Camera.Defaults()
	box := NewBox(sc, "box", 1, 1, 1)
	var sld *Solid
	var g *Group
	tree.AddChild(sc, func(n *Group) {
		g = n
		tree.AddChild(n, func(s *Solid) { sld = s })
	})
	sc.Update()
	g.Defaults()
	g.SetEulerRotation(0, 90, 0)
	assert.Equal(t, RClassNone, g.RenderClass())

	require.NoError(t, sld.SetMeshName("box"))
	assert.Equal(t, Mesh(box), sld.Mesh)
	assert.Error(t, sld.SetMeshName("none"))
	assert.NoError(t, sld.SetMeshName(""))
	sld.SetEmissive(color.RGBA{1, 2, 3, 255}).SetBright(1.5).SetEulerRotation(90, 0, 0)
	assert.Equal(t, color.RGBA{1, 2, 3, 255}, sld.Material.Emissive)
	assert.Equal(t, float32(1.5), sld.Material.Bright)
	assert.Nil(t, sld.ParentMaterial(), "a group has no material")
	top := NewSolid(sc)
	assert.Nil(t, top.ParentMaterial(), "the scene has no material")
	inner := NewSolid(top)
	assert.Equal(t, &top.Material, inner.ParentMaterial())
	assert.Equal(t, sld, sld.AsSolid())
	assert.Nil(t, g.AsSolid())
	assert.False(t, g.IsTransparent())

	sld.SetPosePos(math32.Vec3(1, 2, 3))
	sld.SetPoseScale(math32.Vec3(2, 2, 2))
	sld.SetPoseQuat(math32.NewQuatAxisAngle(math32.Vec3(0, 1, 0), 0))
	assert.Equal(t, math32.Vec3(1, 2, 3), sld.Pose.Pos)
	assert.Equal(t, math32.Vec3(2, 2, 2), sld.Pose.Scale)
	assert.Equal(t, &sld.Pose.WorldMatrix, sld.WorldMatrix())
	sld.Config()
	g.Config()

	dl := NewDirectional(sc, "dir", 1, DirectSun)
	dl.Pos.Set(0, 0, 1)
	view := math32.Identity4()
	assert.Equal(t, math32.Vec3(0, 0, 1), dl.ViewDir(view))
	pl := NewPoint(sc, "point", 1, DirectSun)
	assert.Equal(t, pl.Pos, pl.ViewPos(view))
	require.NoError(t, sld.TrackLight("point"))
	assert.Equal(t, pl.Pos, sld.Pose.Pos)
	require.NoError(t, sld.TrackLight("dir"))
	assert.Equal(t, dl.Pos, sld.Pose.Pos)
	assert.Error(t, sld.TrackLight("none"))

	vtx, idx := box.Offsets()
	assert.Zero(t, vtx+idx)
	box.SetOffsets(1, 1)

	sc.ResetMeshes()
	assert.Empty(t, sc.MeshList())

	tex := NewTextureFile(sc, "tex", "testdata/scene.png")
	require.NotNil(t, tex)
	assert.Equal(t, "scene.png", tex.File)
	img := tex.Image()
	require.NotNil(t, img)
	assert.False(t, img.Bounds().Empty())
	missing := NewTextureFile(sc, "missing", "testdata/none.png")
	require.NotNil(t, missing)
	assert.Nil(t, missing.Image(), "not found")
}

// TestShapeDefaults tests the defaults of the shapes.
func TestShapeDefaults(t *testing.T) {
	sc := NewScene()
	sp := NewSphere(sc, "sphere", 3, 8)
	sp.Defaults()
	assert.Equal(t, float32(1), sp.Radius)
	cy := NewCylinder(sc, "cyl", 2, 3, 8, 1, true, true)
	cy.Defaults()
	assert.Equal(t, float32(1), cy.Height)
	ca := NewCapsule(sc, "cap", 2, 3, 8, 1)
	ca.Defaults()
	assert.Equal(t, float32(1), ca.Height)
	to := NewTorus(sc, "torus", 2, 0.5, 8)
	to.Defaults()
	assert.Equal(t, float32(1), to.Radius)
}
