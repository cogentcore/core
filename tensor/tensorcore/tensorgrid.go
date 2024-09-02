// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensorcore

import (
	"image/color"
	"log"
	"strconv"

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

// TensorLayout are layout options for displaying tensors
type TensorLayout struct { //types:add

	// even-numbered dimensions are displayed as Y*X rectangles.
	// This determines along which dimension to display any remaining
	// odd dimension: OddRow = true = organize vertically along row
	// dimension, false = organize horizontally across column dimension.
	OddRow bool

	// if true, then the Y=0 coordinate is displayed from the top-down;
	// otherwise the Y=0 coordinate is displayed from the bottom up,
	// which is typical for emergent network patterns.
	TopZero bool

	// display the data as a bitmap image.  if a 2D tensor, then it will
	// be a greyscale image.  if a 3D tensor with size of either the first
	// or last dim = either 3 or 4, then it is a RGB(A) color image.
	Image bool
}

// TensorDisplay are options for displaying tensors
type TensorDisplay struct { //types:add
	TensorLayout

	// range to plot
	Range minmax.Range64 `display:"inline"`

	// if not using fixed range, this is the actual range of data
	MinMax minmax.F64 `display:"inline"`

	// the name of the color map to use in translating values to colors
	ColorMap core.ColorMapName

	// what proportion of grid square should be filled by color block -- 1 = all, .5 = half, etc
	GridFill float32 `min:"0.1" max:"1" step:"0.1" default:"0.9,1"`

	// amount of extra space to add at dimension boundaries, as a proportion of total grid size
	DimExtra float32 `min:"0" max:"1" step:"0.02" default:"0.1,0.3"`

	// minimum size for grid squares -- they will never be smaller than this
	GridMinSize float32

	// maximum size for grid squares -- they will never be larger than this
	GridMaxSize float32

	// total preferred display size along largest dimension.
	// grid squares will be sized to fit within this size,
	// subject to harder GridMin / Max size constraints
	TotPrefSize float32

	// font size in standard point units for labels (e.g., SimMat)
	FontSize float32

	// our gridview, for update method
	GridView *TensorGrid `copier:"-" json:"-" xml:"-" display:"-"`
}

// Defaults sets defaults for values that are at nonsensical initial values
func (td *TensorDisplay) Defaults() {
	if td.ColorMap == "" {
		td.ColorMap = "ColdHot"
	}
	if td.Range.Max == 0 && td.Range.Min == 0 {
		td.Range.SetMin(-1)
		td.Range.SetMax(1)
	}
	if td.GridMinSize == 0 {
		td.GridMinSize = 2
	}
	if td.GridMaxSize == 0 {
		td.GridMaxSize = 16
	}
	if td.TotPrefSize == 0 {
		td.TotPrefSize = 100
	}
	if td.GridFill == 0 {
		td.GridFill = 0.9
		td.DimExtra = 0.3
	}
	if td.FontSize == 0 {
		td.FontSize = 24
	}
}

// FromMeta sets display options from Tensor meta-data
func (td *TensorDisplay) FromMeta(tsr tensor.Tensor) {
	if op, has := tsr.MetaData("top-zero"); has {
		if op == "+" || op == "true" {
			td.TopZero = true
		}
	}
	if op, has := tsr.MetaData("odd-row"); has {
		if op == "+" || op == "true" {
			td.OddRow = true
		}
	}
	if op, has := tsr.MetaData("image"); has {
		if op == "+" || op == "true" {
			td.Image = true
		}
	}
	if op, has := tsr.MetaData("min"); has {
		mv, _ := strconv.ParseFloat(op, 64)
		td.Range.Min = mv
	}
	if op, has := tsr.MetaData("max"); has {
		mv, _ := strconv.ParseFloat(op, 64)
		td.Range.Max = mv
	}
	if op, has := tsr.MetaData("fix-min"); has {
		if op == "+" || op == "true" {
			td.Range.FixMin = true
		} else {
			td.Range.FixMin = false
		}
	}
	if op, has := tsr.MetaData("fix-max"); has {
		if op == "+" || op == "true" {
			td.Range.FixMax = true
		} else {
			td.Range.FixMax = false
		}
	}
	if op, has := tsr.MetaData("colormap"); has {
		td.ColorMap = core.ColorMapName(op)
	}
	if op, has := tsr.MetaData("grid-fill"); has {
		mv, _ := strconv.ParseFloat(op, 32)
		td.GridFill = float32(mv)
	}
	if op, has := tsr.MetaData("grid-min"); has {
		mv, _ := strconv.ParseFloat(op, 32)
		td.GridMinSize = float32(mv)
	}
	if op, has := tsr.MetaData("grid-max"); has {
		mv, _ := strconv.ParseFloat(op, 32)
		td.GridMaxSize = float32(mv)
	}
	if op, has := tsr.MetaData("dim-extra"); has {
		mv, _ := strconv.ParseFloat(op, 32)
		td.DimExtra = float32(mv)
	}
	if op, has := tsr.MetaData("font-size"); has {
		mv, _ := strconv.ParseFloat(op, 32)
		td.FontSize = float32(mv)
	}
}

////////////////////////////////////////////////////////////////////////////
//  	TensorGrid

// TensorGrid is a widget that displays tensor values as a grid of colored squares.
type TensorGrid struct {
	core.WidgetBase

	// the tensor that we view
	Tensor tensor.Tensor `set:"-"`

	// display options
	Display TensorDisplay

	// the actual colormap
	ColorMap *colormap.Map
}

func (tg *TensorGrid) WidgetValue() any { return &tg.Tensor }

func (tg *TensorGrid) SetWidgetValue(value any) error {
	tg.SetTensor(value.(tensor.Tensor))
	return nil
}

func (tg *TensorGrid) Init() {
	tg.WidgetBase.Init()
	tg.Display.GridView = tg
	tg.Display.Defaults()
	tg.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.DoubleClickable)
		ms := tg.MinSize()
		s.Min.Set(units.Dot(ms.X), units.Dot(ms.Y))
		s.Grow.Set(1, 1)
	})

	tg.OnDoubleClick(func(e events.Event) {
		tg.OpenTensorEditor()
	})
	tg.AddContextMenu(func(m *core.Scene) {
		core.NewFuncButton(m).SetFunc(tg.OpenTensorEditor).SetIcon(icons.Edit)
		core.NewFuncButton(m).SetFunc(tg.EditSettings).SetIcon(icons.Edit)
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
		tg.Display.FromMeta(tg.Tensor)
	}
	return tg
}

