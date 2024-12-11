// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plot

import (
	"cogentcore.org/core/base/metadata"
	"cogentcore.org/core/math32/minmax"
	"cogentcore.org/core/styles/units"
)

// Style contains the plot styling properties relevant across
// most plot types. These properties apply to individual plot elements
// while the Plot properties applies to the overall plot itself.
type Style struct { //types:add -setters

	//	Plot has overall plot-level properties, which can be set by any
	// plot element, and are updated first, before applying element-wise styles.
	Plot PlotStyle `display:"-"`

	// On specifies whether to plot this item, for table-based plots.
	On bool

	// Plotter is the type of plotter to use in plotting this data,
	// for [plot.NewTablePlot] [table.Table] driven plots.
	// Blank means use default ([plots.XY] is overall default).
	Plotter PlotterName

	// Role specifies how a particular column of data should be used,
	// for [plot.NewTablePlot] [table.Table] driven plots.
	Role Roles

	// Group specifies a group of related data items,
	// for [plot.NewTablePlot] [table.Table] driven plots,
	// where different columns of data within the same Group play different Roles.
	Group string

	// Range is the effective range of data to plot, where either end can be fixed.
	Range minmax.Range64 `display:"inline"`

	// Label provides an alternative label to use for axis, if set.
	Label string

	// NoLegend excludes this item from the legend when it otherwise would be included,
	// for [plot.NewTablePlot] [table.Table] driven plots.
	// Role = Y values are included in the Legend by default.
	NoLegend bool

	// NTicks sets the desired number of ticks for the axis, if > 0.
	NTicks int

	// Line has style properties for drawing lines.
	Line LineStyle `display:"add-fields"`

	// Point has style properties for drawing points.
	Point PointStyle `display:"add-fields"`

	// Text has style properties for rendering text.
	Text TextStyle `display:"add-fields"`

	// Width has various plot width properties.
	Width WidthStyle `display:"inline"`
}

// NewStyle returns a new Style object with defaults applied.
func NewStyle() *Style {
	st := &Style{}
	st.Defaults()
	return st
}

func (st *Style) Defaults() {
	st.Plot.Defaults()
	st.Line.Defaults()
	st.Point.Defaults()
	st.Text.Defaults()
	st.Width.Defaults()
}

// WidthStyle contains various plot width properties relevant across
// different plot types.
type WidthStyle struct { //types:add -setters
	// Cap is the width of the caps drawn at the top of error bars.
	// The default is 10dp
	Cap units.Value

	// Offset for Bar plot is the offset added to each X axis value
	// relative to the Stride computed value (X = offset + index * Stride)
	// Defaults to 0.
	Offset float64

	// Stride for Bar plot is distance between bars. Defaults to 1.
	Stride float64

	// Width for Bar plot is the width of the bars, as a fraction of the Stride,
	// to prevent bar overlap. Defaults to .8.
	Width float64 `min:"0.01" max:"1" default:"0.8"`

	// Pad for Bar plot is additional space at start / end of data range,
	// to keep bars from overflowing ends. This amount is subtracted from Offset
	// and added to (len(Values)-1)*Stride -- no other accommodation for bar
	// width is provided, so that should be built into this value as well.
	// Defaults to 1.
	Pad float64
}

func (ws *WidthStyle) Defaults() {
	ws.Cap.Dp(10)
	ws.Offset = 1
	ws.Stride = 1
	ws.Width = .8
	ws.Pad = 1
}

// Stylers is a list of styling functions that set Style properties.
// These are called in the order added.
type Stylers []func(s *Style)

// Add Adds a styling function to the list.
func (st *Stylers) Add(f func(s *Style)) {
	*st = append(*st, f)
}

// Run runs the list of styling functions on given [Style] object.
func (st *Stylers) Run(s *Style) {
	for _, f := range *st {
		f(s)
	}
}

// NewStyle returns a new Style object with styling functions applied
// on top of Style defaults.
func (st *Stylers) NewStyle(ps *PlotStyle) *Style {
	s := NewStyle()
	ps.SetElementStyle(s)
	st.Run(s)
	return s
}

// SetStylersTo sets the [Stylers] into given object's [metadata].
func SetStylersTo(obj any, st Stylers) {
	metadata.SetTo(obj, "PlotStylers", st)
}

// GetStylersFrom returns [Stylers] from given object's [metadata].
// Returns nil if none or no metadata.
func GetStylersFrom(obj any) Stylers {
	st, _ := metadata.GetFrom[Stylers](obj, "PlotStylers")
	return st
}

// SetStylerTo sets the [Styler] function into given object's [metadata],
// replacing anything that might have already been added.
func SetStylerTo(obj any, f func(s *Style)) {
	metadata.SetTo(obj, "PlotStylers", Stylers{f})
}

// AddStylerTo adds the given [Styler] function into given object's [metadata].
func AddStylerTo(obj any, f func(s *Style)) {
	st := GetStylersFrom(obj)
	st.Add(f)
	SetStylersTo(obj, st)
}

// GetStylersFromData returns [Stylers] from given role
// in given [Data]. nil if not present.
func GetStylersFromData(data Data, role Roles) Stylers {
	vr, ok := data[role]
	if !ok {
		return nil
	}
	return GetStylersFrom(vr)
}

////////

// DefaultOffOn specifies whether to use the default value for a bool option,
// or to override the default and set Off or On.
type DefaultOffOn int32 //enums:enum

const (
	// Default means use the default value.
	Default DefaultOffOn = iota

	// Off means to override the default and turn Off.
	Off

	// On means to override the default and turn On.
	On
)
