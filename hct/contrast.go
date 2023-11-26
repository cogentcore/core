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

package hct

import (
	"goki.dev/cam/cie"
	"goki.dev/mat32/v2"
)

// ContrastRatio returns the contrast ratio between the given two tones.
// The contrast ratio will be between 1 and 21, and the tones should be
// between 0 and 100 and will be clamped to such.
func ContrastRatio(a, b float32) float32 {
	a = mat32.Clamp(a, 0, 100)
	b = mat32.Clamp(b, 0, 100)
	return RatioOfYs(cie.LToY(a), cie.LToY(b))
}

// RatioOfYs returns the contrast ratio of two XYZ Y values.
func RatioOfYs(a, b float32) float32 {
	lighter := max(a, b)
	darker := min(a, b)
	return (lighter + 5) / (darker + 5)
}

// Lighter returns a tone greater than or equal to the given tone
// that ensures that given contrast ratio between the two tones is met.
// It returns -1, false if the given ratio can not be achieved with the
// given tone. Tone must be between 0 and 100 and ratio must be between
// 1 and 21.
func Lighter(tone, ratio float32) (float32, bool) {
	if tone < 0 || tone > 100 {
		return -1, false
	}

	darkY := cie.LToY(tone)
	lightY := ratio*(darkY+5) - 5
	realContrast := RatioOfYs(lightY, darkY)
	delta := mat32.Abs(realContrast - ratio)
	if realContrast < ratio && delta > 0.04 {
		return -1, false
	}

	// Ensure gamut mapping, which requires a 'range' on tone, will still result
	// the correct ratio by darkening slightly.
	ret := cie.YToL(lightY) + 0.4
	if ret < 0 || ret > 100 {
		return -1, false
	}
	return ret, true
}

// Darker returns a tone less than or equal to the given tone
// that ensures that given contrast ratio between the two tones is met.
// It returns -1, false if the given ratio can not be achieved with the
// given tone. Tone must be between 0 and 100 and ratio must be between
// 1 and 21.
func Darker(tone, ratio float32) (float32, bool) {
	if tone < 0 || tone > 100 {
		return -1, false
	}

	lightY := cie.LToY(tone)
	darkY := ((lightY + 5) / ratio) - 5
	realContrast := RatioOfYs(lightY, darkY)
	delta := mat32.Abs(realContrast - ratio)
	if realContrast < ratio && delta > 0.04 {
		return -1, false
	}

	// Ensure gamut mapping, which requires a 'range' on tone, will still result
	// the correct ratio by darkening slightly.
	ret := cie.YToL(lightY) + 0.4
	if ret < 0 || ret > 100 {
		return -1, false
	}
	return ret, true
}

// LighterUnsafe returns a tone greater than or equal to the given tone
// that ensures that given contrast ratio between the two tones is met.
// It returns 100 if the given ratio can not be achieved with the
// given tone. Tone must be between 0 and 100 and ratio must be between
// 1 and 21. This function is unsafe because the returned value may not
// satisfy the ratio requirement.
func LighterUnsafe(tone, ratio float32) float32 {
	safe, ok := Lighter(tone, ratio)
	if ok {
		return safe
	}
	return 100
}

// DarkerUnsafe returns a tone less than or equal to the given tone
// that ensures that given contrast ratio between the two tones is met.
// It returns 0 if the given ratio can not be achieved with the
// given tone. Tone must be between 0 and 100 and ratio must be between
// 1 and 21. This function is unsafe because the returned value may not
// satisfy the ratio requirement.
func DarkerUnsafe(tone, ratio float32) float32 {
	safe, ok := Darker(tone, ratio)
	if ok {
		return safe
	}
	return 0
}
