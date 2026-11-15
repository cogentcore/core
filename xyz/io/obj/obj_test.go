// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package obj

import (
	"io"
	"os"
	"strings"
	"testing"

	"cogentcore.org/core/xyz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// decodeFile decodes the given file in testdata.
func decodeFile(t *testing.T, fname string) *Decoder {
	dec := (&Decoder{}).New().(*Decoder)
	files, err := dec.SetFileFS(os.DirFS("testdata"), fname)
	require.NoError(t, err)
	readers := make([]io.Reader, len(files))
	for i, f := range files {
		fh, err := os.Open("testdata/" + f)
		require.NoError(t, err)
		defer fh.Close()
		readers[i] = fh
	}
	require.NoError(t, dec.Decode(readers))
	return dec
}

// TestDecode tests decoding an obj file with its materials.
func TestDecode(t *testing.T) {
	dec := decodeFile(t, "tri.obj")
	assert.Len(t, dec.Objects, 1)
	ob := dec.Objects[0]
	assert.Equal(t, "quad", ob.Name)
	assert.Len(t, ob.Faces, 2)
	assert.Len(t, dec.Vertices, 4*3)
	assert.Len(t, dec.Uvs, 4*2)
	assert.Len(t, dec.Normals, 3)
	require.Contains(t, dec.Materials, "red")
	assert.Equal(t, uint8(255), dec.Materials["red"].Diffuse.R)
	assert.Equal(t, "red", ob.Faces[0].Material)
	assert.False(t, dec.HasScene())
	assert.Contains(t, dec.Desc(), "Wavefront")
}

// TestSetGroup tests setting the decoded objects into a scene, as
// solids with meshes and materials.
func TestSetGroup(t *testing.T) {
	dec := decodeFile(t, "tri.obj")
	sc := xyz.NewScene()
	gp := xyz.NewGroup(sc)
	dec.SetGroup(sc, gp)
	require.Equal(t, 1, gp.NumChildren())
	obgp := gp.Child(0).(*xyz.Group)
	assert.Equal(t, "quad", obgp.Name)
	require.Equal(t, 1, obgp.NumChildren(), "one solid for the one material")
	sld := obgp.Child(0).(*xyz.Solid)
	assert.Equal(t, uint8(255), sld.Material.Color.R, "the diffuse color of the material")
	ms, err := sc.MeshByName(sld.Mesh.AsMeshBase().Name)
	require.NoError(t, err)
	nv, ni, _ := ms.MeshSize()
	assert.Equal(t, 6, ni, "two triangles")
	assert.GreaterOrEqual(t, nv, 4)
	assert.Nil(t, dec.Objects, "the decoder data is freed")
}

// TestMissingFiles tests decoding without a material file, and a
// missing obj file.
func TestMissingFiles(t *testing.T) {
	dir := t.TempDir()
	data, err := os.ReadFile("testdata/tri.obj")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(dir+"/nomtl.obj", data, 0644))
	dec := (&Decoder{}).New().(*Decoder)
	files, err := dec.SetFileFS(os.DirFS(dir), "nomtl.obj")
	require.NoError(t, err)
	assert.Equal(t, []string{"nomtl.obj"}, files, "no material file")

	_, err = dec.SetFileFS(os.DirFS(dir), "none.obj")
	assert.Error(t, err)
}

// TestParseErrors tests decoding invalid obj data.
func TestParseErrors(t *testing.T) {
	for _, src := range []string{
		"v 1 2\n",   // too few coordinates
		"v a b c\n", // not numbers
		"vn 1\n",    // too few
		"vt x y\n",  // not numbers
		"f 1 2\n",   // too few vertexes
		"usemtl\n",  // no name
		"mtllib\n",  // no name
	} {
		dec := (&Decoder{}).New().(*Decoder)
		err := dec.Decode([]io.Reader{strings.NewReader(src)})
		assert.Error(t, err, "%q", src)
	}
}

// TestPolygonFaces tests triangulating faces with more than three
// vertexes, without normals, each triangle indexed once.
func TestPolygonFaces(t *testing.T) {
	src := "o poly\nv 0 0 0\nv 1 0 0\nv 1 1 0\nv 0 1 0\nv -1 1 0\nf 1 2 3\nf 1 2 3 4 5\n"
	dec := (&Decoder{}).New().(*Decoder)
	require.NoError(t, dec.Decode([]io.Reader{strings.NewReader(src)}))
	sc := xyz.NewScene()
	gp := xyz.NewGroup(sc)
	dec.SetGroup(sc, gp)
	sld := gp.Child(0).AsTree().Child(0).(*xyz.Solid)
	ms := sld.Mesh.(*xyz.GenMesh)
	nv, ni, _ := ms.MeshSize()
	assert.Equal(t, 3+3*3, ni, "a triangle, and a pentagon of three triangles")
	assert.Equal(t, nv*3, len(ms.Normal), "a normal for each vertex")
	for i := 0; i+2 < len(ms.Index); i += 3 {
		a, b, c := ms.Index[i], ms.Index[i+1], ms.Index[i+2]
		assert.True(t, a != b && b != c && a != c, "triangle %d: %d %d %d", i/3, a, b, c)
	}
}
