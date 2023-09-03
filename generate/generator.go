// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generate

import (
	"bytes"
	"fmt"
	"strings"

	"goki.dev/goki/config"
	"golang.org/x/tools/go/packages"
)

// Generator holds the state of the generator.
// It is primarily used to buffer the output.
type Generator struct {
	Config *config.Config // The configuration information
	Buf    bytes.Buffer   // The accumulated output.
	Pkg    []*Package     // The packages we are scanning.
	Types  []Type         // The Ki types
}

// NewGenerator returns a new generator with the
// given configuration information.
func NewGenerator(config *config.Config) *Generator {
	return &Generator{Config: config}
}

// ParsePackage parses the single package located in the configuration directory.
func (g *Generator) ParsePackage() error {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedTypes | packages.NeedTypesSizes | packages.NeedSyntax | packages.NeedTypesInfo,
		// TODO: Need to think about constants in test files. Maybe write type_string_test.go
		// in a separate pass? For later.
		Tests: false,
	}
	pkgs, err := packages.Load(cfg, g.Config.Generate.Dir)
	if err != nil {
		return err
	}
	if len(pkgs) == 0 {
		return fmt.Errorf("expected at least 1 package, but found 0")
	}
	g.Pkg = []*Package{}
	for _, pkg := range pkgs {
		g.AddPackage(pkg)
	}
	return nil
}

// AddPackage adds a type-checked Package and its syntax files to the generator.
func (g *Generator) AddPackage(pkg *packages.Package) {
	p := &Package{
		Name:  pkg.Name,
		Defs:  pkg.TypesInfo.Defs,
		Files: make([]*File, 0),
	}

	for _, file := range pkg.Syntax {
		// ignore generated code
		isGen := false
		for _, c := range file.Comments {
			if strings.Contains(c.Text(), "; DO NOT EDIT.") {
				isGen = true
				break
			}
		}
		if isGen {
			continue
		}
		// need to use append and 0 initial length
		// because we don't know if it has generated code
		p.Files = append(p.Files, &File{
			File: file,
			Pkg:  p,
		})
	}
	g.Pkg = append(g.Pkg, p)
}

// Printf prints the formatted string to the
// accumulated output in [Generator.Buf]
func (g *Generator) Printf(format string, args ...any) {
	fmt.Fprintf(&g.Buf, format, args...)
}
