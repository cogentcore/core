// Copyright 2022 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shape

import (
	"fmt"

	"cogentcore.org/core/math32"
)

// HeightfieldData2D is an interface for accessing the 2D Heightfield data.
type HeightfieldData2D interface {
	// HeightfieldValue returns the data value at given coordinate.
	HeightfieldValue(x, y int) float32
}

// Heightfield represents planar 2D data with the third dimension
// provided by the data values, which can be visualized as either
// a triangulated plane or discrete bars.
type Heightfield struct { //types:add -setters
	ShapeBase

	// Bars represents the data using discrete bars, else triangles on the plane.
	Bars bool

	// axis along which the normal perpendicular to the plane points.
	// E.g., if the Y axis is specified, then it is a standard X-Z ground plane.
	// See also NormalNeg for whether it is facing in the positive or
	// negative of the given axis.
	NormalAxis math32.Dims

	// if false, the plane normal is facing in the positive direction
	// along specified NormalAxis, otherwise it faces in the negative.
	NormalNeg bool

	// Size is the 2D size of the heightfield data values.
	Size math32.Vector2i

	// 3D visual size, with the Z value providing scaling for data heights.
	VisSize math32.Vector3

	// Data for the heights of points in the field.
	Data HeightfieldData2D

	// BarPct is proportion (0-1) that the bar occupies within its unit square
	// (i.e., 1-BarPct is the spacing between bars).
	BarPct float32

	// Offset from origin along direction of normal to the plane.
	Offset float32
}

// NewHeightfield returns a Heightfield with given data and visual size.
func NewHeightfield(axis math32.Dims, data HeightfieldData2D, nx, ny int, width, height, zscale float32) *Heightfield {
	hf := &Heightfield{}
	hf.Defaults()
	hf.NormalAxis = axis
	hf.Data = data
	hf.Size.Set(int32(nx), int32(ny))
	hf.VisSize.Set(width, height, zscale)
	return hf
}

func (hf *Heightfield) Defaults() {
	hf.NormalAxis = math32.Y
	hf.NormalNeg = false
	hf.VisSize.Set(1, 1, 1)
	hf.BarPct = 0.9
	hf.Offset = 0
}

func (hf *Heightfield) MeshSize() (numVertex, nIndex int, hasColor bool) {
	if hf.Bars {
		numVertex, nIndex = HeightfieldBarsN(int(hf.Size.X), int(hf.Size.Y))
	} else {
		numVertex, nIndex = HeightfieldN(int(hf.Size.X), int(hf.Size.Y))
	}
	return
}

// Set sets points in given allocated arrays.
func (hf *Heightfield) Set(vertex, normal, texcoord, clrs math32.ArrayF32, index math32.ArrayU32) {
	waxis, haxis, wdir, hdir := NormPlaneAxes(hf.NormalAxis, hf.NormalNeg)
	if hf.Bars {
		SetHeightfieldBars(vertex, normal, texcoord, index, 0, 0, int(hf.Size.X), int(hf.Size.Y), hf.Data, waxis, haxis, wdir, hdir, hf.VisSize.X, hf.VisSize.Y, 0, 0, hf.Offset, hf.VisSize.Z, hf.BarPct, math32.Vector3{})
	} else {
		SetHeightfield(vertex, normal, texcoord, index, 0, 0, int(hf.Size.X), int(hf.Size.Y), hf.Data, waxis, haxis, wdir, hdir, hf.VisSize.X, hf.VisSize.Y, 0, 0, hf.Offset, hf.VisSize.Z, math32.Vector3{})
	}
}

//////// Planar representation

func HeightfieldN(nx, ny int) (numVertex, nIndex int) {
	numVertex = ny * nx
	nIndex = 6 * (ny - 1) * (nx - 1) // two triangles per square
	return
}

