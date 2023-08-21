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
	"bytes"
	"fmt"
	"go/ast"
	"go/importer"
	"go/token"
	"go/types"
	"log"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/iancoleman/strcase"
)

// check type-checks the package. The package must be OK to proceed.
func (pkg *Package) check(fs *token.FileSet, astFiles []*ast.File) {
	pkg.Defs = make(map[*ast.Ident]types.Object)
	config := types.Config{Importer: importer.Default(), FakeImportC: true}
	info := &types.Info{
		Defs: pkg.Defs,
	}
	typesPkg, err := config.Check(pkg.Dir, fs, astFiles, info)
	if err != nil {
		log.Fatalf("checking package: %s", err)
	}
	pkg.TypesPkg = typesPkg
}

func (g *Generator) TransformValueNames(values []Value, transformMethod string) {
	var fn func(src string) string
	switch transformMethod {
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
	default:
		return
	}

	for i, v := range values {
		after := fn(v.name)
		// If the original one was "" or the one before the transformation
		// was "" (most commonly if linecomment defines it as empty) we
		// do not care if it's empty.
		// But if any of them was not empty before then it means that
		// the transformed emptied the value
		if v.originalName != "" && v.name != "" && after == "" {
			log.Fatalf("transformation of %q (%s) got an empty result", v.name, v.originalName)
		}
		values[i].name = after
	}
}

// trimValueNames removes a prefix from each name
func (g *Generator) trimValueNames(values []Value, prefix string) {
	for i := range values {
		values[i].name = strings.TrimPrefix(values[i].name, prefix)
	}
}

// prefixValueNames adds a prefix to each name
func (g *Generator) prefixValueNames(values []Value, prefix string) {
	for i := range values {
		values[i].name = prefix + values[i].name
	}
}

// splitIntoRuns breaks the values into runs of contiguous sequences.
// For example, given 1,2,3,5,6,7 it returns {1,2,3},{5,6,7}.
// The input slice is known to be non-empty.
func splitIntoRuns(values []Value) [][]Value {
	// We use stable sort so the lexically first name is chosen for equal elements.
	sort.Stable(byValue(values))
	// Remove duplicates. Stable sort has put the one we want to print first,
	// so use that one. The String method won't care about which named constant
	// was the argument, so the first name for the given value is the only one to keep.
	// We need to do this because identical values would cause the switch or map
	// to fail to compile.
	j := 1
	for i := 1; i < len(values); i++ {
		if values[i].value != values[i-1].value {
			values[j] = values[i]
			j++
		}
	}
	values = values[:j]
	runs := make([][]Value, 0, 10)
	for len(values) > 0 {
		// One contiguous sequence per outer loop.
		i := 1
		for i < len(values) && values[i].value == values[i-1].value+1 {
			i++
		}
		runs = append(runs, values[:i])
		values = values[i:]
	}
	return runs
}

// Value represents a declared constant.
type Value struct {
	originalName string // The name of the constant before transformation
	name         string // The name of the constant after transformation (i.e. camel case => snake case)
	// The value is stored as a bit pattern alone. The boolean tells us
	// whether to interpret it as an int64 or a uint64; the only place
	// this matters is when sorting.
	// Much of the time the str field is all we need; it is printed
	// by Value.String.
	value  uint64 // Will be converted to int64 when needed.
	signed bool   // Whether the constant is a signed type.
	str    string // The string representation given by the "go/exact" package.
}

func (v *Value) String() string {
	return v.str
}

// byValue lets us sort the constants into increasing order.
// We take care in the Less method to sort in signed or unsigned order,
// as appropriate.
type byValue []Value

func (b byValue) Len() int      { return len(b) }
func (b byValue) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b byValue) Less(i, j int) bool {
	if b[i].signed {
		return int64(b[i].value) < int64(b[j].value)
	}
	return b[i].value < b[j].value
}

// Helpers

// usize returns the number of bits of the smallest unsigned integer
// type that will hold n. Used to create the smallest possible slice of
// integers to use as indexes into the concatenated strings.
func usize(n int) int {
	switch {
	case n < 1<<8:
		return 8
	case n < 1<<16:
		return 16
	default:
		// 2^32 is enough constants for anyone.
		return 32
	}
}

