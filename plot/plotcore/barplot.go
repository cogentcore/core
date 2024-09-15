// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plotcore

import (
	"fmt"
	"log"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/math32/minmax"
	"cogentcore.org/core/plot"
	"cogentcore.org/core/plot/plots"
)

// bar plot is on integer positions, with different Y values and / or
// legend values interleaved

// genPlotBar generates a Bar plot, setting GPlot variable
func (pl *PlotEditor) genPlotBar() {
	plt := plot.New() // note: not clear how to re-use, due to newtablexynames
	if pl.Options.BarWidth > 1 {
		pl.Options.BarWidth = .8
	}

	// process xaxis first
	xi, xview, err := pl.plotXAxis(plt, pl.table)
	if err != nil {
		return
	}
	xp := pl.Columns[xi]

	// var lsplit *table.Splits
	nleg := 1
	if pl.Options.Legend != "" {
		lcol := pl.table.Columns.IndexByKey(pl.Options.Legend)
		if lcol < 0 {
			log.Println("plot.Legend not found: " + pl.Options.Legend)
		} else {
			// xview.SortColumnNames([]string{pl.Options.Legend, xp.Column}, tensor.Ascending) // make it fit!
			// lsplit = split.GroupBy(xview, pl.Options.Legend)
			// nleg = max(lsplit.Len(), 1)
		}
	}

	var firstXY *tableXY
	var strCols []*ColumnOptions
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
			yc := pl.table.Column(cp.Column)
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
			// if lsplit != nil && len(lsplit.Values) > li {
			// 	leg = lsplit.Values[li][0]
			// 	lview = lsplit.Splits[li]
			// }
			nidx := 1
			stidx := cp.TensorIndex
			if cp.TensorIndex < 0 { // do all
				yc := pl.table.Column(cp.Column)
				_, sz := yc.RowCellSize()
				nidx = sz
				stidx = 0
			}
			for ii := 0; ii < nidx; ii++ {
				idx := stidx + ii
				xy, _ := newTableXYName(lview, xi, xp.TensorIndex, cp.Column, idx, cp.Range)
				if xy == nil {
					continue
				}
				maxx = max(maxx, lview.NumRows())
				if firstXY == nil {
					firstXY = xy
				}
				lbl := cp.getLabel()
				clr := cp.Color
				if leg != "" {
					lbl = leg + " " + lbl
				}
				if nleg > 1 {
					cidx := yidx*nleg + li
					clr = colors.Uniform(colors.Spaced(cidx))
				}
				if nidx > 1 {
					clr = colors.Uniform(colors.Spaced(idx))
					lbl = fmt.Sprintf("%s_%02d", lbl, idx)
				}
				ec := -1
				if cp.ErrColumn != "" {
					ec = pl.table.Columns.IndexByKey(cp.ErrColumn)
				}
				var bar *plots.BarChart
				if ec >= 0 {
					exy, _ := newTableXY(lview, ec, 0, ec, 0, minmax.Range32{})
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
				bar.Width = pl.Options.BarWidth
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
		firstXY.table = xview
		n := xview.NumRows()
		for _, cp := range strCols {
			xy, _ := newTableXY(xview, xi, xp.TensorIndex, firstXY.yColumn, cp.TensorIndex, firstXY.yRange)
			xy.labelColumn = xview.Columns.IndexByKey(cp.Column)
			xy.yIndex = firstXY.yIndex

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

	netn := pl.table.NumRows() * stride
	xc := pl.table.ColumnIndex(xi)
	vals := make([]string, netn)
	for i, dx := range pl.table.Indexes {
		pi := mid + i*stride
		if pi < netn && dx < xc.Len() {
			vals[pi] = xc.String1D(dx)
		}
	}
	plt.NominalX(vals...)

	pl.configPlot(plt)
	pl.plot = plt
}
