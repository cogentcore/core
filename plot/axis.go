// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Adapted initially from gonum/plot:
// Copyright ©2015 The Gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plot

import (
	"math"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
)

// Normalizer rescales values from the data coordinate system to the
// normalized coordinate system.
type Normalizer interface {
	// Normalize transforms a value x in the data coordinate system to
	// the normalized coordinate system.
	Normalize(min, max, x float64) float64
}

// Axis represents either a horizontal or vertical
// axis of a plot.
type Axis struct {
	// Min and Max are the minimum and maximum data
	// values represented by the axis.
	Min, Max float64

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

	// text style for rendering tick labels
	TickStyle TextStyle

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
}

// Sets Defaults, range is (∞, ­∞), and thus any finite
// value is less than Min and greater than Max.
func (ax *Axis) Defaults(dim math32.Dims) {
	ax.Min = math.Inf(+1)
	ax.Max = math.Inf(-1)
	ax.Axis = dim
	ax.Line.Defaults()
	ax.Label.Defaults()
	ax.Label.Style.Align = styles.Center
	ax.Padding.Pt(5)
	ax.TickStyle.Defaults()
	ax.TickStyle.Size.Pt(10)
	ax.TickLine.Defaults()
	ax.TickLength.Pt(8)
	ax.Scale = LinearScale{}
	ax.Ticker = DefaultTicks{}
}

// SanitizeRange ensures that the range of the axis makes sense.
func (ax *Axis) SanitizeRange() {
	if math.IsInf(ax.Min, 0) {
		ax.Min = 0
	}
	if math.IsInf(ax.Max, 0) {
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
			ax.Min = math.Min(ax.Min, t.Value)
			ax.Max = math.Max(ax.Max, t.Value)
		}
	}
}

// LinearScale an be used as the value of an Axis.Scale function to
// set the axis to a standard linear scale.
type LinearScale struct{}

var _ Normalizer = LinearScale{}

// Normalize returns the fractional distance of x between min and max.
func (LinearScale) Normalize(min, max, x float64) float64 {
	return (x - min) / (max - min)
}

// LogScale can be used as the value of an Axis.Scale function to
// set the axis to a log scale.
type LogScale struct{}

var _ Normalizer = LogScale{}

// Normalize returns the fractional logarithmic distance of
// x between min and max.
func (LogScale) Normalize(min, max, x float64) float64 {
	if min <= 0 || max <= 0 || x <= 0 {
		panic("Values must be greater than 0 for a log scale.")
	}
	logMin := math.Log(min)
	return (math.Log(x) - logMin) / (math.Log(max) - logMin)
}

// InvertedScale can be used as the value of an Axis.Scale function to
// invert the axis using any Normalizer.
type InvertedScale struct{ Normalizer }

var _ Normalizer = InvertedScale{}

// Normalize returns a normalized [0, 1] value for the position of x.
func (is InvertedScale) Normalize(min, max, x float64) float64 {
	return is.Normalizer.Normalize(max, min, x)
}

// Norm returns the value of x, given in the data coordinate
// system, normalized to its distance as a fraction of the
// range of this axis.  For example, if x is a.Min then the return
// value is 0, and if x is a.Max then the return value is 1.
func (ax *Axis) Norm(x float64) float64 {
	return ax.Scale.Normalize(ax.Min, ax.Max, x)
}

// drawTicks returns true if the tick marks should be drawn.
func (ax *Axis) drawTicks() bool {
	return ax.TickLine.Width.Value > 0 && ax.TickLength.Value > 0
}

// size returns the Height of X axis or Width of Y axis
func (ax *Axis) Size(pt *Plot) float32 {
	if ax.Axis == math32.X {
		return ax.SizeX(pt)
	} else {
		return ax.SizeY(pt)
	}
}

func (ax *Axis) SizeX(pt *Plot) float32 {
	h := float32(0)
	if ax.Label.Text != "" { // We assume that the label isn't rotated.
		// h += ax.Label.TextStyle.FontExtents().Descent
		// h += ax.Label.TextStyle.Height(ax.Label.Text)
		h += ax.Padding.Dots
	}

	marks := ax.Ticker.Ticks(ax.Min, ax.Max)
	if len(marks) > 0 {
		if ax.drawTicks() {
			h += ax.TickLength.Dots
		}
		// h += tickLabelHeight(ax.Tick.Label, marks)
	}
	h += ax.Line.Width.Dots / 2
	h += ax.Padding.Dots

	return h
}

func (ax *Axis) SizeY(pt *Plot) float32 {
	w := float32(0)
	if ax.Label.Text != "" { // We assume that the label isn't rotated.
		// w += ax.Label.TextStyle.FontExtents().Descent
		// w += ax.Label.TextStyle.Height(ax.Label.Text)
		w += ax.Padding.Dots
	}

	marks := ax.Ticker.Ticks(ax.Min, ax.Max)
	if len(marks) > 0 {
		// if lwidth := tickLabelWidth(ax.Tick.Label, marks); lwidth > 0 {
		// 	w += lwidth
		// 	w += ax.Label.TextStyle.Width(" ")
		// }
		if ax.drawTicks() {
			w += ax.TickLength.Dots
		}
	}
	w += ax.Line.Width.Dots / 2
	w += ax.Padding.Dots

	return w
}

func (ax *Axis) Draw(pt *Plot) {
	if ax.Axis == math32.X {
		ax.DrawX(pt)
	} else {
		ax.DrawY(pt)
	}
}

