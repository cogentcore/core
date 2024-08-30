// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensorcore

import (
	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/stats/simat"
)

const LabelSpace = float32(8)

// SimMatGrid is a widget that displays a similarity / distance matrix
// with tensor values as a grid of colored squares, and labels for rows and columns.
type SimMatGrid struct { //types:add
	TensorGrid

	// the similarity / distance matrix
	SimMat *simat.SimMat `set:"-"`

	rowMaxSz    math32.Vector2 // maximum label size
	rowMinBlank int            // minimum number of blank rows
	rowNGps     int            // number of groups in row (non-blank after blank)
	colMaxSz    math32.Vector2 // maximum label size
	colMinBlank int            // minimum number of blank cols
	colNGps     int            // number of groups in col (non-blank after blank)
}

// Defaults sets defaults for values that are at nonsensical initial values
func (tg *SimMatGrid) Init() {
	tg.TensorGrid.Init()
	tg.Display.GridView = &tg.TensorGrid
	tg.Display.Defaults()
	tg.Display.TopZero = true

}

// SetSimMat sets the similarity matrix and triggers a display update
func (tg *SimMatGrid) SetSimMat(smat *simat.SimMat) *SimMatGrid {
	tg.SimMat = smat
	tg.Tensor = smat.Mat
	if tg.Tensor != nil {
		tg.Display.FromMeta(tg.Tensor)
	}
	tg.Update()
	return tg
}

func (tg *SimMatGrid) SizeLabel(lbs []string, col bool) (minBlank, ngps int, sz math32.Vector2) {
	mx := 0
	mxi := 0
	minBlank = len(lbs)
	if minBlank == 0 {
		return
	}
	curblk := 0
	ngps = 0
	for i, lb := range lbs {
		l := len(lb)
		if l == 0 {
			curblk++
		} else {
			if curblk > 0 {
				ngps++
			}
			if i > 0 {
				minBlank = min(minBlank, curblk)
			}
			curblk = 0
			if l > mx {
				mx = l
				mxi = i
			}
		}
	}
	minBlank = min(minBlank, curblk)
	tr := paint.Text{}
	fr := tg.Styles.FontRender()
	if col {
		tr.SetStringRot90(lbs[mxi], fr, &tg.Styles.UnitContext, &tg.Styles.Text, true, 0)
	} else {
		tr.SetString(lbs[mxi], fr, &tg.Styles.UnitContext, &tg.Styles.Text, true, 0, 0)
	}
	tsz := tg.Geom.Size.Actual.Content
	if !col {
		tr.LayoutStdLR(&tg.Styles.Text, fr, &tg.Styles.UnitContext, tsz)
	}
	return minBlank, ngps, tr.BBox.Size()
}

func (tg *SimMatGrid) SizeUp() {
	tg.rowMinBlank, tg.rowNGps, tg.rowMaxSz = tg.SizeLabel(tg.SimMat.Rows, false)
	tg.colMinBlank, tg.colNGps, tg.colMaxSz = tg.SizeLabel(tg.SimMat.Columns, true)

	tg.colMaxSz.Y += tg.rowMaxSz.Y // needs one more for some reason

	rtxtsz := tg.rowMaxSz.Y / float32(tg.rowMinBlank+1)
	ctxtsz := tg.colMaxSz.X / float32(tg.colMinBlank+1)
	txtsz := math32.Max(rtxtsz, ctxtsz)

	rows, cols, _, _ := tensor.Projection2DShape(tg.Tensor.Shape(), tg.Display.OddRow)
	rowEx := tg.rowNGps
	colEx := tg.colNGps
	frw := float32(rows) + float32(rowEx)*tg.Display.DimExtra // extra spacing
	fcl := float32(cols) + float32(colEx)*tg.Display.DimExtra // extra spacing
	max := float32(math32.Max(frw, fcl))
	gsz := tg.Display.TotPrefSize / max
	gsz = math32.Max(gsz, tg.Display.GridMinSize)
	gsz = math32.Max(gsz, txtsz)
	gsz = math32.Min(gsz, tg.Display.GridMaxSize)
	minsz := math32.Vec2(tg.rowMaxSz.X+LabelSpace+gsz*float32(cols), tg.colMaxSz.Y+LabelSpace+gsz*float32(rows))
	sz := &tg.Geom.Size
	sz.FitSizeMax(&sz.Actual.Content, minsz)
}