// declareIndexAndNameVars declares the index slices and concatenated names
// strings representing the runs of values.
func (g *Generator) declareIndexAndNameVars(runs [][]Value, typeName string) {
	var indexes, names []string
	for i, run := range runs {
		index, n := g.createIndexAndNameDecl(run, typeName, fmt.Sprintf("_%d", i))
		indexes = append(indexes, index)
		names = append(names, n)
		_, n = g.createLowerIndexAndNameDecl(run, typeName, fmt.Sprintf("_%d", i))
		names = append(names, n)
	}
	g.Printf("const (\n")
	for _, n := range names {
		g.Printf("\t%s\n", n)
	}
	g.Printf(")\n\n")
	g.Printf("var (")
	for _, index := range indexes {
		g.Printf("\t%s\n", index)
	}
	g.Printf(")\n\n")
}

// declareIndexAndNameVar is the single-run version of declareIndexAndNameVars
func (g *Generator) declareIndexAndNameVar(run []Value, typeName string) {
	index, n := g.createIndexAndNameDecl(run, typeName, "")
	g.Printf("const %s\n", n)
	g.Printf("var %s\n", index)
	index, n = g.createLowerIndexAndNameDecl(run, typeName, "")
	g.Printf("const %s\n", n)
	//g.Printf("var %s\n", index)
}

// createIndexAndNameDecl returns the pair of declarations for the run. The caller will add "const" and "var".
func (g *Generator) createLowerIndexAndNameDecl(run []Value, typeName string, suffix string) (string, string) {
	b := new(bytes.Buffer)
	indexes := make([]int, len(run))
	for i := range run {
		b.WriteString(strings.ToLower(run[i].name))
		indexes[i] = b.Len()
	}
	nameConst := fmt.Sprintf("_%sLowerName%s = %q", typeName, suffix, b.String())
	nameLen := b.Len()
	b.Reset()
	_, _ = fmt.Fprintf(b, "_%sLowerIndex%s = [...]uint%d{0, ", typeName, suffix, usize(nameLen))
	for i, v := range indexes {
		if i > 0 {
			_, _ = fmt.Fprintf(b, ", ")
		}
		_, _ = fmt.Fprintf(b, "%d", v)
	}
	_, _ = fmt.Fprintf(b, "}")
	return b.String(), nameConst
}

// createIndexAndNameDecl returns the pair of declarations for the run. The caller will add "const" and "var".
func (g *Generator) createIndexAndNameDecl(run []Value, typeName string, suffix string) (string, string) {
	b := new(bytes.Buffer)
	indexes := make([]int, len(run))
	for i := range run {
		b.WriteString(run[i].name)
		indexes[i] = b.Len()
	}
	nameConst := fmt.Sprintf("_%sName%s = %q", typeName, suffix, b.String())
	nameLen := b.Len()
	b.Reset()
	_, _ = fmt.Fprintf(b, "_%sIndex%s = [...]uint%d{0, ", typeName, suffix, usize(nameLen))
	for i, v := range indexes {
		if i > 0 {
			_, _ = fmt.Fprintf(b, ", ")
		}
		_, _ = fmt.Fprintf(b, "%d", v)
	}
	_, _ = fmt.Fprintf(b, "}")
	return b.String(), nameConst
}

// declareNameVars declares the concatenated names string representing all the values in the runs.
func (g *Generator) declareNameVars(runs [][]Value, typeName string, suffix string) {
	g.Printf("const _%sName%s = \"", typeName, suffix)
	for _, run := range runs {
		for i := range run {
			g.Printf("%s", run[i].name)
		}
	}
	g.Printf("\"\n")
	g.Printf("const _%sLowerName%s = \"", typeName, suffix)
	for _, run := range runs {
		for i := range run {
			g.Printf("%s", strings.ToLower(run[i].name))
		}
	}
	g.Printf("\"\n")
}

