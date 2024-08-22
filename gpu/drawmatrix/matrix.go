// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package drawmatrix

import (
	"image"

	"cogentcore.org/core/math32"
)

// Matrix holds the projection matricies.
type Matrix struct {
	// MVP is the vertex projection matrix,
	// for positioning the vertex points
	MVP math32.Matrix4

	// UVP is the U,V texture coordinate projection matrix
	// for positioning the texture. For fill mode, the
	// last column holds the fill color.
	UVP math32.Matrix4
}

// Config configures the draw matrix for given draw parameters:
// xform is the transform mapping source to destination
// coordinates (translation, scaling), txsz is the size of the texture to draw,
// sr is the source region (set to tex.Format.Bounds() for all)
// flipY inverts the Y axis of the source image.
func Config(destSz image.Point, xform math32.Matrix3, txsz image.Point, sr image.Rectangle, flipY bool) *Matrix {
	var tmat Matrix

	sr = sr.Intersect(image.Rectangle{Max: txsz})
	if sr.Empty() {
		tmat.MVP.SetIdentity()
		tmat.UVP.SetIdentity()
		return &tmat
	}

	// Start with src-space left, top, right and bottom.
	srcL := float32(sr.Min.X)
	srcT := float32(sr.Min.Y)
	srcR := float32(sr.Max.X)
	srcB := float32(sr.Max.Y)

	// Transform to dst-space via the xform matrix, then to a MVP matrix.
	matMVP := calcMVP(destSz.X, destSz.Y,
		xform[0]*srcL+xform[3]*srcT+xform[6],
		xform[1]*srcL+xform[4]*srcT+xform[7],
		xform[0]*srcR+xform[3]*srcT+xform[6],
		xform[1]*srcR+xform[4]*srcT+xform[7],
		xform[0]*srcL+xform[3]*srcB+xform[6],
		xform[1]*srcL+xform[4]*srcB+xform[7],
	)
	tmat.MVP.SetFromMatrix3(&matMVP) // todo render direct

	// OpenGL's fragment shaders' UV coordinates run from (0,0)-(1,1),
	// unlike vertex shaders' XY coordinates running from (-1,+1)-(+1,-1).
	//
	// We are drawing a rectangle PQRS, defined by two of its
	// corners, onto the entire texture. The two quads may actually
	// be equal, but in the general case, PQRS can be smaller.
	//
	//	(0,0) +---------------+ (1,0)
	//	      |  P +-----+ Q  |
	//	      |    |     |    |
	//	      |  S +-----+ R  |
	//	(0,1) +---------------+ (1,1)
	//
	// The PQRS quad is always axis-aligned. First of all, convert
	// from pixel space to texture space.
	tw := float32(txsz.X)
	th := float32(txsz.Y)
	px := float32(sr.Min.X-0) / tw
	py := float32(sr.Min.Y-0) / th
	qx := float32(sr.Max.X-0) / tw
	sy := float32(sr.Max.Y-0) / th
	// Due to axis alignment, qy = py and sx = px.
	//
	// The simultaneous equations are:
	//	  0 +   0 + a02 = px
	//	  0 +   0 + a12 = py
	//	a00 +   0 + a02 = qx
	//	a10 +   0 + a12 = qy = py
	//	  0 + a01 + a02 = sx = px
	//	  0 + a11 + a12 = sy

	if !flipY { // note: reversed from openGL for vulkan
		tmat.UVP.SetFromMatrix3(&math32.Matrix3{
			qx - px, 0, 0,
			0, sy - py, 0, // sy - py
			px, py, 1})
	} else {
		tmat.UVP.SetFromMatrix3(&math32.Matrix3{
			qx - px, 0, 0,
			0, py - sy, 0, // py - sy
			px, sy, 1})
	}

	return &tmat
}

