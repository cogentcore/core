// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Adapted from github.com/gonum/plot:
// Copyright ©2015 The Gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plot

//go:generate core generate -add-types

import (
	"image"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/styles"
)

// Plot is the basic type representing a plot.
// It renders into its own image.RGBA Pixels image,
// and can also save a corresponding SVG version.
// The Axis ranges are updated automatically when plots
// are added, so setting a fixed range should happen
// after that point.  See [UpdateRange] method as well.
type Plot struct {
	// Title of the plot
	Title Text

	// Background is the background of the plot.
	// The default is [colors.Scheme.Surface].
	Background image.Image

	// standard text style with default options
	StandardTextStyle styles.Text

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

	// DPI is the dots per inch for rendering the image.
	// Larger numbers result in larger scaling of the plot contents
	// which is strongly recommended for print (e.g., use 300 for print)
	DPI float32 `default:"96,160,300"`

	// painter for rendering
	Paint *paint.Context

	// pixels that we render into
	Pixels *image.RGBA `copier:"-" json:"-" xml:"-" edit:"-"`

	// Current plot bounding box in image coordinates, for plotting coordinates
	PlotBox math32.Box2
}

// Defaults sets defaults
func (pt *Plot) Defaults() {
	pt.Title.Defaults()
	pt.Title.Style.Size.Dp(24)
	pt.Background = colors.Scheme.Surface
	pt.X.Defaults(math32.X)
	pt.Y.Defaults(math32.Y)
	pt.Legend.Defaults()
	pt.DPI = 96
	pt.Size = image.Point{1280, 1024}
	pt.StandardTextStyle.Defaults()
	pt.StandardTextStyle.WhiteSpace = styles.WhiteSpaceNowrap
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
	pt.Plotters = append(pt.Plotters, ps...)
}

// SetPixels sets the backing pixels image to given image.RGBA
func (pt *Plot) SetPixels(img *image.RGBA) {
	pt.Pixels = img
	pt.Paint = paint.NewContextFromImage(pt.Pixels)
	pt.Paint.UnitContext.DPI = pt.DPI
	pt.Size = pt.Pixels.Bounds().Size()
	pt.UpdateRange() // needs context, to automatically update for labels
}

// Resize sets the size of the output image to given size.
// Does nothing if already the right size.
func (pt *Plot) Resize(sz image.Point) {
	if pt.Pixels != nil {
		ib := pt.Pixels.Bounds().Size()
		if ib == sz {
			pt.Size = sz
			pt.Paint.UnitContext.DPI = pt.DPI
			return // already good
		}
	}
	pt.SetPixels(image.NewRGBA(image.Rectangle{Max: sz}))
}

func (pt *Plot) SaveImage(filename string) error {
	return imagex.Save(pt.Pixels, filename)
}

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
		ticks[i] = Tick{float32(i), name}
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
		ticks[i] = Tick{float32(i), name}
	}
	pt.Y.Ticker = ConstantTicks(ticks)
}

// UpdateRange updates the axis range values based on current Plot values.
// This first resets the range so any fixed additional range values should
// be set after this point.
func (pt *Plot) UpdateRange() {
	pt.X.Min = math32.Inf(+1)
	pt.X.Max = math32.Inf(-1)
	pt.Y.Min = math32.Inf(+1)
	pt.Y.Max = math32.Inf(-1)
	for _, d := range pt.Plotters {
		pt.UpdateRangeFromPlotter(d)
	}
}

func (pt *Plot) UpdateRangeFromPlotter(d Plotter) {
	if x, ok := d.(DataRanger); ok {
		xmin, xmax, ymin, ymax := x.DataRange(pt)
		pt.X.Min = math32.Min(pt.X.Min, xmin)
		pt.X.Max = math32.Max(pt.X.Max, xmax)
		pt.Y.Min = math32.Min(pt.Y.Min, ymin)
		pt.Y.Max = math32.Max(pt.Y.Max, ymax)
	}
}

// PX returns the X-axis plotting coordinate for given raw data value
// using the current plot bounding region
func (pt *Plot) PX(v float32) float32 {
	return pt.PlotBox.ProjectX(pt.X.Norm(v))
}

// PY returns the Y-axis plotting coordinate for given raw data value
func (pt *Plot) PY(v float32) float32 {
	return pt.PlotBox.ProjectY(1 - pt.Y.Norm(v))
}

// ClosestDataToPixel returns the Plotter data point closest to given pixel point,
// in the Pixels image.
func (pt *Plot) ClosestDataToPixel(px, py int) (plt Plotter, idx int, dist float32, data, pixel math32.Vector2, legend string) {
	tp := math32.Vec2(float32(px), float32(py))
	dist = float32(math32.MaxFloat32)
	for _, p := range pt.Plotters {
		dts, pxls := p.XYData()
		for i := range pxls.Len() {
			ptx, pty := pxls.XY(i)
			pxy := math32.Vec2(ptx, pty)
			d := pxy.DistanceTo(tp)
			if d < dist {
				dist = d
				pixel = pxy
				plt = p
				idx = i
				dx, dy := dts.XY(i)
				data = math32.Vec2(dx, dy)
				legend = pt.Legend.LegendForPlotter(p)
			}
		}
	}
	return
}
