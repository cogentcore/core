// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"html/template"
	"os"
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
	return MakeFiles(c)
}

// MakeFiles makes the necessary static web files based on the given configuration information.
func MakeFiles(c *config.Config) error {
	odir := filepath.Dir(c.Build.Output)

	wej := []byte(WASMExecJS())
	err := os.WriteFile(filepath.Join(odir, "wasm_exec.js"), wej, 0666)
	if err != nil {
		return err
	}

	ajs, err := MakeAppJS(c)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(odir, "app.js"), ajs, 0666)
	if err != nil {
		return err
	}

	awjs, err := MakeAppWorkerJS(c)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(odir, "app-worker.js"), awjs, 0666)
	if err != nil {
		return err
	}

	man, err := MakeManifestJSON(c)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(odir, "manifest.webmanifest"), man, 0666)
	if err != nil {
		return err
	}

	acs := []byte(AppCSS)
	err = os.WriteFile(filepath.Join(odir, "app.css"), acs, 0666)
	if err != nil {
		return err
	}

	iht, err := MakeIndexHTML(c)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(odir, "index.html"), iht, 0666)
	if err != nil {
		return err
	}

	return nil
}
