// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Adapted initially from gonum/plot:
// Copyright ©2015 The Gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plot

import (
	"image"
	"image/color"
	"math"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
)

// Plot is the basic type representing a plot.
// It renders into
type Plot struct {
	// Title of the plot
	Title Text

	// Background is the background color of the plot.
	// The default is White.
	Background color.Color

	// X and Y are the horizontal and vertical axes
	// of the plot respectively.
	X, Y Axis

	// Legend is the plot's legend.
	Legend Legend

	// plotters are drawn by calling their Plot method
	// after the axes are drawn.
	Plotters []Plotter

	// size is the target size of the image to render to
	Size image.Point

	// painter for rendering
	Paint paint.Context

	// pixels that we render into
	Pixels *image.RGBA `copier:"-" json:"-" xml:"-" edit:"-"`

	// unit context: parameters necessary for anchoring relative units
	UnitContext units.Context

	// standard text style with default params
	StdTextStyle styles.Text
}

// Defaults sets defaults
func (pt *Plot) Defaults() {
	pt.Title.Defaults()
	pt.Title.Style.Size.Pt(16)
	pt.Title.Style.Align = styles.Center
	pt.Background = color.White
	pt.X.Defaults(math32.X)
	pt.Y.Defaults(math32.X)
	pt.Legend.Defaults()
	pt.Size = image.Point{1280, 1024}
	pt.StdTextStyle.Defaults()
	pt.StdTextStyle.WhiteSpace = styles.WhiteSpaceNowrap
}

// New returns a new plot with some reasonable default settings.
func New() *Plot {
	pt := &Plot{}
	pt.Defaults()
	return pt
}

// Add adds a Plotters to the plot.
//
// If the plotters implements DataRanger then the
// minimum and maximum values of the X and Y
// axes are changed if necessary to fit the range of
// the data.
//
// When drawing the plot, Plotters are drawn in the
// order in which they were added to the plot.
func (pt *Plot) Add(ps ...Plotter) {
	for _, d := range ps {
		if x, ok := d.(DataRanger); ok {
			xmin, xmax, ymin, ymax := x.DataRange()
			pt.X.Min = math.Min(pt.X.Min, xmin)
			pt.X.Max = math.Max(pt.X.Max, xmax)
			pt.Y.Min = math.Min(pt.Y.Min, ymin)
			pt.Y.Max = math.Max(pt.Y.Max, ymax)
		}
	}

	pt.Plotters = append(pt.Plotters, ps...)
}

// DrawConfig configures everything for drawing
func (pt *Plot) DrawConfig() {
	// todo: ensure image, do units

}

// Draw draws the plot to image.
// Plotters are drawn in the order in which they were
// added to the plot.
func (pt *Plot) Draw() {
	pt.DrawConfig()
	if pt.Background != nil {
		// c.SetColor(p.Background)
		// c.Fill(c.Rectangle.Path())
	}

	if pt.Title.Text != "" {
		pt.Title.Config(pt)
		// descent := pt.Title.TextStyle.FontExtents().Descent
		// c.FillText(pt.Title.TextStyle, vg.Point{X: c.Center().X, Y: c.Max.Y + descent}, pt.Title.Text)
		// rect := pt.Title.TextStyle.Rectangle(pt.Title.Text)
		// c.Max.Y -= rect.Size().Y
		// c.Max.Y -= pt.Title.Padding
	}

	pt.X.SanitizeRange()
	pt.Y.SanitizeRange()

	ywidth := pt.Y.Size(pt)
	xheight := pt.X.Size(pt)
	_, _ = ywidth, xheight

	// x.draw(padX(pt, draw.Crop(c, ywidth, 0, 0, 0)))
	// y.draw(padY(pt, draw.Crop(c, 0, 0, xheight, 0)))

	// dataC := padY(pt, padX(pt, draw.Crop(c, ywidth, 0, xheight, 0)))
	for _, plt := range pt.Plotters {
		plt.Plot(pt)
	}

	pt.Legend.Draw(pt)
}

// DataCanvas returns a new draw.Canvas that
// is the subset of the given draw area into which
// the plot data will be drawn.
// func (pt *Plot) DataCanvas(da draw.Canvas) draw.Canvas {
// 	if pt.Title.Text != "" {
// 		rect := pt.Title.TextStyle.Rectangle(pt.Title.Text)
// 		da.Max.Y -= rect.Size().Y
// 		da.Max.Y -= pt.Title.Padding
// 	}
// 	pt.X.sanitizeRange()
// 	x := horizontalAxis{pt.X}
// 	pt.Y.sanitizeRange()
// 	y := verticalAxis{pt.Y}
// 	return padY(pt, padX(pt, draw.Crop(da, y.size(), 0, x.size(), 0)))
// }

