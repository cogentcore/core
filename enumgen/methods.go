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
	"text/template"
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
func (g *Generator) DeclareIndexAndNameVars(runs [][]Value, typ *Type) {
	var indexes, names []string
	for i, run := range runs {
		index, n := g.CreateIndexAndNameDecl(run, typ, fmt.Sprintf("_%d", i))
		indexes = append(indexes, index)
		names = append(names, n)
		_, n = g.CreateLowerIndexAndNameDecl(run, typ, fmt.Sprintf("_%d", i))
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
func (g *Generator) DeclareIndexAndNameVar(run []Value, typ *Type) {
	index, n := g.CreateIndexAndNameDecl(run, typ, "")
	g.Printf("const %s\n", n)
	g.Printf("var %s\n", index)
	index, n = g.CreateLowerIndexAndNameDecl(run, typ, "")
	g.Printf("const %s\n", n)
	// g.Printf("var %s\n", index)
}

// createIndexAndNameDecl returns the pair of declarations for the run. The caller will add "const" and "var".
func (g *Generator) CreateLowerIndexAndNameDecl(run []Value, typ *Type, suffix string) (string, string) {
	b := new(bytes.Buffer)
	indexes := make([]int, len(run))
	for i := range run {
		b.WriteString(strings.ToLower(run[i].Name))
		indexes[i] = b.Len()
	}
	nameConst := fmt.Sprintf("_%sLowerName%s = %q", typ.Name, suffix, b.String())
	nameLen := b.Len()
	b.Reset()
	_, _ = fmt.Fprintf(b, "_%sLowerIndex%s = [...]uint%d{0, ", typ.Name, suffix, Usize(nameLen))
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
func (g *Generator) CreateIndexAndNameDecl(run []Value, typ *Type, suffix string) (string, string) {
	b := new(bytes.Buffer)
	indexes := make([]int, len(run))
	for i := range run {
		b.WriteString(run[i].Name)
		indexes[i] = b.Len()
	}
	nameConst := fmt.Sprintf("_%sName%s = %q", typ.Name, suffix, b.String())
	nameLen := b.Len()
	b.Reset()
	_, _ = fmt.Fprintf(b, "_%sIndex%s = [...]uint%d{0, ", typ.Name, suffix, Usize(nameLen))
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
func (g *Generator) DeclareNameVars(runs [][]Value, typ *Type, suffix string) {
	g.Printf("const _%sName%s = \"", typ.Name, suffix)
	for _, run := range runs {
		for i := range run {
			g.Printf("%s", run[i].Name)
		}
	}
	g.Printf("\"\n")
	g.Printf("const _%sLowerName%s = \"", typ.Name, suffix)
	for _, run := range runs {
		for i := range run {
			g.Printf("%s", strings.ToLower(run[i].Name))
		}
	}
	g.Printf("\"\n")
}

// BuildOneRun generates the variables and String method for a single run of contiguous values.
func (g *Generator) BuildOneRun(runs [][]Value, typ *Type) {
	values := runs[0]
	g.Printf("\n")
	g.DeclareIndexAndNameVar(values, typ)
	// The generated code is simple enough to write as a template.
	d := &TmplData{
		TypeName:         typ.Name,
		MinValue:         values[0].String(),
		IndexElementSize: Usize(len(values)),
	}
	if values[0].Signed {
		d.LessThanZeroCheck = "i < 0 || "
	}
	d.SetMethod(typ.IsBitFlag)
	d.SetIfInvalidForString(typ.Extends, d.MinValue)
	if values[0].Value == 0 { // Signed or unsigned, 0 is still 0.
		g.ExecTmpl(StringMethodOneRunTmpl, d)
	} else {
		g.ExecTmpl(StringMethodOneRunWithOffsetTmpl, d)
	}
}

var StringMethodOneRunTmpl = template.Must(template.New("StringMethodOneRun").Parse(
	`{{.MethodComment}}
func (i {{.TypeName}}) {{.MethodName}}() string {
	if {{.LessThanZeroCheck}}i >= {{.TypeName}}(len(_{{.TypeName}}Index)-1) {
		{{.IfInvalid}}
	}
	return _{{.TypeName}}Name[_{{.TypeName}}Index[i]:_{{.TypeName}}Index[i+1]]
}
`))

var StringMethodOneRunWithOffsetTmpl = template.Must(template.New("StringMethodOneRunWithOffset").Parse(
	`{{.MethodComment}}
func (i {{.TypeName}}) {{.MethodName}}() string {
	i -= {{.MinValue}}
	if {{.LessThanZeroCheck}}i >= {{.TypeName}}(len(_{{.TypeName}}Index)-1) {
		{{.IfInvalid}}
	}
	return _{{.TypeName}}Name[_{{.TypeName}}Index[i] : _{{.TypeName}}Index[i+1]]
}
`))

// BuildMultipleRuns generates the variables and String method for multiple runs of contiguous values.
// For this pattern, a single Printf format won't do.
func (g *Generator) BuildMultipleRuns(runs [][]Value, typ *Type) {
	g.Printf("\n")
	g.DeclareIndexAndNameVars(runs, typ)
	d := &TmplData{
		TypeName: typ.Name,
	}
	d.SetMethod(typ.IsBitFlag)
	d.SetIfInvalidForString(typ.Extends, "")
	g.Printf(d.MethodComment)
	g.Printf("\n")
	if typ.IsBitFlag {
		g.Printf("func (i %s) BitIndexString() string {\n", typ.Name)
	} else {
		g.Printf("func (i %s) String() string {\n", typ.Name)
	}
	g.Printf("\tswitch {\n")
	for i, values := range runs {
		if len(values) == 1 {
			g.Printf("\tcase i == %s:\n", &values[0])
			g.Printf("\t\treturn _%sName_%d\n", typ.Name, i)
			continue
		}
		g.Printf("\tcase %s <= i && i <= %s:\n", &values[0], &values[len(values)-1])
		if values[0].Value != 0 {
			g.Printf("\t\ti -= %s\n", &values[0])
		}
		g.Printf("\t\treturn _%sName_%d[_%sIndex_%d[i]:_%sIndex_%d[i+1]]\n",
			typ.Name, i, typ.Name, i, typ.Name, i)
	}

	g.Printf("\tdefault:\n")
	g.Printf(d.IfInvalid)
	g.Printf("\t}\n")
	g.Printf("}\n")
}

// BuildMap handles the case where the space is so sparse a map is a reasonable fallback.
// It's a rare situation but has simple code.
func (g *Generator) BuildMap(runs [][]Value, typ *Type) {
	g.Printf("\n")
	g.DeclareNameVars(runs, typ, "")
	g.Printf("\nvar _%sMap = map[%s]string{\n", typ.Name, typ.Name)
	n := 0
	for _, values := range runs {
		for _, value := range values {
			g.Printf("\t%s: _%sName[%d:%d],\n", &value, typ.Name, n, n+len(value.Name))
			n += len(value.Name)
		}
	}
	g.Printf("}\n\n")
	d := &TmplData{
		TypeName: typ.Name,
	}
	d.SetMethod(typ.IsBitFlag)
	d.SetIfInvalidForString(typ.Extends, "")
	g.ExecTmpl(StringMethodMapTmpl, d)
}

// BuildNoOpOrderChangeDetect lets the compiler and the user know if the order/value of the enum values has changed.
func (g *Generator) BuildNoOpOrderChangeDetect(runs [][]Value, typ *Type) {
	g.Printf("\n")

	g.Printf(`
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the enumgen command to generate them again.
	`)
	g.Printf("func _%sNoOp (){ ", typ.Name)
	g.Printf("\n var x [1]struct{}\n")
	for _, values := range runs {
		for _, value := range values {
			g.Printf("\t_ = x[%s-(%s)]\n", value.OriginalName, value.Str)
		}
	}
	g.Printf("}\n\n")
}

var StringMethodMapTmpl = template.Must(template.New("StringMethodMap").Parse(
	`{{.MethodComment}}
func (i {{.TypeName}}) {{.MethodName}}() string {
	if str, ok := _{{.TypeName}}Map[i]; ok {
		return str
	}
	{{.IfInvalid}}
}
`))

var NConstantTmpl = template.Must(template.New("StringNConstant").Parse(
	`//{{.TypeName}}N is the highest valid value
// for type {{.TypeName}}, plus one.
const {{.TypeName}}N {{.TypeName}} = {{.MaxValueP1}}
`))

var SetStringMethodTmpl = template.Must(template.New("SetStringMethod").Parse(
	`// SetString sets the {{.TypeName}} value from its
// string representation, and returns an
// error if the string is invalid.
func (i *{{.TypeName}}) SetString(s string) error {
	if val, ok := _{{.TypeName}}NameToValueMap[s]; ok {
		*i = val
		return nil
	}

	if val, ok := _{{.TypeName}}NameToValueMap[strings.ToLower(s)]; ok {
		*i = val
		return nil
	}
	{{.IfInvalid}}
}
`))

var Int64MethodTmpl = template.Must(template.New("Int64Method").Parse(
	`// Int64 returns the {{.TypeName}} value as an int64.
func (i {{.TypeName}}) Int64() int64 {
	return int64(i)
}
`))

var SetInt64MethodTmpl = template.Must(template.New("SetInt64Method").Parse(
	`// SetInt64 sets the {{.TypeName}} value from an int64.
func (i *{{.TypeName}}) SetInt64(in int64) {
	*i = {{.TypeName}}(in)
}
`))

var DescMethodTmpl = template.Must(template.New("DescMethod").Parse(`// Desc returns the description of the {{.TypeName}} value.
func (i {{.TypeName}}) Desc() string {
	if str, ok := _{{.TypeName}}DescMap[i]; ok {
		return str
	}
	return i.String()
}
`))

var ValuesGlobalTmpl = template.Must(template.New("ValuesGlobal").Parse(
	`// {{.TypeName}}Values returns all possible values for
// the type {{.TypeName}}.
func {{.TypeName}}Values() []{{.TypeName}} {
	return _{{.TypeName}}Values
}
`))

var ValuesMethodTmpl = template.Must(template.New("ValuesMethod").Parse(
	`// Values returns all possible values for
// the type {{.TypeName}}.
func (i {{.TypeName}}) Values() []enums.Enum {
	res := make([]enums.Enum, len(_{{.TypeName}}Values))
	for i, d := range _{{.TypeName}}Values {
		res[i] = d
	}
	return res 
}
`))

var IsValidMethodLoopTmpl = template.Must(template.New("IsValidMethodLoop").Parse(
	`// IsValid returns whether the value is a
// valid option for type {{.TypeName}}.
func (i {{.TypeName}}) IsValid() bool {
	for _, v := range _{{.TypeName}}Values {
		if i == v {
			return true
		}
	}
	return false
}
`))

var IsValidMethodMapTmpl = template.Must(template.New("IsValidMethodMap").Parse(
	`// IsValid returns whether the value is a
// valid option for type {{.TypeName}}.
func (i {{.TypeName}}) IsValid() bool {
	_, ok := _{{.TypeName}}Map[i] 
	return ok
}
`))

// BuildBasicExtras builds methods common to all types, like Desc and SetString.
func (g *Generator) BuildBasicExtras(runs [][]Value, typ *Type) {
	// At this moment, either "g.declareIndexAndNameVars()" or "g.declareNameVars()" has been called

	// Print the slice of values
	max := uint64(0)
	g.Printf("\nvar _%sValues = []%s{", typ.Name, typ.Name)
	for _, values := range runs {
		for _, value := range values {
			g.Printf("\t%s, ", value.OriginalName)
			if value.Value > max {
				max = value.Value
			}
		}
	}
	g.Printf("}\n\n")

	d := &TmplData{
		TypeName:   typ.Name,
		MaxValueP1: fmt.Sprintf("%d", max+1),
	}

	g.ExecTmpl(NConstantTmpl, d)

	// Print the map between name and value
	g.PrintValueMap(runs, typ)

	// Print the slice of names
	g.PrintNamesSlice(runs, typ)

	// Print the map of values to descriptions
	g.PrintDescMap(runs, typ)

	// Print the basic extra methods
	d.SetIfInvalidForSetString(typ.Extends, typ.IsBitFlag)
	if typ.IsBitFlag {
		g.ExecTmpl(SetStringMethodBitFlagTmpl, d)
	} else {
		g.ExecTmpl(SetStringMethodTmpl, d)
	}
	g.ExecTmpl(Int64MethodTmpl, d)
	g.ExecTmpl(SetInt64MethodTmpl, d)
	g.ExecTmpl(DescMethodTmpl, d)
	g.ExecTmpl(ValuesGlobalTmpl, d)
	g.ExecTmpl(ValuesMethodTmpl, d)
	if len(runs) <= typ.RunsThreshold {
		g.ExecTmpl(IsValidMethodLoopTmpl, d)
	} else { // There is a map of values, the code is simpler then
		g.ExecTmpl(IsValidMethodMapTmpl, d)
	}
}

// PrintValueMap prints the map between name and value
func (g *Generator) PrintValueMap(runs [][]Value, typ *Type) {
	thereAreRuns := len(runs) > 1 && len(runs) <= typ.RunsThreshold
	g.Printf("\nvar _%sNameToValueMap = map[string]%s{\n", typ.Name, typ.Name)

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
			g.Printf("\t_%sName%s[%d:%d]: %s,\n", typ.Name, runID, n, n+len(value.Name), value.OriginalName)
			g.Printf("\t_%sLowerName%s[%d:%d]: %s,\n", typ.Name, runID, n, n+len(value.Name), value.OriginalName)
			n += len(value.Name)
		}
	}
	g.Printf("}\n\n")
}

// PrintNamesSlice prints the slice of names
func (g *Generator) PrintNamesSlice(runs [][]Value, typ *Type) {
	thereAreRuns := len(runs) > 1 && len(runs) <= typ.RunsThreshold
	g.Printf("\nvar _%sNames = []string{\n", typ.Name)

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
			g.Printf("\t_%sName%s[%d:%d],\n", typ.Name, runID, n, n+len(value.Name))
			n += len(value.Name)
		}
	}
	g.Printf("}\n\n")
}

// PrintDescMap prints the map of values to descriptions
func (g *Generator) PrintDescMap(runs [][]Value, typ *Type) {
	g.Printf("\n")
	g.Printf("\nvar _%sDescMap = map[%s]string{\n", typ.Name, typ.Name)
	i := 0
	for _, values := range runs {
		for _, value := range values {
			g.Printf("\t%s: `%s`,\n", &value, value.Desc)
			i++
		}
	}
	g.Printf("}\n\n")
}
