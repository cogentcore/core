// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package config contains the configuration
// structs for the GoKi tool.
package config

// The is the singular instance of the
// [Main] configuration object
var The = &Main{}

// Main is the main config struct
// that contains all of the configuration
// options for the GoKi tool
type Main struct {

	// the name of the app/library
	Name string `desc:"the name of the app/library"`

	// the description of the app/library
	Desc string `desc:"the description of the app/library"`

	// the version of the app/library
	Version string `desc:"the version of the app/library"`

	// the configuration options for the build command
	Build Build `desc:"the configuration options for the build command"`

	// the configuration options for the colorgen command
	Colorgen Colorgen `desc:"the configuration options for the colorgen command"`
}

// Build contains the configuration options
// for the build command.
type Build struct {

	// the path of the package to build
	Path string `desc:"the path of the package to build"`

	// the target platforms to build executables for, in os[/arch] format
	Target []string `desc:"the target platforms to build executables for, in os[/arch] format"`
}

// Colorgen contains the configuration options
// for the colorgen command.
type Colorgen struct {

	// the package in which the color schemes will be used
	Package string `desc:"the package in which the color schemes will be used"`

	// the comment for the color schemes variable
	Comment string `desc:"the comment for the color schemes variable"`
}
