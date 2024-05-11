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

	"cogentcore.org/core/base/generate"
	"cogentcore.org/core/base/ordmap"
	"cogentcore.org/core/cmd/core/config"
	"cogentcore.org/core/enums/enumgen"
	"cogentcore.org/core/types"
	"cogentcore.org/core/types/typegen"
	"golang.org/x/tools/go/packages"
)

// TreeMethodsTmpl is a template that contains the methods
// and functions specific to [tree.Node] types.
var TreeMethodsTmpl = template.Must(template.New("TreeMethods").
	Funcs(template.FuncMap{
		"HasEmbedDirective": HasEmbedDirective,
		"HasNoNewDirective": HasNoNewDirective,
		"DocToComment":      typegen.DocToComment,
		"TreePkg":           TreePkg,
	}).Parse(
	`
	{{if not (HasNoNewDirective .)}}
	// New{{.LocalName}} adds a new [{{.LocalName}}] to the given optional parent:
	{{DocToComment .Doc}}
	func New{{.LocalName}}(parent ...{{TreePkg .}}Node) *{{.LocalName}} {
		return {{TreePkg .}}New[*{{.LocalName}}](parent...)
	}
	{{end}}

	// NodeType returns the [*types.Type] of [{{.LocalName}}]
	func (t *{{.LocalName}}) NodeType() *types.Type { return {{.LocalName}}Type }

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
func TreePkg(typ *typegen.Type) string {
	if typ.Pkg == "tree" { // we are already in tree
		return ""
	}
	return "tree."
}

// HasEmbedDirective returns whether the given [typegen.Type] has a "core:embedder"
// comment directive. This function is used in [TreeMethodsTmpl].
func HasEmbedDirective(typ *typegen.Type) bool {
	return slices.ContainsFunc(typ.Directives, func(d types.Directive) bool {
		return d.Tool == "core" && d.Directive == "embedder"
	})
}

// HasNoNewDirective returns whether the given [typegen.Type] has a "core:no-new"
// comment directive. This function is used in [TreeMethodsTmpl].
func HasNoNewDirective(typ *typegen.Type) bool {
	return slices.ContainsFunc(typ.Directives, func(d types.Directive) bool {
		return d.Tool == "core" && d.Directive == "no-new"
	})
}

// Generate is the main entry point to code generation
// that does all of the generation according to the
// given config info. It overrides the
// [config.Config.Generate.Typegen.InterfaceConfigs] info.
func Generate(c *config.Config) error { //types:add
	c.Generate.Typegen.InterfaceConfigs = &ordmap.Map[string, *typegen.Config]{}

	c.Generate.Typegen.InterfaceConfigs.Add("cogentcore.org/core/tree.Node", &typegen.Config{
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
	err = typegen.GeneratePkgs(&c.Generate.Typegen, pkgs)
	if err != nil {
		return fmt.Errorf("error running typegen: %w", err)
	}
	err = Pages(c)
	if err != nil {
		return fmt.Errorf("error running pagegen: %w", err)
	}
	return nil
}

// ParsePackages parses the package(s) based on the given config info.
func ParsePackages(cfg *config.Config) ([]*packages.Package, error) {
	pcfg := &packages.Config{
		Mode: enumgen.PackageModes() | typegen.PackageModes(&cfg.Generate.Typegen), // need superset of both
		// TODO: Need to think about constants in test files. Maybe write type_string_test.go
		// in a separate pass? For later.
		Tests: false,
	}
	return generate.Load(pcfg, cfg.Generate.Dir)
}
