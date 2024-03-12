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

var ValueMethodTmpl = template.Must(template.New("ValueMethod").Parse(
	`// Value implements the [driver.Valuer] interface.
func (i {{.Name}}) Value() (driver.Value, error) { return i.String(), nil }
`))

var ScanMethodTmpl = template.Must(template.New("ScanMethod").Parse(
	`// Scan implements the [sql.Scanner] interface.
func (i *{{.Name}}) Scan(value any) error { return enums.Scan(i, value, "{{.Name}}") }
`))

func (g *Generator) AddValueAndScanMethod(typ *Type) {
	g.Printf("\n")
	g.ExecTmpl(ValueMethodTmpl, typ)
	g.Printf("\n\n")
	g.ExecTmpl(ScanMethodTmpl, typ)
}
