// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plotcore

import (
	"fmt"
	"log/slog"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/plot"
	"cogentcore.org/core/plot/plots"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/stats/split"
	"cogentcore.org/core/tensor/table"
)

// genPlotXY generates an XY (lines, points) plot, setting Plot variable
func (pl *PlotEditor) genPlotXY() {
	plt := plot.New()

	// process xaxis first
	xi, xview, err := pl.plotXAxis(plt, pl.table)
	if err != nil {
		return
	}
	xp := pl.Columns[xi]

	var lsplit *table.Splits
	nleg := 1
	if pl.Options.Legend != "" {
		_, err = pl.table.Table.ColumnIndexTry(pl.Options.Legend)
		if err != nil {
			slog.Error("plot.Legend", "err", err.Error())
		} else {
			errors.Log(xview.SortStableColumnNames([]string{pl.Options.Legend, xp.Column}, table.Ascending))
			lsplit = split.GroupBy(xview, pl.Options.Legend)
			nleg = max(lsplit.Len(), 1)
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
			yc := pl.table.Table.ColumnByName(cp.Column)
			_, sz := yc.RowCellSize()
			nys += sz
		} else {
			nys++
		}
	}

	if nys == 0 {
		return
	}

	firstXY = nil
	yidx := 0
	for _, cp := range pl.Columns {
		if !cp.On || cp == xp {
			continue
		}
		if cp.IsString {
			continue
		}
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
				yc := pl.table.Table.ColumnByName(cp.Column)
				_, sz := yc.RowCellSize()
				nidx = sz
				stidx = 0
			}
			for ii := 0; ii < nidx; ii++ {
				idx := stidx + ii
				tix := lview.Clone()
				xy, _ := newTableXYName(tix, xi, xp.TensorIndex, cp.Column, idx, cp.Range)
				if xy == nil {
					continue
				}
				if firstXY == nil {
					firstXY = xy
				}
				var pts *plots.Scatter
				var lns *plots.Line
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
				if cp.Lines.Or(pl.Options.Lines) && cp.Points.Or(pl.Options.Points) {
					lns, pts, _ = plots.NewLinePoints(xy)
				} else if cp.Points.Or(pl.Options.Points) {
					pts, _ = plots.NewScatter(xy)
				} else {
					lns, _ = plots.NewLine(xy)
				}
				if lns != nil {
					lns.LineStyle.Width.Pt(float32(cp.LineWidth.Or(pl.Options.LineWidth)))
					lns.LineStyle.Color = clr
					lns.NegativeXDraw = pl.Options.NegativeXDraw
					plt.Add(lns)
					if pts != nil {
						plt.Legend.Add(lbl, lns, pts)
					} else {
						plt.Legend.Add(lbl, lns)
					}
				}
				if pts != nil {
					pts.LineStyle.Color = clr
					pts.LineStyle.Width.Pt(float32(cp.LineWidth.Or(pl.Options.LineWidth)))
					pts.PointSize.Pt(float32(cp.PointSize.Or(pl.Options.PointSize)))
					pts.PointShape = cp.PointShape.Or(pl.Options.PointShape)
					plt.Add(pts)
					if lns == nil {
						plt.Legend.Add(lbl, pts)
					}
				}
				if cp.ErrColumn != "" {
					ec := pl.table.Table.ColumnIndex(cp.ErrColumn)
					if ec >= 0 {
						xy.errColumn = ec
						eb, _ := plots.NewYErrorBars(xy)
						eb.LineStyle.Color = clr
						plt.Add(eb)
					}
				}
			}
		}
		yidx++
	}
	if firstXY != nil && len(strCols) > 0 {
		for _, cp := range strCols {
			xy, _ := newTableXY(xview, xi, xp.TensorIndex, firstXY.yColumn, cp.TensorIndex, firstXY.yRange)
			xy.labelColumn = xview.Table.ColumnIndex(cp.Column)
			xy.yIndex = firstXY.yIndex
			lbls, _ := plots.NewLabels(xy)
			if lbls != nil {
				plt.Add(lbls)
			}
		}
	}

	// Use string labels for X axis if X is a string
	xc := pl.table.Table.Columns[xi]
	if xc.IsString() {
		xcs := xc.(*tensor.String)
		vals := make([]string, pl.table.Len())
		for i, dx := range pl.table.Indexes {
			vals[i] = xcs.Values[dx]
		}
		plt.NominalX(vals...)
	}

	pl.configPlot(plt)
	pl.plot = plt
}
