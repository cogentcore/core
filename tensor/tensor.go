// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

//go:generate core generate

import (
	"fmt"
	"reflect"

	"cogentcore.org/core/base/keylist"
	"cogentcore.org/core/base/metadata"
	"gonum.org/v1/gonum/mat"
)

// DataTypes are the primary tensor data types with specific support.
// Any numerical type can also be used.  bool is represented using an
// efficient bit slice.
type DataTypes interface {
	string | bool | float32 | float64 | int | int32 | byte
}

// todo: add a conversion function to copy data from Column-Major to a tensor:
// It is also possible to use Column-Major order, which is used in R, Julia, and MATLAB
// where the inner-most index is first and outermost last.

// Tensor is the interface for n-dimensional tensors.
// Per C / Go / Python conventions, indexes are Row-Major, ordered from
// outer to inner left-to-right, so the inner-most is right-most.
// It is implemented by the Base and Number generic types specialized
// by different concrete types: float64, float32, int, int32, byte,
// string, bits (bools).
// For float32 and float64 values, use NaN to indicate missing values.
// All of the data analysis and plot packages skip NaNs.
type Tensor interface {
	fmt.Stringer
	mat.Matrix

	// Label satisfies the core.Labeler interface for a summary description of the tensor.
	Label() string

	// Shape returns a pointer to the Shape that fully parametrizes
	// the tensor shape.
	Shape() *Shape

	// SetShape sets the sizes parameters of the tensor, and resizes
	// backing storage appropriately, retaining all existing data that fits.
	SetShape(sizes ...int)

	// SetNames sets the dimension names of the tensor shape.
	SetNames(names ...string)

	// Len returns the number of elements in the tensor,
	// which is the product of all shape dimensions.
	Len() int

	// NumDims returns the total number of dimensions.
	NumDims() int

	// DimSize returns size of given dimension.
	DimSize(dim int) int

	// RowCellSize returns the size of the outermost Row shape dimension,
	// and the size of all the remaining inner dimensions (the "cell" size).
	// Commonly used to organize multiple instances (rows) of higher-dimensional
	// patterns (cells), and the [Indexed] type operates on the outer row dimension.
	RowCellSize() (rows, cells int)

	// DataType returns the type of the data elements in the tensor.
	// Bool is returned for the Bits tensor type.
	DataType() reflect.Kind

	// Sizeof returns the number of bytes contained in the Values of this tensor.
	// For String types, this is just the string pointers, not the string content.
	Sizeof() int64

	// Bytes returns the underlying byte representation of the tensor values.
	// This is the actual underlying data, so make a copy if it can be
	// unintentionally modified or retained more than for immediate use.
	Bytes() []byte

	// IsString returns true if the data type is a String; otherwise it is numeric.
	IsString() bool

	/////////////////////  Floats

	// Float returns the value of given n-dimensional index (matching Shape) as a float64.
	Float(i ...int) float64

	// SetFloat sets the value of given n-dimensional index (matching Shape) as a float64.
	SetFloat(val float64, i ...int)

	// Float1D returns the value of given 1-dimensional index (0-Len()-1) as a float64.
	Float1D(i int) float64

	// SetFloat1D sets the value of given 1-dimensional index (0-Len()-1) as a float64.
	SetFloat1D(val float64, i int)

	// FloatRowCell returns the value at given row and cell, where row is the outermost
	// dimension, and cell is a 1D index into remaining inner dimensions.
	// [Indexed] tensors index along the row, and use this interface extensively.
	// This is useful for lists of patterns, and the [table.Table] container.
	FloatRowCell(row, cell int) float64

	// SetFloatRowCell sets the value at given row and cell, where row is the outermost
	// dimension, and cell is a 1D index into remaining inner dimensions.
	// [Indexed] tensors index along the row, and use this interface extensively.
	// This is useful for lists of patterns, and the [table.Table] container.
	SetFloatRowCell(val float64, row, cell int)

	/////////////////////  Strings

	// StringValue returns the value of given n-dimensional index (matching Shape) as a string.
	// 'String' conflicts with [fmt.Stringer], so we have to use StringValue here.
	StringValue(i ...int) string

	// SetString sets the value of given n-dimensional index (matching Shape) as a string.
	SetString(val string, i ...int)

	// String1D returns the value of given 1-dimensional index (0-Len()-1) as a string.
	String1D(i int) string

	// SetString1D sets the value of given 1-dimensional index (0-Len()-1) as a string.
	SetString1D(val string, i int)

	// StringRowCell returns the value at given row and cell, where row is the outermost
	// dimension, and cell is a 1D index into remaining inner dimensions.
	// [Indexed] tensors index along the row, and use this interface extensively.
	// This is useful for lists of patterns, and the [table.Table] container.
	StringRowCell(row, cell int) string

	// SetStringRowCell sets the value at given row and cell, where row is the outermost
	// dimension, and cell is a 1D index into remaining inner dimensions.
	// [Indexed] tensors index along the row, and use this interface extensively.
	// This is useful for lists of patterns, and the [table.Table] container.
	SetStringRowCell(val string, row, cell int)

	/////////////////////  Ints

	// Int returns the value of given n-dimensional index (matching Shape) as a int.
	Int(i ...int) int

	// SetInt sets the value of given n-dimensional index (matching Shape) as a int.
	SetInt(val int, i ...int)

	// Int1D returns the value of given 1-dimensional index (0-Len()-1) as a int.
	Int1D(i int) int

	// SetInt1D sets the value of given 1-dimensional index (0-Len()-1) as a int.
	SetInt1D(val int, i int)

	// IntRowCell returns the value at given row and cell, where row is the outermost
	// dimension, and cell is a 1D index into remaining inner dimensions.
	// [Indexed] tensors index along the row, and use this interface extensively.
	// This is useful for lists of patterns, and the [table.Table] container.
	IntRowCell(row, cell int) int

	// SetIntRowCell sets the value at given row and cell, where row is the outermost
	// dimension, and cell is a 1D index into remaining inner dimensions.
	// [Indexed] tensors index along the row, and use this interface extensively.
	// This is useful for lists of patterns, and the [table.Table] container.
	SetIntRowCell(val int, row, cell int)

	/////////////////////  SubSpaces

	// SubSpace returns a new tensor with innermost subspace at given
	// offset(s) in outermost dimension(s) (len(offs) < [NumDims]).
	// The new tensor points to the values of the this tensor (i.e., modifications
	// will affect both), as its Values slice is a view onto the original (which
	// is why only inner-most contiguous supsaces are supported).
	// Use Clone() method to separate the two. See [Slice] function to
	// extract arbitrary subspaces along ranges of each dimension.
	SubSpace(offs ...int) Tensor

	// RowTensor is a convenience version of [Tensor.SubSpace] to return the
	// SubSpace for the outermost row dimension. [Indexed] defines a version
	// of this that indirects through the row indexes.
	RowTensor(row int) Tensor

	// SetRowTensor sets the values of the [Tensor.SubSpace] at given row to given values.
	SetRowTensor(val Tensor, row int)

	// Range returns the min, max (and associated indexes, -1 = no values) for the tensor.
	// This is needed for display and is thus in the core api in optimized form.
	// See the [stats] package for many more statistics functions.
	Range() (min, max float64, minIndex, maxIndex int)

	// SetZeros is a simple convenience function initialize all values to the
	// zero value of the type (empty strings for string type).
	SetZeros()

	// Clone clones this tensor, creating a duplicate copy of itself with its
	// own separate memory representation of all the values, and returns
	// that as a Tensor (which can be converted into the known type as needed).
	Clone() Tensor

	// View clones this tensor, *keeping the same underlying Values slice*,
	// instead of making a copy like [Tensor.Clone] does. A major use of this
	// is to then change the shape of the view to provide a different way
	// of accessing the same data.  See [New1DViewOf] for example.
	View() Tensor

	// CopyFrom copies all values from other tensor into this tensor, with an
	// optimized implementation if the other tensor is of the same type, and
	// otherwise it goes through the appropriate standard type (Float, Int, String).
	CopyFrom(from Tensor)

	// AppendFrom appends all values from other tensor into this tensor, with an
	// optimized implementation if the other tensor is of the same type, and
	// otherwise it goes through the appropriate standard type (Float, Int, String).
	AppendFrom(from Tensor) error

	// SetShapeFrom sets our shape from given source tensor, calling
	// [Tensor.SetShape] with the shape params from source.
	SetShapeFrom(from Tensor)

	// CopyCellsFrom copies given range of values from other tensor into this tensor,
	// using flat 1D indexes: to = starting index in this Tensor to start copying into,
	// start = starting index on from Tensor to start copying from, and n = number of
	// values to copy.  Uses an optimized implementation if the other tensor is
	// of the same type, and otherwise it goes through appropriate standard type.
	CopyCellsFrom(from Tensor, to, start, n int)

	// SetNumRows sets the number of rows (outermost dimension).
	SetNumRows(rows int)

	// Metadata returns the metadata for this tensor, which can be used
	// to encode plotting options, etc.
	Metadata() *metadata.Data
}

