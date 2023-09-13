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
	{{if .Config.TypeVar}} // {{.Name}}Type is the [gti.Type] for [{{.Name}}]
	var {{.Name}}Type {{else}} var _ {{end}} = gti.AddType(&gti.Type{
		Name: "{{.FullName}}",
		Doc: {{printf "%q" .Doc}},
		Directives: {{printf "%#v" .Directives}},
		{{if ne .Fields nil}} Fields: {{printf "%#v" .Fields}}, {{end}}
		Methods: {{printf "%#v" .Methods}},
		{{if .Config.Instance}} Instance: &{{.Name}}{}, {{end}}
	})
	{{if .Config.TypeMethod}}
	// Type returns the [*gti.Type] of [{{.Name}}]
	func (t *{{.Name}}) Type() *gti.Type {
		return {{.Name}}Type
	} {{end}}
	{{if .Config.NewMethod}}
	// New returns a new [*{{.Name}}] value
	func (t *{{.Name}}) New() any {
		return &{{.Name}}{}
	} {{end}}
	`))

// FuncTmpl is the template for [gti.Func] declarations.
// It takes a [*gti.Func] as its value.
var FuncTmpl = template.Must(template.New("Func").Parse(
	`
	var _ = gti.AddFunc(&gti.Func{
		Name: "{{.Name}}",
		Doc: {{printf "%q" .Doc}},
		Directives: {{printf "%#v" .Directives}},
		Args: {{printf "%#v" .Args}},
		Returns: {{printf "%#v" .Returns}},
	})
	`))
