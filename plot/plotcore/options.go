// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plotcore

import (
	"image"
	"strings"

	"cogentcore.org/core/base/option"
	"cogentcore.org/core/base/reflectx"
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
	// LegendColumn is also present, then an extra space will be put between X values.
	XAxisColumn string

	// optional column for adding a separate colored / styled line or bar
	// according to this value, and acts just like a separate Y variable,
	// crossed with Y variables.
	LegendColumn string

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
	po.fromMetaMap(dt.MetaData)
}

// metaMapLower tries meta data access by lower-case version of key too
func metaMapLower(meta map[string]string, key string) (string, bool) {
	vl, has := meta[key]
	if has {
		return vl, has
	}
	vl, has = meta[strings.ToLower(key)]
	return vl, has
}

// fromMetaMap sets plot options from meta data map.
func (po *PlotOptions) fromMetaMap(meta map[string]string) {
	if typ, has := metaMapLower(meta, "Type"); has {
		po.Type.SetString(typ)
	}
	if op, has := metaMapLower(meta, "Lines"); has {
		if op == "+" || op == "true" {
			po.Lines = true
		} else {
			po.Lines = false
		}
	}
	if op, has := metaMapLower(meta, "Points"); has {
		if op == "+" || op == "true" {
			po.Points = true
		} else {
			po.Points = false
		}
	}
	if lw, has := metaMapLower(meta, "LineWidth"); has {
		po.LineWidth, _ = reflectx.ToFloat32(lw)
	}
	if ps, has := metaMapLower(meta, "PointSize"); has {
		po.PointSize, _ = reflectx.ToFloat32(ps)
	}
	if bw, has := metaMapLower(meta, "BarWidth"); has {
		po.BarWidth, _ = reflectx.ToFloat32(bw)
	}
	if op, has := metaMapLower(meta, "NegativeXDraw"); has {
		if op == "+" || op == "true" {
			po.NegativeXDraw = true
		} else {
			po.NegativeXDraw = false
		}
	}
	if scl, has := metaMapLower(meta, "Scale"); has {
		po.Scale, _ = reflectx.ToFloat32(scl)
	}
	if xc, has := metaMapLower(meta, "XAxisColumn"); has {
		po.XAxisColumn = xc
	}
	if lc, has := metaMapLower(meta, "LegendColumn"); has {
		po.LegendColumn = lc
	}
	if xrot, has := metaMapLower(meta, "XAxisRotation"); has {
		po.XAxisRotation, _ = reflectx.ToFloat32(xrot)
	}
	if lb, has := metaMapLower(meta, "XAxisLabel"); has {
		po.XAxisLabel = lb
	}
	if lb, has := metaMapLower(meta, "YAxisLabel"); has {
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
func (co *ColumnOptions) fromMetaMap(meta map[string]string) {
	if op, has := metaMapLower(meta, co.Column+":On"); has {
		if op == "+" || op == "true" || op == "" {
			co.On = true
		} else {
			co.On = false
		}
	}
	if op, has := metaMapLower(meta, co.Column+":Off"); has {
		if op == "+" || op == "true" || op == "" {
			co.On = false
		} else {
			co.On = true
		}
	}
	if op, has := metaMapLower(meta, co.Column+":FixMin"); has {
		if op == "+" || op == "true" {
			co.Range.FixMin = true
		} else {
			co.Range.FixMin = false
		}
	}
	if op, has := metaMapLower(meta, co.Column+":FixMax"); has {
		if op == "+" || op == "true" {
			co.Range.FixMax = true
		} else {
			co.Range.FixMax = false
		}
	}
	if op, has := metaMapLower(meta, co.Column+":FloatMin"); has {
		if op == "+" || op == "true" {
			co.Range.FixMin = false
		} else {
			co.Range.FixMin = true
		}
	}
	if op, has := metaMapLower(meta, co.Column+":FloatMax"); has {
		if op == "+" || op == "true" {
			co.Range.FixMax = false
		} else {
			co.Range.FixMax = true
		}
	}
	if vl, has := metaMapLower(meta, co.Column+":Max"); has {
		co.Range.Max, _ = reflectx.ToFloat32(vl)
	}
	if vl, has := metaMapLower(meta, co.Column+":Min"); has {
		co.Range.Min, _ = reflectx.ToFloat32(vl)
	}
	if lb, has := metaMapLower(meta, co.Column+":Label"); has {
		co.Label = lb
	}
	if lb, has := metaMapLower(meta, co.Column+":ErrColumn"); has {
		co.ErrColumn = lb
	}
	if vl, has := metaMapLower(meta, co.Column+":TensorIndex"); has {
		iv, _ := reflectx.ToInt(vl)
		co.TensorIndex = int(iv)
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
