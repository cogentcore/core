// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package enumgen provides functions for generating
// enum methods for enum types.
package enumgen

// Generate generates enum methods using
// the given configuration object. It reads
// all Go files in the config source directory
// and writes the result to the config output file.
func Generate(config Config) error {
	g := NewGenerator(config)
	return g.ParsePackage()
}
