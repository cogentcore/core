// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensorcore

import (
	"cogentcore.org/core/base/metadata"
	"cogentcore.org/core/core"
	"cogentcore.org/core/math32/minmax"
)

// Layout are layout options for displaying tensors.
type Layout struct { //types:add --setters

	// OddRow means that even-numbered dimensions are displayed as Y*X rectangles.
	// This determines along which dimension to display any remaining
	// odd dimension: OddRow = true = organize vertically along row
	// dimension, false = organize horizontally across column dimension.
	OddRow bool

	// TopZero means that the Y=0 coordinate is displayed from the top-down;
	// otherwise the Y=0 coordinate is displayed from the bottom up,
	// which is typical for emergent network patterns.
	TopZero bool

	// Image will display the data as a bitmap image. If a 2D tensor, then it will
	// be a greyscale image. If a 3D tensor with size of either the first
	// or last dim = either 3 or 4, then it is a RGB(A) color image.
	Image bool
}

// GridStyle are options for displaying tensors
type GridStyle struct { //types:add --setters
	Layout

	// Range to plot
	Range minmax.Range64 `display:"inline"`

	// MinMax has the actual range of data, if not using fixed Range.
	MinMax minmax.F64 `display:"inline"`

	// ColorMap is the name of the color map to use in translating values to colors.
	ColorMap core.ColorMapName

	// GridFill sets proportion of grid square filled by the color block:
	// 1 = all, .5 = half, etc.
	GridFill float32 `min:"0.1" max:"1" step:"0.1" default:"0.9,1"`

	// DimExtra is the amount of extra space to add at dimension boundaries,
	// as a proportion of total grid size.
	DimExtra float32 `min:"0" max:"1" step:"0.02" default:"0.1,0.3"`

	// Size sets the minimum and maximum size for grid squares.
	Size minmax.F32 `display:"inline"`

	// TotalSize sets the total preferred display size along largest dimension.
	// Grid squares will be sized to fit within this size,
	// subject to the Size.Min / Max constraints, which have precedence.
	TotalSize float32

	// FontSize is the font size in standard point units for labels.
	FontSize float32
}

// Defaults sets defaults for values that are at nonsensical initial values
func (gs *GridStyle) Defaults() {
	gs.Range.SetMin(-1).SetMax(1)
	gs.ColorMap = "ColdHot"
	gs.GridFill = 0.9
	gs.DimExtra = 0.3
	gs.Size.Set(2, 32)
	gs.TotalSize = 100
	gs.FontSize = 24
}

// NewGridStyle returns a new GridStyle with defaults.
func NewGridStyle() *GridStyle {
	gs := &GridStyle{}
	gs.Defaults()
	return gs
}

func (gs *GridStyle) ApplyStylersFrom(obj any) {
	st := GetGridStylersFrom(obj)
	if st == nil {
		return
	}
	st.Run(gs)
}

// GridStylers is a list of styling functions that set GridStyle properties.
// These are called in the order added.
type GridStylers []func(s *GridStyle)

// Add Adds a styling function to the list.
func (st *GridStylers) Add(f func(s *GridStyle)) {
	*st = append(*st, f)
}

// Run runs the list of styling functions on given [GridStyle] object.
func (st *GridStylers) Run(s *GridStyle) {
	for _, f := range *st {
		f(s)
	}
}

// SetGridStylersTo sets the [GridStylers] into given object's [metadata].
func SetGridStylersTo(obj any, st GridStylers) {
	metadata.SetTo(obj, "GridStylers", st)
}

// GetGridStylersFrom returns [GridStylers] from given object's [metadata].
// Returns nil if none or no metadata.
func GetGridStylersFrom(obj any) GridStylers {
	st, _ := metadata.GetFrom[GridStylers](obj, "GridStylers")
	return st
}

// AddGridStylerTo adds the given [GridStyler] function into given object's [metadata].
func AddGridStylerTo(obj any, f func(s *GridStyle)) {
	st := GetGridStylersFrom(obj)
	st.Add(f)
	SetGridStylersTo(obj, st)
}