/*
// padX returns a draw.Canvas that is padded horizontally
// so that glyphs will no be clipped.
func padX(pt *Plot, c draw.Canvas) draw.Canvas {
	glyphs := pt.GlyphBoxes(pt)
	l := leftMost(&c, glyphs)
	xAxis := horizontalAxis{pt.X}
	glyphs = append(glyphs, xAxis.GlyphBoxes(pt)...)
	r := rightMost(&c, glyphs)

	minx := c.Min.X - l.Min.X
	maxx := c.Max.X - (r.Min.X + r.Size().X)
	lx := vg.Length(l.X)
	rx := vg.Length(r.X)
	n := (lx*maxx - rx*minx) / (lx - rx)
	m := ((lx-1)*maxx - rx*minx + minx) / (lx - rx)
	return draw.Canvas{
		Canvas: vg.Canvas(c),
		Rectangle: vg.Rectangle{
			Min: vg.Point{X: n, Y: c.Min.Y},
			Max: vg.Point{X: m, Y: c.Max.Y},
		},
	}
}

// padY returns a draw.Canvas that is padded vertically
// so that glyphs will no be clipped.
func padY(pt *Plot, c draw.Canvas) draw.Canvas {
	glyphs := pt.GlyphBoxes(pt)
	b := bottomMost(&c, glyphs)
	yAxis := verticalAxis{pt.Y}
	glyphs = append(glyphs, yAxis.GlyphBoxes(pt)...)
	t := topMost(&c, glyphs)

	miny := c.Min.Y - b.Min.Y
	maxy := c.Max.Y - (t.Min.Y + t.Size().Y)
	by := vg.Length(b.Y)
	ty := vg.Length(t.Y)
	n := (by*maxy - ty*miny) / (by - ty)
	m := ((by-1)*maxy - ty*miny + miny) / (by - ty)
	return draw.Canvas{
		Canvas: vg.Canvas(c),
		Rectangle: vg.Rectangle{
			Min: vg.Point{Y: n, X: c.Min.X},
			Max: vg.Point{Y: m, X: c.Max.X},
		},
	}
}

// Transforms returns functions to transfrom
// from the x and y data coordinate system to
// the draw coordinate system of the given
// draw area.
func (pt *Plot) Transforms(c *draw.Canvas) (x, y func(float64) vg.Length) {
	x = func(x float64) vg.Length { return c.X(pt.X.Norm(x)) }
	y = func(y float64) vg.Length { return c.Y(pt.Y.Norm(y)) }
	return
}
*/

// NominalX configures the plot to have a nominal X
// axis—an X axis with names instead of numbers.  The
// X location corresponding to each name are the integers,
// e.g., the x value 0 is centered above the first name and
// 1 is above the second name, etc.  Labels for x values
// that do not end up in range of the X axis will not have
// tick marks.
func (pt *Plot) NominalX(names ...string) {
	pt.X.TickLine.Width.Pt(0)
	pt.X.TickLength.Pt(0)
	pt.X.Line.Width.Pt(0)
	// pt.Y.Padding.Pt(pt.X.Tick.Label.Width(names[0]) / 2)
	ticks := make([]Tick, len(names))
	for i, name := range names {
		ticks[i] = Tick{float64(i), name}
	}
	pt.X.Ticker = ConstantTicks(ticks)
}

// HideX configures the X axis so that it will not be drawn.
func (pt *Plot) HideX() {
	pt.X.TickLength.Pt(0)
	pt.X.Line.Width.Pt(0)
	pt.X.Ticker = ConstantTicks([]Tick{})
}

// HideY configures the Y axis so that it will not be drawn.
func (pt *Plot) HideY() {
	pt.Y.TickLength.Pt(0)
	pt.Y.Line.Width.Pt(0)
	pt.Y.Ticker = ConstantTicks([]Tick{})
}

// HideAxes hides the X and Y axes.
func (pt *Plot) HideAxes() {
	pt.HideX()
	pt.HideY()
}

// NominalY is like NominalX, but for the Y axis.
func (pt *Plot) NominalY(names ...string) {
	pt.Y.TickLine.Width.Pt(0)
	pt.Y.TickLength.Pt(0)
	pt.Y.Line.Width.Pt(0)
	// pt.X.Padding = pt.Y.Tick.Label.Height(names[0]) / 2
	ticks := make([]Tick, len(names))
	for i, name := range names {
		ticks[i] = Tick{float64(i), name}
	}
	pt.Y.Ticker = ConstantTicks(ticks)
}
