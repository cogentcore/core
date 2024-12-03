// Copyright (c) 2022, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sltype

import "cogentcore.org/core/math32"

// Int32Vec2 is a length 2 vector of int32
type Int32Vec2 = math32.Vector2i

// Int32Vec3 is a length 3 vector of int32
type IntVec3 = math32.Vector3i

// Int32Vec4 is a length 4 vector of int32
type Int32Vec4 struct {
	X int32
	Y int32
	Z int32
	W int32
}

////////  Unsigned

// Uint32Vec2 is a length 2 vector of uint32
type Uint32Vec2 struct {
	X uint32
	Y uint32
}

// Uint32Vec3 is a length 3 vector of uint32
type Uint32Vec3 struct {
	X uint32
	Y uint32
	Z uint32
}

// Uint32Vec4 is a length 4 vector of uint32
type Uint32Vec4 struct {
	X uint32
	Y uint32
	Z uint32
	W uint32
}

func (u *Uint32Vec4) SetFromVec2(u2 Uint32Vec2) {
	u.X = u2.X
	u.Y = u2.Y
	u.Z = 0
	u.W = 1
}
