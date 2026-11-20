// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shape

//core:js-test

import (
	"testing"

	"cogentcore.org/core/math32"
	"github.com/stretchr/testify/assert"
)

// testHeights is a [HeightfieldData2D] of x + y.
type testHeights struct{}

func (testHeights) HeightfieldValue(x, y int) float32 { return float32(x + y) }

// checkMesh checks the general properties of the data of a mesh:
// indexes in range, unit normals, texture coordinates in range,
// and a bounding box that is that of the vertexes.
func checkMesh(t *testing.T, name string, m Mesh) *MeshData {
	t.Helper()
	md := NewMeshData(m)
	assert.Equal(t, md.NumVertex*3, len(md.Vertex), name)
	assert.Equal(t, md.NumIndex, len(md.Index), name)
	assert.Equal(t, 0, md.NumIndex%3, "%s: triangles", name)
	for i, ix := range md.Index {
		if !assert.Less(t, int(ix), md.NumVertex, "%s: index %d", name, i) {
			break
		}
	}
	for i := range md.NumVertex {
		var n math32.Vector3
		n.FromSlice(md.Normal, i*3)
		if !assert.InDelta(t, 1, n.Length(), 1e-3, "%s: normal %d %v", name, i, n) {
			break
		}
	}
	bb := BBoxFromVtxs(md.Vertex, 0, md.NumVertex)
	assert.InDelta(t, 0, bb.Min.Sub(md.MeshBBox.Min).Length(), 1e-4, "%s: bbox min %v %v", name, bb.Min, md.MeshBBox.Min)
	assert.InDelta(t, 0, bb.Max.Sub(md.MeshBBox.Max).Length(), 1e-4, "%s: bbox max %v %v", name, bb.Max, md.MeshBBox.Max)
	return md
}

// TestMeshes tests the general properties and the sizes of the meshes.
func TestMeshes(t *testing.T) {
	cases := []struct {
		name string
		mesh Mesh
		min  math32.Vector3
		max  math32.Vector3
	}{
		{"box", NewBox(2, 4, 6), math32.Vec3(-1, -2, -3), math32.Vec3(1, 2, 3)},
		{"plane", NewPlane(math32.Y, 4, 2), math32.Vec3(-2, 0, -1), math32.Vec3(2, 0, 1)},
		{"sphere", NewSphere(1.5, 16), math32.Vec3(-1.5, -1.5, -1.5), math32.Vec3(1.5, 1.5, 1.5)},
		{"cylinder", NewCylinder(2, 1, 16, 2, WithTop, WithBottom), math32.Vec3(-1, -1, -1), math32.Vec3(1, 1, 1)},
		{"cone", NewCone(2, 1, 16, 2, WithBottom), math32.Vec3(-1, -1, -1), math32.Vec3(1, 1, 1)},
		{"capsule", NewCapsule(2, 0.5, 16, 2), math32.Vec3(-0.5, -1.5, -0.5), math32.Vec3(0.5, 1.5, 0.5)},
		{"torus", NewTorus(2, 0.5, 16), math32.Vec3(-2.5, -2.5, -0.5), math32.Vec3(2.5, 2.5, 0.5)}, // in the XY plane
	}
	for _, c := range cases {
		md := checkMesh(t, c.name, c.mesh)
		assert.InDelta(t, 0, md.MeshBBox.Min.Sub(c.min).Length(), 0.05, "%s min %v", c.name, md.MeshBBox.Min)
		assert.InDelta(t, 0, md.MeshBBox.Max.Sub(c.max).Length(), 0.05, "%s max %v", c.name, md.MeshBBox.Max)
	}
}

// TestCone tests that the top of a cone is a point.
func TestCone(t *testing.T) {
	md := checkMesh(t, "cone", NewCone(2, 1, 8, 1, NoBottom))
	for i := range md.NumVertex {
		var v math32.Vector3
		v.FromSlice(md.Vertex, i*3)
		if v.Y > 0.99 {
			assert.InDelta(t, 0, math32.Vec2(v.X, v.Z).Length(), 1e-5, "the top is a point: %v", v)
		}
	}
}

// TestTriangleQuad tests setting triangles and quads into the arrays.
func TestTriangleQuad(t *testing.T) {
	nv, ni := TriangleN()
	qv, qi := QuadN()
	vtx := make(math32.ArrayF32, (nv+qv)*3)
	norm := make(math32.ArrayF32, (nv+qv)*3)
	tex := make(math32.ArrayF32, (nv+qv)*2)
	idx := make(math32.ArrayU32, ni+qi)
	pos := math32.Vec3(0, 0, 1)
	bb := SetTriangle(vtx, norm, tex, idx, 0, 0, math32.Vec3(0, 0, 0), math32.Vec3(1, 0, 0), math32.Vec3(0, 1, 0),
		[]math32.Vector2{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 0, Y: 1}}, pos)
	assert.Equal(t, math32.Vec3(1, 1, 0), bb.Max)
	assert.Equal(t, math32.ArrayU32{0, 1, 2}, idx[:3])
	assert.Equal(t, float32(1), vtx[2], "offset by pos")
	assert.Equal(t, math32.ArrayF32{0, 0, 1}, norm[:3], "counter-clockwise faces +z")

	quad := []math32.Vector3{{X: 0, Y: 0, Z: 0}, {X: 2, Y: 0, Z: 0}, {X: 2, Y: 2, Z: 0}, {X: 0, Y: 2, Z: 0}}
	bb = SetQuad(vtx, norm, tex, idx, nv, ni, quad, []math32.Vector2{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 1, Y: 1}, {X: 0, Y: 1}}, math32.Vector3{})
	assert.Equal(t, math32.Vec3(2, 2, 0), bb.Max)
	assert.Equal(t, math32.ArrayU32{3, 4, 5, 3, 5, 6}, idx[3:])
	assert.Equal(t, float32(1), tex[(nv+2)*2], "texture coordinates")
}

