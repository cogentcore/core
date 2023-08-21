// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on http://github.com/dmarkham/enumer and
// golang.org/x/tools/cmd/stringer:

// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package enumgen

// Value represents a declared constant.
type Value struct {
	OriginalName string // The name of the constant before transformation
	Name         string // The name of the constant after transformation (i.e. camel case => snake case)
	// The Value is stored as a bit pattern alone. The boolean tells us
	// whether to interpret it as an int64 or a uint64; the only place
	// this matters is when sorting.
	// Much of the time the str field is all we need; it is printed
	// by Value.String.
	Value  uint64 // Will be converted to int64 when needed.
	Signed bool   // Whether the constant is a signed type.
	Str    string // The string representation given by the "go/constant" package.
}

func (v *Value) String() string {
	return v.Str
}

// ByValue is a sorting method that sorts the constants into increasing order.
// We take care in the Less method to sort in signed or unsigned order,
// as appropriate.
type ByValue []Value

func (b ByValue) Len() int      { return len(b) }
func (b ByValue) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b ByValue) Less(i, j int) bool {
	if b[i].Signed {
		return int64(b[i].Value) < int64(b[j].Value)
	}
	return b[i].Value < b[j].Value
}
