// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package styles

import "cogentcore.org/core/math32"

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

// ClampMaxVector returns given Vector2 values, not greater than given max _only if_ max > 0
func ClampMaxVector(v, mx math32.Vector2) math32.Vector2 {
	var nv math32.Vector2
	nv.X = ClampMax(v.X, mx.X)
	nv.Y = ClampMax(v.Y, mx.Y)
	return nv
}

// ClampMinVector returns given Vector2 values, not less than given min _only if_ min > 0
func ClampMinVector(v, mn math32.Vector2) math32.Vector2 {
	var nv math32.Vector2
	nv.X = ClampMin(v.X, mn.X)
	nv.Y = ClampMin(v.Y, mn.Y)
	return nv
}

// SetClampMaxVector ensures the given Vector2 values are not greater than given max _only if_ max > 0
func SetClampMaxVector(v *math32.Vector2, mx math32.Vector2) {
	SetClampMax(&v.X, mx.X)
	SetClampMax(&v.Y, mx.Y)
}

// SetClampMinVector ensures the given Vector2 values are not less than given min _only if_ min > 0
func SetClampMinVector(v *math32.Vector2, mn math32.Vector2) {
	SetClampMin(&v.X, mn.X)
	SetClampMin(&v.Y, mn.Y)
}
