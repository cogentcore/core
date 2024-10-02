// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package matrix

import (
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/num"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/vector"
)

// offCols is a helper function to process the optional offset_cols args
func offCols(size int, offset_cols ...int) (off, cols int) {
	off = 0
	cols = size
	if len(offset_cols) >= 1 {
		off = offset_cols[0]
	}
	if len(offset_cols) == 2 {
		cols = offset_cols[1]
	}
	return
}

// Identity returns a new 2D Float64 tensor with 1s along the diagonal and
// 0s elsewhere, with the given row and column size.
//   - If one additional parameter is passed, it is the offset,
//     to set values above (positive) or below (negative) the diagonal.
//   - If a second additional parameter is passed, it is the number of columns
//     for a non-square matrix (first size parameter = number of rows).
func Identity(size int, offset_cols ...int) *tensor.Float64 {
	off, cols := offCols(size, offset_cols...)
	tsr := tensor.NewFloat64(size, cols)
	for r := range size {
		c := r + off
		if c < 0 || c >= cols {
			continue
		}
		tsr.SetFloat(1, r, c)
	}
	return tsr
}

// DiagonalN returns the number of elements in the along the diagonal
// of a 2D matrix of given row and column size.
//   - If one additional parameter is passed, it is the offset,
//     to include values above (positive) or below (negative) the diagonal.
//   - If a second additional parameter is passed, it is the number of columns
//     for a non-square matrix (first size parameter = number of rows).
func DiagonalN(size int, offset_cols ...int) int {
	off, cols := offCols(size, offset_cols...)
	rows := size
	if num.Abs(off) > 0 {
		oa := num.Abs(off)
		if off > 0 {
			if cols > rows {
				return DiagonalN(rows, 0, cols-oa)
			} else {
				return DiagonalN(rows-oa, 0, cols-oa)
			}
		} else {
			if rows > cols {
				return DiagonalN(rows-oa, 0, cols)
			} else {
				return DiagonalN(rows-oa, 0, cols-oa)
			}
		}
	}
	n := min(rows, cols)
	return n
}

// DiagonalIndices returns a list of indices for the diagonal elements of
// a 2D matrix of given row and column size.
// The result is a 2D list of indices, where the outer (row) dimension
// is the number of indices, and the inner dimension is 2 for the r, c coords.
//   - If one additional parameter is passed, it is the offset,
//     to set values above (positive) or below (negative) the diagonal.
//   - If a second additional parameter is passed, it is the number of columns
//     for a non-square matrix (first size parameter = number of rows).
func DiagonalIndices(size int, offset_cols ...int) *tensor.Int {
	off, cols := offCols(size, offset_cols...)
	dn := DiagonalN(size, off, cols)
	tsr := tensor.NewInt(dn, 2)
	idx := 0
	for r := range size {
		c := r + off
		if c < 0 || c >= cols {
			continue
		}
		tsr.SetInt(r, idx, 0)
		tsr.SetInt(c, idx, 1)
		idx++
	}
	return tsr
}

// Diagonal returns an [Indexed] view of the given tensor for the diagonal
// values, as a 1D list. An error is logged if the tensor is not 2D.
// Use the optional offset parameter to get values above (positive) or
// below (negative) the diagonal.
func Diagonal(tsr tensor.Tensor, offset ...int) *tensor.Indexed {
	if tsr.NumDims() != 2 {
		errors.Log(errors.New("matrix.TriLView requires a 2D tensor"))
		return nil
	}
	off := 0
	if len(offset) == 1 {
		off = offset[0]
	}
	return tensor.NewIndexed(tsr, DiagonalIndices(tsr.DimSize(0), off, tsr.DimSize(1)))
}

// Trace returns the sum of the [Diagonal] elements of the given
// tensor, as a tensor scalar.
// An error is logged if the tensor is not 2D.
// Use the optional offset parameter to get values above (positive) or
// below (negative) the diagonal.
func Trace(tsr tensor.Tensor, offset ...int) tensor.Values {
	return vector.Sum(Diagonal(tsr, offset...))
}

// Tri returns a new 2D Float64 tensor with 1s along the diagonal and
// below it, and 0s elsewhere (i.e., a filled lower triangle).
//   - If one additional parameter is passed, it is the offset,
//     to include values above (positive) or below (negative) the diagonal.
//   - If a second additional parameter is passed, it is the number of columns
//     for a non-square matrix (first size parameter = number of rows).
func Tri(size int, offset_cols ...int) *tensor.Float64 {
	off, cols := offCols(size, offset_cols...)
	tsr := tensor.NewFloat64(size, cols)
	for r := range size {
		for c := range cols {
			if c <= r+off {
				tsr.SetFloat(1, r, c)
			}
		}
	}
	return tsr
}

// TriUpper returns a new 2D Float64 tensor with 1s along the diagonal and
// above it, and 0s elsewhere (i.e., a filled upper triangle).
//   - If one additional parameter is passed, it is the offset,
//     to include values above (positive) or below (negative) the diagonal.
//   - If a second additional parameter is passed, it is the number of columns
//     for a non-square matrix (first size parameter = number of rows).
func TriUpper(size int, offset_cols ...int) *tensor.Float64 {
	off, cols := offCols(size, offset_cols...)
	tsr := tensor.NewFloat64(size, cols)
	for r := range size {
		for c := range cols {
			if c >= r+off {
				tsr.SetFloat(1, r, c)
			}
		}
	}
	return tsr
}

