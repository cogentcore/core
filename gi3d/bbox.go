// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import "github.com/goki/gi/mat32"

// BBox contains bounding box and other gross object properties
type BBox struct {
	BBox       mat32.Box3   `desc:"Last calculated bounding box in local coords"`
	BSphere    mat32.Sphere `desc:"Last calculated bounding sphere in local coords"`
	Area       float32      `desc:"Last calculated area"`
	Volume     float32      `desc:"Last calculated volume"`
	RotInertia mat32.Mat3   `desc:"Last calculated rotational inertia matrix in local coords"`
}

// SetBounds sets BBox from min, max and updates other factors based on that
func (bb *BBox) SetBounds(min, max mat32.Vec3) {
	bb.BBox.Set(&min, &max)
	bb.BSphere.SetFromBox(bb.BBox)
	sz := bb.BBox.Size()
	bb.Area = 2*sz.X + 2*sz.Y + 2*sz.Z
	bb.Volume = sz.X * sz.Y * sz.Z
	// todo: RotInertia
}
