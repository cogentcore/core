// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package enumgen provides functions for generating
// enum methods for enum types.
package enumgen

import (
	"fmt"

	"goki.dev/enums/enumgen/config"
)

// Generate generates enum methods using
// the given configuration object. It reads
// all Go files in the config source directory
// and writes the result to the config output file.
// It is a simple entry point to enumgen that does all
// of the steps; for more specific functionality, create
// a new [Generator] with [NewGenerator] and call methods on it.
func Generate(config *config.Config) error {
	g := NewGenerator(config)
	err := g.ParsePackage()
	if err != nil {
		return fmt.Errorf("enumgen: Generate: error parsing package: %w", err)
	}
	err = g.FindEnumTypes()
	if err != nil {
		return fmt.Errorf("enumgen: Generate: error finding enum types: %w", err)
	}
	g.PrintHeader()
	err = g.Generate()
	if err != nil {
		return fmt.Errorf("enumgen: Generate: error generating code: %w", err)
	}
	err = g.Write()
	if err != nil {
		return fmt.Errorf("enumgen: Generate: error writing code: %w", err)
	}
	return nil
}
