// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi3d

import "github.com/goki/gi/mat32"

// Pose contains the full specification of a given object's position and orientation
type Pose struct {
	Pos     mat32.Vector3    // position of center of object
	Scale   mat32.Vector3    // scale (relative to parent)
	Dir     mat32.Vector3    // Initial direction (relative to parent)
	Rot     mat32.Vector3    // Node rotation specified in Euler angles (relative to parent)
	Quat    mat32.Quaternion // Node rotation specified as a Quaternion (relative to parent)
	Xform   mat32.Matrix4    // Local transform matrix. Contains all position/rotation/scale information (relative to parent)
	WorldXf mat32.Matrix4    // World transform matrix. Contains all absolute position/rotation/scale information (i.e. relative to very top parent, generally the scene)
}

// todo: compute things..