// buildOneRun generates the variables and String method for a single run of contiguous values.
func (g *Generator) buildOneRun(runs [][]Value, typeName string) {
	values := runs[0]
	g.Printf("\n")
	g.declareIndexAndNameVar(values, typeName)
	// The generated code is simple enough to write as a Printf format.
	lessThanZero := ""
	if values[0].signed {
		lessThanZero = "i < 0 || "
	}
	if values[0].value == 0 { // Signed or unsigned, 0 is still 0.
		g.Printf(stringOneRun, typeName, usize(len(values)), lessThanZero)
	} else {
		g.Printf(stringOneRunWithOffset, typeName, values[0].String(), usize(len(values)), lessThanZero)
	}
}

// Arguments to format are:
//
//	[1]: type name
//	[2]: size of index element (8 for uint8 etc.)
//	[3]: less than zero check (for signed types)
const stringOneRun = `func (i %[1]s) String() string {
	if %[3]si >= %[1]s(len(_%[1]sIndex)-1) {
		return fmt.Sprintf("%[1]s(%%d)", i)
	}
	return _%[1]sName[_%[1]sIndex[i]:_%[1]sIndex[i+1]]
}
`

// Arguments to format are:
//
//	[1]: type name
//	[2]: lowest defined value for type, as a string
//	[3]: size of index element (8 for uint8 etc.)
//	[4]: less than zero check (for signed types)
const stringOneRunWithOffset = `func (i %[1]s) String() string {
	i -= %[2]s
	if %[4]si >= %[1]s(len(_%[1]sIndex)-1) {
		return fmt.Sprintf("%[1]s(%%d)", i + %[2]s)
	}
	return _%[1]sName[_%[1]sIndex[i] : _%[1]sIndex[i+1]]
}
`

// buildMultipleRuns generates the variables and String method for multiple runs of contiguous values.
// For this pattern, a single Printf format won't do.
func (g *Generator) buildMultipleRuns(runs [][]Value, typeName string) {
	g.Printf("\n")
	g.declareIndexAndNameVars(runs, typeName)
	g.Printf("func (i %s) String() string {\n", typeName)
	g.Printf("\tswitch {\n")
	for i, values := range runs {
		if len(values) == 1 {
			g.Printf("\tcase i == %s:\n", &values[0])
			g.Printf("\t\treturn _%sName_%d\n", typeName, i)
			continue
		}
		g.Printf("\tcase %s <= i && i <= %s:\n", &values[0], &values[len(values)-1])
		if values[0].value != 0 {
			g.Printf("\t\ti -= %s\n", &values[0])
		}
		g.Printf("\t\treturn _%sName_%d[_%sIndex_%d[i]:_%sIndex_%d[i+1]]\n",
			typeName, i, typeName, i, typeName, i)
	}
	g.Printf("\tdefault:\n")
	g.Printf("\t\treturn fmt.Sprintf(\"%s(%%d)\", i)\n", typeName)
	g.Printf("\t}\n")
	g.Printf("}\n")
}

// buildMap handles the case where the space is so sparse a map is a reasonable fallback.
// It's a rare situation but has simple code.
func (g *Generator) buildMap(runs [][]Value, typeName string) {
	g.Printf("\n")
	g.declareNameVars(runs, typeName, "")
	g.Printf("\nvar _%sMap = map[%s]string{\n", typeName, typeName)
	n := 0
	for _, values := range runs {
		for _, value := range values {
			g.Printf("\t%s: _%sName[%d:%d],\n", &value, typeName, n, n+len(value.name))
			n += len(value.name)
		}
	}
	g.Printf("}\n\n")
	g.Printf(stringMap, typeName)
}

// buildNoOpOrderChangeDetect try to let the compiler and the user know if the order/value of the ENUMS have changed.
func (g *Generator) buildNoOpOrderChangeDetect(runs [][]Value, typeName string) {
	g.Printf("\n")

	g.Printf(`
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	`)
	g.Printf("func _%sNoOp (){ ", typeName)
	g.Printf("\n var x [1]struct{}\n")
	for _, values := range runs {
		for _, value := range values {
			g.Printf("\t_ = x[%s-(%s)]\n", value.originalName, value.str)
		}
	}
	g.Printf("}\n\n")
}

// Argument to format is the type name.
const stringMap = `func (i %[1]s) String() string {
	if str, ok := _%[1]sMap[i]; ok {
		return str
	}
	return fmt.Sprintf("%[1]s(%%d)", i)
}
`