// OpenTensorEditor pulls up a TensorEditor of our tensor
func (tg *TensorGrid) OpenTensorEditor() { //types:add
	d := core.NewBody("Tensor Editor")
	tb := core.NewToolbar(d)
	te := NewTensorEditor(d).SetTensor(tg.Tensor)
	te.OnChange(func(e events.Event) {
		tg.NeedsRender()
	})
	tb.Maker(te.MakeToolbar)
	d.RunWindowDialog(tg)
}

func (tg *TensorGrid) EditSettings() { //types:add
	d := core.NewBody("Tensor Grid Display Options")
	core.NewForm(d).SetStruct(&tg.Display).
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
	if tg.Display.Image {
		return math32.Vec2(float32(tg.Tensor.DimSize(1)), float32(tg.Tensor.DimSize(0)))
	}
	rows, cols, rowEx, colEx := tensor.Projection2DShape(tg.Tensor.Shape(), tg.Display.OddRow)
	frw := float32(rows) + float32(rowEx)*tg.Display.DimExtra // extra spacing
	fcl := float32(cols) + float32(colEx)*tg.Display.DimExtra // extra spacing
	mx := float32(max(frw, fcl))
	gsz := tg.Display.TotPrefSize / mx
	gsz = max(gsz, tg.Display.GridMinSize)
	gsz = min(gsz, tg.Display.GridMaxSize)
	gsz = max(gsz, 2)
	return math32.Vec2(gsz*float32(fcl), gsz*float32(frw))
}

// EnsureColorMap makes sure there is a valid color map that matches specified name
func (tg *TensorGrid) EnsureColorMap() {
	if tg.ColorMap != nil && tg.ColorMap.Name != string(tg.Display.ColorMap) {
		tg.ColorMap = nil
	}
	if tg.ColorMap == nil {
		ok := false
		tg.ColorMap, ok = colormap.AvailableMaps[string(tg.Display.ColorMap)]
		if !ok {
			tg.Display.ColorMap = ""
			tg.Display.Defaults()
		}
		tg.ColorMap = colormap.AvailableMaps[string(tg.Display.ColorMap)]
	}
}

