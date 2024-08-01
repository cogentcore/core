// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package config contains the configuration
// structs for the Cogent Core tool.
package config

//go:generate core generate

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/exec"
	"cogentcore.org/core/base/iox/tomlx"
	"cogentcore.org/core/base/strcase"
	"cogentcore.org/core/enums/enumgen"
	"cogentcore.org/core/types/typegen"
)

// Config is the main config struct that contains all of the configuration
// options for the Cogent Core command line tool.
type Config struct { //types:add

	// Name is the user-friendly name of the project.
	// The default is based on the current directory name.
	Name string

	// NamePrefix is the prefix to add to the default name of the project
	// and any projects nested below it. A separating space is automatically included.
	NamePrefix string

	// ID is the bundle / package ID to use for the project
	// (required for building for mobile platforms and packaging
	// for desktop platforms). It is typically in the format com.org.app
	// (eg: com.cogent.mail). It defaults to com.parentDirectory.currentDirectory.
	ID string

	// About is the about information for the project, which can be viewed via
	// the "About" button in the app bar. It is also used when packaging the app.
	About string

	// the version of the project to release
	Version string `cmd:"release" posarg:"0" save:"-"`

	// Pages, if specified, indicates that the app has core
	// pages located at this directory. If so, markdown code blocks with
	// language Go (must be uppercase, as that indicates that is an
	// "exported" example) will be collected and stored at pagegen.go, and
	// a directory tree will be made for all of the pages when building
	// for platform web. This defaults to "content" when building an app
	// for platform web that imports pages.
	Pages string

	// the configuration options for the build, install, run, and pack commands
	Build Build `cmd:"build,install,run,pack"`

	// the configuration information for the pack command
	Pack Pack `cmd:"pack"`

	// the configuration information for web
	Web Web `cmd:"build,install,run,pack"`

	// the configuration options for the log command
	Log Log `cmd:"log"`

	// the configuration options for the generate command
	Generate Generate `cmd:"generate"`
}

type Build struct { //types:add

	// the target platforms to build executables for
	Target []Platform `flag:"t,target" posarg:"0" required:"-" save:"-"`

	// Dir is the directory to build the app from.
	// It defaults to the current directory.
	Dir string

	// Output is the directory to output the built app to.
	// It defaults to the current directory for desktop executables
	// and "bin/{platform}" for all other platforms and command "pack".
	Output string `flag:"o,output"`

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

type Pack struct { //types:add

	// whether to build a .dmg file on macOS in addition to a .app file.
	// This is automatically disabled for the install command.
	DMG bool `default:"true"`
}

type Log struct { //types:add

	// the target platform to view the logs for (ios or android)
	Target string `default:"android"`

	// whether to keep the previous log messages or clear them
	Keep bool `default:"false"`

	// messages not generated from your app equal to or above this log level will be shown
	All string `default:"F"`
}

type Generate struct { //types:add

	// the enum generation configuration options passed to enumgen
	Enumgen enumgen.Config

	// the generation configuration options passed to typegen
	Typegen typegen.Config

	// the source directory to run generate on (can be multiple through ./...)
	Dir string `default:"." posarg:"0" required:"-" nest:"-"`

	// Icons, if specified, indicates to generate an icongen.go file with
	// icon variables for the icon svg files in the specified folder.
	Icons string
}

func (c *Config) OnConfig(cmd string) error {
	// if we have no target, we assume it is our current platform,
	// unless we are in init, in which case we do not want to set
	// the config file to be specific to our platform
	if len(c.Build.Target) == 0 && cmd != "init" {
		c.Build.Target = []Platform{{OS: runtime.GOOS, Arch: runtime.GOARCH}}
	}
	if c.Build.Output == "" && len(c.Build.Target) > 0 {
		t := c.Build.Target[0]
		if cmd == "pack" || t.OS == "web" || t.OS == "android" || t.OS == "ios" {
			c.Build.Output = filepath.Join("bin", t.OS)
		}
	}
	// we must make the output dir absolute before changing the current directory
	out, err := filepath.Abs(c.Build.Output)
	if err != nil {
		return err
	}
	c.Build.Output = out
	if c.Build.Dir != "" {
		err := os.Chdir(c.Build.Dir)
		if err != nil {
			return err
		}
		// re-read the config file from the new location if it exists
		err = tomlx.Open(c, "core.toml")
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	// we must do auto-naming after we apply any directory change above
	if c.Name == "" || c.ID == "" {
		cdir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("error finding current directory: %w", err)
		}
		base := filepath.Base(cdir)
		if c.Name == "" {
			c.Name = strcase.ToTitle(base)
			if c.NamePrefix != "" {
				c.Name = c.NamePrefix + " " + c.Name
			}
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
	return nil
}

// LinkerFlags returns the ld linker flags that specify the app and core version,
// the app about information, and the app icon.
func LinkerFlags(c *Config) string {
	res := ""

	if c.About != "" {
		res += "-X 'cogentcore.org/core/core.AppAbout=" + strings.ReplaceAll(c.About, "'", `\'`) + "' "
	}

	b, err := os.ReadFile("icon.svg")
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			errors.Log(err)
		}
	} else {
		res += "-X 'cogentcore.org/core/core.AppIcon=" + strings.ReplaceAll(string(b), "'", `\'`) + "' "
	}

	av, err := exec.Silent().Output("git", "describe", "--tags")
	if err == nil {
		res += "-X cogentcore.org/core/system.AppVersion=" + av + " "
	}

	// workspaces can interfere with getting the right version
	cv, err := exec.Silent().SetEnv("GOWORK", "off").Output("go", "list", "-m", "-f", "{{.Version}}", "cogentcore.org/core")
	if err == nil {
		// we must be in core itself if it is blank
		if cv == "" {
			cv = av
		}
		res += "-X cogentcore.org/core/system.CoreVersion=" + cv
	}
	return res
}
