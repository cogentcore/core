// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gtigen

import (
	"fmt"
	"go/ast"
	"sort"

	"goki.dev/grease"
	"goki.dev/gti"
)

// Type represents a parsed enum type.
type Type struct {
	Name       string         // The name of the type in its package (eg: MyType)
	FullName   string         // The fully-package-path-qualified name of the type (eg: goki.dev/ki/v2.MyType)
	Type       *ast.TypeSpec  // The standard AST type value
	Doc        string         // The documentation for the type
	Directives gti.Directives // The directives for the type; guaranteed to be non-nil
	Fields     *gti.Fields
	Config     *Config // Configuration information set in the comment directive for the type; is initialized to generator config info first
}

// GetFields creates and returns a new [gti.Fields] object
// from the given [ast.FieldList].
func GetFields(list *ast.FieldList) (*gti.Fields, error) {
	res := &gti.Fields{}
	for _, field := range list.List {
		if len(field.Names) == 0 {
			return nil, fmt.Errorf("got unnamed struct field %v", field)
		}
		dirs := gti.Directives{}
		if field.Doc != nil {
			for _, c := range field.Doc.List {
				dir, err := grease.ParseDirective(c.Text)
				if err != nil {
					return nil, fmt.Errorf("error parsing comment directive from %q: %w", c.Text, err)
				}
				if dir == nil {
					continue
				}
				dirs = append(dirs, dir)
			}
		}
		fo := &gti.Field{
			Name:       field.Names[0].Name,
			Doc:        field.Doc.Text(),
			Directives: dirs,
		}
		res.Add(fo.Name, fo)
	}
	return res, nil
}

// Value represents a declared constant.
type Value struct {
	OriginalName string // The name of the constant before transformation
	Name         string // The name of the constant after transformation (i.e. camel case => snake case)
	Desc         string // The comment description of the constant
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

// SortValues sorts the values and ensures there
// are no duplicates. The input slice is known
// to be non-empty.
func SortValues(values []Value) []Value {
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
	return values[:j]
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
