// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plotview

import (
	"image/color"
	"strings"

	"cogentcore.org/core/base/option"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/math32/minmax"
	"cogentcore.org/core/plot"
	"cogentcore.org/core/plot/plots"
	"cogentcore.org/core/tensor/table"
)

// PlotParams are parameters for overall plot
type PlotParams struct { //types:add

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
	LegendPosition plot.LegendPosition `view:"inline"`

	// rotation of the X Axis labels, in degrees
	XAxisRot float32

	// optional label to use for XAxis instead of column name
	XAxisLabel string

	// optional label to use for YAxis -- if empty, first column name is used
	YAxisLabel string

	// our plot, for update method
	Plot *PlotView `copier:"-" json:"-" xml:"-" view:"-"`
}

// Defaults sets defaults if nil vals present
func (pp *PlotParams) Defaults() {
	if pp.LineWidth == 0 {
		pp.LineWidth = 1
		pp.Lines = true
		pp.Points = false
		pp.PointSize = 3
		pp.BarWidth = .8
		pp.LegendPosition.Defaults()
	}
	if pp.Scale == 0 {
		pp.Scale = 1
	}
}

// Update satisfies the core.Updater interface and will trigger display update on edits
func (pp *PlotParams) Update() {
	if pp.BarWidth > 1 {
		pp.BarWidth = .8
	}
	if pp.Plot != nil {
		pp.Plot.Update()
	}
}

// CopyFrom copies from other col params
func (pp *PlotParams) CopyFrom(fr *PlotParams) {
	pl := pp.Plot
	*pp = *fr
	pp.Plot = pl
}

// FromMeta sets plot params from meta data
func (pp *PlotParams) FromMeta(dt *table.Table) {
	pp.FromMetaMap(dt.MetaData)
}

// MetaMapLower tries meta data access by lower-case version of key too
func MetaMapLower(meta map[string]string, key string) (string, bool) {
	vl, has := meta[key]
	if has {
		return vl, has
	}
	vl, has = meta[strings.ToLower(key)]
	return vl, has
}

// FromMetaMap sets plot params from meta data map
func (pp *PlotParams) FromMetaMap(meta map[string]string) {
	if typ, has := MetaMapLower(meta, "Type"); has {
		pp.Type.SetString(typ)
	}
	if op, has := MetaMapLower(meta, "Lines"); has {
		if op == "+" || op == "true" {
			pp.Lines = true
		} else {
			pp.Lines = false
		}
	}
	if op, has := MetaMapLower(meta, "Points"); has {
		if op == "+" || op == "true" {
			pp.Points = true
		} else {
			pp.Points = false
		}
	}
	if lw, has := MetaMapLower(meta, "LineWidth"); has {
		pp.LineWidth, _ = reflectx.ToFloat32(lw)
	}
	if ps, has := MetaMapLower(meta, "PointSize"); has {
		pp.PointSize, _ = reflectx.ToFloat32(ps)
	}
	if bw, has := MetaMapLower(meta, "BarWidth"); has {
		pp.BarWidth, _ = reflectx.ToFloat32(bw)
	}
	if op, has := MetaMapLower(meta, "NegativeXDraw"); has {
		if op == "+" || op == "true" {
			pp.NegativeXDraw = true
		} else {
			pp.NegativeXDraw = false
		}
	}
	if scl, has := MetaMapLower(meta, "Scale"); has {
		pp.Scale, _ = reflectx.ToFloat32(scl)
	}
	if xc, has := MetaMapLower(meta, "XAxisColumn"); has {
		pp.XAxisColumn = xc
	}
	if lc, has := MetaMapLower(meta, "LegendColumn"); has {
		pp.LegendColumn = lc
	}
	if xrot, has := MetaMapLower(meta, "XAxisRot"); has {
		pp.XAxisRot, _ = reflectx.ToFloat32(xrot)
	}
	if lb, has := MetaMapLower(meta, "XAxisLabel"); has {
		pp.XAxisLabel = lb
	}
	if lb, has := MetaMapLower(meta, "YAxisLabel"); has {
		pp.YAxisLabel = lb
	}
}

