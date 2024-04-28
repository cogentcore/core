// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plot

import (
	"image"
	"image/color"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/styles/units"
)

// LineStyle has style properties for line drawing
type LineStyle struct {

	// stroke color image specification; stroking is off if nil
	Color image.Image

	// line width
	Width units.Value

	// Dashes are the dashes of the stroke. Each pair of values specifies
	// the amount to paint and then the amount to skip.
	Dashes []float32
}

func (ls *LineStyle) Defaults() {
	ls.Color = colors.C(color.Black)
	ls.Width.Pt(0.5)
}
