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
	"cogentcore.org/core/math32/minmax"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
)

// XAxisStyle has overall plot level styling properties for the XAxis.
type XAxisStyle struct { //types:add -setters
	// Column specifies the column to use for the common X axis in a table based plot.
	// if empty or not found, the row number is used.
	// This optional for Bar plots, if present and Legend is also present,
	// then an extra space will be put between X values.
	Column string

	// Rotation is the rotation of the X Axis labels, in degrees.
	Rotation float32

	// Label is the optional label to use for the XAxis instead of the default.
	Label string

	// Range is the effective range of XAxis data to plot, where either end can be fixed.
	Range minmax.Range64 `display:"inline"`

	// Scale specifies how values are scaled along the X axis:
	// Linear, Log, Inverted
	Scale AxisScales
}

// PlotStyle has overall plot level styling properties.
// Some properties provide defaults for individual elements, which can
// then be overwritten by element-level properties.
type PlotStyle struct { //types:add -setters

	// Title is the overall title of the plot.
	Title string

	// TitleStyle is the text styling parameters for the title.
	TitleStyle TextStyle

	// Background is the background of the plot.
	// The default is [colors.Scheme.Surface].
	Background image.Image

	// Scale multiplies the plot DPI value, to change the overall scale
	// of the rendered plot.  Larger numbers produce larger scaling.
	// Typically use larger numbers when generating plots for inclusion in
	// documents or other cases where the overall plot size will be small.
	Scale float32 `default:"1,2"`

	// Legend has the styling properties for the Legend.
	Legend LegendStyle

	// Axis has the styling properties for the Axes.
	Axis AxisStyle

	// XAxis has plot-level XAxis style properties.
	XAxis XAxisStyle

	// YAxisLabel is the optional label to use for the YAxis instead of the default.
	YAxisLabel string

	// LinesOn determines whether lines are plotted by default,
	// for elements that plot lines (e.g., plots.XY).
	LinesOn DefaultOffOn

	// LineWidth sets the default line width for data plotting lines.
	LineWidth units.Value

	// PointsOn determines whether points are plotted by default,
	// for elements that plot points (e.g., plots.XY).
	PointsOn DefaultOffOn

	// PointSize sets the default point size.
	PointSize units.Value

	// LabelSize sets the default label text size.
	LabelSize units.Value

	// BarWidth for Bar plot sets the default width of the bars,
	// which should be less than the Stride (1 typically) to prevent
	// bar overlap. Defaults to .8.
	BarWidth float64
}

func (ps *PlotStyle) Defaults() {
	ps.TitleStyle.Defaults()
	ps.TitleStyle.Size.Dp(24)
	ps.Background = colors.Scheme.Surface
	ps.Scale = 1
	ps.Legend.Defaults()
	ps.Axis.Defaults()
	ps.LineWidth.Pt(1)
	ps.PointSize.Pt(4)
	ps.LabelSize.Dp(16)
	ps.BarWidth = .8
}

// SetElementStyle sets the properties for given element's style
// based on the global default settings in this PlotStyle.
func (ps *PlotStyle) SetElementStyle(es *Style) {
	if ps.LinesOn != Default {
		es.Line.On = ps.LinesOn
	}
	if ps.PointsOn != Default {
		es.Point.On = ps.PointsOn
	}
	es.Line.Width = ps.LineWidth
	es.Point.Size = ps.PointSize
	es.Width.Width = ps.BarWidth
	es.Text.Size = ps.LabelSize
}

