// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package szalloc

import (
	"image"

	"github.com/goki/mat32"
)

// Idxs contains the indexes where a given item image size is allocated
// there is one of these per each ItemSizes
type Idxs struct {
	PctSize mat32.Vec2 `desc:"percent size of this image relative to max size allocated"`
	GpIdx   int        `desc:"group index"`
	ItemIdx int        `desc:"item index within group (e.g., Layer)"`
}

func NewIdxs(gpi, itmi int, sz, mxsz image.Point) *Idxs {
	ii := &Idxs{}
	ii.Set(gpi, itmi, sz, mxsz)
	return ii
}

func (ii *Idxs) Set(gpi, itmi int, sz, mxsz image.Point) {
	ii.GpIdx = gpi
	ii.ItemIdx = itmi
	ii.PctSize.X = float32(sz.X) / float32(mxsz.X)
	ii.PctSize.Y = float32(sz.Y) / float32(mxsz.Y)
}
