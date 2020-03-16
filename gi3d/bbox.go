// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import (
	"github.com/goki/mat32"
)

// BBox contains bounding box and other gross solid properties
type BBox struct {
	BBox    mat32.Box3   `desc:"bounding box in local coords"`
	BSphere mat32.Sphere `desc:"bounding sphere in local coords"`
	Area    float32      `desc:"area"`
	Volume  float32      `desc:"volume"`
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
