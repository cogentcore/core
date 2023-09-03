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
	"fmt"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/iancoleman/strcase"
	"goki.dev/enums/enumgen/config"
)

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

// TrimValueNames removes the prefixes specified
// in [config.Config.TrimPrefix] from each name
// of the given values.
func (g *Generator) TrimValueNames(values []Value, c *config.Config) {
	for _, prefix := range strings.Split(c.TrimPrefix, ",") {
		for i := range values {
			values[i].Name = strings.TrimPrefix(values[i].Name, prefix)
		}
	}

}

// PrefixValueNames adds the prefix specified in
// [config.Config.AddPrefix] to each name of
// the given values.
func (g *Generator) PrefixValueNames(values []Value, c *config.Config) {
	for i := range values {
		values[i].Name = c.AddPrefix + values[i].Name
	}
}

// TransformValueNames transforms the names of the given values according
// to the transform method specified in [config.Config.Transform]
func (g *Generator) TransformValueNames(values []Value, c *config.Config) error {
	var fn func(src string) string
	switch c.Transform {
	case "snake":
		fn = strcase.ToSnake
	case "snake_upper", "snake-upper":
		fn = strcase.ToScreamingSnake
	case "kebab":
		fn = strcase.ToKebab
	case "kebab_upper", "kebab-upper":
		fn = strcase.ToScreamingKebab
	case "upper":
		fn = strings.ToUpper
	case "lower":
		fn = strings.ToLower
	case "title":
		fn = strings.Title
	case "title-lower":
		fn = func(s string) string {
			title := []rune(strings.Title(s))
			title[0] = unicode.ToLower(title[0])
			return string(title)
		}
	case "first":
		fn = func(s string) string {
			r, _ := utf8.DecodeRuneInString(s)
			return string(r)
		}
	case "first_upper", "first-upper":
		fn = func(s string) string {
			r, _ := utf8.DecodeRuneInString(s)
			return strings.ToUpper(string(r))
		}
	case "first_lower", "first-lower":
		fn = func(s string) string {
			r, _ := utf8.DecodeRuneInString(s)
			return strings.ToLower(string(r))
		}
	case "whitespace":
		fn = func(s string) string {
			return strcase.ToDelimited(s, ' ')
		}
	case "":
		return nil
	default:
		return fmt.Errorf("unknown transformation method: %q", c.Transform)
	}

	for i, v := range values {
		after := fn(v.Name)
		// If the original one was "" or the one before the transformation
		// was "" (most commonly if linecomment defines it as empty) we
		// do not care if it's empty.
		// But if any of them was not empty before then it means that
		// the transformed emptied the value
		if v.OriginalName != "" && v.Name != "" && after == "" {
			return fmt.Errorf("transformation of %q (%s) got an empty result", v.Name, v.OriginalName)
		}
		values[i].Name = after
	}
	return nil
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
