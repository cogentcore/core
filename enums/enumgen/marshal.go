// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on http://github.com/dmarkham/enumer and
// golang.org/x/tools/cmd/stringer:

// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package enumgen

import "text/template"

var TextMethodsTmpl = template.Must(template.New("TextMethods").Parse(
	`
// MarshalText implements the [encoding.TextMarshaler] interface.
func (i {{.Name}}) MarshalText() ([]byte, error) { return []byte(i.String()), nil }

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
func (i *{{.Name}}) UnmarshalText(text []byte) error { return enums.UnmarshalText(i, text, "{{.Name}}") }
`))

func (g *Generator) BuildTextMethods(runs []Value, typ *Type) {
	g.ExecTmpl(TextMethodsTmpl, typ)
}
