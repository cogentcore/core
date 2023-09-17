// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package enumgen provides functions for generating
// enum methods for enum types.
package enumgen

//go:generate gtigen

import (
	"fmt"

	"goki.dev/gengo"
	"golang.org/x/tools/go/packages"
)

// ParsePackage parses the package(s) located in the configuration source directory.
func ParsePackage(cfg *Config) ([]*packages.Package, error) {
	pcfg := &packages.Config{
		Mode: PackageModes(),
		// TODO: Need to think about constants in test files. Maybe write type_string_test.go
		// in a separate pass? For later.
		Tests: false,
	}
	pkgs, err := gengo.Load(pcfg, cfg.Dir)
	if err != nil {
		return nil, fmt.Errorf("enumgen: Generate: error parsing package: %w", err)
	}
	return pkgs, err
}

// Generate generates enum methods, using the
// configuration information, loading the packages from the
// configuration source directory, and writing the result
// to the configuration output file.
//
// It is a simple entry point to enumgen that does all
// of the steps; for more specific functionality, create
// a new [Generator] with [NewGenerator] and call methods on it.
//
//gti:add
//grease:cmd -root
func Generate(cfg *Config) error {
	pkgs, err := ParsePackage(cfg)
	if err != nil {
		return err
	}
	return GeneratePkgs(cfg, pkgs)
}

// GeneratePkgs generates enum methods using
// the given configuration object and packages parsed
// from the configuration source directory,
// and writes the result to the config output file.
// It is a simple entry point to enumgen that does all
// of the steps; for more specific functionality, create
// a new [Generator] with [NewGenerator] and call methods on it.
func GeneratePkgs(cfg *Config, pkgs []*packages.Package) error {
	g := NewGenerator(cfg, pkgs)
	for _, pkg := range g.Pkgs {
		g.Pkg = pkg
		g.Buf.Reset()
		err := g.FindEnumTypes()
		if err != nil {
			return fmt.Errorf("enumgen: Generate: error finding enum types for package %q: %w", pkg.Name, err)
		}
		g.PrintHeader()
		has, err := g.Generate()
		if !has {
			continue
		}
		if err != nil {
			return fmt.Errorf("enumgen: Generate: error generating code for package %q: %w", pkg.Name, err)
		}
		err = g.Write()
		if err != nil {
			return fmt.Errorf("enumgen: Generate: error writing code for package %q: %w", pkg.Name, err)
		}
	}
	return nil
}