// Plot is the basic type representing a plot.
// It renders into its own image.RGBA Pixels image,
// and can also save a corresponding SVG version.
type Plot struct {
	// Title of the plot
	Title Text

	// Style has the styling properties for the plot.
	Style PlotStyle

	// standard text style with default options
	StandardTextStyle styles.Text

	// X, Y, and Z are the horizontal, vertical, and depth axes
	// of the plot respectively.
	X, Y, Z Axis

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

// New returns a new plot with some reasonable default settings.
func New() *Plot {
	pt := &Plot{}
	pt.Defaults()
	return pt
}

// Defaults sets defaults
func (pt *Plot) Defaults() {
	pt.Style.Defaults()
	pt.Title.Defaults()
	pt.Title.Style.Size.Dp(24)
	pt.X.Defaults(math32.X)
	pt.Y.Defaults(math32.Y)
	pt.Legend.Defaults()
	pt.DPI = 96
	pt.Size = image.Point{1280, 1024}
	pt.StandardTextStyle.Defaults()
	pt.StandardTextStyle.WhiteSpace = styles.WhiteSpaceNowrap
}

// applyStyle applies all the style parameters
func (pt *Plot) applyStyle() {
	// first update the global plot style settings
	var st Style
	st.Defaults()
	st.Plot = pt.Style
	for _, plt := range pt.Plotters {
		stlr := plt.Stylers()
		stlr.Run(&st)
	}
	pt.Style = st.Plot
	// then apply to elements
	for _, plt := range pt.Plotters {
		plt.ApplyStyle(&pt.Style)
	}
	// now style plot:
	pt.DPI *= pt.Style.Scale
	pt.Title.Style = pt.Style.TitleStyle
	if pt.Style.Title != "" {
		pt.Title.Text = pt.Style.Title
	}
	pt.Legend.Style = pt.Style.Legend
	pt.Legend.Style.Text.openFont(pt)
	pt.X.Style = pt.Style.Axis
	pt.X.Style.Scale = pt.Style.XAxis.Scale
	pt.Y.Style = pt.Style.Axis
	if pt.Style.XAxis.Label != "" {
		pt.X.Label.Text = pt.Style.XAxis.Label
	}
	if pt.Style.YAxisLabel != "" {
		pt.Y.Label.Text = pt.Style.YAxisLabel
	}
	pt.X.Label.Style = pt.Style.Axis.Text
	pt.Y.Label.Style = pt.Style.Axis.Text
	pt.X.TickText.Style = pt.Style.Axis.TickText
	pt.X.TickText.Style.Rotation = pt.Style.XAxis.Rotation
	pt.Y.TickText.Style = pt.Style.Axis.TickText
	pt.Y.Label.Style.Rotation = -90
	pt.Y.Style.TickText.Align = styles.End
	pt.UpdateRange()
}

// Add adds Plotter element(s) to the plot.
// When drawing the plot, Plotters are drawn in the
// order in which they were added to the plot.
func (pt *Plot) Add(ps ...Plotter) {
	pt.Plotters = append(pt.Plotters, ps...)
}

// SetPixels sets the backing pixels image to given image.RGBA.
func (pt *Plot) SetPixels(img *image.RGBA) {
	pt.Pixels = img
	pt.Paint = paint.NewContextFromImage(pt.Pixels)
	pt.Paint.UnitContext.DPI = pt.DPI
	pt.Size = pt.Pixels.Bounds().Size()
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
	pt.X.Style.TickLine.Width.Pt(0)
	pt.X.Style.TickLength.Pt(0)
	pt.X.Style.Line.Width.Pt(0)
	// pt.Y.Padding.Pt(pt.X.Style.Tick.Label.Width(names[0]) / 2)
	ticks := make([]Tick, len(names))
	for i, name := range names {
		ticks[i] = Tick{float64(i), name}
	}
	pt.X.Ticker = ConstantTicks(ticks)
}

// HideX configures the X axis so that it will not be drawn.
func (pt *Plot) HideX() {
	pt.X.Style.TickLength.Pt(0)
	pt.X.Style.Line.Width.Pt(0)
	pt.X.Ticker = ConstantTicks([]Tick{})
}

// HideY configures the Y axis so that it will not be drawn.
func (pt *Plot) HideY() {
	pt.Y.Style.TickLength.Pt(0)
	pt.Y.Style.Line.Width.Pt(0)
	pt.Y.Ticker = ConstantTicks([]Tick{})
}

// HideAxes hides the X and Y axes.
func (pt *Plot) HideAxes() {
	pt.HideX()
	pt.HideY()
}

// NominalY is like NominalX, but for the Y axis.
func (pt *Plot) NominalY(names ...string) {
	pt.Y.Style.TickLine.Width.Pt(0)
	pt.Y.Style.TickLength.Pt(0)
	pt.Y.Style.Line.Width.Pt(0)
	// pt.X.Padding = pt.Y.Tick.Label.Height(names[0]) / 2
	ticks := make([]Tick, len(names))
	for i, name := range names {
		ticks[i] = Tick{float64(i), name}
	}
	pt.Y.Ticker = ConstantTicks(ticks)
}

// UpdateRange updates the axis range values based on current Plot values.
// This first resets the range so any fixed additional range values should
// be set after this point.
func (pt *Plot) UpdateRange() {
	pt.X.Range.SetInfinity()
	pt.Y.Range.SetInfinity()
	pt.Z.Range.SetInfinity()
	if pt.Style.XAxis.Range.FixMin {
		pt.X.Range.Min = pt.Style.XAxis.Range.Min
	}
	if pt.Style.XAxis.Range.FixMax {
		pt.X.Range.Max = pt.Style.XAxis.Range.Max
	}
	for _, pl := range pt.Plotters {
		pl.UpdateRange(pt, &pt.X.Range, &pt.Y.Range, &pt.Z.Range)
	}
}

// PX returns the X-axis plotting coordinate for given raw data value
// using the current plot bounding region
func (pt *Plot) PX(v float64) float32 {
	return pt.PlotBox.ProjectX(float32(pt.X.Norm(v)))
}

// PY returns the Y-axis plotting coordinate for given raw data value
func (pt *Plot) PY(v float64) float32 {
	return pt.PlotBox.ProjectY(float32(1 - pt.Y.Norm(v)))
}

// ClosestDataToPixel returns the Plotter data point closest to given pixel point,
// in the Pixels image.
func (pt *Plot) ClosestDataToPixel(px, py int) (plt Plotter, idx int, dist float32, pixel math32.Vector2, data Data, legend string) {
	tp := math32.Vec2(float32(px), float32(py))
	dist = float32(math32.MaxFloat32)
	for _, p := range pt.Plotters {
		dts, pxX, pxY := p.Data()
		for i, ptx := range pxX {
			pty := pxY[i]
			pxy := math32.Vec2(ptx, pty)
			d := pxy.DistanceTo(tp)
			if d < dist {
				dist = d
				pixel = pxy
				plt = p
				idx = i
				data = dts
				legend = pt.Legend.LegendForPlotter(p)
			}
		}
	}
	return
}
