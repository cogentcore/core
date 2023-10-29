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
		d := data{Camel: strcase.ToCamel(v.String())}
		grr.Must0(newFuncs.Execute(buf, d))
	}
	grr.Must0(os.WriteFile("unitgen.go", buf.Bytes(), 0666))
}

type data struct {
	Camel string
}

var newFuncs = template.Must(template.New("newFuncs").Parse(
	`
func {{.Camel}}(val float32) Value {
	return Value{Val: val, Un: Unit{{.Camel}}}
}
`))
