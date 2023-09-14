// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package generate provides the generation
// of useful methods, variables, and constants
// for GoKi code.
package generate

import (
	"fmt"
	"text/template"

	"goki.dev/enums/enumgen"
	"goki.dev/gengo"
	"goki.dev/goki/config"
	"goki.dev/gti/gtigen"
	"golang.org/x/tools/go/packages"
)

// Generate is the main entry point to code generation
// that does all of the generation according to the
// given config info. It overrides the
// [config.Config.Generate.Gtigen.InterfaceConfigs] info.
//
//gti:add
func Generate(cfg *config.Config) error {
	cfg.Generate.Gtigen.InterfaceConfigs = make(map[string]*gtigen.Config)
	cfg.Generate.Gtigen.InterfaceConfigs["goki.dev/ki/v2.Ki"] = &gtigen.Config{
		AddTypes: true,
		Instance: true,
		TypeVar:  true,
		Templates: []*template.Template{
			template.Must(template.New("KiMethods").Parse(
				`
				// Type returns the [*gti.Type] of [{{.Name}}]
				func (t *{{.Name}}) Type() *gti.Type {
					return {{.Name}}Type
				}

				// New returns a new [*{{.Name}}] value
				func (t *{{.Name}}) New() ki.Ki {
					return &{{.Name}}{}
				}`,
			)),
		},
	}
	pkgs, err := ParsePackage(cfg)
	if err != nil {
		return fmt.Errorf("Generate: error parsing package: %w", err)
	}

	err = enumgen.GeneratePkgs(&cfg.Generate.Enumgen, pkgs)
	if err != nil {
		return fmt.Errorf("error running enumgen: %w", err)
	}
	err = gtigen.GeneratePkgs(&cfg.Generate.Gtigen, pkgs)
	if err != nil {
		return fmt.Errorf("error running gtigen: %w", err)
	}
	return nil
}

// ParsePackage parses the package(s) based on the given config info.
func ParsePackage(cfg *config.Config) ([]*packages.Package, error) {
	pcfg := &packages.Config{
		Mode: enumgen.PackageModes() | gtigen.PackageModes(&cfg.Generate.Gtigen), // need superset of both
		// TODO: Need to think about constants in test files. Maybe write type_string_test.go
		// in a separate pass? For later.
		Tests: false,
	}
	return gengo.Load(pcfg, cfg.Generate.Dir)
}
