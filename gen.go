// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ignore

// This program generates constants for all of the icon
// svg file names in outlined
package main

import (
	"bytes"
	"io/fs"
	"log"
	"os"
	"strings"
	"text/template"
	"unicode"

	"github.com/iancoleman/strcase"
	"goki.dev/icons"
)

const preamble = `// Code generated by "gen.go"; DO NOT EDIT.

package icons

const (`

// iconData contains the data for an icon
type iconData struct {
	Dir   string // Dir is the directory in which the icon is contained
	Snake string // Snake is the snake_case name of the icon
	Camel string // Camel is the CamelCase name of the icon
}

var iconTmpl = template.Must(template.New("icon").Parse(
	`
	// {{.Camel}} is https://github.com/goki/icons/blob/main/{{.Dir}}{{.Snake}}.svg
	{{.Camel}} Icon = "{{.Snake}}"
`,
))

func main() {
	buf := bytes.NewBufferString(preamble)

	fs.WalkDir(icons.Icons, ".", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		name := strings.TrimSuffix(path, ".svg")
		// ignore fill icons as they are handled separately
		if strings.HasSuffix(name, "-fill") {
			return nil
		}
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
			Dir:   "svg/",
			Snake: name,
			Camel: camel,
		}
		return iconTmpl.Execute(buf, data)
	})
	buf.WriteString(")\n")
	err := os.WriteFile("iconnames.go", buf.Bytes(), 0666)
	if err != nil {
		log.Fatalln("error writing result to iconnames.go:", err)
	}
}
