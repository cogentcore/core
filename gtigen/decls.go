// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gtigen

import "text/template"

var TypeTmpl = template.Must(template.New("Type").Parse(
	`var {{if .Config.TypeVar}} {{.Name}}Type {{else}} _ {{end}} = &gti.Type{
		Name: "{{.Name}}",
		Directives: {{printf "%#v" .Directives}},
	}
	`))
