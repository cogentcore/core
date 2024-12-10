// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

const (
	// OnedRow is for onedRow arguments to Projection2D functions,
	// specifies that the 1D case goes along the row.
	OnedRow = true

	// OnedColumn is for onedRow arguments to Projection2D functions,
	// specifies that the 1D case goes along the column.
	OnedColumn = false
)

// Projection2DShape returns the size of a 2D projection of the given tensor Shape,
// collapsing higher dimensions down to 2D (and 1D up to 2D).
// For the 1D case, onedRow determines if the values are row-wise or not.
// Even multiples of inner-most dimensions are placed along the row, odd in the column.
// If there are an odd number of dimensions, the first dimension is row-wise, and
// the remaining inner dimensions use the above logic from there, as if it was even.
// rowEx returns the number of "extra" (outer-dimensional) rows
// and colEx returns the number of extra cols, to add extra spacing between these dimensions.
func Projection2DShape(shp *Shape, onedRow bool) (rows, cols, rowEx, colEx int) {
	if shp.Len() == 0 {
		return 1, 1, 0, 0
	}
	nd := shp.NumDims()
	if nd == 1 {
		if onedRow {
			return shp.DimSize(0), 1, 0, 0
		}
		return 1, shp.DimSize(0), 0, 0
	}
	if nd == 2 {
		return shp.DimSize(0), shp.DimSize(1), 0, 0
	}
	rowShape, colShape, rowIdxs, colIdxs := Projection2DDimShapes(shp, onedRow)
	rows = rowShape.Len()
	cols = colShape.Len()
	nri := len(rowIdxs)
	if nri > 1 {
		rowEx = 1
		for i := range nri - 1 {
			rowEx *= shp.DimSize(rowIdxs[i])
		}
	}
	nci := len(colIdxs)
	if nci > 1 {
		colEx = 1
		for i := range nci - 1 {
			colEx *= shp.DimSize(colIdxs[i])
		}
	}
	return
}

// Projection2DDimShapes returns the shapes and dimension indexes for a 2D projection
// of given tensor Shape, collapsing higher dimensions down to 2D (and 1D up to 2D).
// For the 1D case, onedRow determines if the values are row-wise or not.
// Even multiples of inner-most dimensions are placed along the row, odd in the column.
// If there are an odd number of dimensions, the first dimension is row-wise, and
// the remaining inner dimensions use the above logic from there, as if it was even.
// This is the main organizing function for all Projection2D calls.
func Projection2DDimShapes(shp *Shape, onedRow bool) (rowShape, colShape *Shape, rowIdxs, colIdxs []int) {
	nd := shp.NumDims()
	if nd == 1 {
		if onedRow {
			return NewShape(shp.DimSize(0)), NewShape(1), []int{0}, nil
		}
		return NewShape(1), NewShape(shp.DimSize(0)), nil, []int{0}
	}
	if nd == 2 {
		return NewShape(shp.DimSize(0)), NewShape(shp.DimSize(1)), []int{0}, []int{1}
	}
	var rs, cs []int
	odd := nd%2 == 1
	sd := 0
	end := nd
	if odd {
		end = nd - 1
		sd = 1
		rs = []int{shp.DimSize(0)}
		rowIdxs = []int{0}
	}
	for d := range end {
		ad := d + sd
		if d%2 == 0 { // even goes to row
			rs = append(rs, shp.DimSize(ad))
			rowIdxs = append(rowIdxs, ad)
		} else {
			cs = append(cs, shp.DimSize(ad))
			colIdxs = append(colIdxs, ad)
		}
	}
	rowShape = NewShape(rs...)
	colShape = NewShape(cs...)
	return
}

// Projection2DIndex returns the flat 1D index for given row, col coords for a 2D projection
// of the given tensor shape, collapsing higher dimensions down to 2D (and 1D up to 2D).
// See [Projection2DShape] for full info.
func Projection2DIndex(shp *Shape, onedRow bool, row, col int) int {
	if shp.Len() == 0 {
		return 0
	}
	nd := shp.NumDims()
	if nd == 1 {
		if onedRow {
			return row
		}
		return col
	}
	if nd == 2 {
		return shp.IndexTo1D(row, col)
	}
	rowShape, colShape, rowIdxs, colIdxs := Projection2DDimShapes(shp, onedRow)
	ris := rowShape.IndexFrom1D(row)
	cis := colShape.IndexFrom1D(col)
	ixs := make([]int, nd)
	for i, ri := range rowIdxs {
		ixs[ri] = ris[i]
	}
	for i, ci := range colIdxs {
		ixs[ci] = cis[i]
	}
	return shp.IndexTo1D(ixs...)
}

// Projection2DCoords returns the corresponding full-dimensional coordinates
// that go into the given row, col coords for a 2D projection of the given tensor,
// collapsing higher dimensions down to 2D (and 1D up to 2D).
// See [Projection2DShape] for full info.
func Projection2DCoords(shp *Shape, onedRow bool, row, col int) (rowCoords, colCoords []int) {
	if shp.Len() == 0 {
		return []int{0}, []int{0}
	}
	idx := Projection2DIndex(shp, onedRow, row, col)
	dims := shp.IndexFrom1D(idx)
	nd := shp.NumDims()
	if nd == 1 {
		if onedRow {
			return dims, []int{0}
		}
		return []int{0}, dims
	}
	if nd == 2 {
		return dims[:1], dims[1:]
	}
	_, _, rowIdxs, colIdxs := Projection2DDimShapes(shp, onedRow)
	rowCoords = make([]int, len(rowIdxs))
	colCoords = make([]int, len(colIdxs))
	for i, ri := range rowIdxs {
		rowCoords[i] = dims[ri]
	}
	for i, ci := range colIdxs {
		colCoords[i] = dims[ci]
	}
	return
}

// Projection2DValue returns the float64 value at given row, col coords for a 2D projection
// of the given tensor, collapsing higher dimensions down to 2D (and 1D up to 2D).
// See [Projection2DShape] for full info.
func Projection2DValue(tsr Tensor, onedRow bool, row, col int) float64 {
	idx := Projection2DIndex(tsr.Shape(), onedRow, row, col)
	return tsr.Float1D(idx)
}

// Projection2DString returns the string value at given row, col coords for a 2D projection
// of the given tensor, collapsing higher dimensions down to 2D (and 1D up to 2D).
// See [Projection2DShape] for full info.
func Projection2DString(tsr Tensor, onedRow bool, row, col int) string {
	idx := Projection2DIndex(tsr.Shape(), onedRow, row, col)
	return tsr.String1D(idx)
}

// Projection2DSet sets a float64 value at given row, col coords for a 2D projection
// of the given tensor, collapsing higher dimensions down to 2D (and 1D up to 2D).
// See [Projection2DShape] for full info.
func Projection2DSet(tsr Tensor, onedRow bool, row, col int, val float64) {
	idx := Projection2DIndex(tsr.Shape(), onedRow, row, col)
	tsr.SetFloat1D(val, idx)
}

// Projection2DSetString sets a string value at given row, col coords for a 2D projection
// of the given tensor, collapsing higher dimensions down to 2D (and 1D up to 2D).
// See [Projection2DShape] for full info.
func Projection2DSetString(tsr Tensor, onedRow bool, row, col int, val string) {
	idx := Projection2DIndex(tsr.Shape(), onedRow, row, col)
	tsr.SetString1D(val, idx)
}
