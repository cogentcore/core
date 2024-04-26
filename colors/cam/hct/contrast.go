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

package hct

import (
	"image/color"

	"cogentcore.org/core/colors/cam/cie"
	"cogentcore.org/core/math32"
)

const (
	// ContrastAA is the contrast ratio required by WCAG AA for body text
	ContrastAA float32 = 4.5

	// ContrastLargeAA is the contrast ratio required by WCAG AA for large text
	// (at least 120-150% larger than the body text)
	ContrastLargeAA float32 = 3

	// ContrastGraphicsAA is the contrast ratio required by WCAG AA for graphical objects
	// and active user interface components like graphs, icons, and form input borders
	ContrastGraphicsAA float32 = 3

	// ContrastAAA is the contrast ratio required by WCAG AAA for body text
	ContrastAAA float32 = 7

	// ContrastLargeAAA is the contrast ratio required by WCAG AAA for large text
	// (at least 120-150% larger than the body text)
	ContrastLargeAAA float32 = 4.5
)

// ContrastRatio returns the contrast ratio between the given two colors.
// The contrast ratio will be between 1 and 21.
func ContrastRatio(a, b color.Color) float32 {
	ah := FromColor(a)
	bh := FromColor(b)
	return ToneContrastRatio(ah.Tone, bh.Tone)
}

// ToneContrastRatio returns the contrast ratio between the given two tones.
// The contrast ratio will be between 1 and 21, and the tones should be
// between 0 and 100 and will be clamped to such.
func ToneContrastRatio(a, b float32) float32 {
	a = math32.Clamp(a, 0, 100)
	b = math32.Clamp(b, 0, 100)
	return ContrastRatioOfYs(cie.LToY(a), cie.LToY(b))
}

// ContrastColor returns the color that will ensure that the given contrast ratio
// between the given color and the resulting color is met. If the given ratio can
// not be achieved with the given color, it returns the color that would result in
// the highest contrast ratio. The ratio must be between 1 and 21. If the tone of
// the given color is greater than 50, it tries darker tones first, and otherwise
// it tries lighter tones first.
func ContrastColor(c color.Color, ratio float32) color.RGBA {
	h := FromColor(c)
	ct := ContrastTone(h.Tone, ratio)
	return h.WithTone(ct).AsRGBA()
}

// ContrastColorTry returns the color that will ensure that the given contrast ratio
// between the given color and the resulting color is met. It returns color.RGBA{}, false if
// the given ratio can not be achieved with the given color. The ratio must be between
// 1 and 21. If the tone of the given color is greater than 50, it tries darker tones first,
// and otherwise it tries lighter tones first.
func ContrastColorTry(c color.Color, ratio float32) (color.RGBA, bool) {
	h := FromColor(c)
	ct, ok := ContrastToneTry(h.Tone, ratio)
	if !ok {
		return color.RGBA{}, false
	}
	return h.WithTone(ct).AsRGBA(), true
}

// ContrastTone returns the tone that will ensure that the given contrast ratio
// between the given tone and the resulting tone is met. If the given ratio can
// not be achieved with the given tone, it returns the tone that would result in
// the highest contrast ratio. The tone must be between 0 and 100 and the ratio must be
// between 1 and 21. If the given tone is greater than 50, it tries darker tones first,
// and otherwise it tries lighter tones first.
func ContrastTone(tone, ratio float32) float32 {
	ct, ok := ContrastToneTry(tone, ratio)
	if ok {
		return ct
	}
	dcr := ToneContrastRatio(tone, 0)
	lcr := ToneContrastRatio(tone, 100)
	if dcr > lcr {
		return 0
	}
	return 100
}