// DrawX draws the horizontal axis
func (ax *Axis) DrawX(pt *Plot) {
	/*
		var (
			x vg.Length
			y = c.Min.Y
		)
		switch ax.Label.Position {
		case draw.PosCenter:
			x = c.Center().X
		case draw.PosRight:
			x = c.Max.X
			x -= ax.Label.TextStyle.Width(ax.Label.Text) / 2
		}
		if ax.Label.Text != "" {
			descent := ax.Label.TextStyle.FontExtents().Descent
			c.FillText(ax.Label.TextStyle, vg.Point{X: x, Y: y + descent}, ax.Label.Text)
			y += ax.Label.TextStyle.Height(ax.Label.Text)
			y += ax.Label.Padding
		}

		marks := ax.Ticker.Ticks(ax.Min, ax.Max)
		ticklabelheight := tickLabelHeight(ax.Tick.Label, marks)
		descent := ax.Tick.Label.FontExtents().Descent
		for _, t := range marks {
			x := c.X(ax.Norm(t.Value))
			if !c.ContainsX(x) || t.IsMinor() {
				continue
			}
			c.FillText(ax.Tick.Label, vg.Point{X: x, Y: y + ticklabelheight + descent}, t.Label)
		}

		if len(marks) > 0 {
			y += ticklabelheight
		} else {
			y += ax.Width / 2
		}

		if len(marks) > 0 && ax.drawTicks() {
			len := ax.Tick.Length
			for _, t := range marks {
				x := c.X(ax.Norm(t.Value))
				if !c.ContainsX(x) {
					continue
				}
				start := t.lengthOffset(len)
				c.StrokeLine2(ax.Tick.LineStyle, x, y+start, x, y+len)
			}
			y += len
		}

		c.StrokeLine2(ax.LineStyle, c.Min.X, y, c.Max.X, y)
	*/
}

/*
// GlyphBoxes returns the GlyphBoxes for the tick labels.
func (ax horizontalAxis) GlyphBoxes(p *Plot) []GlyphBox {
	var (
		boxes []GlyphBox
		yoff  font.Length
	)

	if ax.Label.Text != "" {
		x := ax.Norm(p.X.Max)
		switch ax.Label.Position {
		case draw.PosCenter:
			x = ax.Norm(0.5 * (p.X.Max + p.X.Min))
		case draw.PosRight:
			x -= ax.Norm(0.5 * ax.Label.TextStyle.Width(ax.Label.Text).Points()) // FIXME(sbinet): want data coordinates
		}
		descent := ax.Label.TextStyle.FontExtents().Descent
		boxes = append(boxes, GlyphBox{
			X:         x,
			Rectangle: ax.Label.TextStyle.Rectangle(ax.Label.Text).Add(vg.Point{Y: yoff + descent}),
		})
		yoff += ax.Label.TextStyle.Height(ax.Label.Text)
		yoff += ax.Label.Padding
	}

	var (
		marks   = ax.Ticker.Ticks(ax.Min, ax.Max)
		height  = tickLabelHeight(ax.Tick.Label, marks)
		descent = ax.Tick.Label.FontExtents().Descent
	)
	for _, t := range marks {
		if t.IsMinor() {
			continue
		}
		box := GlyphBox{
			X:         ax.Norm(t.Value),
			Rectangle: ax.Tick.Label.Rectangle(t.Label).Add(vg.Point{Y: yoff + height + descent}),
		}
		boxes = append(boxes, box)
	}
	return boxes
}
*/

// DrawY draws the Y axis along the left side
func (ax *Axis) DrawY(pt *Plot) {
	/*
		var (
			x = c.Min.X
			y vg.Length
		)
		if ax.Label.Text != "" {
			sty := ax.Label.TextStyle
			sty.Rotation += math.Pi / 2
			x += ax.Label.TextStyle.Height(ax.Label.Text)
			switch ax.Label.Position {
			case draw.PosCenter:
				y = c.Center().Y
			case draw.PosTop:
				y = c.Max.Y
				y -= ax.Label.TextStyle.Width(ax.Label.Text) / 2
			}
			descent := ax.Label.TextStyle.FontExtents().Descent
			c.FillText(sty, vg.Point{X: x - descent, Y: y}, ax.Label.Text)
			x += descent
			x += ax.Label.Padding
		}
		marks := ax.Ticker.Ticks(ax.Min, ax.Max)
		if w := tickLabelWidth(ax.Tick.Label, marks); len(marks) > 0 && w > 0 {
			x += w
		}

		major := false
		descent := ax.Tick.Label.FontExtents().Descent
		for _, t := range marks {
			y := c.Y(ax.Norm(t.Value))
			if !c.ContainsY(y) || t.IsMinor() {
				continue
			}
			c.FillText(ax.Tick.Label, vg.Point{X: x, Y: y + descent}, t.Label)
			major = true
		}
		if major {
			x += ax.Tick.Label.Width(" ")
		}
		if ax.drawTicks() && len(marks) > 0 {
			len := ax.Tick.Length
			for _, t := range marks {
				y := c.Y(ax.Norm(t.Value))
				if !c.ContainsY(y) {
					continue
				}
				start := t.lengthOffset(len)
				c.StrokeLine2(ax.Tick.LineStyle, x+start, y, x+len, y)
			}
			x += len
		}

		c.StrokeLine2(ax.LineStyle, x, c.Min.Y, x, c.Max.Y)
	*/
}
