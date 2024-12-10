// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensorcore

import (
	"image/color"
	"log"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/colormap"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/math32/minmax"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tensor"
)

// TensorGrid is a widget that displays tensor values as a grid of colored squares.
type TensorGrid struct {
	core.WidgetBase

	// Tensor is the tensor that we view.
	Tensor tensor.Tensor `set:"-"`

	// GridStyle has grid display style properties.
	GridStyle GridStyle

	// ColorMap is the colormap displayed (based on)
	ColorMap *colormap.Map
}

func (tg *TensorGrid) WidgetValue() any { return &tg.Tensor }

func (tg *TensorGrid) SetWidgetValue(value any) error {
	tg.SetTensor(value.(tensor.Tensor))
	return nil
}

func (tg *TensorGrid) Init() {
	tg.WidgetBase.Init()
	tg.GridStyle.Defaults()
	tg.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.DoubleClickable)
		ms := tg.MinSize()
		s.Min.Set(units.Dot(ms.X), units.Dot(ms.Y))
		s.Grow.Set(1, 1)
	})

	tg.OnDoubleClick(func(e events.Event) {
		tg.TensorEditor()
	})
	tg.AddContextMenu(func(m *core.Scene) {
		core.NewFuncButton(m).SetFunc(tg.TensorEditor).SetIcon(icons.Edit)
		core.NewFuncButton(m).SetFunc(tg.EditStyle).SetIcon(icons.Edit)
	})
}

// SetTensor sets the tensor.  Must call Update after this.
func (tg *TensorGrid) SetTensor(tsr tensor.Tensor) *TensorGrid {
	if _, ok := tsr.(*tensor.String); ok {
		log.Printf("TensorGrid: String tensors cannot be displayed using TensorGrid\n")
		return tg
	}
	tg.Tensor = tsr
	if tg.Tensor != nil {
		tg.GridStyle.ApplyStylersFrom(tg.Tensor)
	}
	return tg
}

// TensorEditor pulls up a TensorEditor of our tensor
func (tg *TensorGrid) TensorEditor() { //types:add
	d := core.NewBody("Tensor editor")
	tb := core.NewToolbar(d)
	te := NewTensorEditor(d).SetTensor(tg.Tensor)
	te.OnChange(func(e events.Event) {
		tg.NeedsRender()
	})
	tb.Maker(te.MakeToolbar)
	d.RunWindowDialog(tg)
}

func (tg *TensorGrid) EditStyle() { //types:add
	d := core.NewBody("Tensor grid style")
	core.NewForm(d).SetStruct(&tg.GridStyle).
		OnChange(func(e events.Event) {
			tg.NeedsRender()
		})
	d.RunWindowDialog(tg)
}

// MinSize returns minimum size based on tensor and display settings
func (tg *TensorGrid) MinSize() math32.Vector2 {
	if tg.Tensor == nil || tg.Tensor.Len() == 0 {
		return math32.Vector2{}
	}
	if tg.GridStyle.Image {
		return math32.Vec2(float32(tg.Tensor.DimSize(1)), float32(tg.Tensor.DimSize(0)))
	}
	rows, cols, rowEx, colEx := tensor.Projection2DShape(tg.Tensor.Shape(), tg.GridStyle.OddRow)
	frw := float32(rows) + float32(rowEx)*tg.GridStyle.DimExtra // extra spacing
	fcl := float32(cols) + float32(colEx)*tg.GridStyle.DimExtra // extra spacing
	mx := float32(max(frw, fcl))
	gsz := tg.GridStyle.TotalSize / mx
	gsz = tg.GridStyle.Size.ClampValue(gsz)
	gsz = max(gsz, 2)
	return math32.Vec2(gsz*float32(fcl), gsz*float32(frw))
}

// EnsureColorMap makes sure there is a valid color map that matches specified name
func (tg *TensorGrid) EnsureColorMap() {
	if tg.ColorMap != nil && tg.ColorMap.Name != string(tg.GridStyle.ColorMap) {
		tg.ColorMap = nil
	}
	if tg.ColorMap == nil {
		ok := false
		tg.ColorMap, ok = colormap.AvailableMaps[string(tg.GridStyle.ColorMap)]
		if !ok {
			tg.GridStyle.ColorMap = ""
			tg.GridStyle.Defaults()
		}
		tg.ColorMap = colormap.AvailableMaps[string(tg.GridStyle.ColorMap)]
	}
}

func (tg *TensorGrid) Color(val float64) (norm float64, clr color.Color) {
	if tg.ColorMap.Indexed {
		clr = tg.ColorMap.MapIndex(int(val))
	} else {
		norm = tg.GridStyle.Range.ClipNormValue(val)
		clr = tg.ColorMap.Map(float32(norm))
	}
	return
}

func (tg *TensorGrid) UpdateRange() {
	if !tg.GridStyle.Range.FixMin || !tg.GridStyle.Range.FixMax {
		min, max, _, _ := tensor.Range(tg.Tensor.AsValues())
		if !tg.GridStyle.Range.FixMin {
			nmin := minmax.NiceRoundNumber(min, true) // true = below #
			tg.GridStyle.Range.Min = nmin
		}
		if !tg.GridStyle.Range.FixMax {
			nmax := minmax.NiceRoundNumber(max, false) // false = above #
			tg.GridStyle.Range.Max = nmax
		}
	}
}

