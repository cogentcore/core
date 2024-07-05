// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generate

import (
	"bytes"
	"io/fs"
	"os"
	"strings"
	"text/template"
	"unicode"

	"cogentcore.org/core/base/generate"
	"cogentcore.org/core/base/strcase"
	"cogentcore.org/core/cmd/core/config"
)

// iconData contains the data for an icon
type iconData struct {
	Dir   string // Dir is the directory in which the icon is contained
	Snake string // Snake is the snake_case name of the icon
	Camel string // Camel is the CamelCase name of the icon
}

var iconTmpl = template.Must(template.New("icon").Parse(
	`
	//go:embed {{.Dir}}/{{.Snake}}.svg
	{{.Camel}} Icon`,
))

// Icons does any necessary generation for icons.
func Icons(c *config.Config) error {
	if c.Generate.Icons == "" {
		return nil
	}
	b := &bytes.Buffer{}
	generate.PrintHeader(b, "icons")
	b.WriteString(`import _ "embed"

var (`)

	fs.WalkDir(os.DirFS(c.Generate.Icons), ".", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		name := strings.TrimSuffix(path, ".svg")
		// ignore blank icon, as we define the constant for that separately
		if name == "blank" {
			return nil
		}
		camel := strcase.ToCamel(name)
		// identifier names can't start with a digit
		if unicode.IsDigit([]rune(camel)[0]) {
			camel = "X" + camel
		}
		data := iconData{
			Dir:   c.Generate.Icons,
			Snake: name,
			Camel: camel,
		}
		return iconTmpl.Execute(b, data)
	})
	b.WriteString("\n)\n")
	return generate.Write("icongen.go", b.Bytes(), nil)
}
