// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on http://github.com/dmarkham/enumer and
// golang.org/x/tools/cmd/stringer:

// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package enumgen

import (
	"sort"
	"strings"
)

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

// SplitIntoRuns breaks the values into runs of contiguous sequences.
// For example, given 1,2,3,5,6,7 it returns {1,2,3},{5,6,7}.
// The input slice is known to be non-empty.
func SplitIntoRuns(values []Value) [][]Value {
	// We use stable sort so the lexically first name is chosen for equal elements.
	sort.Stable(ByValue(values))
	// Remove duplicates. Stable sort has put the one we want to print first,
	// so use that one. The String method won't care about which named constant
	// was the argument, so the first name for the given value is the only one to keep.
	// We need to do this because identical values would cause the switch or map
	// to fail to compile.
	j := 1
	for i := 1; i < len(values); i++ {
		if values[i].Value != values[i-1].Value {
			values[j] = values[i]
			j++
		}
	}
	values = values[:j]
	runs := make([][]Value, 0, 10)
	for len(values) > 0 {
		// One contiguous sequence per outer loop.
		i := 1
		for i < len(values) && values[i].Value == values[i-1].Value+1 {
			i++
		}
		runs = append(runs, values[:i])
		values = values[i:]
	}
	return runs
}

// TrimValueNames removes the prefixes specified in
// [Generator.Config.TrimPrefix] from each name of
// the given values.
func (g *Generator) TrimValueNames(values []Value) {
	for _, prefix := range strings.Split(g.Config.TrimPrefix, ",") {
		for i := range values {
			values[i].Name = strings.TrimPrefix(values[i].Name, prefix)
		}
	}

}

// PrefixValueNames adds the prefix specified in
// [Generator.Config.AddPrefix] to each name of
// the given values.
func (g *Generator) PrefixValueNames(values []Value) {
	for i := range values {
		values[i].Name = g.Config.AddPrefix + values[i].Name
	}
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
