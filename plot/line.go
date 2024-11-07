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

// LineStyle has style properties for drawing lines.
type LineStyle struct { //types:add -setters
	// On indicates whether to plot lines.
	On DefaultOffOn

	// Color is the stroke color image specification.
	// Setting to nil turns line off.
	Color image.Image

	// Width is the line width, with a default of 1 Pt (point).
	// Setting to 0 turns line off.
	Width units.Value

	// Dashes are the dashes of the stroke. Each pair of values specifies
	// the amount to paint and then the amount to skip.
	Dashes []float32

	// Fill is the color to fill solid regions, in a plot-specific
	// way (e.g., the area below a Line plot, the bar color).
	// Use nil to disable filling.
	Fill image.Image

	// NegativeX specifies whether to draw lines that connect points with a negative
	// X-axis direction; otherwise there is a break in the line.
	// default is false, so that repeated series of data across the X axis
	// are plotted separately.
	NegativeX bool

	// Step specifies how to step the line between points.
	Step StepKind
}

func (ls *LineStyle) Defaults() {
	ls.Color = colors.Scheme.OnSurface
	ls.Width.Pt(1)
}

// SetStroke sets the stroke style in plot paint to current line style.
// returns false if either the Width = 0 or Color is nil
func (ls *LineStyle) SetStroke(pt *Plot) bool {
	if ls.On == Off || ls.Color == nil {
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

// StepKind specifies a form of a connection of two consecutive points.
type StepKind int32 //enums:enum

const (
	// NoStep connects two points by simple line.
	NoStep StepKind = iota

	// PreStep connects two points by following lines: vertical, horizontal.
	PreStep

	// MidStep connects two points by following lines: horizontal, vertical, horizontal.
	// Vertical line is placed in the middle of the interval.
	MidStep

	// PostStep connects two points by following lines: horizontal, vertical.
	PostStep
)
