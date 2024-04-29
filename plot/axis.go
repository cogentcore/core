// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Adapted initially from gonum/plot:
// Copyright ©2015 The Gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plot

import (
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
)

// Normalizer rescales values from the data coordinate system to the
// normalized coordinate system.
type Normalizer interface {
	// Normalize transforms a value x in the data coordinate system to
	// the normalized coordinate system.
	Normalize(min, max, x float32) float32
}

// Axis represents either a horizontal or vertical
// axis of a plot.
type Axis struct {
	// Min and Max are the minimum and maximum data
	// values represented by the axis.
	Min, Max float32

	// specifies which axis this is: X or Y
	Axis math32.Dims

	// Label for the axis
	Label Text

	// Line styling properties for the axis line.
	Line LineStyle

	// Padding between the axis line and the data.  Having
	// non-zero padding ensures that the data is never drawn
	// on the axis, thus making it easier to see.
	Padding units.Value

	// has the text style for rendering tick labels, and is shared for actual rendering
	TickText Text

	// line style for drawing tick lines
	TickLine LineStyle

	// length of tick lines
	TickLength units.Value

	// Ticker generates the tick marks.  Any tick marks
	// returned by the Marker function that are not in
	// range of the axis are not drawn.
	Ticker Ticker

	// Scale transforms a value given in the data coordinate system
	// to the normalized coordinate system of the axis—its distance
	// along the axis as a fraction of the axis range.
	Scale Normalizer

	// AutoRescale enables an axis to automatically adapt its minimum
	// and maximum boundaries, according to its underlying Ticker.
	AutoRescale bool

	// cached list of ticks, set in size
	ticks []Tick
}

// Sets Defaults, range is (∞, ­∞), and thus any finite
// value is less than Min and greater than Max.
func (ax *Axis) Defaults(dim math32.Dims) {
	ax.Min = math32.Inf(+1)
	ax.Max = math32.Inf(-1)
	ax.Axis = dim
	ax.Line.Defaults()
	ax.Label.Defaults()
	ax.Label.Style.Size.Dp(20)
	ax.Padding.Pt(5)
	ax.TickText.Defaults()
	ax.TickText.Style.Size.Dp(16)
	ax.TickText.Style.Padding.Dp(2)
	ax.TickLine.Defaults()
	ax.TickLength.Pt(8)
	if dim == math32.Y {
		ax.Label.Style.Rotation = -90
		ax.TickText.Style.Align = styles.End
	}
	ax.Scale = LinearScale{}
	ax.Ticker = DefaultTicks{}
}

// SanitizeRange ensures that the range of the axis makes sense.
func (ax *Axis) SanitizeRange() {
	if math32.IsInf(ax.Min, 0) {
		ax.Min = 0
	}
	if math32.IsInf(ax.Max, 0) {
		ax.Max = 0
	}
	if ax.Min > ax.Max {
		ax.Min, ax.Max = ax.Max, ax.Min
	}
	if ax.Min == ax.Max {
		ax.Min--
		ax.Max++
	}

	if ax.AutoRescale {
		marks := ax.Ticker.Ticks(ax.Min, ax.Max)
		for _, t := range marks {
			ax.Min = math32.Min(ax.Min, t.Value)
			ax.Max = math32.Max(ax.Max, t.Value)
		}
	}
}

// LinearScale an be used as the value of an Axis.Scale function to
// set the axis to a standard linear scale.
type LinearScale struct{}

var _ Normalizer = LinearScale{}

// Normalize returns the fractional distance of x between min and max.
func (LinearScale) Normalize(min, max, x float32) float32 {
	return (x - min) / (max - min)
}

// LogScale can be used as the value of an Axis.Scale function to
// set the axis to a log scale.
type LogScale struct{}

var _ Normalizer = LogScale{}

// Normalize returns the fractional logarithmic distance of
// x between min and max.
func (LogScale) Normalize(min, max, x float32) float32 {
	if min <= 0 || max <= 0 || x <= 0 {
		panic("Values must be greater than 0 for a log scale.")
	}
	logMin := math32.Log(min)
	return (math32.Log(x) - logMin) / (math32.Log(max) - logMin)
}

// InvertedScale can be used as the value of an Axis.Scale function to
// invert the axis using any Normalizer.
type InvertedScale struct{ Normalizer }

var _ Normalizer = InvertedScale{}

// Normalize returns a normalized [0, 1] value for the position of x.
func (is InvertedScale) Normalize(min, max, x float32) float32 {
	return is.Normalizer.Normalize(max, min, x)
}

// Norm returns the value of x, given in the data coordinate
// system, normalized to its distance as a fraction of the
// range of this axis.  For example, if x is a.Min then the return
// value is 0, and if x is a.Max then the return value is 1.
func (ax *Axis) Norm(x float32) float32 {
	return ax.Scale.Normalize(ax.Min, ax.Max, x)
}
