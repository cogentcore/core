// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

const (
	// OddRow is for oddRow arguments to Projection2D functions,
	// specifies that the odd dimension goes along the row.
	OddRow = true

	// OddColumn is for oddRow arguments to Projection2D functions,
	// specifies that the odd dimension goes along the column.
	OddColumn = false
)

// Projection2DShape returns the size of a 2D projection of the given tensor Shape,
// collapsing higher dimensions down to 2D (and 1D up to 2D).
// For any odd number of dimensions, the remaining outer-most dimension
// can either be multipliexed across the row or column, given the oddRow arg.
// Even multiples of inner-most dimensions are assumed to be row, then column.
// rowEx returns the number of "extra" (higher dimensional) rows
// and colEx returns the number of extra cols
func Projection2DShape(shp *Shape, oddRow bool) (rows, cols, rowEx, colEx int) {
	if shp.Len() == 0 {
		return 1, 1, 0, 0
	}
	nd := shp.NumDims()
	switch nd {
	case 1:
		if oddRow {
			return shp.DimSize(0), 1, 0, 0
		} else {
			return 1, shp.DimSize(0), 0, 0
		}
	case 2:
		return shp.DimSize(0), shp.DimSize(1), 0, 0
	case 3:
		if oddRow {
			return shp.DimSize(0) * shp.DimSize(1), shp.DimSize(2), shp.DimSize(0), 0
		} else {
			return shp.DimSize(1), shp.DimSize(0) * shp.DimSize(2), 0, shp.DimSize(0)
		}
	case 4:
		return shp.DimSize(0) * shp.DimSize(2), shp.DimSize(1) * shp.DimSize(3), shp.DimSize(0), shp.DimSize(1)
	case 5:
		if oddRow {
			return shp.DimSize(0) * shp.DimSize(1) * shp.DimSize(3), shp.DimSize(2) * shp.DimSize(4), shp.DimSize(0) * shp.DimSize(1), 0
		} else {
			return shp.DimSize(1) * shp.DimSize(3), shp.DimSize(0) * shp.DimSize(2) * shp.DimSize(4), 0, shp.DimSize(0) * shp.DimSize(1)
		}
	}
	return 1, 1, 0, 0
}

// Projection2DIndex returns the flat 1D index for given row, col coords for a 2D projection
// of the given tensor shape, collapsing higher dimensions down to 2D (and 1D up to 2D).
// For any odd number of dimensions, the remaining outer-most dimension
// can either be multipliexed across the row or column, given the oddRow arg.
// Even multiples of inner-most dimensions are assumed to be row, then column.
func Projection2DIndex(shp *Shape, oddRow bool, row, col int) int {
	nd := shp.NumDims()
	switch nd {
	case 1:
		if oddRow {
			return row
		} else {
			return col
		}
	case 2:
		return shp.Offset([]int{row, col})
	case 3:
		if oddRow {
			ny := shp.DimSize(1)
			yy := row / ny
			y := row % ny
			return shp.Offset([]int{yy, y, col})
		} else {
			nx := shp.DimSize(2)
			xx := col / nx
			x := col % nx
			return shp.Offset([]int{xx, row, x})
		}
	case 4:
		ny := shp.DimSize(2)
		yy := row / ny
		y := row % ny
		nx := shp.DimSize(3)
		xx := col / nx
		x := col % nx
		return shp.Offset([]int{yy, xx, y, x})
	case 5:
		// todo: oddRows version!
		nyy := shp.DimSize(1)
		ny := shp.DimSize(3)
		yyy := row / (nyy * ny)
		yy := row % (nyy * ny)
		y := yy % ny
		yy = yy / ny
		nx := shp.DimSize(4)
		xx := col / nx
		x := col % nx
		return shp.Offset([]int{yyy, yy, xx, y, x})
	}
	return 0
}

// Projection2DCoords returns the corresponding full-dimensional coordinates
// that go into the given row, col coords for a 2D projection of the given tensor,
// collapsing higher dimensions down to 2D (and 1D up to 2D).
func Projection2DCoords(shp *Shape, oddRow bool, row, col int) (rowCoords, colCoords []int) {
	idx := Projection2DIndex(shp, oddRow, row, col)
	dims := shp.Index(idx)
	nd := shp.NumDims()
	switch nd {
	case 1:
		if oddRow {
			return dims, []int{0}
		} else {
			return []int{0}, dims
		}
	case 2:
		return dims[:1], dims[1:]
	case 3:
		if oddRow {
			return dims[:2], dims[2:]
		} else {
			return dims[:1], dims[1:]
		}
	case 4:
		return []int{dims[0], dims[2]}, []int{dims[1], dims[3]}
	case 5:
		if oddRow {
			return []int{dims[0], dims[1], dims[3]}, []int{dims[2], dims[4]}
		} else {
			return []int{dims[1], dims[3]}, []int{dims[0], dims[2], dims[4]}
		}
	}
	return nil, nil
}

// Projection2DValue returns the float64 value at given row, col coords for a 2D projection
// of the given tensor, collapsing higher dimensions down to 2D (and 1D up to 2D).
// For any odd number of dimensions, the remaining outer-most dimension
// can either be multipliexed across the row or column, given the oddRow arg.
// Even multiples of inner-most dimensions are assumed to be row, then column.
func Projection2DValue(tsr Tensor, oddRow bool, row, col int) float64 {
	idx := Projection2DIndex(tsr.Shape(), oddRow, row, col)
	return tsr.Float1D(idx)
}

// Projection2DString returns the string value at given row, col coords for a 2D projection
// of the given tensor, collapsing higher dimensions down to 2D (and 1D up to 2D).
// For any odd number of dimensions, the remaining outer-most dimension
// can either be multipliexed across the row or column, given the oddRow arg.
// Even multiples of inner-most dimensions are assumed to be row, then column.
func Projection2DString(tsr Tensor, oddRow bool, row, col int) string {
	idx := Projection2DIndex(tsr.Shape(), oddRow, row, col)
	return tsr.String1D(idx)
}

// Projection2DSet sets a float64 value at given row, col coords for a 2D projection
// of the given tensor, collapsing higher dimensions down to 2D (and 1D up to 2D).
// For any odd number of dimensions, the remaining outer-most dimension
// can either be multipliexed across the row or column, given the oddRow arg.
// Even multiples of inner-most dimensions are assumed to be row, then column.
func Projection2DSet(tsr Tensor, oddRow bool, row, col int, val float64) {
	idx := Projection2DIndex(tsr.Shape(), oddRow, row, col)
	tsr.SetFloat1D(idx, val)
}

// Projection2DSetString sets a string value at given row, col coords for a 2D projection
// of the given tensor, collapsing higher dimensions down to 2D (and 1D up to 2D).
// For any odd number of dimensions, the remaining outer-most dimension
// can either be multipliexed across the row or column, given the oddRow arg.
// Even multiples of inner-most dimensions are assumed to be row, then column.
func Projection2DSetString(tsr Tensor, oddRow bool, row, col int, val string) {
	idx := Projection2DIndex(tsr.Shape(), oddRow, row, col)
	tsr.SetString1D(idx, val)
}
