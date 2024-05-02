// Copyright (c) 2022, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sltype

import "cogentcore.org/core/math32"

// Int is identical to an int32
type Int = int32

// Int2 is a length 2 vector of int32
type Int2 = math32.Vector2i

// Int3 is a length 3 vector of int32
type Int3 = math32.Vector3i

// Int4 is a length 4 vector of int32
type Int4 struct {
	X int32
	Y int32
	Z int32
	W int32
}

////////////////////////////////////////
// Unsigned

// Uint is identical to a uint32
type Uint = uint32

// Uint2 is a length 2 vector of uint32
type Uint2 struct {
	X uint32
	Y uint32
}

// Uint3 is a length 3 vector of uint32
type Uint3 struct {
	X uint32
	Y uint32
	Z uint32
}

// Uint4 is a length 4 vector of uint32
type Uint4 struct {
	X uint32
	Y uint32
	Z uint32
	W uint32
}

func (u *Uint4) SetFrom2(u2 Uint2) {
	u.X = u2.X
	u.Y = u2.Y
	u.Z = 0
	u.W = 1
}
