// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package config contains the configuration
// structs for the Cogent Core tool.
package config

//go:generate core generate

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"

	"cogentcore.org/core/enums/enumgen"
	"cogentcore.org/core/glop/sentence"
	"cogentcore.org/core/gti/gtigen"
)

// TODO: make all of the target fields enums

// Config is the main config struct
// that contains all of the configuration
// options for the Cogent Core tool
type Config struct { //gti:add

	// the user-friendly name of the project
	Name string

	// the bundle / package ID to use of the project (required for building for mobile platforms
	// and packaging for desktop platforms). It is typically in the format com.org.app (eg: com.core.mail)
	ID string

	// the description of the project
	Desc string

	// the version of the project
	Version string `cmd:"set-version" posarg:"0" default:"v0.0.0"`

	// TODO: add def TypeApp for type once fix SetString

	// the type of the project (app/library)
	Type Types

	// the configuration options for the build, install, run, and pack commands
	Build Build `cmd:"build,install,run,pack" view:"add-fields"`

	// the configuration information for the pack command
	Pack Pack `cmd:"pack" view:"add-fields"`

	// the configuration information for web
	Web Web `cmd:"build,install,run,pack" view:"add-fields"`

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

	// the target platforms to build executables for
	Target []Platform `flag:"t,target" posarg:"0" required:"-" save:"-"`

	// whether to build/run the app in debug mode, which sets
	// the "debug" tag when building. On iOS and Android, this
	// also prints the program output.
	Debug bool `flag:"d,debug"`

	// the minimum version of the iOS SDK to compile against
	IOSVersion string `default:"13.0"`

	// the minimum supported Android SDK (uses-sdk/android:minSdkVersion in AndroidManifest.xml)
	AndroidMinSDK int `default:"23" min:"23"`

	// the target Android SDK version (uses-sdk/android:targetSdkVersion in AndroidManifest.xml)
	AndroidTargetSDK int `default:"29"`
}

type Pack struct { //gti:add

	// whether to build a .dmg file on macOS in addition to a .app file.
	// This is automatically disabled for the install command.
	DMG bool `default:"true"`
}

type Setup struct { //gti:add

	// the platform to set things up for
	Platform Platform `posarg:"0"`
}

type Log struct { //gti:add

	// the target platform to view the logs for (ios or android)
	Target string `default:"android"`

	// whether to keep the previous log messages or clear them
	Keep bool `default:"false"`

	// messages not generated from your app equal to or above this log level will be shown
	All string `default:"F"`
}

type Release struct { //gti:add

	// the Go file to store version information in
	VersionFile string `default:"version.go"`

	// the Go package in which the version file will be stored
	Package string `default:"main"`
}

type Generate struct { //gti:add

	// the enum generation configuration options passed to enumgen
	Enumgen enumgen.Config

	// the generation configuration options passed to gtigen
	Gtigen gtigen.Config

	// the source directory to run generate on (can be multiple through ./...)
	Dir string `default:"." posarg:"0" required:"-" nest:"-"`
}

func (c *Config) OnConfig(cmd string) error {
	if c.Name == "" || c.ID == "" {
		cdir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("error finding current directory: %w", err)
		}
		base := filepath.Base(cdir)
		if c.Name == "" {
			c.Name = sentence.Case(base)
		}

		if c.ID == "" {
			dir := filepath.Dir(cdir)
			// if our directory starts with a v and then has only digits, it is a version directory
			// so we go up another directory to get to the actual directory
			if len(dir) > 1 && dir[0] == 'v' && !strings.ContainsFunc(dir[1:], func(r rune) bool {
				return !unicode.IsDigit(r)
			}) {
				dir = filepath.Dir(dir)
			}
			dir = filepath.Base(dir)
			// we ignore anything after any dot in the directory name
			dir, _, _ = strings.Cut(dir, ".")
			// the default ID is "com.dir.base", which is relatively likely
			// to be close to "com.org.app", the intended format
			c.ID = "com." + dir + "." + base
		}
	}
	// if we have no target, we assume it is our current platform,
	// unless we are in init, in which case we do not want to set
	// the config file to be specific to our platform
	if len(c.Build.Target) == 0 && cmd != "init" {
		c.Build.Target = []Platform{{OS: runtime.GOOS, Arch: runtime.GOARCH}}
	}
	return nil
}
