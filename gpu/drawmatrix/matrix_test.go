// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package drawmatrix

import (
	"image"
	"testing"

	"cogentcore.org/core/base/tolassert"
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
	pts := DrawFromMatrixMVP4(&tmat.MVP)
	dr := sr.Add(dp)
	CompareRect(t, pts, dr)
}

func TestScaleMatrix(t *testing.T) {
	sr := image.Rectangle{Min: image.Point{0, 0}, Max: image.Point{190, 280}}
	destSz := image.Point{1000, 500}
	dr := image.Rectangle{Max: destSz}
	mat := Transform(dr, sr, 0)
	tmat := Config(destSz, mat, sr.Max, sr, false)
	pts := DrawFromMatrixMVP4(&tmat.MVP)
	CompareRect(t, pts, dr)
}

// mulXform applies the transform the way Config does, which is the
// standard affine multiply with the point on the right.
func mulXform(x math32.Matrix3, p math32.Vector2) math32.Vector2 {
	return math32.Vec2(
		x[0]*p.X+x[3]*p.Y+x[6],
		x[1]*p.X+x[4]*p.Y+x[7])
}

func rectCorners(r image.Rectangle) []math32.Vector2 {
	return []math32.Vector2{
		math32.Vec2(float32(r.Min.X), float32(r.Min.Y)),
		math32.Vec2(float32(r.Max.X), float32(r.Min.Y)),
		math32.Vec2(float32(r.Min.X), float32(r.Max.Y)),
		math32.Vec2(float32(r.Max.X), float32(r.Max.Y)),
	}
}

// TestTransformMapsSourceOntoDest checks the defining property of
// Transform: it maps the corners of the source rectangle onto the
// corners of the destination rectangle, for each supported rotation.
func TestTransformMapsSourceOntoDest(t *testing.T) {
	tests := []struct {
		rotDeg float32
		dr, sr image.Rectangle
	}{
		{0, image.Rect(0, 0, 100, 50), image.Rect(0, 0, 100, 50)},
		{0, image.Rect(10, 20, 110, 70), image.Rect(0, 0, 50, 25)},
		{90, image.Rect(0, 0, 100, 50), image.Rect(0, 0, 50, 100)},
		{-90, image.Rect(0, 0, 100, 50), image.Rect(0, 0, 50, 100)},
		{180, image.Rect(0, 0, 100, 50), image.Rect(0, 0, 100, 50)},
	}
	for _, tst := range tests {
		x := Transform(tst.dr, tst.sr, tst.rotDeg)
		// every source corner lands on a distinct destination corner
		want := rectCorners(tst.dr)
		used := make([]bool, len(want))
		for _, c := range rectCorners(tst.sr) {
			g := mulXform(x, c)
			found := false
			for i, w := range want {
				if !used[i] && math32.Abs(g.X-w.X) < 0.01 && math32.Abs(g.Y-w.Y) < 0.01 {
					used[i] = true
					found = true
					break
				}
			}
			if !found {
				t.Errorf("rot %g: source corner %v mapped to %v, which is not an unused destination corner of %v",
					tst.rotDeg, c, g, tst.dr)
			}
		}
	}
}

// TestTransformNoRotation checks the unrotated transform exactly.
func TestTransformNoRotation(t *testing.T) {
	x := Transform(image.Rect(10, 20, 110, 70), image.Rect(0, 0, 50, 25), 0)
	p := mulXform(x, math32.Vec2(0, 0))
	tolassert.Equal(t, float32(10), p.X)
	tolassert.Equal(t, float32(20), p.Y)
	p = mulXform(x, math32.Vec2(50, 25))
	tolassert.Equal(t, float32(110), p.X)
	tolassert.Equal(t, float32(70), p.Y)
}
