// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package generate provides the generation
// of useful methods, variables, and constants
// for Cogent Core code.
package generate

//go:generate core generate

import (
	"fmt"
	"slices"
	"text/template"

	"cogentcore.org/core/cmd/core/config"
	"cogentcore.org/core/enums/enumgen"
	"cogentcore.org/core/gengo"
	"cogentcore.org/core/gti"
	"cogentcore.org/core/gti/gtigen"
	"cogentcore.org/core/ordmap"
	"golang.org/x/tools/go/packages"
)

// TreeMethodsTmpl is a template that contains the methods
// and functions specific to [tree.Node] types.
var TreeMethodsTmpl = template.Must(template.New("TreeMethods").
	Funcs(template.FuncMap{
		"HasEmbedDirective": HasEmbedDirective,
		"HasNoNewDirective": HasNoNewDirective,
		"DocToComment":      gtigen.DocToComment,
		"TreePkg":           TreePkg,
	}).Parse(
	`
	{{if not (HasNoNewDirective .)}}
	// New{{.LocalName}} adds a new [{{.LocalName}}] with the given name to the given parent:
	{{DocToComment .Doc}}
	func New{{.LocalName}}(parent {{TreePkg .}}Node, name ...string) *{{.LocalName}} {
		return parent.NewChild({{.LocalName}}Type, name...).(*{{.LocalName}})
	}
	{{end}}

	// KiType returns the [*gti.Type] of [{{.LocalName}}]
	func (t *{{.LocalName}}) KiType() *gti.Type { return {{.LocalName}}Type }

	// New returns a new [*{{.LocalName}}] value
	func (t *{{.LocalName}}) New() {{TreePkg .}}Node { return &{{.LocalName}}{} }
	
	{{if HasEmbedDirective .}}
	// {{.LocalName}}Embedder is an interface that all types that embed {{.LocalName}} satisfy
	type {{.LocalName}}Embedder interface {
		As{{.LocalName}}() *{{.LocalName}}
	}
	
	// As{{.LocalName}} returns the given value as a value of type {{.LocalName}} if the type
	// of the given value embeds {{.LocalName}}, or nil otherwise
	func As{{.LocalName}}(k {{TreePkg .}}Node) *{{.LocalName}} {
		if k == nil || k.This() == nil {
			return nil
		}
		if t, ok := k.({{.LocalName}}Embedder); ok {
			return t.As{{.LocalName}}()
		}
		return nil
	}
	
	// As{{.LocalName}} satisfies the [{{.LocalName}}Embedder] interface
	func (t *{{.LocalName}}) As{{.LocalName}}() *{{.LocalName}} { return t }
	{{end}}
	`,
))

// TreePkg returns the package identifier for the tree package in
// the context of the given type ("" if it is already in the tree
// package, and "tree." otherwise)
func TreePkg(typ *gtigen.Type) string {
	if typ.Pkg == "tree" { // we are already in tree
		return ""
	}
	return "tree."
}

// HasEmbedDirective returns whether the given [gtigen.Type] has a "core:embedder"
// comment directive. This function is used in [TreeMethodsTmpl].
func HasEmbedDirective(typ *gtigen.Type) bool {
	return slices.ContainsFunc(typ.Directives, func(d gti.Directive) bool {
		return d.Tool == "core" && d.Directive == "embedder"
	})
}

// HasNoNewDirective returns whether the given [gtigen.Type] has a "core:no-new"
// comment directive. This function is used in [TreeMethodsTmpl].
func HasNoNewDirective(typ *gtigen.Type) bool {
	return slices.ContainsFunc(typ.Directives, func(d gti.Directive) bool {
		return d.Tool == "core" && d.Directive == "no-new"
	})
}

// Generate is the main entry point to code generation
// that does all of the generation according to the
// given config info. It overrides the
// [config.Config.Generate.Gtigen.InterfaceConfigs] info.
func Generate(c *config.Config) error { //gti:add
	c.Generate.Gtigen.InterfaceConfigs = &ordmap.Map[string, *gtigen.Config]{}

	c.Generate.Gtigen.InterfaceConfigs.Add("cogentcore.org/core/tree.Node", &gtigen.Config{
		AddTypes:  true,
		Instance:  true,
		TypeVar:   true,
		Setters:   true,
		Templates: []*template.Template{TreeMethodsTmpl},
	})

	pkgs, err := ParsePackages(c)
	if err != nil {
		return fmt.Errorf("Generate: error parsing package: %w", err)
	}

	err = enumgen.GeneratePkgs(&c.Generate.Enumgen, pkgs)
	if err != nil {
		return fmt.Errorf("error running enumgen: %w", err)
	}
	err = gtigen.GeneratePkgs(&c.Generate.Gtigen, pkgs)
	if err != nil {
		return fmt.Errorf("error running gtigen: %w", err)
	}
	err = Webcore(c)
	if err != nil {
		return fmt.Errorf("error running webcoregen: %w", err)
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