// TriUNum returns the number of elements in the upper triangular region
// of a 2D matrix of given row and column size, where the triangle includes the
// elements along the diagonal.
//   - If one additional parameter is passed, it is the offset,
//     to include values above (positive) or below (negative) the diagonal.
//   - If a second additional parameter is passed, it is the number of columns
//     for a non-square matrix (first size parameter = number of rows).
func TriUNum(size int, offset_cols ...int) int {
	off, cols := offCols(size, offset_cols...)
	rows := size
	if off > 0 {
		if cols > rows {
			return TriUNum(rows, 0, cols-off)
		} else {
			return TriUNum(rows-off, 0, cols-off)
		}
	} else if off < 0 { // invert
		return cols*rows - TriUNum(cols, -(off-1), rows)
	}
	if cols <= size {
		return cols + (cols*(cols-1))/2
	}
	return rows + (rows*(2*cols-rows-1))/2
}

// TriLNum returns the number of elements in the lower triangular region
// of a 2D matrix of given row and column size, where the triangle includes the
// elements along the diagonal.
//   - If one additional parameter is passed, it is the offset,
//     to include values above (positive) or below (negative) the diagonal.
//   - If a second additional parameter is passed, it is the number of columns
//     for a non-square matrix (first size parameter = number of rows).
func TriLNum(size int, offset_cols ...int) int {
	off, cols := offCols(size, offset_cols...)
	return TriUNum(cols, -off, size)
}

// TriLIndicies returns the list of r, c indexes for the lower triangular
// portion of a square matrix of size n, including the diagonal.
// The result is a 2D list of indices, where the outer (row) dimension
// is the number of indices, and the inner dimension is 2 for the r, c coords.
//   - If one additional parameter is passed, it is the offset,
//     to include values above (positive) or below (negative) the diagonal.
//   - If a second additional parameter is passed, it is the number of columns
//     for a non-square matrix.
func TriLIndicies(size int, offset_cols ...int) *tensor.Int {
	off, cols := offCols(size, offset_cols...)
	trin := TriLNum(size, off, cols)
	coords := tensor.NewInt(trin, 2)
	i := 0
	for r := range size {
		for c := range cols {
			if c <= r+off {
				coords.SetInt(r, i, 0)
				coords.SetInt(c, i, 1)
				i++
			}
		}
	}
	return coords
}

// TriUIndicies returns the list of r, c indexes for the upper triangular
// portion of a square matrix of size n, including the diagonal.
// If one additional parameter is passed, it is the offset,
// to include values above (positive) or below (negative) the diagonal.
// If a second additional parameter is passed, it is the number of columns
// for a non-square matrix.
// The result is a 2D list of indices, where the outer (row) dimension
// is the number of indices, and the inner dimension is 2 for the r, c coords.
func TriUIndicies(size int, offset_cols ...int) *tensor.Int {
	off, cols := offCols(size, offset_cols...)
	trin := TriUNum(size, off, cols)
	coords := tensor.NewInt(trin, 2)
	i := 0
	for r := range size {
		for c := range cols {
			if c >= r+off {
				coords.SetInt(r, i, 0)
				coords.SetInt(c, i, 1)
				i++
			}
		}
	}
	return coords
}

// TriLView returns an [Indexed] view of the given tensor for the lower triangular
// region of values, as a 1D list. An error is logged if the tensor is not 2D.
// Use the optional offset parameter to get values above (positive) or
// below (negative) the diagonal.
func TriLView(tsr tensor.Tensor, offset ...int) *tensor.Indexed {
	if tsr.NumDims() != 2 {
		errors.Log(errors.New("matrix.TriLView requires a 2D tensor"))
		return nil
	}
	off := 0
	if len(offset) == 1 {
		off = offset[0]
	}
	return tensor.NewIndexed(tsr, TriLIndicies(tsr.DimSize(0), off, tsr.DimSize(1)))
}

// TriUView returns an [Indexed] view of the given tensor for the upper triangular
// region of values, as a 1D list. An error is logged if the tensor is not 2D.
// Use the optional offset parameter to get values above (positive) or
// below (negative) the diagonal.
func TriUView(tsr tensor.Tensor, offset ...int) *tensor.Indexed {
	if tsr.NumDims() != 2 {
		errors.Log(errors.New("matrix.TriUView requires a 2D tensor"))
		return nil
	}
	off := 0
	if len(offset) == 1 {
		off = offset[0]
	}
	return tensor.NewIndexed(tsr, TriUIndicies(tsr.DimSize(0), off, tsr.DimSize(1)))
}

// TriL returns a copy of the given tensor containing the lower triangular
// region of values (including the diagonal), with the lower triangular region
// zeroed. An error is logged if the tensor is not 2D.
// Use the optional offset parameter to include values above (positive) or
// below (negative) the diagonal.
func TriL(tsr tensor.Tensor, offset ...int) tensor.Tensor {
	if tsr.NumDims() != 2 {
		errors.Log(errors.New("matrix.TriL requires a 2D tensor"))
		return nil
	}
	off := 0
	if len(offset) == 1 {
		off = offset[0]
	}
	off += 1
	tc := tensor.Clone(tsr)
	tv := TriUView(tc, off) // opposite
	tensor.SetAllFloat64(tv, 0)
	return tc
}

// TriU returns a copy of the given tensor containing the upper triangular
// region of values (including the diagonal), with the lower triangular region
// zeroed. An error is logged if the tensor is not 2D.
// Use the optional offset parameter to include values above (positive) or
// below (negative) the diagonal.
func TriU(tsr tensor.Tensor, offset ...int) tensor.Tensor {
	if tsr.NumDims() != 2 {
		errors.Log(errors.New("matrix.TriU requires a 2D tensor"))
		return nil
	}
	off := 0
	if len(offset) == 1 {
		off = offset[0]
	}
	off -= 1
	tc := tensor.Clone(tsr)
	tv := TriLView(tc, off) // opposite
	tensor.SetAllFloat64(tv, 0)
	return tc
}
