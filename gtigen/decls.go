// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gtigen

import (
	"strings"
	"text/template"

	"goki.dev/gti"
	"goki.dev/ordmap"
)

// TypeTmpl is the template for [gti.Type] declarations.
// It takes a [*Type] as its value.
var TypeTmpl = template.Must(template.New("Type").Parse(
	`
	{{if .Config.TypeVar}} // {{.Name}}Type is the [gti.Type] for [{{.Name}}]
	var {{.Name}}Type {{else}} var _ {{end}} = gti.AddType(&gti.Type{
		Name: "{{.FullName}}",
		ShortName: "{{.ShortName}}",
		IDName: "{{.IDName}}",
		Doc: {{printf "%q" .Doc}},
		Directives: {{printf "%#v" .Directives}},
		{{if ne .Fields nil}} Fields: {{printf "%#v" .Fields}}, {{end}}
		{{if ne .Embeds nil}} Embeds: {{printf "%#v" .Embeds}}, {{end}}
		Methods: {{printf "%#v" .Methods}},
		{{if .Config.Instance}} Instance: &{{.Name}}{}, {{end}}
	})
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

// SetterMethodsTmpl is the template for setter methods for a type.
// It takes a [*Type] as its value.
var SetterMethodsTmpl = template.Must(template.New("SetterMethods").
	Funcs(template.FuncMap{
		"SetterFields": SetterFields,
		"DocToComment": DocToComment,
	}).Parse(
	`
	{{$typ := .}}
	{{range (SetterFields .)}}
	// Set{{.Name}} sets the [{{$typ.Name}}.{{.Name}}]:
	{{DocToComment .Doc}}
	func (t *{{$typ.Name}}) Set{{.Name}}(v {{.LocalType}}) *{{$typ.Name}} {
		t.{{.Name}} = v
		return t
	}
	{{end}}
`))

// SetterFields returns all of the fields and embedded fields of the given type
// that don't have a `set:"-"` struct tag.
func SetterFields(typ *Type) []*gti.Field {
	res := []*gti.Field{}
	do := func(fields *ordmap.Map[string, *gti.Field]) {
		for _, kv := range fields.Order {
			f := kv.Val
			// unspecified indicates to add a set method; only "-" means no set
			hasSetter := f.Tag.Get("set") != "-"
			if hasSetter {
				res = append(res, f)
			}
		}
	}
	do(typ.Fields)
	do(typ.EmbeddedFields)
	return res
}

// DocToComment converts the given doc string to an appropriate comment string.
func DocToComment(doc string) string {
	return "// " + strings.ReplaceAll(doc, "\n", "\n// ")
}
