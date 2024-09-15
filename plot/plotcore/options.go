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
	"cogentcore.org/core/tensor/table"
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

// defaults sets defaults if unset values are present.
func (po *PlotOptions) defaults() {
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

// fromMeta sets plot options from meta data.
func (po *PlotOptions) fromMeta(dt *table.Table) {
	po.FromMetaMap(dt.Meta)
}

// FromMetaMap sets plot options from meta data map.
func (po *PlotOptions) FromMetaMap(meta metadata.Data) {
	if typ, err := metadata.Get[string](meta, "Type"); err == nil {
		po.Type.SetString(typ)
	}
	if op, err := metadata.Get[bool](meta, "Lines"); err == nil {
		po.Lines = op
	}
	if op, err := metadata.Get[bool](meta, "Points"); err == nil {
		po.Points = op
	}
	if lw, err := metadata.Get[float64](meta, "LineWidth"); err == nil {
		po.LineWidth = float32(lw)
	}
	if ps, err := metadata.Get[float64](meta, "PointSize"); err == nil {
		po.PointSize = float32(ps)
	}
	if bw, err := metadata.Get[float64](meta, "BarWidth"); err == nil {
		po.BarWidth = float32(bw)
	}
	if op, err := metadata.Get[bool](meta, "NegativeXDraw"); err == nil {
		po.NegativeXDraw = op
	}
	if scl, err := metadata.Get[float64](meta, "Scale"); err == nil {
		po.Scale = float32(scl)
	}
	if xc, err := metadata.Get[string](meta, "XAxis"); err == nil {
		po.XAxis = xc
	}
	if lc, err := metadata.Get[string](meta, "Legend"); err == nil {
		po.Legend = lc
	}
	if xrot, err := metadata.Get[float64](meta, "XAxisRotation"); err == nil {
		po.XAxisRotation = float32(xrot)
	}
	if lb, err := metadata.Get[string](meta, "XAxisLabel"); err == nil {
		po.XAxisLabel = lb
	}
	if lb, err := metadata.Get[string](meta, "YAxisLabel"); err == nil {
		po.YAxisLabel = lb
	}
}

// ColumnOptions are options for plotting one column of data.
type ColumnOptions struct { //types:add
	// whether to plot this column
	On bool

	// name of column being plotting
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

// defaults sets defaults if unset values are present.
func (co *ColumnOptions) defaults() {
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

// fromMetaMap sets column options from meta data map.
func (co *ColumnOptions) fromMetaMap(meta metadata.Data) {
	if op, err := metadata.Get[bool](meta, co.Column+":On"); err == nil {
		co.On = op
	}
	if op, err := metadata.Get[bool](meta, co.Column+":Off"); err == nil {
		co.On = op
	}
	if op, err := metadata.Get[bool](meta, co.Column+":FixMin"); err == nil {
		co.Range.FixMin = op
	}
	if op, err := metadata.Get[bool](meta, co.Column+":FixMax"); err == nil {
		co.Range.FixMax = op
	}
	if op, err := metadata.Get[bool](meta, co.Column+":FloatMin"); err == nil {
		co.Range.FixMin = op
	}
	if op, err := metadata.Get[bool](meta, co.Column+":FloatMax"); err == nil {
		co.Range.FixMax = op
	}
	if vl, err := metadata.Get[float64](meta, co.Column+":Max"); err == nil {
		co.Range.Max = float32(vl)
	}
	if vl, err := metadata.Get[float64](meta, co.Column+":Min"); err == nil {
		co.Range.Min = float32(vl)
	}
	if lb, err := metadata.Get[string](meta, co.Column+":Label"); err == nil {
		co.Label = lb
	}
	if lb, err := metadata.Get[string](meta, co.Column+":ErrColumn"); err == nil {
		co.ErrColumn = lb
	}
	if vl, err := metadata.Get[int](meta, co.Column+":TensorIndex"); err == nil {
		co.TensorIndex = vl
	}
}

// PlotTypes are different types of plots.
type PlotTypes int32 //enums:enum

const (
	// XY is a standard line / point plot.
	XY PlotTypes = iota

	// Bar plots vertical bars.
	Bar
)
