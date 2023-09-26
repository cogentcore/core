// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/material-foundation/material-color-utilities/blob/main/dart/lib/palettes/core_palette.dart
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

package matcolor

import (
	"image/color"

	"goki.dev/cam/hct"
)

// Key contains the set of key colors used to generate
// a [Scheme] and [Palette]
type Key struct {

	// the primary accent key color
	Primary color.RGBA `desc:"the primary accent key color"`

	// the secondary accent key color
	Secondary color.RGBA `desc:"the secondary accent key color"`

	// the tertiary accent key color
	Tertiary color.RGBA `desc:"the tertiary accent key color"`

	// the error accent key color
	Error color.RGBA `desc:"the error accent key color"`

	// the success accent key color
	Success color.RGBA `desc:"the success accent key color"`

	// the warn accent key color
	Warn color.RGBA `desc:"the warn accent key color"`

	// the neutral key color used to generate surface and surface container colors
	Neutral color.RGBA `desc:"the neutral key color used to generate surface and surface container colors"`

	// the neutral variant key color used to generate surface variant and outline colors
	NeutralVariant color.RGBA `desc:"the neutral variant key color used to generate surface variant and outline colors"`

	// an optional map of custom accent key colors
	Custom map[string]color.RGBA `desc:"an optional map of custom accent key colors"`
}

// Key returns a new [Key] from the given primary accent key color.
func KeyFromPrimary(primary color.RGBA) *Key {
	k := &Key{}
	p := hct.FromColor(primary)
	p.SetTone(40)

	k.Primary = p.WithChroma(max(p.Chroma, 48)).AsRGBA()
	k.Secondary = p.WithChroma(16).AsRGBA()
	// Material adds 60, but we subtract 60 to get green instead of pink when specifying
	// blue (TODO: is this a good idea, or should we just follow Material?)
	k.Tertiary = p.WithHue(p.Hue - 60).WithChroma(24).AsRGBA()
	k.Error = color.RGBA{179, 38, 30, 255} // #B3261E (Material default error color)
	k.Success = color.RGBA{0, 255, 0, 255}
	k.Warn = color.RGBA{255, 255, 0, 255}
	k.Neutral = p.WithChroma(4).AsRGBA()
	k.NeutralVariant = p.WithChroma(8).AsRGBA()
	return k
}
