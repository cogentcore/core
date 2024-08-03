// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package drawmatrix

import (
	"image"
	"testing"

	"cogentcore.org/core/math32"
	"github.com/stretchr/testify/assert"
)

// NDCtoFramebuffer converts points in NDC coordinates to
// framebuffer pixel coordinates.
// NDC is TL: -1, 1; TR: 1,1; BL: -1,-1; BR: 1,-1
// FB is TL:  0,  0; TR: w,0; BL:  0, h; BR: w, h
func NDCToFramebuffer(pts []math32.Vector2) []math32.Vector2 {
	wd := float32(1000)
	ht := float32(500)

	fbp := make([]math32.Vector2, len(pts))
	for i, pt := range pts {
		fb := pt
		fb.X = 0.5 * (pt.X + 1) * wd
		fb.Y = 0.5 * (-pt.Y + 1) * ht
		fbp[i] = fb
		// fmt.Println(i, pt, fb)
	}
	return fbp
}

func TestNDC(t *testing.T) {
	pts := []math32.Vector2{ // tl, tr, bl, br
		{-1, 1}, {1, 1}, {-1, -1}, {1, -1},
	}
	// fmt.Println("NDC:")
	fbp := NDCToFramebuffer(pts)
	trg := []math32.Vector2{ // tl, tr, bl, br
		{0, 0}, {1000, 0}, {0, 500}, {1000, 500},
	}
	assert.Equal(t, trg, fbp)
}

func TestTriangle(t *testing.T) {
	pts := make([]math32.Vector2, 3)
	for i, pt := range pts {
		pt.X = float32(i - 1)
		pt.Y = float32((i&1)*2 - 1)
		pts[i] = pt
	}
	fbp := NDCToFramebuffer(pts)
	trg := []math32.Vector2{ // tl, tr, bl, br
		{0, 500}, {500, 0}, {1000, 500},
	}
	assert.Equal(t, trg, fbp)
}

func DrawFromMatrixMVP4(mat *math32.Matrix4) []math32.Vector2 {
	m3 := math32.Matrix3FromMatrix4(mat)
	// fmt.Println(m3)
	pts := []math32.Vector2{ // tl, tr, bl, br
		{0, 0}, {0, 1}, {1, 0}, {1, 1},
	}
	cpts := make([]math32.Vector2, 4) // clip coords
	for i, pt := range pts {
		pt3 := math32.Vector3{pt.X, pt.Y, 1} // depends on this!
		cp := pt3.MulMatrix3(&m3)
		cpts[i] = math32.Vector2{cp.X, cp.Y}
		// fmt.Println(i, pt, cp)
	}
	return NDCToFramebuffer(cpts)
}

func CompareRect(t *testing.T, pts []math32.Vector2, rect image.Rectangle) {
	trg := []math32.Vector2{ // tl, tr, bl, br
		{float32(rect.Min.X), float32(rect.Min.Y)}, {float32(rect.Min.X), float32(rect.Max.Y)}, {float32(rect.Max.X), float32(rect.Min.Y)}, {float32(rect.Max.X), float32(rect.Max.Y)},
	}
	for i, tp := range trg {
		pp := pts[i]
		// fmt.Println(i, tp, pp)
		assert.InDelta(t, tp.X, pp.X, .001)
		assert.InDelta(t, tp.Y, pp.Y, .001)
	}
}

func TestFillMatrix(t *testing.T) {
	dr := image.Rectangle{Min: image.Point{10, 20}, Max: image.Point{200, 300}}
	destSz := image.Point{1000, 500}
	tmat := Config(destSz, math32.Identity3(), destSz, dr, false)
	pts := DrawFromMatrixMVP4(&tmat.MVP)
	CompareRect(t, pts, dr)
}

func TestDrawMatrix(t *testing.T) {
	sr := image.Rectangle{Min: image.Point{0, 0}, Max: image.Point{190, 280}}
	dp := image.Point{10, 20}
	destSz := image.Point{1000, 500}
	mat := math32.Matrix3{
		1, 0, 0,
		0, 1, 0,
		float32(dp.X - sr.Min.X), float32(dp.Y - sr.Min.Y), 1,
	}
	tmat := Config(destSz, mat, sr.Max, sr, false)
	// fmt.Println("draw:")
	pts := DrawFromMatrixMVP4(&tmat.MVP)
	dr := sr.Add(dp)
	CompareRect(t, pts, dr)
}
