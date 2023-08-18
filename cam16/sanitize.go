// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Adapted from https://github.com/material-foundation/material-color-utilities
// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cam16

import "github.com/goki/mat32"

// SanitizeDegInt ensures that degrees is in [0-360) range
func SanitizeDegInt(deg int) int {
	if deg < 0 {
		return (deg % 360) + 360
	} else if deg >= 360.0 {
		return deg % 360
	} else {
		return deg
	}
}

// SanitizeDeg ensures that degrees is in [0-360) range
func SanitizeDeg(deg float32) float32 {
	if deg < 0 {
		return mat32.Mod(deg, 360) + 360
	} else if deg >= 360 {
		return mat32.Mod(deg, 360)
	} else {
		return deg
	}
}

// SanitizeRadians Sanitizes a small enough angle in radians.
// Takes an angle in radians; must not deviate too much from 0,
// and returns a coterminal angle between 0 and 2pi.
func SanitizeRadians(angle float32) float32 {
	return mat32.Mod(angle+mat32.Pi*8, mat32.Pi*2)
}

// InCyclicOrder returns true a, b, c are in order around a circle
func InCyclicOrder(a, b, c float32) bool {
	delta_a_b := SanitizeRadians(b - a)
	delta_a_c := SanitizeRadians(c - a)
	return delta_a_b < delta_a_c
}

func DiffDeg(a, b float32) float32 {
	return 180 - mat32.Abs(mat32.Abs(a-b)-180)
}

// RotationDirection returns 1 if to-from < 180 deg, else -1
func RotationDirection(from, to float32) float32 {
	inc := SanitizeDeg(to - from)
	if inc < 180 {
		return 1
	}
	return -1
}
