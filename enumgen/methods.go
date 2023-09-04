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
	"strings"
)

// Usize returns the number of bits of the smallest unsigned integer
// type that will hold n. Used to create the smallest possible slice of
// integers to use as indexes into the concatenated strings.
func Usize(n int) int {
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

// DeclareIndexAndNameVars declares the index slices and concatenated names
// strings representing the runs of values.
func (g *Generator) DeclareIndexAndNameVars(runs [][]Value, typeName string) {
	var indexes, names []string
	for i, run := range runs {
		index, n := g.CreateIndexAndNameDecl(run, typeName, fmt.Sprintf("_%d", i))
		indexes = append(indexes, index)
		names = append(names, n)
		_, n = g.CreateLowerIndexAndNameDecl(run, typeName, fmt.Sprintf("_%d", i))
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

// DeclareIndexAndNameVar is the single-run version of declareIndexAndNameVars
func (g *Generator) DeclareIndexAndNameVar(run []Value, typeName string) {
	index, n := g.CreateIndexAndNameDecl(run, typeName, "")
	g.Printf("const %s\n", n)
	g.Printf("var %s\n", index)
	index, n = g.CreateLowerIndexAndNameDecl(run, typeName, "")
	g.Printf("const %s\n", n)
	// g.Printf("var %s\n", index)
}

// createIndexAndNameDecl returns the pair of declarations for the run. The caller will add "const" and "var".
func (g *Generator) CreateLowerIndexAndNameDecl(run []Value, typeName string, suffix string) (string, string) {
	b := new(bytes.Buffer)
	indexes := make([]int, len(run))
	for i := range run {
		b.WriteString(strings.ToLower(run[i].Name))
		indexes[i] = b.Len()
	}
	nameConst := fmt.Sprintf("_%sLowerName%s = %q", typeName, suffix, b.String())
	nameLen := b.Len()
	b.Reset()
	_, _ = fmt.Fprintf(b, "_%sLowerIndex%s = [...]uint%d{0, ", typeName, suffix, Usize(nameLen))
	for i, v := range indexes {
		if i > 0 {
			_, _ = fmt.Fprintf(b, ", ")
		}
		_, _ = fmt.Fprintf(b, "%d", v)
	}
	_, _ = fmt.Fprintf(b, "}")
	return b.String(), nameConst
}

// CreateIndexAndNameDecl returns the pair of declarations for the run. The caller will add "const" and "var".
func (g *Generator) CreateIndexAndNameDecl(run []Value, typeName string, suffix string) (string, string) {
	b := new(bytes.Buffer)
	indexes := make([]int, len(run))
	for i := range run {
		b.WriteString(run[i].Name)
		indexes[i] = b.Len()
	}
	nameConst := fmt.Sprintf("_%sName%s = %q", typeName, suffix, b.String())
	nameLen := b.Len()
	b.Reset()
	_, _ = fmt.Fprintf(b, "_%sIndex%s = [...]uint%d{0, ", typeName, suffix, Usize(nameLen))
	for i, v := range indexes {
		if i > 0 {
			_, _ = fmt.Fprintf(b, ", ")
		}
		_, _ = fmt.Fprintf(b, "%d", v)
	}
	_, _ = fmt.Fprintf(b, "}")
	return b.String(), nameConst
}

// DeclareNameVars declares the concatenated names string representing all the values in the runs.
func (g *Generator) DeclareNameVars(runs [][]Value, typeName string, suffix string) {
	g.Printf("const _%sName%s = \"", typeName, suffix)
	for _, run := range runs {
		for i := range run {
			g.Printf("%s", run[i].Name)
		}
	}
	g.Printf("\"\n")
	g.Printf("const _%sLowerName%s = \"", typeName, suffix)
	for _, run := range runs {
		for i := range run {
			g.Printf("%s", strings.ToLower(run[i].Name))
		}
	}
	g.Printf("\"\n")
}

// BuildOneRun generates the variables and String method for a single run of contiguous values.
func (g *Generator) BuildOneRun(runs [][]Value, typeName string, isBitFlag bool) {
	values := runs[0]
	g.Printf("\n")
	g.DeclareIndexAndNameVar(values, typeName)
	// The generated code is simple enough to write as a Printf format.
	lessThanZero := ""
	if values[0].Signed {
		lessThanZero = "i < 0 || "
	}
	if values[0].Value == 0 { // Signed or unsigned, 0 is still 0.
		if isBitFlag {
			g.Printf(StringOneRunBitFlag, typeName, Usize(len(values)), lessThanZero)
		} else {
			g.Printf(StringOneRun, typeName, Usize(len(values)), lessThanZero)
		}
	} else {
		if isBitFlag {
			g.Printf(StringOneRunWithOffsetBitFlag, typeName, values[0].String(), Usize(len(values)), lessThanZero)
		} else {
			g.Printf(StringOneRunWithOffset, typeName, values[0].String(), Usize(len(values)), lessThanZero)
		}
	}
}

const (
	// StringMethodName is the name of the String method
	StringMethodName = `String`
	// StringMethodComment is the comment for the String method
	StringMethodComment = `// String returns the string representation
// of this %[1]s value.`
	// BitIndexStringMethodName is the name of the BitIndexString method
	BitIndexStringMethodName = `BitIndexString`
	// BitIndexStringMethodComment is the comment for the BitIndexString method
	BitIndexStringMethodComment = `// BitIndexString returns the string
// representation of the bit flag if
// the bit flag is a bit index value
// (typically an enum constant), and
// not an actual bit flag value.`
)

// Arguments to format are:
//
//	[1]: type name
//	[2]: size of index element (8 for uint8 etc.)
//	[3]: less than zero check (for signed types)
//	[4]: method name (String or BitIndexString)
//	[5]: method comment
const StringOneRun = `%[5]s
func (i %[1]s) %[4]s() string {
	if %[3]si >= %[1]s(len(_%[1]sIndex)-1) {
		return "%[1]s(" + strconv.FormatInt(int64(i), 10) + ")"
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
const StringOneRunWithOffset = `// String returns the string representation
// of this %[1]s value.
func (i %[1]s) String() string {
	i -= %[2]s
	if %[4]si >= %[1]s(len(_%[1]sIndex)-1) {
		return "%[1]s(" + strconv.FormatInt(int64(i + %[2]s), 10) + ")"
	}
	return _%[1]sName[_%[1]sIndex[i] : _%[1]sIndex[i+1]]
}
`

// BuildMultipleRuns generates the variables and String method for multiple runs of contiguous values.
// For this pattern, a single Printf format won't do.
func (g *Generator) BuildMultipleRuns(runs [][]Value, typeName string, isBitFlag bool) {
	g.Printf("\n")
	g.DeclareIndexAndNameVars(runs, typeName)
	g.Printf(`
	// String returns the string representation
	// of this %[1]s value.
	`, typeName)
	g.Printf("func (i %s) String() string {\n", typeName)
	g.Printf("\tswitch {\n")
	for i, values := range runs {
		if len(values) == 1 {
			g.Printf("\tcase i == %s:\n", &values[0])
			g.Printf("\t\treturn _%sName_%d\n", typeName, i)
			continue
		}
		g.Printf("\tcase %s <= i && i <= %s:\n", &values[0], &values[len(values)-1])
		if values[0].Value != 0 {
			g.Printf("\t\ti -= %s\n", &values[0])
		}
		g.Printf("\t\treturn _%sName_%d[_%sIndex_%d[i]:_%sIndex_%d[i+1]]\n",
			typeName, i, typeName, i, typeName, i)
	}
	if isBitFlag {
		g.Printf(StringMultipleRunsBitFlagDefault, typeName)
	} else {
		g.Printf("\tdefault:\n")
		g.Printf("\t\treturn \"%s(\" + strconv.FormatInt(int64(i), 10) + \")\"\n", typeName)
		g.Printf("\t}\n")
		g.Printf("}\n")
	}
}

// BuildMap handles the case where the space is so sparse a map is a reasonable fallback.
// It's a rare situation but has simple code.
func (g *Generator) BuildMap(runs [][]Value, typeName string, isBitFlag bool) {
	g.Printf("\n")
	g.DeclareNameVars(runs, typeName, "")
	g.Printf("\nvar _%sMap = map[%s]string{\n", typeName, typeName)
	n := 0
	for _, values := range runs {
		for _, value := range values {
			g.Printf("\t%s: _%sName[%d:%d],\n", &value, typeName, n, n+len(value.Name))
			n += len(value.Name)
		}
	}
	g.Printf("}\n\n")
	if isBitFlag {
		g.Printf(StringMapBitFlag, typeName)
	} else {
		g.Printf(StringMap, typeName)
	}
}

// BuildNoOpOrderChangeDetect lets the compiler and the user know if the order/value of the enum values has changed.
func (g *Generator) BuildNoOpOrderChangeDetect(runs [][]Value, typeName string) {
	g.Printf("\n")

	g.Printf(`
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the enumgen command to generate them again.
	`)
	g.Printf("func _%sNoOp (){ ", typeName)
	g.Printf("\n var x [1]struct{}\n")
	for _, values := range runs {
		for _, value := range values {
			g.Printf("\t_ = x[%s-(%s)]\n", value.OriginalName, value.Str)
		}
	}
	g.Printf("}\n\n")
}

// Arguments to format are:
//
//	[1]: type name
const StringMap = `// String returns the string representation
// of this %[1]s value.
func (i %[1]s) String() string {
	if str, ok := _%[1]sMap[i]; ok {
		return str
	}
	return "%[1]s(" + strconv.FormatInt(int64(i), 10) + ")"
}
`

// Arguments to format are:
//
//	[1]: type name
//	[2]: number of constants for type
const StringNConstant = `//%[1]sN is the highest valid value
// for type %[1]s, plus one.
const %[1]sN %[1]s = %[2]d
`

// Arguments to format are:
//
//	[1]: type name
const StringSetStringMethod = `// SetString sets the %[1]s value from its
// string representation, and returns an
// error if the string is invalid.
func (i *%[1]s) SetString(s string) error {
	if val, ok := _%[1]sNameToValueMap[s]; ok {
		*i = val
		return nil
	}

	if val, ok := _%[1]sNameToValueMap[strings.ToLower(s)]; ok {
		*i = val
		return nil
	}
	return errors.New(s+" does not belong to %[1]s values")
}
`

// Arguments to format are:
//
//	[1]: type name
const StringInt64Method = `// Int64 returns the %[1]s value as an int64.
func (i %[1]s) Int64() int64 {
	return int64(i)
}
`

// Arguments to format are:
//
//	[1]: type name
const StringSetInt64Method = `// SetInt64 sets the %[1]s value from an int64.
func (i *%[1]s) SetInt64(in int64) {
	*i = %[1]s(in)
}
`

// Arguments to format are:
//
//	[1]: type name
const StringDescMethod = `// Desc returns the description of the %[1]s value.
func (i %[1]s) Desc() string {
	if str, ok := _%[1]sDescMap[i]; ok {
		return str
	}
	return i.String()
}
`

// Arguments to format are:
//
//	[1]: type name
const StringDescsMethod = `	// Descs returns the descriptions of all
// possible values of type %[1]s.
// This slice will be in the same order as
// those returned by Values and Strings.
func (i %[1]s) Descs() []string {
	return _%[1]sDescs
}
`

// Arguments to format are:
//
//	[1]: type name
const StringValuesGlobal = `// %[1]sValues returns all possible values of
// the type %[1]s. This slice will be in the
// same order as those returned by the Values,
// Strings, and Descs methods on %[1]s.
func %[1]sValues() []%[1]s {
	return _%[1]sValues
}
`

// Arguments to format are:
//
//	[1]: type name
const StringValuesMethod = `// Values returns all possible values of
// type %[1]s. This slice will be in the
// same order as those returned by Strings and Descs.
func (i %[1]s) Values() []enums.Enum {
	res := make([]enums.Enum, len(_%[1]sValues))
	for i, d := range _%[1]sValues {
		res[i] = d
	}
	return res 
}
`

// Arguments to format are:
//
//	[1]: type name
const StringStringsMethod = `// Strings returns the string representations of
// all possible values of type %[1]s.
// This slice will be in the same order as
// those returned by Values and Descs.
func (i %[1]s) Strings() []string {
	return _%[1]sNames
}
`

// Arguments to format are:
//
//	[1]: type name
const StringBelongsMethodLoop = `// IsValid returns whether the value is a
// valid option for type %[1]s.
func (i %[1]s) IsValid() bool {
	for _, v := range _%[1]sValues {
		if i == v {
			return true
		}
	}
	return false
}
`

// Arguments to format are:
//
//	[1]: type name
const StringIsValidMethod = `// IsValid returns whether the value is a
// valid option for type %[1]s.
func (i %[1]s) IsValid() bool {
	_, ok := _%[1]sMap[i] 
	return ok
}
`

// BuildBasicExtras builds methods common to all types, like Desc and SetString.
func (g *Generator) BuildBasicExtras(runs [][]Value, typeName string, isBitFlag bool, runsThreshold int) {
	// At this moment, either "g.declareIndexAndNameVars()" or "g.declareNameVars()" has been called

	// Print the slice of values
	max := uint64(0)
	g.Printf("\nvar _%sValues = []%s{", typeName, typeName)
	for _, values := range runs {
		for _, value := range values {
			g.Printf("\t%s, ", value.OriginalName)
			if value.Value > max {
				max = value.Value
			}
		}
	}
	g.Printf("}\n\n")

	g.Printf(StringNConstant, typeName, max+1)

	// Print the map between name and value
	g.PrintValueMap(runs, typeName, runsThreshold)

	// Print the slice of names
	g.PrintNamesSlice(runs, typeName, runsThreshold)

	// Print the map of values to descriptions
	g.PrintDescMap(runs, typeName)
	g.PrintDescSlice(runs, typeName)

	// Print the basic extra methods
	if isBitFlag {
		g.Printf(StringSetStringBitFlagMethod, typeName)
	} else {
		g.Printf(StringSetStringMethod, typeName)
	}
	g.Printf(StringInt64Method, typeName)
	g.Printf(StringSetInt64Method, typeName)
	g.Printf(StringDescMethod, typeName)
	g.Printf(StringValuesGlobal, typeName)
	g.Printf(StringValuesMethod, typeName)
	g.Printf(StringStringsMethod, typeName)
	g.Printf(StringDescsMethod, typeName)
	if len(runs) <= runsThreshold {
		g.Printf(StringBelongsMethodLoop, typeName)
	} else { // There is a map of values, the code is simpler then
		g.Printf(StringIsValidMethod, typeName)
	}
}

// PrintValueMap prints the map between name and value
func (g *Generator) PrintValueMap(runs [][]Value, typeName string, runsThreshold int) {
	thereAreRuns := len(runs) > 1 && len(runs) <= runsThreshold
	g.Printf("\nvar _%sNameToValueMap = map[string]%s{\n", typeName, typeName)

	var n int
	var runID string
	for i, values := range runs {
		if thereAreRuns {
			runID = "_" + fmt.Sprintf("%d", i)
			n = 0
		} else {
			runID = ""
		}

		for _, value := range values {
			g.Printf("\t_%sName%s[%d:%d]: %s,\n", typeName, runID, n, n+len(value.Name), value.OriginalName)
			g.Printf("\t_%sLowerName%s[%d:%d]: %s,\n", typeName, runID, n, n+len(value.Name), value.OriginalName)
			n += len(value.Name)
		}
	}
	g.Printf("}\n\n")
}

// PrintNamesSlice prints the slice of names
func (g *Generator) PrintNamesSlice(runs [][]Value, typeName string, runsThreshold int) {
	thereAreRuns := len(runs) > 1 && len(runs) <= runsThreshold
	g.Printf("\nvar _%sNames = []string{\n", typeName)

	var n int
	var runID string
	for i, values := range runs {
		if thereAreRuns {
			runID = "_" + fmt.Sprintf("%d", i)
			n = 0
		} else {
			runID = ""
		}

		for _, value := range values {
			g.Printf("\t_%sName%s[%d:%d],\n", typeName, runID, n, n+len(value.Name))
			n += len(value.Name)
		}
	}
	g.Printf("}\n\n")
}

// PrintDescMap prints the map of values to descriptions
func (g *Generator) PrintDescMap(runs [][]Value, typeName string) {
	g.Printf("\n")
	g.Printf("\nvar _%sDescMap = map[%s]string{\n", typeName, typeName)
	i := 0
	for _, values := range runs {
		for _, value := range values {
			g.Printf("\t%s: _%sDescs[%d],\n", &value, typeName, i)
			i++
		}
	}
	g.Printf("}\n\n")
}

// PrintDescSlice prints the slice of descriptions
func (g *Generator) PrintDescSlice(runs [][]Value, typeName string) {
	g.Printf("\n")
	g.Printf("\nvar _%sDescs = []string{\n", typeName)
	for _, values := range runs {
		for _, value := range values {
			g.Printf("\t`%s`,\n", value.Desc)
		}
	}
	g.Printf("}\n\n")
}
