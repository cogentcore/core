// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package config contains the configuration
// structs for the GoKi tool.
package config

import (
	"goki.dev/enums/enumgen"
	"goki.dev/gti/gtigen"
)

// TODO: make all of the target fields enums

// Config is the main config struct
// that contains all of the configuration
// options for the GoKi tool
type Config struct {

	// the name of the project
	Name string `desc:"the name of the project"`

	// the description of the project
	Desc string `desc:"the description of the project"`

	// [def: v0.0.0] the version of the project
	Version string `def:"v0.0.0" desc:"the version of the project"`

	// TODO: add def TypeApp for type once fix SetString

	// the type of the project (app/library)
	Type Types `desc:"the type of the project (app/library)"`

	// [view: add-fields] the configuration options for the build command
	Build Build `cmd:"build" view:"add-fields" desc:"the configuration options for the build command"`

	// [view: add-fields] the configuration options for the colorgen command
	Colorgen Colorgen `cmd:"colorgen" view:"add-fields" desc:"the configuration options for the colorgen command"`

	// [view: add-fields] the configuration options for the install command
	Install Install `cmd:"install" view:"add-fields" desc:"the configuration options for the install command"`

	// [view: add-fields] the configuration options for the log command
	Log Log `cmd:"log" view:"add-fields" desc:"the configuration options for the log command"`

	// [view: add-fields] the configuration options for the release command
	Release Release `cmd:"release" view:"add-fields" desc:"the configuration options for the release command"`

	// [view: add-fields] the configuration options for the generate command
	Generate Generate `cmd:"generate" view:"add-fields" desc:"the configuration options for the generate command"`
}

type Build struct {

	// [def: .] the path of the package to build
	Package string `def:"." desc:"the path of the package to build"`

	// the target platforms to build executables for
	Platform []Platform `desc:"the target platforms to build executables for"`
}

type Colorgen struct {

	// [def: colors.xml] the source file path to generate the color schemes from
	Source string `def:"colors.xml" desc:"the source file path to generate the color schemes from"`

	// [def: colorgen.go] the output file to store the resulting Go file in
	Output string `nest:"+" def:"colorgen.go" desc:"the output file to store the resulting Go file in"`

	// [def: main] the package in which the color schemes will be used
	Package string `def:"main" nest:"+" desc:"the package in which the color schemes will be used"`

	// the comment for the color schemes variable
	Comment string `desc:"the comment for the color schemes variable"`
}

type Install struct {

	// [def: .] the name/path of the package to install
	Package string `def:"." nest:"+" desc:"the name/path of the package to install"`

	// the target platforms to install the executables on, as a list of operating systems (this should include no more than the operating system you are on, android, and ios)
	Target []string `desc:"the target platforms to install the executables on, as a list of operating systems (this should include no more than the operating system you are on, android, and ios)"`
}

type Log struct {

	// [def: android] the target platform to view the logs for (ios or android)
	Target string `def:"android" nest:"+" desc:"the target platform to view the logs for (ios or android)"`

	// [def: false] whether to keep the previous log messages or clear them
	Keep bool `def:"false" desc:"whether to keep the previous log messages or clear them"`

	// [def: F] messages not generated from your app equal to or above this log level will be shown
	All string `def:"F" desc:"messages not generated from your app equal to or above this log level will be shown"`
}

type Release struct {

	// [def: version.go] the Go file to store version information in
	VersionFile string `def:"version.go" desc:"the Go file to store version information in"`

	// [def: main] the Go package in which the version file will be stored
	Package string `def:"main" nest:"+" desc:"the Go package in which the version file will be stored"`
}

type Generate struct {

	// the enum generation configuration options passed to enumgen
	Enumgen enumgen.Config `nest:"+" desc:"the enum generation configuration options passed to enumgen"`

	// the generation configuration options passed to gtigen
	Gtigen gtigen.Config `desc:"the generation configuration options passed to gtigen"`

	// [def: .] the source directory to run generate on (can be multiple through ./...)
	Dir string `def:"." desc:"the source directory to run generate on (can be multiple through ./...)"`

	// [def: gokigen.go] the output file location relative to the package on which generate is being called
	Output string `def:"gokigen.go" desc:"the output file location relative to the package on which generate is being called"`
}