func (tg *SimMatGrid) Render() {
	if tg.SimMat == nil || tg.SimMat.Mat.Len() == 0 {
		return
	}
	tg.EnsureColorMap()
	tg.UpdateRange()
	pc := &tg.Scene.PaintContext

	pos := tg.Geom.Pos.Content
	sz := tg.Geom.Size.Actual.Content

	effsz := sz
	effsz.X -= tg.rowMaxSz.X + LabelSpace
	effsz.Y -= tg.colMaxSz.Y + LabelSpace

	pc.FillBox(pos, sz, tg.Styles.Background)

	tsr := tg.SimMat.Mat

	rows, cols, _, _ := tensor.Projection2DShape(tsr.Shape(), tg.Display.OddRow)
	rowEx := tg.rowNGps
	colEx := tg.colNGps
	frw := float32(rows) + float32(rowEx)*tg.Display.DimExtra // extra spacing
	fcl := float32(cols) + float32(colEx)*tg.Display.DimExtra // extra spacing
	tsz := math32.Vec2(fcl, frw)
	gsz := effsz.Div(tsz)

	// Render Rows
	epos := pos
	epos.Y += tg.colMaxSz.Y + LabelSpace
	nr := len(tg.SimMat.Rows)
	mx := min(nr, rows)
	tr := paint.Text{}
	txsty := tg.Styles.Text
	txsty.AlignV = styles.Start
	ygp := 0
	prvyblk := false
	fr := tg.Styles.FontRender()
	for y := 0; y < mx; y++ {
		lb := tg.SimMat.Rows[y]
		if len(lb) == 0 {
			prvyblk = true
			continue
		}
		if prvyblk {
			ygp++
			prvyblk = false
		}
		yex := float32(ygp) * tg.Display.DimExtra
		tr.SetString(lb, fr, &tg.Styles.UnitContext, &txsty, true, 0, 0)
		tr.LayoutStdLR(&txsty, fr, &tg.Styles.UnitContext, tg.rowMaxSz)
		cr := math32.Vec2(0, float32(y)+yex)
		pr := epos.Add(cr.Mul(gsz))
		tr.Render(pc, pr)
	}

	// Render Cols
	epos = pos
	epos.X += tg.rowMaxSz.X + LabelSpace
	nc := len(tg.SimMat.Columns)
	mx = min(nc, cols)
	xgp := 0
	prvxblk := false
	for x := 0; x < mx; x++ {
		lb := tg.SimMat.Columns[x]
		if len(lb) == 0 {
			prvxblk = true
			continue
		}
		if prvxblk {
			xgp++
			prvxblk = false
		}
		xex := float32(xgp) * tg.Display.DimExtra
		tr.SetStringRot90(lb, fr, &tg.Styles.UnitContext, &tg.Styles.Text, true, 0)
		cr := math32.Vec2(float32(x)+xex, 0)
		pr := epos.Add(cr.Mul(gsz))
		tr.Render(pc, pr)
	}

	pos.X += tg.rowMaxSz.X + LabelSpace
	pos.Y += tg.colMaxSz.Y + LabelSpace
	ssz := gsz.MulScalar(tg.Display.GridFill) // smaller size with margin
	prvyblk = false
	ygp = 0
	for y := 0; y < rows; y++ {
		ylb := tg.SimMat.Rows[y]
		if len(ylb) > 0 && prvyblk {
			ygp++
			prvyblk = false
		}
		yex := float32(ygp) * tg.Display.DimExtra
		prvxblk = false
		xgp = 0
		for x := 0; x < cols; x++ {
			xlb := tg.SimMat.Columns[x]
			if len(xlb) > 0 && prvxblk {
				xgp++
				prvxblk = false
			}
			xex := float32(xgp) * tg.Display.DimExtra
			ey := y
			if !tg.Display.TopZero {
				ey = (rows - 1) - y
			}
			val := tensor.Projection2DValue(tsr, tg.Display.OddRow, ey, x)
			cr := math32.Vec2(float32(x)+xex, float32(y)+yex)
			pr := pos.Add(cr.Mul(gsz))
			_, clr := tg.Color(val)
			pc.FillBox(pr, ssz, colors.Uniform(clr))
			if len(xlb) == 0 {
				prvxblk = true
			}
		}
		if len(ylb) == 0 {
			prvyblk = true
		}
	}
}
