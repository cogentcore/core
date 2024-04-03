// Copyright (c) 2022, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package szalloc

import (
	"image"

	"cogentcore.org/core/mat32"
)

// Indexs contains the indexes where a given item image size is allocated
// there is one of these per each ItemSizes
type Indexs struct {

	// percent size of this image relative to max size allocated
	PctSize mat32.Vec2

	// group index
	GpIndex int

	// item index within group (e.g., Layer)
	ItemIndex int
}

func NewIndexs(gpi, itmi int, sz, mxsz image.Point) *Indexs {
	ii := &Indexs{}
	ii.Set(gpi, itmi, sz, mxsz)
	return ii
}

func (ii *Indexs) Set(gpi, itmi int, sz, mxsz image.Point) {
	ii.GpIndex = gpi
	ii.ItemIndex = itmi
	ii.PctSize.X = float32(sz.X) / float32(mxsz.X)
	ii.PctSize.Y = float32(sz.Y) / float32(mxsz.Y)
}