// calcMVP returns the Model View Projection matrix that maps the quadCoords
// unit square, (0, 0) to (1, 1), to a quad QV, such that QV in vertex shader
// space corresponds to the quad QP in pixel space, where QP is defined by
// three of its four corners - the arguments to this function. The three
// corners are nominally the top-left, top-right and bottom-left, but there is
// no constraint that e.g. tlx < trx.
//
// In pixel space, the window ranges from (0, 0) to (widthPx, heightPx). The
// Y-axis points downwards (unless flipped).
//
// In vertex shader space, the window ranges from (-1, +1) to (+1, -1), which
// is a 2-unit by 2-unit square. The Y-axis points upwards.
func calcMVP(widthPx, heightPx int, tlx, tly, trx, try, blx, bly float32) math32.Matrix3 {
	// Convert from pixel coords to vertex shader coords.
	invHalfWidth := 2 / float32(widthPx)
	invHalfHeight := 2 / float32(heightPx)
	tlx = tlx*invHalfWidth - 1
	tly = 1 - tly*invHalfHeight // 1 - min
	trx = trx*invHalfWidth - 1
	try = 1 - try*invHalfHeight // 1 - min
	blx = blx*invHalfWidth - 1
	bly = 1 - bly*invHalfHeight // 1 - (min + max)

	// The resultant affine matrix:
	//	- maps (0, 0) to (tlx, tly).
	//	- maps (1, 0) to (trx, try).
	//	- maps (0, 1) to (blx, bly).
	return math32.Matrix3{
		trx - tlx, try - tly, 0,
		blx - tlx, bly - tly, 0,
		tlx, tly, 1,
	}
}

// Transform returns a transformation matrix for the
// generic Draw function that scales, translates, and rotates
// the source image by the given degrees, to make it fit within
// the destination rectangle dr, given its original size sr (unrotated).
// To avoid scaling, ensure that the dr and sr are the same
// dimensions (post rotation).
// rotDeg = rotation degrees to apply in the mapping:
// 90 = left, -90 = right, 180 = invert.
func Transform(dr image.Rectangle, sr image.Rectangle, rotDeg float32) math32.Matrix3 {
	sx := float32(dr.Dx()) / float32(sr.Dx())
	sy := float32(dr.Dy()) / float32(sr.Dy())
	tx := float32(dr.Min.X) - sx*float32(sr.Min.X)
	ty := float32(dr.Min.Y) - sy*float32(sr.Min.Y)

	if rotDeg == 0 {
		return math32.Matrix3{
			sx, 0, 0,
			0, sy, 0,
			tx, ty, 1,
		}
	}
	rad := math32.DegToRad(rotDeg)
	dsz := math32.FromPoint(dr.Size())
	rmat := math32.Rotate2D(rad)

	dmnr := rmat.MulVector2AsPoint(math32.FromPoint(dr.Min))
	dmxr := rmat.MulVector2AsPoint(math32.FromPoint(dr.Max))
	sx = math32.Abs(dmxr.X-dmnr.X) / float32(sr.Dx())
	sy = math32.Abs(dmxr.Y-dmnr.Y) / float32(sr.Dy())
	tx = dmnr.X - sx*float32(sr.Min.X)
	ty = dmnr.Y - sy*float32(sr.Min.Y)

	if rotDeg < -45 && rotDeg > -135 {
		ty -= dsz.X
	} else if rotDeg > 45 && rotDeg < 135 {
		tx -= dsz.Y
	} else if rotDeg > 135 || rotDeg < -135 {
		ty -= dsz.Y
		tx -= dsz.X
	}

	mat := math32.Matrix3{
		sx, 0, 0,
		0, sy, 0,
		tx, ty, 1,
	}

	return mat.Mul(math32.Matrix3FromMatrix2(rmat))

	/*  stuff that didn't work, but theoretically should?
	rad := math32.DegToRad(rotDeg)
	dsz := math32.FromPoint(dr.Size())
	dctr := dsz.MulScalar(0.5)
	_ = dctr
	// mat2 := math32.Translate2D(dctr.X, 0).Mul(math32.Rotate2D(rad)).Mul(math32.Translate2D(tx, ty)).Mul(math32.Scale2D(sx, sy))
	mat2 := math32.Translate2D(tx, ty).Mul(math32.Scale2D(sx, sy)).Mul(math32.Translate2D(dctr.X, 0)).Mul(math32.Rotate2D(rad))
	// mat2 := math32.Rotate2D(rad).MulCtr(math32.Translate2D(tx, ty).Mul(math32.Scale2D(sx, sy)), dctr)
	mat := math32.Matrix3FromMatrix2(mat2)
	*/
}
