// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stats

import (
	"reflect"

	"cogentcore.org/core/tensor/table"
)

// MeanTables returns an table.Table with the mean values across all float
// columns of the input tables, which must have the same columns but not
// necessarily the same number of rows.
func MeanTables(dts []*table.Table) *table.Table {
	nt := len(dts)
	if nt == 0 {
		return nil
	}
	maxRows := 0
	var maxdt *table.Table
	for _, dt := range dts {
		if dt.Rows > maxRows {
			maxRows = dt.Rows
			maxdt = dt
		}
	}
	if maxRows == 0 {
		return nil
	}
	ot := maxdt.Clone()

	// N samples per row
	rns := make([]int, maxRows)
	for _, dt := range dts {
		dnr := dt.Rows
		mx := min(dnr, maxRows)
		for ri := 0; ri < mx; ri++ {
			rns[ri]++
		}
	}
	for ci, cl := range ot.Columns {
		if cl.DataType() != reflect.Float32 && cl.DataType() != reflect.Float64 {
			continue
		}
		_, cells := cl.RowCellSize()
		for di, dt := range dts {
			if di == 0 {
				continue
			}
			dc := dt.Columns[ci]
			dnr := dt.Rows
			mx := min(dnr, maxRows)
			for ri := 0; ri < mx; ri++ {
				si := ri * cells
				for j := 0; j < cells; j++ {
					ci := si + j
					cv := cl.Float1D(ci)
					cv += dc.Float1D(ci)
					cl.SetFloat1D(ci, cv)
				}
			}
		}
		for ri := 0; ri < maxRows; ri++ {
			si := ri * cells
			for j := 0; j < cells; j++ {
				ci := si + j
				cv := cl.Float1D(ci)
				if rns[ri] > 0 {
					cv /= float64(rns[ri])
					cl.SetFloat1D(ci, cv)
				}
			}
		}
	}
	return ot
}
