// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
package bools does conversion to / from booleans and other go standard types
*/
package bools

// ToFloat32 converts a bool to a 1 (true) or 0 (false)
func ToFloat32(b bool) float32 {
	if b {
		return 1
	}
	return 0
}

// ToFloat64 converts a bool to a 1 (true) or 0 (false)
func ToFloat64(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

// ToInt converts a bool to a 1 (true) or 0 (false)
func ToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ToInt32 converts a bool to a 1 (true) or 0 (false)
func ToInt32(b bool) int32 {
	if b {
		return 1
	}
	return 0
}

// ToInt64 converts a bool to a 1 (true) or 0 (false)
func ToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

//////////////////////////////////////////////

// FromFloat32 converts value to a bool, 0 = false, else true
func FromFloat32(v float32) bool {
	if v == 0 {
		return false
	}
	return true
}

// FromFloat64 converts value to a bool, 0 = false, else true
func FromFloat64(v float64) bool {
	if v == 0 {
		return false
	}
	return true
}

// FromInt converts value to a bool, 0 = false, else true
func FromInt(v int) bool {
	if v == 0 {
		return false
	}
	return true
}

// FromInt32 converts value to a bool, 0 = false, else true
func FromInt32(v int32) bool {
	if v == 0 {
		return false
	}
	return true
}

// FromInt64 converts value to a bool, 0 = false, else true
func FromInt64(v int64) bool {
	if v == 0 {
		return false
	}
	return true
}
