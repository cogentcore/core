// Copyright (c) 2023, Cogent Core. All rights reserved.
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

import "cogentcore.org/core/math32"

// SanitizeDegrees ensures that degrees is in [0-360) range
func SanitizeDegrees(deg float32) float32 {
	if deg < 0 {
		return math32.Mod(deg, 360) + 360
	} else if deg >= 360 {
		return math32.Mod(deg, 360)
	}

	return deg
}

// SanitizeRadians sanitizes a small enough angle in radians.
// Takes an angle in radians; must not deviate too much from 0,
// and returns a coterminal angle between 0 and 2pi.
func SanitizeRadians(angle float32) float32 {
	return math32.Mod(angle+math32.Pi*8, math32.Pi*2)
}

// InCyclicOrder returns true a, b, c are in order around a circle
func InCyclicOrder(a, b, c float32) bool {
	delta_a_b := SanitizeRadians(b - a)
	delta_a_c := SanitizeRadians(c - a)
	return delta_a_b < delta_a_c
}
