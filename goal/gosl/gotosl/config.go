// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gotosl

//go:generate core generate -add-types -add-funcs

// Keep these in sync with go/format/format.go.
const (
	tabWidth    = 8
	printerMode = UseSpaces | TabIndent | printerNormalizeNumbers

	// printerNormalizeNumbers means to canonicalize number literal prefixes
	// and exponents while printing. See https://golang.org/doc/go1.13#gosl.
	//
	// This value is defined in go/printer specifically for go/format and cmd/gosl.
	printerNormalizeNumbers = 1 << 30
)

// Config has the configuration info for the gosl system.
type Config struct {

	// Output is the output directory for shader code,
	// relative to where gosl is invoked; must not be an empty string.
	Output string `flag:"out" default:"shaders"`

	// Exclude is a comma-separated list of names of functions to exclude from exporting to WGSL.
	Exclude string `default:"Update,Defaults"`

	// Keep keeps temporary converted versions of the source files, for debugging.
	Keep bool

	//	Debug enables debugging messages while running.
	Debug bool
}

//cli:cmd -root
func Run(cfg *Config) error { //types:add
	st := &State{}
	st.Init(cfg)
	return st.Run()
}