func (tg *TensorGrid) Render() {
	if tg.Tensor == nil || tg.Tensor.Len() == 0 {
		return
	}
	tg.EnsureColorMap()
	tg.UpdateRange()

	pc := &tg.Scene.PaintContext

	pos := tg.Geom.Pos.Content
	sz := tg.Geom.Size.Actual.Content
	// sz.SetSubScalar(tg.Disp.BotRtSpace.Dots)

	pc.FillBox(pos, sz, tg.Styles.Background)

	tsr := tg.Tensor

	if tg.GridStyle.Image {
		ysz := tsr.DimSize(0)
		xsz := tsr.DimSize(1)
		nclr := 1
		outclr := false // outer dimension is color
		if tsr.NumDims() == 3 {
			if tsr.DimSize(0) == 3 || tsr.DimSize(0) == 4 {
				outclr = true
				ysz = tsr.DimSize(1)
				xsz = tsr.DimSize(2)
				nclr = tsr.DimSize(0)
			} else {
				nclr = tsr.DimSize(2)
			}
		}
		tsz := math32.Vec2(float32(xsz), float32(ysz))
		gsz := sz.Div(tsz)
		for y := 0; y < ysz; y++ {
			for x := 0; x < xsz; x++ {
				ey := y
				if !tg.GridStyle.TopZero {
					ey = (ysz - 1) - y
				}
				switch {
				case outclr:
					var r, g, b, a float64
					a = 1
					r = tg.GridStyle.Range.ClipNormValue(tsr.Float(0, y, x))
					g = tg.GridStyle.Range.ClipNormValue(tsr.Float(1, y, x))
					b = tg.GridStyle.Range.ClipNormValue(tsr.Float(2, y, x))
					if nclr > 3 {
						a = tg.GridStyle.Range.ClipNormValue(tsr.Float(3, y, x))
					}
					cr := math32.Vec2(float32(x), float32(ey))
					pr := pos.Add(cr.Mul(gsz))
					pc.StrokeStyle.Color = colors.Uniform(colors.FromFloat64(r, g, b, a))
					pc.FillBox(pr, gsz, pc.StrokeStyle.Color)
				case nclr > 1:
					var r, g, b, a float64
					a = 1
					r = tg.GridStyle.Range.ClipNormValue(tsr.Float(y, x, 0))
					g = tg.GridStyle.Range.ClipNormValue(tsr.Float(y, x, 1))
					b = tg.GridStyle.Range.ClipNormValue(tsr.Float(y, x, 2))
					if nclr > 3 {
						a = tg.GridStyle.Range.ClipNormValue(tsr.Float(y, x, 3))
					}
					cr := math32.Vec2(float32(x), float32(ey))
					pr := pos.Add(cr.Mul(gsz))
					pc.StrokeStyle.Color = colors.Uniform(colors.FromFloat64(r, g, b, a))
					pc.FillBox(pr, gsz, pc.StrokeStyle.Color)
				default:
					val := tg.GridStyle.Range.ClipNormValue(tsr.Float(y, x))
					cr := math32.Vec2(float32(x), float32(ey))
					pr := pos.Add(cr.Mul(gsz))
					pc.StrokeStyle.Color = colors.Uniform(colors.FromFloat64(val, val, val, 1))
					pc.FillBox(pr, gsz, pc.StrokeStyle.Color)
				}
			}
		}
		return
	}
	rows, cols, rowEx, colEx := tensor.Projection2DShape(tsr.Shape(), tg.GridStyle.OddRow)
	frw := float32(rows) + float32(rowEx)*tg.GridStyle.DimExtra // extra spacing
	fcl := float32(cols) + float32(colEx)*tg.GridStyle.DimExtra // extra spacing
	rowsInner := rows
	colsInner := cols
	if rowEx > 0 {
		rowsInner = rows / rowEx
	}
	if colEx > 0 {
		colsInner = cols / colEx
	}
	tsz := math32.Vec2(fcl, frw)
	gsz := sz.Div(tsz)

	ssz := gsz.MulScalar(tg.GridStyle.GridFill) // smaller size with margin
	for y := 0; y < rows; y++ {
		yex := float32(int(y/rowsInner)) * tg.GridStyle.DimExtra
		for x := 0; x < cols; x++ {
			xex := float32(int(x/colsInner)) * tg.GridStyle.DimExtra
			ey := y
			if !tg.GridStyle.TopZero {
				ey = (rows - 1) - y
			}
			val := tensor.Projection2DValue(tsr, tg.GridStyle.OddRow, ey, x)
			cr := math32.Vec2(float32(x)+xex, float32(y)+yex)
			pr := pos.Add(cr.Mul(gsz))
			_, clr := tg.Color(val)
			pc.FillBox(pr, ssz, colors.Uniform(clr))
		}
	}
}
