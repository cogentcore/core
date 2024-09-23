// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"fmt"
	"reflect"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/metadata"
)

// Values is an extended [Tensor] interface for raw value tensors.
// This supports direct setting of the shape of the underlying values,
// sub-space access to inner-dimensional subspaces of values, etc.
type Values interface {
	RowMajor

	// SetShapeSizes sets the dimension sizes of the tensor, and resizes
	// backing storage appropriately, retaining all existing data that fits.
	SetShapeSizes(sizes ...int)

	// SetNumRows sets the number of rows (outermost dimension).
	// It is safe to set this to 0. For incrementally growing tensors (e.g., a log)
	// it is best to first set the anticipated full size, which allocates the
	// full amount of memory, and then set to 0 and grow incrementally.
	SetNumRows(rows int)

	// Sizeof returns the number of bytes contained in the Values of this tensor.
	// For String types, this is just the string pointers, not the string content.
	Sizeof() int64

	// Bytes returns the underlying byte representation of the tensor values.
	// This is the actual underlying data, so make a copy if it can be
	// unintentionally modified or retained more than for immediate use.
	Bytes() []byte

	// SetZeros is a convenience function initialize all values to the
	// zero value of the type (empty strings for string type).
	// New tensors always start out with zeros.
	SetZeros()

	// Clone clones this tensor, creating a duplicate copy of itself with its
	// own separate memory representation of all the values.
	Clone() Values

	// CopyFrom copies all values from other tensor into this tensor, with an
	// optimized implementation if the other tensor is of the same type, and
	// otherwise it goes through the appropriate standard type (Float, Int, String).
	CopyFrom(from Values)

	// CopyCellsFrom copies given range of values from other tensor into this tensor,
	// using flat 1D indexes: to = starting index in this Tensor to start copying into,
	// start = starting index on from Tensor to start copying from, and n = number of
	// values to copy.  Uses an optimized implementation if the other tensor is
	// of the same type, and otherwise it goes through appropriate standard type.
	CopyCellsFrom(from Values, to, start, n int)

	// AppendFrom appends all values from other tensor into this tensor, with an
	// optimized implementation if the other tensor is of the same type, and
	// otherwise it goes through the appropriate standard type (Float, Int, String).
	AppendFrom(from Values) error
}

// New returns a new n-dimensional tensor of given value type
// with the given sizes per dimension (shape).
func New[T DataTypes](sizes ...int) Values {
	var v T
	switch any(v).(type) {
	case string:
		return NewString(sizes...)
	case bool:
		return NewBool(sizes...)
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
// Supported types are in [DataTypes].
func NewOfType(typ reflect.Kind, sizes ...int) Values {
	switch typ {
	case reflect.String:
		return NewString(sizes...)
	case reflect.Bool:
		return NewBool(sizes...)
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

// metadata helpers

// SetShapeNames sets the tensor shape dimension names into given metadata.
func SetShapeNames(md *metadata.Data, names ...string) {
	md.Set("ShapeNames", names)
}

// ShapeNames gets the tensor shape dimension names from given metadata.
func ShapeNames(md *metadata.Data) []string {
	return errors.Log1(metadata.Get[[]string](*md, "ShapeNames"))
}
