// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"bytes"
	"html/template"
	"path/filepath"

	"goki.dev/goki/config"
	"goki.dev/xe"
)

// Templates for web files
var (
	DefaultAppWorkerJSTmpl = template.Must(template.New("DefaultAppWorkerJS").Parse(DefaultAppWorkerJS))
)

// Build builds an app for web using the given configuration information.
func Build(c *config.Config) error {
	err := xe.Major().SetEnv("GOOS", "js").SetEnv("GOARCH", "wasm").Run("go", "build", "-o", c.Build.Output, c.Build.Package)
	if err != nil {
		return err
	}

	odir := filepath.Dir(c.Build.Output)

	buf := &bytes.Buffer{}
	err = DefaultAppWorkerJSTmpl.Execute()
}
