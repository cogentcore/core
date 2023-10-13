// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"goki.dev/mat32/v2"
)

// BBox contains bounding box and other gross solid properties
type BBox struct {

	// bounding box in local coords
	BBox mat32.Box3

	// bounding sphere in local coords
	BSphere mat32.Sphere

	// area
	Area float32

	// volume
	Volume float32
}

// SetBounds sets BBox from min, max and updates other factors based on that
func (bb *BBox) SetBounds(min, max mat32.Vec3) {
	bb.BBox.Set(&min, &max)
	bb.UpdateFmBBox()
}

// UpdateFmBBox updates other values from BBox
func (bb *BBox) UpdateFmBBox() {
	bb.BSphere.SetFromBox(bb.BBox)
	sz := bb.BBox.Size()
	bb.Area = 2*sz.X + 2*sz.Y + 2*sz.Z
	bb.Volume = sz.X * sz.Y * sz.Z
}
