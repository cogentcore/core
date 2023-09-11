// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gtigen

import (
	"fmt"
	"text/template"

	"goki.dev/gti"
)

var TypeTmpl = template.Must(template.New("Type").Funcs(template.FuncMap{"FieldsCodeString": OrdmapCodeString[string, *gti.Field]}).Parse(
	`var {{if .Config.TypeVar}} {{.Name}}Type {{else}} _ {{end}} = &gti.Type{
		Name: "{{.FullName}}",
		Doc: {{printf "%q" .Doc}},
		Directives: {{printf "%#v" .Directives}},
		{{if ne .Fields nil}} Fields: {{FieldsCodeString .Fields}}, {{end}}
		{{if .Config.Instance}} Instance: &{{.Name}}{}, {{end}}
	}
	`))

// OrdmapCodeString returns the given ordered map as a string
// of valid Go code that constructs the ordered map
func OrdmapCodeString[K comparable, V any](omp *gti.Fields) string {
	var zk K
	var zv V
	res := fmt.Sprintf("ordmap.Make([]ordmap.KeyVal[%T, %T]{", zk, zv)
	for _, kv := range omp.Order {
		res += fmt.Sprintf("{%#v, %#v},", kv.Key, kv.Val)
	}
	res += "})"
	return res
}
