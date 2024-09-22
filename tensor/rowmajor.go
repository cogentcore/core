// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

// RowMajor is subtype of [Tensor] that maintains a row major memory organization
// that thereby supports efficient access via the outermost 'row' dimension,
// with all remaining inner dimensions comprising the 'cells' of data per row
// (1 scalar value in the case of a 1D tensor).
// It is implemented by raw [Values] tensors, and the [Rows] indexed view of
// raw Values tensors. Other views however do not retain the underlying
// outer to inner row major memory structure and thus do not implement this interface.
type RowMajor interface {
	Tensor

	// RowCellSize returns the size of the outermost Row shape dimension,
	// and the size of all the remaining inner dimensions (the "cell" size).
	// Commonly used to organize multiple instances (rows) of higher-dimensional
	// patterns (cells), and the [Rows] type operates on the outer row dimension.
	RowCellSize() (rows, cells int)

	// SubSpace returns a new tensor with innermost subspace at given
	// offset(s) in outermost dimension(s) (len(offs) < [NumDims]).
	// The new tensor points to the values of the this tensor (i.e., modifications
	// will affect both), as its Values slice is a view onto the original (which
	// is why only inner-most contiguous supsaces are supported).
	// Use AsValues() method to separate the two. See [Slice] function to
	// extract arbitrary subspaces along ranges of each dimension.
	SubSpace(offs ...int) Values

	// RowTensor is a convenience version of [Tensor.SubSpace] to return the
	// SubSpace for the outermost row dimension. [Rows] defines a version
	// of this that indirects through the row indexes.
	RowTensor(row int) Values

	// SetRowTensor sets the values of the [Tensor.SubSpace] at given row to given values.
	SetRowTensor(val Values, row int)

	/////////////////////  Floats

	// FloatRowCell returns the value at given row and cell, where row is the outermost
	// dimension, and cell is a 1D index into remaining inner dimensions.
	// [Rows] tensors index along the row, and use this interface extensively.
	// This is useful for lists of patterns, and the [table.Table] container.
	FloatRowCell(row, cell int) float64

	// SetFloatRowCell sets the value at given row and cell, where row is the outermost
	// dimension, and cell is a 1D index into remaining inner dimensions.
	// [Rows] tensors index along the row, and use this interface extensively.
	// This is useful for lists of patterns, and the [table.Table] container.
	SetFloatRowCell(val float64, row, cell int)

	// FloatRow returns the value at given row (outermost dimension).
	// It is a convenience wrapper for FloatRowCell(row, 0), providing robust
	// operations on 1D and higher-dimensional data (which nevertheless should
	// generally be processed separately in ways that treat it properly).
	FloatRow(row int) float64

	// SetFloatRow sets the value at given row (outermost dimension).
	// It is a convenience wrapper for SetFloatRowCell(row, 0), providing robust
	// operations on 1D and higher-dimensional data (which nevertheless should
	// generally be processed separately in ways that treat it properly).
	SetFloatRow(val float64, row int)

	/////////////////////  Ints

	// IntRowCell returns the value at given row and cell, where row is the outermost
	// dimension, and cell is a 1D index into remaining inner dimensions.
	// [Rows] tensors index along the row, and use this interface extensively.
	// This is useful for lists of patterns, and the [table.Table] container.
	IntRowCell(row, cell int) int

	// SetIntRowCell sets the value at given row and cell, where row is the outermost
	// dimension, and cell is a 1D index into remaining inner dimensions.
	// [Rows] tensors index along the row, and use this interface extensively.
	// This is useful for lists of patterns, and the [table.Table] container.
	SetIntRowCell(val int, row, cell int)

	// IntRow returns the value at given row (outermost dimension).
	// It is a convenience wrapper for IntRowCell(row, 0), providing robust
	// operations on 1D and higher-dimensional data (which nevertheless should
	// generally be processed separately in ways that treat it properly).
	IntRow(row int) int

	// SetIntRow sets the value at given row (outermost dimension).
	// It is a convenience wrapper for SetIntRowCell(row, 0), providing robust
	// operations on 1D and higher-dimensional data (which nevertheless should
	// generally be processed separately in ways that treat it properly).
	SetIntRow(val int, row int)

	/////////////////////  Strings

	// StringRowCell returns the value at given row and cell, where row is the outermost
	// dimension, and cell is a 1D index into remaining inner dimensions.
	// [Rows] tensors index along the row, and use this interface extensively.
	// This is useful for lists of patterns, and the [table.Table] container.
	StringRowCell(row, cell int) string

	// SetStringRowCell sets the value at given row and cell, where row is the outermost
	// dimension, and cell is a 1D index into remaining inner dimensions.
	// [Rows] tensors index along the row, and use this interface extensively.
	// This is useful for lists of patterns, and the [table.Table] container.
	SetStringRowCell(val string, row, cell int)

	// StringRow returns the value at given row (outermost dimension).
	// It is a convenience wrapper for StringRowCell(row, 0), providing robust
	// operations on 1D and higher-dimensional data (which nevertheless should
	// generally be processed separately in ways that treat it properly).
	StringRow(row int) string

	// SetStringRow sets the value at given row (outermost dimension).
	// It is a convenience wrapper for SetStringRowCell(row, 0), providing robust
	// operations on 1D and higher-dimensional data (which nevertheless should
	// generally be processed separately in ways that treat it properly).
	SetStringRow(val string, row int)
}
