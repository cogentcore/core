// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package config contains the configuration
// structs for the GoKi tool.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"goki.dev/enums/enumgen"
	"goki.dev/gti/gtigen"
	"goki.dev/xe"
)

// TODO: make all of the target fields enums

// Config is the main config struct
// that contains all of the configuration
// options for the GoKi tool
//
//gti:add
type Config struct {

	// the name of the project
	Name string

	// the description of the project
	Desc string

	// [def: v0.0.0]
	Version string `cmd:"set-version" posarg:"0" def:"v0.0.0"`

	// TODO: add def TypeApp for type once fix SetString

	// the type of the project (app/library)
	Type Types

	// [view: add-fields]
	Build Build `cmd:"build,install,run" view:"add-fields"`

	// [view: add-fields]
	Setup Setup `cmd:"setup" view:"add-fields"`

	// [view: add-fields]
	Log Log `cmd:"log" view:"add-fields"`

	// [view: add-fields]
	Release Release `cmd:"release" view:"add-fields"`

	// [view: add-fields]
	Generate Generate `cmd:"generate" view:"add-fields"`
}

//gti:add
type Build struct {

	// [def: .]
	Package string `def:"." posarg:"0" required:"-"`

	// the target platforms to build executables for
	Target []Platform `flag:"t,target"`

	// the output file name; if not specified, it depends on the package being built
	Output string `flag:"o,output"`

	// the bundle / package ID to use for the app (required for mobile platforms and N/A otherwise); it is typically in the format com.org.app (eg: com.goki.widgets)
	ID string

	// whether to build/run the app in debug mode; this currently only works on mobile platforms
	Debug bool `flag:"d,debug"`

	// force rebuilding of packages that are already up-to-date
	Rebuild bool `flag:"a,rebuild"`

	// install the generated executable
	Install bool `flag:"i,install"`

	// print the commands but do not run them
	PrintOnly bool `flag:"n,print-only"`

	// print the commands
	Print bool `flag:"x,print"`

	// arguments to pass on each go tool compile invocation
	GCFlags []string

	// arguments to pass on each go tool link invocation
	LDFlags []string

	// a comma-separated list of additional build tags to consider satisfied during the build
	Tags []string

	// remove all file system paths from the resulting executable. Instead of absolute file system paths, the recorded file names will begin either a module path@version (when using modules), or a plain import path (when using the standard library, or GOPATH).
	Trimpath bool

	// print the name of the temporary work directory and do not delete it when exiting
	Work bool

	// [def: 13.0]
	IOSVersion string `def:"13.0"`

	// [def: 23] [min: 23]
	AndroidMinSDK int `def:"23" min:"23"`

	// [def: 29]
	AndroidTargetSDK int `def:"29"`
}

//gti:add
type Setup struct {

	// the platform to set things up for
	Platform Platform `posarg:"0"`
}

//gti:add
type Log struct {

	// [def: android]
	Target string `def:"android"`

	// [def: false]
	Keep bool `def:"false"`

	// [def: F]
	All string `def:"F"`
}

type Release struct {

	// [def: version.go]
	VersionFile string `def:"version.go"`

	// [def: main]
	Package string `def:"main"`
}

//gti:add
type Generate struct {

	// the enum generation configuration options passed to enumgen
	Enumgen enumgen.Config

	// the generation configuration options passed to gtigen
	Gtigen gtigen.Config

	// [def: .]
	Dir string `def:"." posarg:"0" required:"-" nest:"-"`

	// [def: gokigen.go]
	Output string `def:"gokigen.go"`

	// [def: true]
	AddKiTypes bool `def:"true"`
}

func (c *Config) OnConfig(cmd string) error {
	xe.SetMajor(xe.Major().SetPrintOnly(c.Build.PrintOnly))
	if c.Name == "" {
		cdir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("error finding current directory: %w", err)
		}
		base := filepath.Base(cdir)
		c.Name = base
	}
	if c.Build.ID == "" {
		c.Build.ID = "com.org.todo." + c.Name
	}
	return nil
}