// SetHeightfield sets planar Heightfield vertex, normal, tex, index data
// at given starting *vertex* index
// (i.e., multiply this *3 to get actual float offset in Vtx array), and starting Index index.
// nx, ny are number of data points along each axis of the data (wrap in
// a struct that implements HeightfieldData2D): a min of 2 in each dim required.
// waxis, haxis = width, height axis, wdir, hdir are the directions for width
// and height dimensions. zscale multiplies the heightfield values.
// offset is the distance to place the plane along the orthogonal axis.
// pos is a 3D position offset.
func SetHeightfield(vertex, normal, texcoord math32.ArrayF32, index math32.ArrayU32, vtxOff, idxOff int, nx, ny int, data HeightfieldData2D, waxis, haxis math32.Dims, wdir, hdir int, width, height, woff, hoff, zoff, zscale float32, pos math32.Vector3) error {
	if ny < 2 || nx < 2 {
		return fmt.Errorf("SetHeightfield: must have at least 2 data points along each dimension.")
	}
	zaxis := OrthoAxis(waxis, haxis)

	fwdir := float32(wdir) * width
	fhdir := float32(hdir) * height
	if wdir < 0 {
		woff = width + woff
	}
	if hdir < 0 {
		hoff = height + hoff
	}

	vidx := vtxOff // vertex index
	var vtx, norm, a, b, c, d math32.Vector3
	var tex math32.Vector2

	// have to write all verticies first, so have access to all the data
	// for computing norms!
	for yi := ny - 1; yi >= 0; yi-- {
		for xi := range nx {
			val := data.HeightfieldValue(xi, yi)
			vtx.SetDim(waxis, float32(xi)*fwdir+woff)
			vtx.SetDim(haxis, float32(yi)*fhdir+hoff)
			vtx.SetDim(zaxis, val*zscale)
			vtx.Add(pos).ToSlice(vertex, vidx*3)

			tex.Set(float32(xi)/float32(nx-1), (float32(yi) / float32(ny-1)))
			tex.ToSlice(texcoord, vidx*2)
			vidx++
		}
	}

	// b c
	// a d
	// abd, bcd
	// because y is inverted, a is higher y, b is lower
	ny1 := ny - 1
	vIdx := func(y, x int) int {
		return ((ny1-y)*nx + x)
	}

	nidx := vtxOff * 3
	iidx := idxOff // index index
	for yi := ny1; yi >= 0; yi-- {
		for xi := 0; xi < nx; xi++ {
			if yi == 0 { // no index
				if xi == nx-1 { // no right
					vertex.GetVector3(vIdx(1, xi-1)*3, &a) // a = yi=1, xi=left
					vertex.GetVector3(vIdx(0, xi-1)*3, &b) // b = yi=0, xi=left
					vertex.GetVector3(vIdx(0, xi)*3, &c)   // c = yi=0, xi=us
				} else {
					vertex.GetVector3(vIdx(1, xi)*3, &a)   // a = yi=1, xi=us
					vertex.GetVector3(vIdx(0, xi)*3, &b)   // b = yi=0, xi=us
					vertex.GetVector3(vIdx(0, xi+1)*3, &c) // c = yi=0, xi=right
				}
			} else { // do indexes
				if xi == nx-1 { // no right, no index
					vertex.GetVector3(vIdx(yi, xi-1)*3, &a)   // a = yi, xi=left
					vertex.GetVector3(vIdx(yi-1, xi-1)*3, &b) // b = yi-1, xi=left
					vertex.GetVector3(vIdx(yi-1, xi)*3, &c)   // c = yi-1, xi=us
				} else {
					ai := vIdx(yi, xi)     // a = yi, xi=us
					bi := vIdx(yi-1, xi)   // b = yi-1, xi=us
					ci := vIdx(yi-1, xi+1) // c = yi-1, xi=right
					di := vIdx(yi, xi+1)   // d = yi, xi=right
					vertex.GetVector3(ai*3, &a)
					vertex.GetVector3(bi*3, &b)
					vertex.GetVector3(ci*3, &c)
					vertex.GetVector3(di*3, &d)

					index.Set(iidx, uint32(ai), uint32(bi), uint32(di), uint32(bi), uint32(ci), uint32(di))
					iidx += 6
				}
			}
			norm = math32.Normal(a, b, c)
			norm.ToSlice(normal, nidx)
			nidx += 3
		}
	}
	return nil
}

//////// Bars representation

func HeightfieldBarsN(nx, ny int) (numVertex, nIndex int) {
	nper := 5
	vtxSz, idxSz := PlaneN(1, 1)
	numVertex = vtxSz * nper * ny * nx
	nIndex = idxSz * nper * ny * nx
	return
}

