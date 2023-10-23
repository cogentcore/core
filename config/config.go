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
type Config struct { //gti:add

	// the name of the project
	Name string

	// the description of the project
	Desc string

	// the version of the project
	Version string `cmd:"set-version" posarg:"0" def:"v0.0.0"`

	// TODO: add def TypeApp for type once fix SetString

	// the type of the project (app/library)
	Type Types

	// the configuration options for the build, install, and run commands
	Build Build `cmd:"build,install,run" view:"add-fields"`

	// the configuration options for the setup command
	Setup Setup `cmd:"setup" view:"add-fields"`

	// the configuration options for the log command
	Log Log `cmd:"log" view:"add-fields"`

	// the configuration options for the release command
	Release Release `cmd:"release" view:"add-fields"`

	// the configuration options for the generate command
	Generate Generate `cmd:"generate" view:"add-fields"`
}

type Build struct { //gti:add

	// the path of the package to build
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

	// the minimal version of the iOS SDK to compile against
	IOSVersion string `def:"13.0"`

	// the minimum supported Android SDK (uses-sdk/android:minSdkVersion in AndroidManifest.xml)
	AndroidMinSDK int `def:"23" min:"23"`

	// the target Android SDK version (uses-sdk/android:targetSdkVersion in AndroidManifest.xml)
	AndroidTargetSDK int `def:"29"`
}

type Setup struct { //gti:add

	// the platform to set things up for
	Platform Platform `posarg:"0"`
}

type Log struct { //gti:add

	// the target platform to view the logs for (ios or android)
	Target string `def:"android"`

	// whether to keep the previous log messages or clear them
	Keep bool `def:"false"`

	// messages not generated from your app equal to or above this log level will be shown
	All string `def:"F"`
}

type Release struct { //gti:add

	// the Go file to store version information in
	VersionFile string `def:"version.go"`

	// the Go package in which the version file will be stored
	Package string `def:"main"`
}

type Generate struct { //gti:add

	// the enum generation configuration options passed to enumgen
	Enumgen enumgen.Config

	// the generation configuration options passed to gtigen
	Gtigen gtigen.Config

	// the source directory to run generate on (can be multiple through ./...)
	Dir string `def:"." posarg:"0" required:"-" nest:"-"`

	// the output file location relative to the package on which generate is being called
	Output string `def:"gokigen.go"`
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