func (tg *TensorGrid) Color(val float64) (norm float64, clr color.Color) {
	if tg.ColorMap.Indexed {
		clr = tg.ColorMap.MapIndex(int(val))
	} else {
		norm = tg.Display.Range.ClipNormValue(val)
		clr = tg.ColorMap.Map(float32(norm))
	}
	return
}

func (tg *TensorGrid) UpdateRange() {
	if !tg.Display.Range.FixMin || !tg.Display.Range.FixMax {
		min, max, _, _ := tg.Tensor.Range()
		if !tg.Display.Range.FixMin {
			nmin := minmax.NiceRoundNumber(min, true) // true = below #
			tg.Display.Range.Min = nmin
		}
		if !tg.Display.Range.FixMax {
			nmax := minmax.NiceRoundNumber(max, false) // false = above #
			tg.Display.Range.Max = nmax
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

	if tg.Display.Image {
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
				if !tg.Display.TopZero {
					ey = (ysz - 1) - y
				}
				switch {
				case outclr:
					var r, g, b, a float64
					a = 1
					r = tg.Display.Range.ClipNormValue(tsr.Float([]int{0, y, x}))
					g = tg.Display.Range.ClipNormValue(tsr.Float([]int{1, y, x}))
					b = tg.Display.Range.ClipNormValue(tsr.Float([]int{2, y, x}))
					if nclr > 3 {
						a = tg.Display.Range.ClipNormValue(tsr.Float([]int{3, y, x}))
					}
					cr := math32.Vec2(float32(x), float32(ey))
					pr := pos.Add(cr.Mul(gsz))
					pc.StrokeStyle.Color = colors.Uniform(colors.FromFloat64(r, g, b, a))
					pc.FillBox(pr, gsz, pc.StrokeStyle.Color)
				case nclr > 1:
					var r, g, b, a float64
					a = 1
					r = tg.Display.Range.ClipNormValue(tsr.Float([]int{y, x, 0}))
					g = tg.Display.Range.ClipNormValue(tsr.Float([]int{y, x, 1}))
					b = tg.Display.Range.ClipNormValue(tsr.Float([]int{y, x, 2}))
					if nclr > 3 {
						a = tg.Display.Range.ClipNormValue(tsr.Float([]int{y, x, 3}))
					}
					cr := math32.Vec2(float32(x), float32(ey))
					pr := pos.Add(cr.Mul(gsz))
					pc.StrokeStyle.Color = colors.Uniform(colors.FromFloat64(r, g, b, a))
					pc.FillBox(pr, gsz, pc.StrokeStyle.Color)
				default:
					val := tg.Display.Range.ClipNormValue(tsr.Float([]int{y, x}))
					cr := math32.Vec2(float32(x), float32(ey))
					pr := pos.Add(cr.Mul(gsz))
					pc.StrokeStyle.Color = colors.Uniform(colors.FromFloat64(val, val, val, 1))
					pc.FillBox(pr, gsz, pc.StrokeStyle.Color)
				}
			}
		}
		return
	}
	rows, cols, rowEx, colEx := tensor.Projection2DShape(tsr.Shape(), tg.Display.OddRow)
	frw := float32(rows) + float32(rowEx)*tg.Display.DimExtra // extra spacing
	fcl := float32(cols) + float32(colEx)*tg.Display.DimExtra // extra spacing
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

	ssz := gsz.MulScalar(tg.Display.GridFill) // smaller size with margin
	for y := 0; y < rows; y++ {
		yex := float32(int(y/rowsInner)) * tg.Display.DimExtra
		for x := 0; x < cols; x++ {
			xex := float32(int(x/colsInner)) * tg.Display.DimExtra
			ey := y
			if !tg.Display.TopZero {
				ey = (rows - 1) - y
			}
			val := tensor.Projection2DValue(tsr, tg.Display.OddRow, ey, x)
			cr := math32.Vec2(float32(x)+xex, float32(y)+yex)
			pr := pos.Add(cr.Mul(gsz))
			_, clr := tg.Color(val)
			pc.FillBox(pr, ssz, colors.Uniform(clr))
		}
	}
}
