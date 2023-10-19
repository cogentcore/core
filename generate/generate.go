// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package generate provides the generation
// of useful methods, variables, and constants
// for GoKi code.
package generate

import (
	"fmt"
	"slices"
	"text/template"

	"goki.dev/enums/enumgen"
	"goki.dev/gengo"
	"goki.dev/goki/config"
	"goki.dev/gti"
	"goki.dev/gti/gtigen"
	"golang.org/x/tools/go/packages"
)

// GeneralMethodsTmpl is a template that contains the methods
// and functions applicable to all types.
var GeneralMethodsTmpl = template.Must(template.New("GeneralMethods").
	Funcs(template.FuncMap{
		"SetterFields": SetterFields,
	}).Parse(
	`
	{{$typ := .}}
	{{range (SetterFields .)}}
	// Set{{.Name}} sets the {{.Name}} of the {{$typ.Name}} and
	// returns it to allow chaining together set calls.
	func (t *{{$typ.Name}}) Set{{.Name}}(v {{.Type}}) *{{$typ.Name}} {
		t.{{.Name}} = v
		return t
	}
	{{end}}
	`))

// KiMethodsTmpl is a template that contains the methods
// and functions specific to Ki types.
var KiMethodsTmpl = template.Must(template.New("KiMethods").
	Funcs(template.FuncMap{
		"HasEmbedDirective": HasEmbedDirective,
		"HasNoNewDirective": HasNoNewDirective,
		"KiPkg":             KiPkg,
	}).Parse(
	`
	{{if not (HasNoNewDirective .)}}
	// New{{.Name}} adds a new [{{.Name}}] with the given name
	// to the given parent. If the name is unspecified, it defaults
	// to the ID (kebab-case) name of the type, plus the
	// [{{KiPkg .}}Ki.NumLifetimeChildren] of the given parent.
	func New{{.Name}}(par {{KiPkg .}}Ki, name ...string) *{{.Name}} {
		return par.NewChild({{.Name}}Type, name...).(*{{.Name}})
	}
	{{end}}

	// KiType returns the [*gti.Type] of [{{.Name}}]
	func (t *{{.Name}}) KiType() *gti.Type {
		return {{.Name}}Type
	}

	// New returns a new [*{{.Name}}] value
	func (t *{{.Name}}) New() {{KiPkg .}}Ki {
		return &{{.Name}}{}
	}
	
	{{if HasEmbedDirective .}}
	// {{.Name}}Embedder is an interface that all types that embed {{.Name}} satisfy
	type {{.Name}}Embedder interface {
		As{{.Name}}() *{{.Name}}
	}
	
	// As{{.Name}} returns the given value as a value of type {{.Name}} if the type
	// of the given value embeds {{.Name}}, or nil otherwise
	func As{{.Name}}(k {{KiPkg .}}Ki) *{{.Name}} {
		if k == nil || k.This() == nil {
			return nil
		}
		if t, ok := k.({{.Name}}Embedder); ok {
			return t.As{{.Name}}()
		}
		return nil
	}
	
	// As{{.Name}} satisfies the [{{.Name}}Embedder] interface
	func (t *{{.Name}}) As{{.Name}}() *{{.Name}} {
		return t
	}
	{{end}}
	`,
))

// KiPkg returns the package identifier for the ki package in
// the context of the given type ("" if it is already in the ki
// package, and "ki." otherwise)
func KiPkg(typ *gtigen.Type) string {
	if typ.Pkg == "ki" { // we are already in ki
		return ""
	}
	return "ki."
}

// HasEmbedDirective returns whether the given [gtigen.Type] has a "goki:embedder"
// comment directive. This function is used in [KiMethodsTmpl].
func HasEmbedDirective(typ *gtigen.Type) bool {
	return slices.ContainsFunc(typ.Directives, func(d *gti.Directive) bool {
		return d.Tool == "goki" && d.Directive == "embedder"
	})
}

// HasNoNewDirective returns whether the given [gtigen.Type] has a "goki:no-new"
// comment directive. This function is used in [KiMethodsTmpl].
func HasNoNewDirective(typ *gtigen.Type) bool {
	return slices.ContainsFunc(typ.Directives, func(d *gti.Directive) bool {
		return d.Tool == "goki" && d.Directive == "no-new"
	})
}

// SetterFields returns all of the fields of the given type
// that have a `setter:"+"` struct tag.
func SetterFields(typ *gtigen.Type) []*gti.Field {
	res := []*gti.Field{}
	for _, kv := range typ.Fields.Order {
		f := kv.Val
		s, ok := f.Tag.Lookup("setter")
		hasSetter := ok && s == "+"
		if hasSetter {
			res = append(res, f)
		}
	}
	return res
}

// Generate is the main entry point to code generation
// that does all of the generation according to the
// given config info. It overrides the
// [config.Config.Generate.Gtigen.InterfaceConfigs] info.
//
//gti:add
func Generate(cfg *config.Config) error {
	gtigen.AddDirectives = append(gtigen.AddDirectives, &gti.Directive{Tool: "gi", Directive: "toolbar"})

	cfg.Generate.Gtigen.Templates = []*template.Template{GeneralMethodsTmpl}
	cfg.Generate.Gtigen.InterfaceConfigs = make(map[string]*gtigen.Config)
	if cfg.Generate.AddKiTypes {
		cfg.Generate.Gtigen.InterfaceConfigs["goki.dev/ki/v2.Ki"] = &gtigen.Config{
			AddTypes:  true,
			Instance:  true,
			TypeVar:   true,
			Templates: []*template.Template{KiMethodsTmpl, GeneralMethodsTmpl},
		}
	}
	pkgs, err := ParsePackages(cfg)
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

// ParsePackages parses the package(s) based on the given config info.
func ParsePackages(cfg *config.Config) ([]*packages.Package, error) {
	pcfg := &packages.Config{
		Mode: enumgen.PackageModes() | gtigen.PackageModes(&cfg.Generate.Gtigen), // need superset of both
		// TODO: Need to think about constants in test files. Maybe write type_string_test.go
		// in a separate pass? For later.
		Tests: false,
	}
	return gengo.Load(pcfg, cfg.Generate.Dir)
}