// ColumnParams are parameters for plotting one column of data
type ColumnParams struct { //types:add

	// whether to plot this column
	On bool

	// name of column we're plotting
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
	Range minmax.Range32 `view:"inline"`

	// full actual range of data -- only valid if specifically computed
	FullRange minmax.F32 `view:"inline"`

	// color to use when plotting the line / column
	Color color.Color

	// desired number of ticks
	NTicks int

	// if non-empty, this is an alternative label to use in plotting
	Lbl string `label:"Label"`

	// if column has n-dimensional tensor cells in each row, this is the index within each cell to plot -- use -1 to plot *all* indexes as separate lines
	TensorIndex int

	// specifies a column containing error bars for this column
	ErrColumn string

	// if true this is a string column -- plots as labels
	IsString bool `edit:"-"`

	// our plot, for update method
	Plot *PlotView `copier:"-" json:"-" xml:"-" view:"-"`
}

// Defaults sets defaults if nil vals present
func (cp *ColumnParams) Defaults() {
	if cp.NTicks == 0 {
		cp.NTicks = 10
	}
}

// Update satisfies the core.Updater interface and will trigger display update on edits
func (cp *ColumnParams) Update() {
	if cp.Plot != nil {
		cp.Plot.Update()
	}
}

// CopyFrom copies from other col params
func (cp *ColumnParams) CopyFrom(fr *ColumnParams) {
	pl := cp.Plot
	*cp = *fr
	cp.Plot = pl
}

func (cp *ColumnParams) Label() string {
	if cp.Lbl != "" {
		return cp.Lbl
	}
	return cp.Column
}

// FromMetaMap sets plot params from meta data map
func (cp *ColumnParams) FromMetaMap(meta map[string]string) {
	if op, has := MetaMapLower(meta, cp.Column+":On"); has {
		if op == "+" || op == "true" || op == "" {
			cp.On = true
		} else {
			cp.On = false
		}
	}
	if op, has := MetaMapLower(meta, cp.Column+":Off"); has {
		if op == "+" || op == "true" || op == "" {
			cp.On = false
		} else {
			cp.On = true
		}
	}
	if op, has := MetaMapLower(meta, cp.Column+":FixMin"); has {
		if op == "+" || op == "true" {
			cp.Range.FixMin = true
		} else {
			cp.Range.FixMin = false
		}
	}
	if op, has := MetaMapLower(meta, cp.Column+":FixMax"); has {
		if op == "+" || op == "true" {
			cp.Range.FixMax = true
		} else {
			cp.Range.FixMax = false
		}
	}
	if op, has := MetaMapLower(meta, cp.Column+":FloatMin"); has {
		if op == "+" || op == "true" {
			cp.Range.FixMin = false
		} else {
			cp.Range.FixMin = true
		}
	}
	if op, has := MetaMapLower(meta, cp.Column+":FloatMax"); has {
		if op == "+" || op == "true" {
			cp.Range.FixMax = false
		} else {
			cp.Range.FixMax = true
		}
	}
	if vl, has := MetaMapLower(meta, cp.Column+":Max"); has {
		cp.Range.Max, _ = reflectx.ToFloat32(vl)
	}
	if vl, has := MetaMapLower(meta, cp.Column+":Min"); has {
		cp.Range.Min, _ = reflectx.ToFloat32(vl)
	}
	if lb, has := MetaMapLower(meta, cp.Column+":Label"); has {
		cp.Lbl = lb
	}
	if lb, has := MetaMapLower(meta, cp.Column+":ErrColumn"); has {
		cp.ErrColumn = lb
	}
	if vl, has := MetaMapLower(meta, cp.Column+":TensorIndex"); has {
		iv, _ := reflectx.ToInt(vl)
		cp.TensorIndex = int(iv)
	}
}

// PlotTypes are different types of plots
type PlotTypes int32 //enums:enum

const (
	// XY is a standard line / point plot
	XY PlotTypes = iota

	// Bar plots vertical bars
	Bar
)