// MinBarHeight ensures that there is always at least some dimensionality
// to the bars. Affects transparency rendering etc
var MinBarHeight = float32(1.0e-6)

// SetHeightfieldBars sets bars Heightfield vertex, normal, index data
// at given starting *vertex* index
// (i.e., multiply this *3 to get actual float offset in Vtx array), and starting Index index.
// nx, ny are number of data points along each axis of the data (wrap in
// a struct that implements HeightfieldData2D): a min of 2 in each dim required.
// waxis, haxis = width, height axis, wdir, hdir are the directions for width
// and height dimensions. zscale multiplies the heightfield values,
// and barPct is proportion (0-1) that the bar occupies within its unit square
// (i.e., 1-barPct is the spacing between bars).
// offset is the distance to place the plane along the orthogonal axis.
// pos is a 3D position offset.
func SetHeightfieldBars(vertex, normal, texcoord math32.ArrayF32, index math32.ArrayU32, vtxOff, idxOff int, nx, ny int, data HeightfieldData2D, waxis, haxis math32.Dims, wdir, hdir int, width, height, woff, hoff, zoff, zscale, barPct float32, pos math32.Vector3) error {
	zaxis := OrthoAxis(waxis, haxis)
	barSpc := (1.0 - barPct)
	segs := 1

	vtxSz, idxSz := PlaneN(segs, segs)

	pidx := 0
	for yi := ny - 1; yi >= 0; yi-- {
		y0 := barSpc - float32(yi+1)
		for xi := range nx {
			poff := vtxOff + pidx*vtxSz*5
			ioff := idxOff + pidx*idxSz*5
			x0 := barSpc + float32(xi)

			val := data.HeightfieldValue(xi, yi)
			ht := zscale * math32.Abs(val)
			if ht < MinBarHeight {
				ht = MinBarHeight
			}
			base := float32(0)
			if val >= 0 {
				// back
				SetPlane(vertex, normal, texcoord, index, poff, ioff, waxis, zaxis, -1, -1, barPct, ht, x0, base, y0, segs, segs, pos)
				// left
				SetPlane(vertex, normal, texcoord, index, poff+1*vtxSz, ioff+1*idxSz, haxis, zaxis, -1, -1, barPct, ht, y0, base, x0+barPct, segs, segs, pos)
				// right
				SetPlane(vertex, normal, texcoord, index, poff+2*vtxSz, ioff+2*idxSz, haxis, zaxis, 1, -1, barPct, ht, y0, base, x0, segs, segs, pos)
				// top
				SetPlane(vertex, normal, texcoord, index, poff+3*vtxSz, ioff+3*idxSz, waxis, haxis, 1, 1, barPct, barPct, x0, y0, ht, segs, segs, pos)
				// front
				SetPlane(vertex, normal, texcoord, index, poff+4*vtxSz, ioff+4*idxSz, waxis, zaxis, 1, -1, barPct, ht, x0, base, y0+barPct, segs, segs, pos)
			} else {
				base = -ht
				// back
				SetPlane(vertex, normal, texcoord, index, poff, ioff, waxis, zaxis, 1, -1, barPct, ht, x0, base, y0, segs, segs, pos)
				// bottom
				SetPlane(vertex, normal, texcoord, index, poff+3*vtxSz, ioff+3*idxSz, waxis, haxis, 1, 1, barPct, barPct, x0, y0, base, segs, segs, pos)
				// left
				SetPlane(vertex, normal, texcoord, index, poff+1*vtxSz, ioff+1*idxSz, haxis, zaxis, 1, -1, barPct, ht, y0, base, x0+barPct, segs, segs, pos)
				// right
				SetPlane(vertex, normal, texcoord, index, poff+2*vtxSz, ioff+2*idxSz, haxis, zaxis, 1, -1, barPct, ht, y0, base, x0, segs, segs, pos)
				// front
				SetPlane(vertex, normal, texcoord, index, poff+4*vtxSz, ioff+4*idxSz, waxis, zaxis, 1, -1, barPct, ht, x0, base, y0+barPct, segs, segs, pos)
			}
			pidx++
		}
	}
	return nil
}
