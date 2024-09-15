// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plotcore

import (
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/metadata"
	"cogentcore.org/core/tensor/table"
)

// todo: make a more systematic version driven by reflect fields
// key issue is the same nullable type issue: you don't want to set everything
// in md.

// also the Column name and IsString should not be in the struct!

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

// PlotColumnZeroOne returns plot options with a fixed 0-1 range
func PlotColumnZeroOne() *ColumnOptions {
	opts := &ColumnOptions{}
	opts.Range.SetMin(0)
	opts.Range.SetMax(1)
	return opts
}

// SetPlotColumnOptions sets given plotting options for named items
// within this directory, in Metadata.
func SetPlotColumnOptions(md metadata.Data, opts *ColumnOptions) {
	md.Set("PlotColumnOptions", opts)
}

// PlotColumnOptions returns plotting options if they have been set, else nil.
func PlotColumnOptions(md metadata.Data) *ColumnOptions {
	return errors.Ignore1(metadata.Get[*ColumnOptions](md, "PlotColumnOptions"))
}
