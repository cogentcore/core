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

// BuildString builds the string function using a map access approach.
func (g *Generator) BuildString(values []Value, typ *Type) {
	g.Printf("\n")
	g.Printf("\nvar _%sMap = map[%s]string{\n", typ.Name, typ.Name)
	n := 0
	for _, value := range values {
		g.Printf("\t%s: `%s`,\n", &value, value.Name)
		n += len(value.Name)
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
func (g *Generator) BuildNoOpOrderChangeDetect(values []Value, typ *Type) {
	g.Printf("\n")

	g.Printf(`
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the enumgen command to generate them again.
	`)
	g.Printf("func _%sNoOp (){ ", typ.Name)
	g.Printf("\n var x [1]struct{}\n")
	for _, value := range values {
		g.Printf("\t_ = x[%s-(%s)]\n", value.OriginalName, value.Str)
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
	`// {{.TypeName}}Values returns all possible values
// for the type {{.TypeName}}.
func {{.TypeName}}Values() []{{.TypeName}} {
	return _{{.TypeName}}Values
}
`))

var ValuesMethodTmpl = template.Must(template.New("ValuesMethod").Parse(
	`// Values returns all possible values
// for the type {{.TypeName}}.
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

// BuildBasicMethods builds methods common to all types, like Desc and SetString.
func (g *Generator) BuildBasicMethods(values []Value, typ *Type) {

	// Print the slice of values
	max := uint64(0)
	g.Printf("\nvar _%sValues = []%s{", typ.Name, typ.Name)
	for _, value := range values {
		g.Printf("\t%s, ", &value)
		if value.Value > max {
			max = value.Value
		}
	}
	g.Printf("}\n\n")

	d := &TmplData{
		TypeName:   typ.Name,
		MaxValueP1: fmt.Sprintf("%d", max+1),
	}

	g.ExecTmpl(NConstantTmpl, d)

	g.BuildNoOpOrderChangeDetect(values, typ)

	// Print the map between name and value
	g.PrintValueMap(values, typ)

	// Print the map of values to descriptions
	g.PrintDescMap(values, typ)

	g.BuildString(values, typ)

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
	if len(values) <= typ.RunsThreshold {
		g.ExecTmpl(IsValidMethodLoopTmpl, d)
	} else { // There is a map of values, the code is simpler then
		g.ExecTmpl(IsValidMethodMapTmpl, d)
	}
}

// PrintValueMap prints the map between name and value
func (g *Generator) PrintValueMap(values []Value, typ *Type) {
	g.Printf("\nvar _%sNameToValueMap = map[string]%s{\n", typ.Name, typ.Name)
	for _, value := range values {
		g.Printf("\t`%s`: %s,\n", value.Name, &value)
		l := strings.ToLower(value.Name)
		if l != value.Name { // avoid duplicate keys
			g.Printf("\t`%s`: %s,\n", l, &value)
		}
	}
	g.Printf("}\n\n")
}

// PrintDescMap prints the map of values to descriptions
func (g *Generator) PrintDescMap(values []Value, typ *Type) {
	g.Printf("\n")
	g.Printf("\nvar _%sDescMap = map[%s]string{\n", typ.Name, typ.Name)
	i := 0
	for _, value := range values {
		g.Printf("\t%s: `%s`,\n", &value, value.Desc)
		i++
	}
	g.Printf("}\n\n")
}
