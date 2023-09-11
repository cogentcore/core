// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gtigen

import (
	"fmt"
	"text/template"

	"goki.dev/gti"
)

var TypeTmpl = template.Must(template.New("Type").Funcs(template.FuncMap{"OrdmapCodeString": OrdmapCodeString}).Parse(
	`var {{if .Config.TypeVar}} {{.Name}}Type {{else}} _ {{end}} = &gti.Type{
		Name: "{{.FullName}}",
		Doc: ` + "`" + `{{.Doc}}` + "`" + `,
		Directives: gti.Directives{ {{range .Directives}} {{printf "%#v" .}}, {{end}} },
		{{if ne .Fields nil}} Fields: {{OrdmapCodeString .Fields}}, {{end}}
		{{if .Config.Instance}} Instance: &{{.Name}}{}, {{end}}
	}
	`)).Funcs(template.FuncMap{})

// OrdmapCodeString returns the given ordered map as a string
// of valid Go code that constructs the ordered map
func OrdmapCodeString(omp *gti.Fields) string {
	res := "ordmap.Make([]ordmap.KeyVal[string, *gti.Field]{"
	for _, kv := range omp.Order {
		res += fmt.Sprintf("{%q, %#v},", kv.Key, kv.Val)
	}
	res += "})"
	return res
}
