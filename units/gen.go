// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ignore

package main

import (
	"bytes"
	"os"
	"text/template"

	"github.com/iancoleman/strcase"
	"goki.dev/girl/units"
	"goki.dev/grr"
)

func main() {
	buf := &bytes.Buffer{}
	buf.WriteString(`
// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package units
	`)
	for _, v := range units.UnitsValues() {
		s := v.String()
		d := data{
			Lower: s,
			Camel: strcase.ToCamel(s),
			Desc:  v.Desc(),
		}
		grr.Must0(newFuncs.Execute(buf, d))
	}
	grr.Must0(os.WriteFile("unitgen.go", buf.Bytes(), 0666))
}

type data struct {
	Lower string
	Camel string
	Desc  string
}

var newFuncs = template.Must(template.New("newFuncs").Parse(
	`
// {{.Camel}} returns a new {{.Lower}} value:
// {{.Desc}}
func {{.Camel}}(val float32) Value {
	return Value{Val: val, Un: Unit{{.Camel}}}
}
`))
