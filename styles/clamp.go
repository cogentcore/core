// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package styles

import "goki.dev/mat32/v2"

// ClampMax returns given value, not greater than given max _only if_ max > 0
func ClampMax(v, mx float32) float32 {
	if mx <= 0 {
		return v
	}
	return min(v, mx)
}

// ClampMin returns given value, not less than given min _only if_ min > 0
func ClampMin(v, mn float32) float32 {
	if mn <= 0 {
		return v
	}
	return max(v, mn)
}

// SetClampMax ensures the given value is not greater than given max _only if_ max > 0
func SetClampMax(v *float32, mx float32) {
	if mx <= 0 {
		return
	}
	*v = min(*v, mx)
}

// SetClampMin ensures the given value is not less than given min _only if_ min > 0
func SetClampMin(v *float32, mn float32) {
	if mn <= 0 {
		return
	}
	*v = max(*v, mn)
}

// ClampMaxVec returns given Vec2 values, not greater than given max _only if_ max > 0
func ClampMaxVec(v, mx mat32.Vec2) mat32.Vec2 {
	var nv mat32.Vec2
	nv.X = ClampMax(v.X, mx.X)
	nv.Y = ClampMax(v.Y, mx.Y)
	return nv
}

// ClampMinVec returns given Vec2 values, not less than given min _only if_ min > 0
func ClampMinVec(v, mn mat32.Vec2) mat32.Vec2 {
	var nv mat32.Vec2
	nv.X = ClampMin(v.X, mn.X)
	nv.Y = ClampMin(v.Y, mn.Y)
	return nv
}

// SetClampMaxVec ensures the given Vec2 values are not greater than given max _only if_ max > 0
func SetClampMaxVec(v *mat32.Vec2, mx mat32.Vec2) {
	SetClampMax(&v.X, mx.X)
	SetClampMax(&v.Y, mx.Y)
}

// SetClampMinVec ensures the given Vec2 values are not less than given min _only if_ min > 0
func SetClampMinVec(v *mat32.Vec2, mn mat32.Vec2) {
	SetClampMin(&v.X, mn.X)
	SetClampMin(&v.Y, mn.Y)
}
