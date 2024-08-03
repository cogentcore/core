// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpudraw

import (
	"fmt"
	"image"
	"testing"

	"cogentcore.org/core/math32"
)

// NDCtoFramebuffer converts points in NDC coordinates to
// framebuffer pixel coordinates.
// NDC is TL: -1, 1; TR: 1,1; BL: -1,-1; BR: 1,-1
// FB is TL:  0,  0; TR: w,0; BL:  0, h; BR: w, h
func NDCToFramebuffer(pts []math32.Vector2) {
	wd := float32(1000)
	ht := float32(500)

	for i, pt := range pts {
		fb := pt
		fb.X = 0.5 * (pt.X + 1) * wd
		fb.Y = 0.5 * (-pt.Y + 1) * ht
		fmt.Println(i, pt, fb)
	}
}

func TestNDC(t *testing.T) {
	pts := []math32.Vector2{ // tl, tr, bl, br
		{-1, 1}, {1, 1}, {-1, -1}, {1, -1},
	}
	fmt.Println("NDC:")
	NDCToFramebuffer(pts)
}

func TestTriangle(t *testing.T) {
	pts := make([]math32.Vector2, 3)
	for i, pt := range pts {
		pt.X = float32(i - 1)
		pt.Y = float32((i&1)*2 - 1)
		pts[i] = pt
	}
	fmt.Println("Triangle:")
	NDCToFramebuffer(pts)
}

func DrawFromMatrixMVP4(mat *math32.Matrix4) {
	m3 := math32.Matrix3FromMatrix4(mat)
	fmt.Println(m3)
	pts := []math32.Vector2{ // tl, tr, bl, br
		{0, 0}, {0, 1}, {1, 0}, {1, 1},
	}
	cpts := make([]math32.Vector2, 4) // clip coords
	for i, pt := range pts {
		cp := m3.MulVector2AsPoint(pt)
		cpts[i] = cp
		fmt.Println(i, pt, cp)
	}
	NDCToFramebuffer(cpts)
}

// fill: (10,20)-(200,300) s2d: [1 0 0 0 1 0 10 20 1]
// m3: [0.37109375 0 0 0 -0.7291667 0 -0.9609375 0.8958333 1]

func TestFillMatrix(t *testing.T) {
	sr := image.Rectangle{Min: image.Point{10, 20}, Max: image.Point{200, 300}}
	destSz := image.Point{1024, 768}
	mat := math32.Matrix3{
		1, 0, 0,
		0, 1, 0,
		float32(sr.Min.X), float32(sr.Min.Y), 1,
	}
	tmat := ConfigMatrix(destSz, mat, destSz, sr, false)
	fmt.Println("fill:")
	DrawFromMatrixMVP4(&tmat.MVP)
}

func TestDrawMatrix(t *testing.T) {
	sr := image.Rectangle{Min: image.Point{0, 0}, Max: image.Point{190, 280}}
	dp := image.Point{10, 20}
	destSz := image.Point{1024, 768}
	mat := math32.Matrix3{
		1, 0, 0,
		0, 1, 0,
		float32(dp.X - sr.Min.X), float32(dp.Y - sr.Min.Y), 1,
	}
	tmat := ConfigMatrix(destSz, mat, sr.Max, sr, false)
	fmt.Println("draw:")
	DrawFromMatrixMVP4(&tmat.MVP)
}
