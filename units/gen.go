// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ignore

package main

import (
	"bytes"
	"html"
	"os"
	"strings"
	"text/template"

	"github.com/iancoleman/strcase"
	"goki.dev/girl/units"
	"goki.dev/grr"
)

func main() {
	buf := &bytes.Buffer{}
	buf.WriteString(
		`// Code generated by "go run gen.go"; DO NOT EDIT.

// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package units
	`)
	for _, v := range units.UnitsValues() {
		// we ignore dots because they also set the dots field
		// as a special case
		if v == units.UnitDot {
			continue
		}
		s := v.String()
		d := data{
			Lower: s,
			Camel: strcase.ToCamel(s),
		}
		// actual desc after =
		_, d.Desc, _ = strings.Cut(v.Desc(), " = ")
		d.Desc = html.UnescapeString(d.Desc)
		grr.Must0(funcs.Execute(buf, d))
	}
	grr.Must0(os.WriteFile("unitgen.go", buf.Bytes(), 0666))
}

type data struct {
	Lower string
	Camel string
	Desc  string
}

var funcs = template.Must(template.New("funcs").Parse(
	`
// {{.Camel}} returns a new {{.Lower}} value.
// {{.Camel}} is {{.Desc}}.
func {{.Camel}}(val float32) Value {
	return Value{Val: val, Un: Unit{{.Camel}}}
}

// {{.Camel}} sets the value in terms of {{.Lower}}.
// {{.Camel}} is {{.Desc}}.
func (v *Value) {{.Camel}}(val float32) {
	v.Val = val
	v.Un = Unit{{.Camel}}
}

// {{.Camel}} converts the given {{.Lower}} value to dots.
// {{.Camel}} is {{.Desc}}.
func (uc *Context) {{.Camel}}(val float32) float32 {
	return uc.ToDots(val, Unit{{.Camel}})
}
`))
