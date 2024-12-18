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

	// SubSpace returns a new tensor with innermost subspace at given
	// offset(s) in outermost dimension(s) (len(offs) < [NumDims]).
	// The new tensor points to the values of the this tensor (i.e., modifications
	// will affect both), as its Values slice is a view onto the original (which
	// is why only inner-most contiguous supsaces are supported).
	// Use AsValues() method to separate the two. See [Slice] function to
	// extract arbitrary subspaces along ranges of each dimension.
	SubSpace(offs ...int) Values

	// RowTensor is a convenience version of [RowMajor.SubSpace] to return the
	// SubSpace for the outermost row dimension. [Rows] defines a version
	// of this that indirects through the row indexes.
	RowTensor(row int) Values

	// SetRowTensor sets the values of the [RowMajor.SubSpace] at given row to given values.
	SetRowTensor(val Values, row int)

	// AppendRow adds a row and sets values to given values.
	AppendRow(val Values)

	////////  Floats

	// FloatRow returns the value at given row and cell, where row is the outermost
	// dimension, and cell is a 1D index into remaining inner dimensions (0 for scalar).
	FloatRow(row, cell int) float64

	// SetFloatRow sets the value at given row and cell, where row is the outermost
	// dimension, and cell is a 1D index into remaining inner dimensions.
	SetFloatRow(val float64, row, cell int)

	// AppendRowFloat adds a row and sets float value(s), up to number of cells.
	AppendRowFloat(val ...float64)

	////////  Ints

	// IntRow returns the value at given row and cell, where row is the outermost
	// dimension, and cell is a 1D index into remaining inner dimensions.
	IntRow(row, cell int) int

	// SetIntRow sets the value at given row and cell, where row is the outermost
	// dimension, and cell is a 1D index into remaining inner dimensions.
	SetIntRow(val int, row, cell int)

	// AppendRowInt adds a row and sets int value(s), up to number of cells.
	AppendRowInt(val ...int)

	////////  Strings

	// StringRow returns the value at given row and cell, where row is the outermost
	// dimension, and cell is a 1D index into remaining inner dimensions.
	// [Rows] tensors index along the row, and use this interface extensively.
	// This is useful for lists of patterns, and the [table.Table] container.
	StringRow(row, cell int) string

	// SetStringRow sets the value at given row and cell, where row is the outermost
	// dimension, and cell is a 1D index into remaining inner dimensions.
	// [Rows] tensors index along the row, and use this interface extensively.
	// This is useful for lists of patterns, and the [table.Table] container.
	SetStringRow(val string, row, cell int)

	// AppendRowString adds a row and sets string value(s), up to number of cells.
	AppendRowString(val ...string)
}
