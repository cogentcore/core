// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plotcore

import (
	"image"

	"cogentcore.org/core/base/metadata"
	"cogentcore.org/core/base/option"
	"cogentcore.org/core/math32/minmax"
	"cogentcore.org/core/plot"
	"cogentcore.org/core/plot/plots"
)

// PlotOptions are options for the overall plot.
type PlotOptions struct { //types:add

	// optional title at top of plot
	Title string

	// type of plot to generate.  For a Bar plot, items are plotted ordinally by row and the XAxis is optional
	Type PlotTypes

	// whether to plot lines
	Lines bool `default:"true"`

	// whether to plot points with symbols
	Points bool

	// width of lines
	LineWidth float32 `default:"1"`

	// size of points
	PointSize float32 `default:"3"`

	// the shape used to draw points
	PointShape plots.Shapes

	// width of bars for bar plot, as fraction of available space (1 = no gaps)
	BarWidth float32 `min:"0.01" max:"1" default:"0.8"`

	// if true, draw lines that connect points with a negative X-axis direction;
	// otherwise there is a break in the line.
	// default is false, so that repeated series of data across the X axis
	// are plotted separately.
	NegativeXDraw bool

	// Scale multiplies the plot DPI value, to change the overall scale
	// of the rendered plot.  Larger numbers produce larger scaling.
	// Typically use larger numbers when generating plots for inclusion in
	// documents or other cases where the overall plot size will be small.
	Scale float32 `default:"1,2"`

	// what column to use for the common X axis. if empty or not found,
	// the row number is used.  This optional for Bar plots, if present and
	// Legend is also present, then an extra space will be put between X values.
	XAxis string

	// optional column for adding a separate colored / styled line or bar
	// according to this value, and acts just like a separate Y variable,
	// crossed with Y variables.
	Legend string

	// position of the Legend
	LegendPosition plot.LegendPosition `display:"inline"`

	// rotation of the X Axis labels, in degrees
	XAxisRotation float32

	// optional label to use for XAxis instead of column name
	XAxisLabel string

	// optional label to use for YAxis -- if empty, first column name is used
	YAxisLabel string
}

// Defaults sets defaults if unset values are present.
func (po *PlotOptions) Defaults() {
	if po.LineWidth == 0 {
		po.LineWidth = 1
		po.Lines = true
		po.Points = false
		po.PointSize = 3
		po.BarWidth = .8
		po.LegendPosition.Defaults()
	}
	if po.Scale == 0 {
		po.Scale = 1
	}
}

// ColumnOptions are options for plotting one column of data.
type ColumnOptions struct { //types:add
	// whether to plot this column
	On bool

	// name of column being plotted
	Column string

	// whether to plot lines; uses the overall plot option if unset
	Lines option.Option[bool]

	// whether to plot points with symbols; uses the overall plot option if unset
	Points option.Option[bool]

	// the width of lines; uses the overall plot option if unset
	LineWidth option.Option[float32]

	// the size of points; uses the overall plot option if unset
	PointSize option.Option[float32]

	// the shape used to draw points; uses the overall plot option if unset
	PointShape option.Option[plots.Shapes]

	// effective range of data to plot -- either end can be fixed
	Range minmax.Range32 `display:"inline"`

	// full actual range of data -- only valid if specifically computed
	FullRange minmax.F32 `display:"inline"`

	// color to use when plotting the line / column
	Color image.Image

	// desired number of ticks
	NTicks int

	// if specified, this is an alternative label to use when plotting
	Label string

	// if column has n-dimensional tensor cells in each row, this is the index within each cell to plot -- use -1 to plot *all* indexes as separate lines
	TensorIndex int

	// specifies a column containing error bars for this column
	ErrColumn string

	// if true this is a string column -- plots as labels
	IsString bool `edit:"-"`
}

// Defaults sets defaults if unset values are present.
func (co *ColumnOptions) Defaults() {
	if co.NTicks == 0 {
		co.NTicks = 10
	}
}

// getLabel returns the effective label of the column.
func (co *ColumnOptions) getLabel() string {
	if co.Label != "" {
		return co.Label
	}
	return co.Column
}

// PlotTypes are different types of plots.
type PlotTypes int32 //enums:enum

const (
	// XY is a standard line / point plot.
	XY PlotTypes = iota

	// Bar plots vertical bars.
	Bar
)

//////// Stylers

// PlotStylers are plot styling functions.
type PlotStylers struct {
	Plot   []func(po *PlotOptions)
	Column map[string][]func(co *ColumnOptions)
}

// PlotStyler adds a plot styling function.
func (ps *PlotStylers) PlotStyler(f func(po *PlotOptions)) {
	ps.Plot = append(ps.Plot, f)
}

// ColumnStyler adds a column styling function for given column name.
func (ps *PlotStylers) ColumnStyler(col string, f func(co *ColumnOptions)) {
	if ps.Column == nil {
		ps.Column = make(map[string][]func(co *ColumnOptions))
	}
	cs := ps.Column[col]
	ps.Column[col] = append(cs, f)
}

// ApplyToPlot applies stylers to plot options.
func (ps *PlotStylers) ApplyToPlot(po *PlotOptions) {
	for _, f := range ps.Plot {
		f(po)
	}
}

// ApplyToColumn applies stylers to column of given name
func (ps *PlotStylers) ApplyToColumn(co *ColumnOptions) {
	if ps.Column == nil {
		return
	}
	fs := ps.Column[co.Column]
	for _, f := range fs {
		f(co)
	}
}

// SetPlotStylers sets the PlotStylers into given metadata.
func SetShapeNames(md *metadata.Data, ps *PlotStylers) {
	md.Set("PlotStylers", ps)
}

// GetPlotStylers gets the PlotStylers from given metadata (nil if none).
func GetPlotStylers(md *metadata.Data) *PlotStylers {
	ps, _ := metadata.Get[*PlotStylers](*md, "PlotStylers")
	return ps
}