// New returns a new n-dimensional tensor of given value type
// with the given sizes per dimension (shape).
func New[T DataTypes](sizes ...int) Tensor {
	var v T
	switch any(v).(type) {
	case string:
		return NewString(sizes...)
	case bool:
		return NewBits(sizes...)
	case float64:
		return NewNumber[float64](sizes...)
	case float32:
		return NewNumber[float32](sizes...)
	case int:
		return NewNumber[int](sizes...)
	case int32:
		return NewNumber[int32](sizes...)
	case byte:
		return NewNumber[byte](sizes...)
	default:
		panic("tensor.New: unexpected error: type not supported")
	}
}

// NewOfType returns a new n-dimensional tensor of given reflect.Kind type
// with the given sizes per dimension (shape).
// Supported types are string, bool (for [Bits]), float32, float64, int, int32, and byte.
func NewOfType(typ reflect.Kind, sizes ...int) Tensor {
	switch typ {
	case reflect.String:
		return NewString(sizes...)
	case reflect.Bool:
		return NewBits(sizes...)
	case reflect.Float64:
		return NewNumber[float64](sizes...)
	case reflect.Float32:
		return NewNumber[float32](sizes...)
	case reflect.Int:
		return NewNumber[int](sizes...)
	case reflect.Int32:
		return NewNumber[int32](sizes...)
	case reflect.Uint8:
		return NewNumber[byte](sizes...)
	default:
		panic(fmt.Sprintf("tensor.NewOfType: type not supported: %v", typ))
	}
}

// New1DViewOf returns a 1D view into the given tensor, using the same
// underlying values, and just changing the shape to a 1D view.
// This can be useful e.g., for stats and metric functions that report
// on the 1D list of values.
func New1DViewOf(tsr Tensor) Tensor {
	vw := tsr.View()
	vw.SetShape(tsr.Len())
	return vw
}

// CopyDense copies a gonum mat.Dense matrix into given Tensor
// using standard Float64 interface
func CopyDense(to Tensor, dm *mat.Dense) {
	nr, nc := dm.Dims()
	to.SetShape(nr, nc)
	idx := 0
	for ri := 0; ri < nr; ri++ {
		for ci := 0; ci < nc; ci++ {
			v := dm.At(ri, ci)
			to.SetFloat1D(v, idx)
			idx++
		}
	}
}

// List is an ordered list of Tensors with name lookup.
type List = keylist.List[string, Tensor]
