// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package typgen provides the generation of type information for
// Go types, methods, and functions.
package typegen

//go:generate go run ../cmd/typegen -output typegen_gen.go

import (
	"fmt"

	"cogentcore.org/core/base/generate"
	"golang.org/x/tools/go/packages"
)

// ParsePackages parses the package(s) located in the configuration source directory.
func ParsePackages(cfg *Config) ([]*packages.Package, error) {
	pcfg := &packages.Config{
		Mode: PackageModes(cfg),
		// TODO: Need to think about types and functions in test files. Maybe write typegen_test.go
		// in a separate pass? For later.
		Tests: false,
	}
	pkgs, err := generate.Load(pcfg, cfg.Dir)
	if err != nil {
		return nil, fmt.Errorf("typegen: Generate: error parsing package: %w", err)
	}
	return pkgs, err
}

// Generate generates typegen type info, using the
// configuration information, loading the packages from the
// configuration source directory, and writing the result
// to the configuration output file.
//
// It is a simple entry point to typegen that does all
// of the steps; for more specific functionality, create
// a new [Generator] with [NewGenerator] and call methods on it.
//
//cli:cmd -root
func Generate(cfg *Config) error { //types:add
	pkgs, err := ParsePackages(cfg)
	if err != nil {
		return err
	}
	return GeneratePkgs(cfg, pkgs)
}

// GeneratePkgs generates enum methods using
// the given configuration object and packages parsed
// from the configuration source directory,
// and writes the result to the config output file.
// It is a simple entry point to typegen that does all
// of the steps; for more specific functionality, create
// a new [Generator] with [NewGenerator] and call methods on it.
func GeneratePkgs(cfg *Config, pkgs []*packages.Package) error {
	g := NewGenerator(cfg, pkgs)
	for _, pkg := range g.Pkgs {
		g.Pkg = pkg
		g.Buf.Reset()
		err := g.Find()
		if err != nil {
			return fmt.Errorf("typegen: Generate: error finding types for package %q: %w", pkg.Name, err)
		}
		g.PrintHeader()
		has, err := g.Generate()
		if !has {
			continue
		}
		if err != nil {
			return fmt.Errorf("typegen: Generate: error generating code for package %q: %w", pkg.Name, err)
		}
		err = g.Write()
		if err != nil {
			return fmt.Errorf("typegen: Generate: error writing code for package %q: %w", pkg.Name, err)
		}
	}
	return nil
}
