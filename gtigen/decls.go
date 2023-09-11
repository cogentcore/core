// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gtigen

import (
	"text/template"
)

// TypeTmpl is the template for [gti.Type] declarations.
// It takes a [*Type] as its value
var TypeTmpl = template.Must(template.New("Type").Parse(
	`
	var {{if .Config.TypeVar}} {{.Name}}Type {{else}} _ {{end}} = &gti.Type{
		Name: "{{.FullName}}",
		Doc: {{printf "%q" .Doc}},
		Directives: {{printf "%#v" .Directives}},
		{{if ne .Fields nil}} Fields: {{printf "%#v" .Fields}}, {{end}}
		Methods: {{printf "%#v" .Methods}},
		{{if .Config.Instance}} Instance: &{{.Name}}{}, {{end}}
	}
	`))

// FuncTmpl is the template for [gti.Func] declarations.
// It takes a [*gti.Func] as its value.
var FuncTmpl = template.Must(template.New("Func").Parse(
	`
	var _ = &gti.Func{
		Name: "{{.Name}}",
		Doc: {{printf "%q" .Doc}},
		Directives: {{printf "%#v" .Directives}},
		Args: {{printf "%#v" .Args}},
		Returns: {{printf "%#v" .Returns}},
	}
	`))
