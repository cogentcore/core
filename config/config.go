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
type Config struct {

	// the name of the project
	Name string `desc:"the name of the project"`

	// the description of the project
	Desc string `desc:"the description of the project"`

	// [def: v0.0.0] the version of the project
	Version string `cmd:"set-version" posarg:"0" def:"v0.0.0" desc:"the version of the project"`

	// TODO: add def TypeApp for type once fix SetString

	// the type of the project (app/library)
	Type Types `desc:"the type of the project (app/library)"`

	// [view: add-fields] the configuration options for the build, install, and run commands
	Build Build `cmd:"build,install,run" view:"add-fields" desc:"the configuration options for the build, install, and run commands"`

	// [view: add-fields] the configuration options for the setup command
	Setup Setup `cmd:"setup" view:"add-fields" desc:"the configuration options for the setup command"`

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
	Target []Platform `flag:"t,target" desc:"the target platforms to build executables for"`

	// the output file name; if not specified, it depends on the package being built
	Output string `flag:"o,output" desc:"the output file name; if not specified, it depends on the package being built"`

	// the bundle / package ID to use for the app (required for mobile platforms and N/A otherwise); it is typically in the format com.org.app (eg: com.goki.widgets)
	ID string `desc:"the bundle / package ID to use for the app (required for mobile platforms and N/A otherwise); it is typically in the format com.org.app (eg: com.goki.widgets)"`

	// whether to build/run the app in debug mode; this currently only works on mobile platforms
	Debug bool `flag:"d,debug" desc:"whether to build/run the app in debug mode; this currently only works on mobile platforms"`

	// force rebuilding of packages that are already up-to-date
	Rebuild bool `flag:"a,rebuild" desc:"force rebuilding of packages that are already up-to-date"`

	// install the generated executable
	Install bool `flag:"i,install" desc:"install the generated executable"`

	// print the commands but do not run them
	PrintOnly bool `flag:"n,print-only" desc:"print the commands but do not run them"`

	// print the commands
	Print bool `flag:"x,print" desc:"print the commands"`

	// arguments to pass on each go tool compile invocation
	GCFlags []string `desc:"arguments to pass on each go tool compile invocation"`

	// arguments to pass on each go tool link invocation
	LDFlags []string `desc:"arguments to pass on each go tool link invocation"`

	// a comma-separated list of additional build tags to consider satisfied during the build
	Tags []string `desc:"a comma-separated list of additional build tags to consider satisfied during the build"`

	// remove all file system paths from the resulting executable. Instead of absolute file system paths, the recorded file names will begin either a module path@version (when using modules), or a plain import path (when using the standard library, or GOPATH).
	Trimpath bool `desc:"remove all file system paths from the resulting executable. Instead of absolute file system paths, the recorded file names will begin either a module path@version (when using modules), or a plain import path (when using the standard library, or GOPATH)."`

	// print the name of the temporary work directory and do not delete it when exiting
	Work bool `desc:"print the name of the temporary work directory and do not delete it when exiting"`

	// [def: 13.0] the minimal version of the iOS SDK to compile against
	IOSVersion string `def:"13.0" desc:"the minimal version of the iOS SDK to compile against"`

	// [def: 23] [min: 23] the minimum supported Android SDK (uses-sdk/android:minSdkVersion in AndroidManifest.xml)
	AndroidMinSDK int `def:"23" min:"23" desc:"the minimum supported Android SDK (uses-sdk/android:minSdkVersion in AndroidManifest.xml)"`

	// [def: 29] the target Android SDK version (uses-sdk/android:targetSdkVersion in AndroidManifest.xml)
	AndroidTargetSDK int `def:"29" desc:"the target Android SDK version (uses-sdk/android:targetSdkVersion in AndroidManifest.xml)"`
}

type Setup struct {

	// the platform to set things up for
	Platform Platform `desc:"the platform to set things up for"`
}

type Log struct {

	// [def: android] the target platform to view the logs for (ios or android)
	Target string `def:"android" desc:"the target platform to view the logs for (ios or android)"`

	// [def: false] whether to keep the previous log messages or clear them
	Keep bool `def:"false" desc:"whether to keep the previous log messages or clear them"`

	// [def: F] messages not generated from your app equal to or above this log level will be shown
	All string `def:"F" desc:"messages not generated from your app equal to or above this log level will be shown"`
}

type Release struct {

	// [def: version.go] the Go file to store version information in
	VersionFile string `def:"version.go" desc:"the Go file to store version information in"`

	// [def: main] the Go package in which the version file will be stored
	Package string `def:"main" desc:"the Go package in which the version file will be stored"`
}

type Generate struct {

	// the enum generation configuration options passed to enumgen
	Enumgen enumgen.Config `desc:"the enum generation configuration options passed to enumgen"`

	// the generation configuration options passed to gtigen
	Gtigen gtigen.Config `desc:"the generation configuration options passed to gtigen"`

	// [def: .] the source directory to run generate on (can be multiple through ./...)
	Dir string `def:"." posarg:"0" required:"-" nest:"-" desc:"the source directory to run generate on (can be multiple through ./...)"`

	// [def: gokigen.go] the output file location relative to the package on which generate is being called
	Output string `def:"gokigen.go" desc:"the output file location relative to the package on which generate is being called"`

	// [def: true] whether to automatically add all types that implement the ki.Ki interface (should be set to false in packages without Ki types)
	AddKiTypes bool `def:"true" desc:"whether to automatically add all types that implement the ki.Ki interface (should be set to false in packages without Ki types)"`
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
