// Copyright (c) 2023, The GoKi Authors. All rights reserved.
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
	`// Scan implements the [driver.Valuer] interface.
func (i {{.TypeName}}) Value() (driver.Value, error) {
	return i.String(), nil
}
`))

var ScanMethodTmpl = template.Must(template.New("ScanMethod").Parse(
	`// Value implements the [sql.Scanner] interface.
func (i *{{.TypeName}}) Scan(value any) error {
	if value == nil {
		return nil
	}

	var str string
	switch v := value.(type) {
	case []byte:
		str = string(v)
	case string:
		str = v
	case fmt.Stringer:
		str = v.String()
	default:
		return fmt.Errorf("invalid value for type {{.TypeName}}: %[1]T(%[1]v)", value)
	}

	return i.SetString(str)
}
`))

func (g *Generator) AddValueAndScanMethod(typeName string) {
	d := &TmplData{
		TypeName: typeName,
	}
	g.Printf("\n")
	g.ExecTmpl(ValueMethodTmpl, d)
	g.Printf("\n\n")
	g.ExecTmpl(ScanMethodTmpl, d)
}