// ContrastToneTry returns the tone that will ensure that the given contrast ratio
// between the given tone and the resulting tone is met. It returns -1, false if
// the given ratio can not be achieved with the given tone. The tone must be between 0
// and 100 and the ratio must be between 1 and 21. If the given tone is greater than 50,
// it tries darker tones first, and otherwise it tries lighter tones first.
func ContrastToneTry(tone, ratio float32) (float32, bool) {
	if tone > 50 {
		d, ok := ContrastToneDarkerTry(tone, ratio)
		if ok {
			return d, true
		}
		l, ok := ContrastToneLighterTry(tone, ratio)
		if ok {
			return l, true
		}
		return -1, false
	}

	l, ok := ContrastToneLighterTry(tone, ratio)
	if ok {
		return l, true
	}
	d, ok := ContrastToneDarkerTry(tone, ratio)
	if ok {
		return d, true
	}
	return -1, false
}

// ContrastToneLighter returns a tone greater than or equal to the given tone
// that ensures that given contrast ratio between the two tones is met.
// It returns 100 if the given ratio can not be achieved with the
// given tone. The tone must be between 0 and 100 and the ratio must be
// between 1 and 21.
func ContrastToneLighter(tone, ratio float32) float32 {
	safe, ok := ContrastToneLighterTry(tone, ratio)
	if ok {
		return safe
	}
	return 100
}

// ContrastToneDarker returns a tone less than or equal to the given tone
// that ensures that given contrast ratio between the two tones is met.
// It returns 0 if the given ratio can not be achieved with the
// given tone. The tone must be between 0 and 100 and the ratio must be
// between 1 and 21.
func ContrastToneDarker(tone, ratio float32) float32 {
	safe, ok := ContrastToneDarkerTry(tone, ratio)
	if ok {
		return safe
	}
	return 0
}

// ContrastToneLighterTry returns a tone greater than or equal to the given tone
// that ensures that given contrast ratio between the two tones is met.
// It returns -1, false if the given ratio can not be achieved with the
// given tone. The tone must be between 0 and 100 and the ratio must be
// between 1 and 21.
func ContrastToneLighterTry(tone, ratio float32) (float32, bool) {
	if tone < 0 || tone > 100 {
		return -1, false
	}

	darkY := cie.LToY(tone)
	lightY := ratio*(darkY+5) - 5
	realContrast := ContrastRatioOfYs(lightY, darkY)
	delta := math32.Abs(realContrast - ratio)
	if realContrast < ratio && delta > 0.04 {
		return -1, false
	}

	// TODO(kai/cam): this +0.4 explained by the comment below only seems to cause problems
	// Ensure gamut mapping, which requires a 'range' on tone, will still result
	// the correct ratio by darkening slightly.
	ret := cie.YToL(lightY) // + 0.4
	if ret < 0 || ret > 100 {
		return -1, false
	}
	return ret, true
}

// ContrastToneDarkerTry returns a tone less than or equal to the given tone
// that ensures that given contrast ratio between the two tones is met.
// It returns -1, false if the given ratio can not be achieved with the
// given tone. The tone must be between 0 and 100 and the ratio must be
// between 1 and 21.
func ContrastToneDarkerTry(tone, ratio float32) (float32, bool) {
	if tone < 0 || tone > 100 {
		return -1, false
	}

	lightY := cie.LToY(tone)
	darkY := ((lightY + 5) / ratio) - 5
	realContrast := ContrastRatioOfYs(lightY, darkY)
	delta := math32.Abs(realContrast - ratio)
	if realContrast < ratio && delta > 0.04 {
		return -1, false
	}

	// TODO(kai/cam): this -0.4 explained by the comment below only seems to cause problems
	// Ensure gamut mapping, which requires a 'range' on tone, will still result
	// the correct ratio by darkening slightly.
	ret := cie.YToL(darkY) // - 0.4
	if ret < 0 || ret > 100 {
		return -1, false
	}
	return ret, true
}

// ContrastRatioOfYs returns the contrast ratio of two XYZ Y values.
func ContrastRatioOfYs(a, b float32) float32 {
	lighter := max(a, b)
	darker := min(a, b)
	return (lighter + 5) / (darker + 5)
}
