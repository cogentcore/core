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
	if typ.IsBitFlag {
		g.ExecTmpl(StringMethodBitFlagTmpl, typ)
	}
	g.ExecTmpl(StringMethodMapTmpl, typ)
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
	`{{if .IsBitFlag}}
	// BitIndexString returns the string
	// representation of this {{.Name}} value
	// if it is a bit index value
	// (typically an enum constant), and
	// not an actual bit flag value.
	{{- else}}
	// String returns the string representation
	// of this {{.Name}} value.
	{{- end}}
func (i {{.Name}}) {{if .IsBitFlag}} BitIndexString {{else}} String {{end}} () string {
	if str, ok := _{{.Name}}Map[i]; ok {
		return str
	} {{if eq .Extends ""}}
	return strconv.FormatInt(int64(i), 10) {{else}}
	return {{.Extends}}(i).{{if .IsBitFlag}} BitIndexString {{else}} String {{end}}() {{end}}
}
`))

var NConstantTmpl = template.Must(template.New("StringNConstant").Parse(
	`//{{.Name}}N is the highest valid value
// for type {{.Name}}, plus one.
const {{.Name}}N {{.Name}} = {{.MaxValueP1}}
`))

var SetStringMethodTmpl = template.Must(template.New("SetStringMethod").Parse(
	`// SetString sets the {{.Name}} value from its
// string representation, and returns an
// error if the string is invalid.
func (i *{{.Name}}) SetString(s string) error {
	if val, ok := _{{.Name}}NameToValueMap[s]; ok {
		*i = val
		return nil
	}

	if val, ok := _{{.Name}}NameToValueMap[strings.ToLower(s)]; ok {
		*i = val
		return nil
	} {{if eq .Extends ""}}
	return errors.New(s+" is not a valid value for type {{.Name}}") {{else}}
	return (*{{.Extends}})(i).SetString(s) {{end}}
}
`))

var Int64MethodTmpl = template.Must(template.New("Int64Method").Parse(
	`// Int64 returns the {{.Name}} value as an int64.
func (i {{.Name}}) Int64() int64 {
	return int64(i)
}
`))

var SetInt64MethodTmpl = template.Must(template.New("SetInt64Method").Parse(
	`// SetInt64 sets the {{.Name}} value from an int64.
func (i *{{.Name}}) SetInt64(in int64) {
	*i = {{.Name}}(in)
}
`))

var DescMethodTmpl = template.Must(template.New("DescMethod").Parse(`// Desc returns the description of the {{.Name}} value.
func (i {{.Name}}) Desc() string {
	if str, ok := _{{.Name}}DescMap[i]; ok {
		return str
	} {{if eq .Extends ""}}
	return i.String() {{else}}
	return {{.Extends}}(i).Desc() {{end}}
}
`))

var ValuesGlobalTmpl = template.Must(template.New("ValuesGlobal").Parse(
	`// {{.Name}}Values returns all possible values
// for the type {{.Name}}.
func {{.Name}}Values() []{{.Name}} { {{if eq .Extends ""}}
	return _{{.Name}}Values {{else}}
	es := {{.Extends}}Values()
	res := make([]{{.Name}}, len(es))
	for i, e := range es {
		res[i] = {{.Name}}(e)
	}
	res = append(res, _{{.Name}}Values...)
	return res {{end}}
}
`))

var ValuesMethodTmpl = template.Must(template.New("ValuesMethod").Parse(
	`// Values returns all possible values
// for the type {{.Name}}.
func (i {{.Name}}) Values() []enums.Enum { {{if eq .Extends ""}}
	res := make([]enums.Enum, len(_{{.Name}}Values))
	for i, d := range _{{.Name}}Values {
		res[i] = d
	} {{else}}
	es := {{.Extends}}Values()
	les := len(es)
	res := make([]enums.Enum, les + len(_{{.Name}}Values))
	for i, d := range es {
		res[i] = d
	}
	for i, d := range _{{.Name}}Values {
		res[i + les] = d
	} {{end}}
	return res 
}
`))

var IsValidMethodMapTmpl = template.Must(template.New("IsValidMethodMap").Parse(
	`// IsValid returns whether the value is a
// valid option for type {{.Name}}.
func (i {{.Name}}) IsValid() bool {
	_, ok := _{{.Name}}Map[i] {{if ne .Extends ""}}
	if !ok {
		return {{.Extends}}(i).IsValid()
	} {{end}}
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

	typ.MaxValueP1 = max + 1

	g.ExecTmpl(NConstantTmpl, typ)

	g.BuildNoOpOrderChangeDetect(values, typ)

	// Print the map between name and value
	g.PrintValueMap(values, typ)

	// Print the map of values to descriptions
	g.PrintDescMap(values, typ)

	g.BuildString(values, typ)

	// Print the basic extra methods
	if typ.IsBitFlag {
		g.ExecTmpl(SetStringMethodBitFlagTmpl, typ)
		g.ExecTmpl(SetStringOrMethodBitFlagTmpl, typ)
	} else {
		g.ExecTmpl(SetStringMethodTmpl, typ)
	}
	g.ExecTmpl(Int64MethodTmpl, typ)
	g.ExecTmpl(SetInt64MethodTmpl, typ)
	g.ExecTmpl(DescMethodTmpl, typ)
	g.ExecTmpl(ValuesGlobalTmpl, typ)
	g.ExecTmpl(ValuesMethodTmpl, typ)
	g.ExecTmpl(IsValidMethodMapTmpl, typ)
}

// PrintValueMap prints the map between name and value
func (g *Generator) PrintValueMap(values []Value, typ *Type) {
	g.Printf("\nvar _%sNameToValueMap = map[string]%s{\n", typ.Name, typ.Name)
	for _, value := range values {
		g.Printf("\t`%s`: %s,\n", value.Name, &value)
		if typ.Config.AcceptLower {
			l := strings.ToLower(value.Name)
			if l != value.Name { // avoid duplicate keys
				g.Printf("\t`%s`: %s,\n", l, &value)
			}
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
