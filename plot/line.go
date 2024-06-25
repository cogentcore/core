// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plot

import (
	"image"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
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
	ls.Color = colors.Scheme.OnSurface
	ls.Width.Pt(1)
}

// SetStroke sets the stroke style in plot paint to current line style.
// returns false if either the Width = 0 or Color is nil
func (ls *LineStyle) SetStroke(pt *Plot) bool {
	if ls.Color == nil {
		return false
	}
	pc := pt.Paint
	uc := &pc.UnitContext
	ls.Width.ToDots(uc)
	if ls.Width.Dots == 0 {
		return false
	}
	pc.StrokeStyle.Width = ls.Width
	pc.StrokeStyle.Color = ls.Color
	pc.StrokeStyle.ToDots(uc)
	return true
}

// Draw draws a line between given coordinates, setting the stroke style
// to current parameters.  Returns false if either Width = 0 or Color = nil
func (ls *LineStyle) Draw(pt *Plot, start, end math32.Vector2) bool {
	if !ls.SetStroke(pt) {
		return false
	}
	pc := pt.Paint
	pc.MoveTo(start.X, start.Y)
	pc.LineTo(end.X, end.Y)
	pc.Stroke()
	return true
}
