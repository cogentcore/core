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

var GQLMethodsTmpl = template.Must(template.New("GQLMethods").Parse(`
// MarshalGQL implements the [graphql.Marshaler] interface.
func (i {{.Name}}) MarshalGQL(w io.Writer) { w.Write([]byte(strconv.Quote(i.String()))) }

// UnmarshalGQL implements the [graphql.Unmarshaler] interface.
func (i *{{.Name}}) UnmarshalGQL(value any) error { return enums.Scan(i, value, "{{.Name}}") }
`))

func (g *Generator) BuildGQLMethods(runs []Value, typ *Type) {
	g.ExecTmpl(GQLMethodsTmpl, typ)
}
