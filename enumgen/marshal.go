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

var JSONMethodsTmpl = template.Must(template.New("JSONMethods").Parse(
	`
// MarshalJSON implements the [json.Marshaler] interface.
func (i {{.TypeName}}) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.String())
}

// UnmarshalJSON implements the [json.Unmarshaler] interface.
func (i *{{.TypeName}}) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return errors.New("{{.TypeName}} should be a string, but got " + string(data) + "instead")
	}
	return i.SetString(s)
}
`))

func (g *Generator) BuildJSONMethods(runs [][]Value, typeName string, runsThreshold int) {
	d := &TmplData{
		TypeName: typeName,
	}
	g.ExecTmpl(JSONMethodsTmpl, d)
}

var TextMethods = template.Must(template.New("TextMethods").Parse(
	`
// MarshalText implements the [encoding.TextMarshaler] interface.
func (i {{.TypeName}}) MarshalText() ([]byte, error) {
	return []byte(i.String()), nil
}

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
func (i *{{.TypeName}}) UnmarshalText(text []byte) error {
	return i.SetString(string(text))
}
`))

func (g *Generator) BuildTextMethods(runs [][]Value, typeName string, runsThreshold int) {
	d := &TmplData{
		TypeName: typeName,
	}
	g.ExecTmpl(TextMethods, d)
}

var YAMLMethods = template.Must(template.New("YAMLMethods").Parse(
	`
// MarshalYAML implements a YAML Marshaler.
func (i {{.TypeName}}) MarshalYAML() (any, error) {
	return i.String(), nil
}

// UnmarshalYAML implements a YAML Unmarshaler.
func (i *{{.TypeName}}) UnmarshalYAML(unmarshal func(any) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	return i.SetString(s)
}
`))

func (g *Generator) BuildYAMLMethods(runs [][]Value, typeName string, runsThreshold int) {
	d := &TmplData{
		TypeName: typeName,
	}
	g.ExecTmpl(YAMLMethods, d)
}
