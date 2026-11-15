// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package obj

import (
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cogentcore.org/core/xyz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// decodeStrings decodes the given obj and (optional) mtl sources.
func decodeStrings(t *testing.T, src ...string) (*Decoder, error) {
	t.Helper()
	dec := (&Decoder{}).New().(*Decoder)
	var rs []io.Reader
	for _, s := range src {
		rs = append(rs, strings.NewReader(s))
	}
	return dec, dec.Decode(rs)
}

const quadSrc = `# comment
mtllib quad.mtl
g first
v 0 0 0
v 1 0 0
v 1 1 0
v 0 1 0
vt 0 0
vt 1 0
vt 1 1
vt 0 1
vn 0 0 1
usemtl shiny
s 1
f 1/1/1 2/2/1 3/3/1 4/4/1
o second
usemtl plain
s off
f -4//-1 -3//-1 -2//-1
f -4/-4 -2/-2 -1/-1
`

const quadMtl = `# materials
newmtl shiny
Ka 0.1 0.1 0.1
Kd 0 1 0
Ks 1 1 1
Ke 0.5 0 0
Ni 1.5
Ns 80
d 0.5
illum 2
map_Kd -s 2 3 -o 0.5 0.25 tex.png
newmtl plain
Kd 0 0 1
unknown 1 2 3
`

// TestDecodeAll tests decoding objects and groups with relative indexes,
// texture coordinates and normals in each combination, smoothing, and
// all of the material properties.
func TestDecodeAll(t *testing.T) {
	dec, err := decodeStrings(t, quadSrc, quadMtl)
	require.NoError(t, err)
	assert.Equal(t, "quad.mtl", dec.Matlib)
	require.Len(t, dec.Objects, 2)
	first, second := dec.Objects[0], dec.Objects[1]
	assert.Equal(t, "first", first.Name)
	require.Len(t, first.Faces, 1)
	f := first.Faces[0]
	assert.Equal(t, []int{0, 1, 2, 3}, f.Vertices)
	assert.Equal(t, []int{0, 1, 2, 3}, f.Uvs)
	assert.Equal(t, []int{0, 0, 0, 0}, f.Normals)
	assert.Equal(t, "shiny", f.Material)
	assert.True(t, f.Smooth)

	require.Len(t, second.Faces, 2)
	f = second.Faces[0]
	assert.Equal(t, []int{0, 1, 2}, f.Vertices, "relative to the last vertex")
	assert.Equal(t, []int{invINDEX, invINDEX, invINDEX}, f.Uvs, "no texture coordinates")
	assert.Equal(t, []int{0, 0, 0}, f.Normals)
	assert.False(t, f.Smooth)
	f = second.Faces[1]
	assert.Equal(t, []int{0, 2, 3}, f.Uvs, "relative texture coordinates")
	assert.Equal(t, []int{invINDEX, invINDEX, invINDEX}, f.Normals)

	m := dec.Materials["shiny"]
	require.NotNil(t, m)
	assert.Equal(t, color.RGBA{0, 255, 0, 255}, m.Diffuse)
	assert.Equal(t, uint8(255), m.Specular.R)
	assert.Equal(t, uint8(128), m.Emissive.R)
	assert.InDelta(t, 26, float64(m.Ambient.R), 1)
	assert.Equal(t, float32(1.5), m.Refraction)
	assert.Equal(t, float32(80), m.Shininess)
	assert.Equal(t, float32(0.5), m.Opacity)
	assert.Equal(t, 2, m.Illum)
	assert.Equal(t, "tex.png", m.MapKd)
	assert.Equal(t, float32(2), m.Tiling.Repeat.X)
	assert.Equal(t, float32(3), m.Tiling.Repeat.Y, "the second value of -s")
	assert.Equal(t, float32(0.5), m.Tiling.Offset.X)
	assert.Equal(t, float32(0.25), m.Tiling.Offset.Y, "the second value of -o")
	assert.Equal(t, color.RGBA{0, 0, 255, 255}, dec.Materials["plain"].Diffuse)
	assert.NotEmpty(t, dec.Warnings, "the unknown material field")
}

// TestMapKdOptions tests the options of a texture with one value each,
// and options without a value.
func TestMapKdOptions(t *testing.T) {
	dec, err := decodeStrings(t, "usemtl m\n", "newmtl m\nmap_Kd -s 2 -o 0.5 tex.png\n")
	require.NoError(t, err)
	m := dec.Materials["m"]
	assert.Equal(t, "tex.png", m.MapKd)
	assert.Equal(t, float32(2), m.Tiling.Repeat.Y, "one value for both")
	assert.Equal(t, float32(0.5), m.Tiling.Offset.Y)

	assert.NotPanics(t, func() {
		dec, err = decodeStrings(t, "usemtl m\n", "newmtl m\nmap_Kd tex.png -s\n")
	})
	assert.Equal(t, defaultMat, dec.Materials["m"], "an error in the material file uses the default")
}

// TestSmoothGroups tests the smoothing group numbers of faces.
func TestSmoothGroups(t *testing.T) {
	dec, err := decodeStrings(t, "v 0 0 0\nv 1 0 0\nv 1 1 0\ns 2\nf 1 2 3\ns 0\nf 1 2 3\n")
	require.NoError(t, err, "a smoothing group number")
	fs := dec.Objects[0].Faces
	assert.True(t, fs[0].Smooth)
	assert.False(t, fs[1].Smooth)
	_, err = decodeStrings(t, "s maybe\n")
	assert.Error(t, err)
	_, err = decodeStrings(t, "s\n")
	assert.Error(t, err)
}

// TestDecodeErrors tests invalid obj and mtl data, which are errors, not
// panics, including faces of vertexes that do not exist and material
// properties before any material.
func TestDecodeErrors(t *testing.T) {
	for _, src := range []string{
		"v 0 0 0\nv 1 0 0\nv 1 1 0\nf 1 2 0\n",     // index 0
		"v 0 0 0\nv 1 0 0\nv 1 1 0\nf 1 2 4\n",     // beyond the vertexes
		"v 0 0 0\nv 1 0 0\nv 1 1 0\nf -1 -2 -4\n",  // before the first one
		"v 0 0 0\nv 1 0 0\nv 1 1 0\nf 1/0 2 3\n",   // texture index 0
		"v 0 0 0\nv 1 0 0\nv 1 1 0\nf 1/1/0 2 3\n", // normal index 0
		"v 0 0 0\nv 1 0 0\nv 1 1 0\nf a 2 3\n",     // not a number
		"v 0 0 0\nv 1 0 0\nv 1 1 0\nf 1/a 2 3\n",   // not a number
		"v 0 0 0\nv 1 0 0\nv 1 1 0\nf 1/1/a 2 3\n", // not a number
		"o\n",        // no name
		"vn a b c\n", // not numbers
		"vt 1\n",     // too few
	} {
		var err error
		assert.NotPanics(t, func() { _, err = decodeStrings(t, src) }, "%q", src)
		assert.Error(t, err, "%q", src)
	}
	for _, mtl := range []string{
		"Kd 1 0 0\n", // before newmtl
		"Ka 1 0 0\n", "Ks 1 0 0\n", "Ke 1 0 0\n", "d 1\n", "Ni 1\n", "Ns 1\n", "illum 1\n", "map_Kd t.png\n",
		"newmtl\n",           // no name
		"newmtl m\nKd 1 0\n", // too few
		"newmtl m\nKa 1\n", "newmtl m\nKs 1\n", "newmtl m\nKe 1\n",
		"newmtl m\nKd a b c\n", "newmtl m\nKa a b c\n", "newmtl m\nKs a b c\n", "newmtl m\nKe a b c\n",
		"newmtl m\nd\n", "newmtl m\nd x\n", "newmtl m\nNi\n", "newmtl m\nNi x\n",
		"newmtl m\nNs\n", "newmtl m\nNs x\n", "newmtl m\nillum\n", "newmtl m\nillum x\n",
		"newmtl m\nmap_Kd\n",
	} {
		dec := (&Decoder{}).New().(*Decoder)
		var err error
		assert.NotPanics(t, func() { err = dec.parse(strings.NewReader(mtl), dec.parseMtlLine) }, "%q", mtl)
		assert.Error(t, err, "%q", mtl)
	}
	dec := (&Decoder{}).New().(*Decoder)
	assert.Error(t, dec.Decode(nil), "no readers")
}

// TestSetScene tests setting the objects into a scene, with a texture of
// a material, a material that is not defined, and the default material
// without a material file.
func TestSetScene(t *testing.T) {
	dir := t.TempDir()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	fh, err := os.Create(filepath.Join(dir, "tex.png"))
	require.NoError(t, err)
	require.NoError(t, png.Encode(fh, img))
	fh.Close()

	dec, err := decodeStrings(t, quadSrc+"usemtl missing\nf 1 2 3\n", quadMtl)
	require.NoError(t, err)
	dec.FS = os.DirFS(dir)
	dec.Objfile = "quad.obj"
	sc := xyz.NewScene()
	dec.SetScene(sc)
	require.Equal(t, 1, sc.NumChildren())
	gp := sc.Child(0).(*xyz.Group)
	assert.Equal(t, "quad.obj", gp.Name)
	require.Equal(t, 2, gp.NumChildren(), "the two objects")
	first := gp.Child(0).(*xyz.Group).Child(0).(*xyz.Solid)
	assert.NotNil(t, first.Material.Texture, "the texture of the material")
	assert.Equal(t, float32(2), first.Material.Tiling.Repeat.X)
	_, err = sc.TextureByName("tex.png")
	assert.NoError(t, err)
	second := gp.Child(1).(*xyz.Group)
	assert.Equal(t, 2, second.NumChildren(), "a solid for each material in turn: plain, missing")
	assert.Equal(t, uint8(127), first.Material.Color.A, "the opacity")
	assert.Equal(t, float32(80), first.Material.Shiny)
	assert.Nil(t, dec.Objects, "freed")

	// without a material file, the default material
	dec, err = decodeStrings(t, quadSrc)
	require.NoError(t, err)
	assert.Equal(t, defaultMat, dec.Materials["shiny"])
}
