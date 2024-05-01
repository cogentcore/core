// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plotview

import (
	"fmt"
	"log"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/math32/minmax"
	"cogentcore.org/core/plot"
	"cogentcore.org/core/plot/plots"
	"cogentcore.org/core/tensor/stats/split"
	"cogentcore.org/core/tensor/table"
)

// bar plot is on integer positions, with different Y values and / or
// legend values interleaved

// GenPlotBar generates a Bar plot, setting GPlot variable
func (pl *PlotView) GenPlotBar() {
	plt := plot.New() // note: not clear how to re-use, due to newtablexynames
	if pl.Params.BarWidth > 1 {
		pl.Params.BarWidth = .8
	}

	// process xaxis first
	xi, xview, err := pl.PlotXAxis(plt, pl.Table)
	if err != nil {
		return
	}
	xp := pl.Columns[xi]

	var lsplit *table.Splits
	nleg := 1
	if pl.Params.LegendColumn != "" {
		_, err = pl.Table.Table.ColumnIndexTry(pl.Params.LegendColumn)
		if err != nil {
			log.Println("plot.LegendColumn: " + err.Error())
		} else {
			xview.SortColumnNames([]string{pl.Params.LegendColumn, xp.Column}, table.Ascending) // make it fit!
			lsplit = split.GroupBy(xview, []string{pl.Params.LegendColumn})
			nleg = max(lsplit.Len(), 1)
		}
	}

	var firstXY *TableXY
	var strCols []*ColumnParams
	nys := 0
	for _, cp := range pl.Columns {
		if !cp.On {
			continue
		}
		if cp.IsString {
			strCols = append(strCols, cp)
			continue
		}
		if cp.TensorIndex < 0 {
			yc := pl.Table.Table.ColumnByName(cp.Column)
			_, sz := yc.RowCellSize()
			nys += sz
		} else {
			nys++
		}
	}

	if nys == 0 {
		return
	}

	stride := nys * nleg
	if stride > 1 {
		stride += 1 // extra gap
	}

	yoff := 0
	yidx := 0
	maxx := 0 // max number of x values
	for _, cp := range pl.Columns {
		if !cp.On || cp == xp {
			continue
		}
		if cp.IsString {
			continue
		}
		start := yoff
		for li := 0; li < nleg; li++ {
			lview := xview
			leg := ""
			if lsplit != nil && len(lsplit.Values) > li {
				leg = lsplit.Values[li][0]
				lview = lsplit.Splits[li]
			}
			nidx := 1
			stidx := cp.TensorIndex
			if cp.TensorIndex < 0 { // do all
				yc := pl.Table.Table.ColumnByName(cp.Column)
				_, sz := yc.RowCellSize()
				nidx = sz
				stidx = 0
			}
			for ii := 0; ii < nidx; ii++ {
				idx := stidx + ii
				xy, _ := NewTableXYName(lview, xi, xp.TensorIndex, cp.Column, idx, cp.Range)
				if xy == nil {
					continue
				}
				maxx = max(maxx, lview.Len())
				if firstXY == nil {
					firstXY = xy
				}
				lbl := cp.Label()
				clr := cp.Color
				if leg != "" {
					lbl = leg + " " + lbl
				}
				if nleg > 1 {
					cidx := yidx*nleg + li
					clr = colors.Spaced(cidx)
				}
				if nidx > 1 {
					clr = colors.Spaced(idx)
					lbl = fmt.Sprintf("%s_%02d", lbl, idx)
				}
				ec := -1
				if cp.ErrColumn != "" {
					ec = pl.Table.Table.ColumnIndex(cp.ErrColumn)
				}
				var bar *plots.BarChart
				if ec >= 0 {
					exy, _ := NewTableXY(lview, ec, 0, ec, 0, minmax.Range32{})
					bar, err = plots.NewBarChart(xy, exy)
					if err != nil {
						log.Println(err)
						continue
					}
				} else {
					bar, err = plots.NewBarChart(xy, nil)
					if err != nil {
						log.Println(err)
						continue
					}
				}
				bar.Color = clr
				bar.Stride = float32(stride)
				bar.Offset = float32(start)
				bar.Width = pl.Params.BarWidth
				plt.Add(bar)
				plt.Legend.Add(lbl, bar)
				start++
			}
		}
		yidx++
		yoff += nleg
	}
	mid := (stride - 1) / 2
	if stride > 1 {
		mid = (stride - 2) / 2
	}
	if firstXY != nil && len(strCols) > 0 {
		firstXY.Table = xview
		n := xview.Len()
		for _, cp := range strCols {
			xy, _ := NewTableXYName(xview, xi, xp.TensorIndex, cp.Column, cp.TensorIndex, firstXY.YRange)
			xy.LabelColumn = xy.YColumn
			xy.YColumn = firstXY.YColumn
			xy.YIndex = firstXY.YIndex

			xyl := plots.XYLabels{}
			xyl.XYs = make(plot.XYs, n)
			xyl.Labels = make([]string, n)

			for i := range xview.Indexes {
				y := firstXY.Value(i)
				x := float32(mid + (i%maxx)*stride)
				xyl.XYs[i] = math32.Vec2(x, y)
				xyl.Labels[i] = xy.Label(i)
			}
			lbls, _ := plots.NewLabels(xyl)
			if lbls != nil {
				plt.Add(lbls)
			}
		}
	}

	netn := pl.Table.Len() * stride
	xc := pl.Table.Table.Columns[xi]
	vals := make([]string, netn)
	for i, dx := range pl.Table.Indexes {
		pi := mid + i*stride
		if pi < netn && dx < xc.Len() {
			vals[pi] = xc.String1D(dx)
		}
	}
	plt.NominalX(vals...)

	pl.ConfigPlot(plt)
	pl.Plot = plt
	if pl.ConfigPlotFunc != nil {
		pl.ConfigPlotFunc()
	}
}