// TestDiskSector tests a disk sector.
func TestDiskSector(t *testing.T) {
	nv, ni := DiskSectorN(8)
	vtx := make(math32.ArrayF32, nv*3)
	norm := make(math32.ArrayF32, nv*3)
	tex := make(math32.ArrayF32, nv*2)
	idx := make(math32.ArrayU32, ni)
	bb := SetDiskSector(vtx, norm, tex, idx, 0, 0, 2, 8, 0, 90, math32.Vector3{})
	assert.InDelta(t, 2, bb.Max.X, 1e-5)
	assert.InDelta(t, 2, bb.Max.Y, 1e-5)
	assert.InDelta(t, 0, bb.Min.X, 1e-5, "a quarter disk")
	assert.Panics(t, func() { SetDiskSector(vtx, norm, tex, idx, 0, 0, 2, 2, 0, 90, math32.Vector3{}) })
}

// TestShapeGroup tests a group of shapes, which is a mesh of all of
// them, each at its own offset.
func TestShapeGroup(t *testing.T) {
	a := NewBox(1, 1, 1)
	b := NewSphere(0.5, 8)
	b.Pos.Set(3, 0, 0)
	grp := &ShapeGroup{Shapes: []Mesh{a, b}}
	av, ai, _ := a.MeshSize()
	bv, bi, _ := b.MeshSize()
	nv, ni, hc := grp.MeshSize()
	assert.Equal(t, av+bv, nv)
	assert.Equal(t, ai+bi, ni)
	assert.False(t, hc)

	md := checkMesh(t, "group", grp)
	vo, io := b.Offsets()
	assert.Equal(t, av, vo)
	assert.Equal(t, ai, io)
	assert.InDelta(t, 3.5, md.MeshBBox.Max.X, 1e-5, "the bbox of all of the shapes")
	assert.InDelta(t, -0.5, md.MeshBBox.Min.X, 1e-5)
	assert.GreaterOrEqual(t, int(md.Index[ai]), av, "the indexes of the second shape are after the first")
}

// TestLines tests lines, open and closed.
func TestLines(t *testing.T) {
	pts := []math32.Vector3{{X: 0, Y: 0, Z: 0}, {X: 1, Y: 0, Z: 0}, {X: 1, Y: 1, Z: 0}}
	open := NewLines(pts, math32.Vec2(0.2, 0.2), false)
	md := checkMesh(t, "lines", open)
	nv, ni := LinesN(3, false)
	assert.Equal(t, nv, md.NumVertex)
	assert.Equal(t, ni, md.NumIndex)
	assert.InDelta(t, 1.1, md.MeshBBox.Max.X, 0.05)

	closed := NewLines(pts, math32.Vec2(0.2, 0.2), true)
	cmd := checkMesh(t, "closed lines", closed)
	assert.Greater(t, cmd.NumVertex, md.NumVertex)
	assert.Len(t, pts, 3, "the points are not changed")

	short := NewLines(pts[:1], math32.Vec2(0.2, 0.2), false)
	vtx := make(math32.ArrayF32, 0)
	bb := SetLines(vtx, vtx, vtx, nil, 0, 0, short.Points, short.Width, false, math32.Vector3{})
	assert.Equal(t, math32.Box3{}, bb, "too few points")
}

// TestHeightfieldSizes tests the sizes of heightfields as planes and bars.
func TestHeightfieldSizes(t *testing.T) {
	hf := NewHeightfield(math32.Y, testHeights{}, 3, 4, 3, 4, 1)
	md := checkMesh(t, "heightfield", hf)
	nv, ni := HeightfieldN(3, 4)
	assert.Equal(t, nv, md.NumVertex)
	assert.Equal(t, ni, md.NumIndex)
	assert.InDelta(t, 5, md.MeshBBox.Max.Y, 1e-5, "the highest value, 2+3")

	hf.Bars = true
	md = checkMesh(t, "heightfield bars", hf)
	nv, ni = HeightfieldBarsN(3, 4)
	assert.Equal(t, nv, md.NumVertex)
	assert.Equal(t, ni, md.NumIndex)
	assert.InDelta(t, 5, md.MeshBBox.Max.Y, 1e-5, "the highest bar")
}

// TestSetColor tests setting the colors of a range of vertexes.
func TestSetColor(t *testing.T) {
	clrs := make(math32.ArrayF32, 4*4)
	SetColor(clrs, 1, 2, math32.Vec4(1, 0.5, 0, 1))
	assert.Equal(t, math32.ArrayF32{0, 0, 0, 0, 1, 0.5, 0, 1, 1, 0.5, 0, 1, 0, 0, 0, 0}, clrs)
}

// TestHeightfieldGroup tests a heightfield in a group after another
// shape, at its offsets and position.
func TestHeightfieldGroup(t *testing.T) {
	hf := NewHeightfield(math32.Y, testHeights{}, 2, 2, 1, 1, 1)
	hf.Pos.Set(10, 0, 0)
	grp := &ShapeGroup{Shapes: []Mesh{NewBox(1, 1, 1), hf}}
	md := checkMesh(t, "group with a heightfield", grp)
	assert.InDelta(t, -0.5, md.MeshBBox.Min.X, 1e-5, "the box is not overwritten")
	assert.Greater(t, md.MeshBBox.Max.X, float32(9), "the heightfield is at its position")
}
