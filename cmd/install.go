// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"cogentcore.org/core/base/exec"
	"cogentcore.org/core/base/logx"
	"cogentcore.org/core/cmd/config"
	"cogentcore.org/core/cmd/generate"
	"cogentcore.org/core/cmd/mobile"
)

// Install installs the package on the local system.
// It uses the same config info as build.
// lsregister is the path of the macOS tool that registers apps
// with Launch Services.
const lsregister = "/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister"

func Install(c *config.Config) error { //types:add
	for i, p := range c.Build.Target {
		err := config.OSSupported(p.OS)
		if err != nil {
			return fmt.Errorf("install: %w", err)
		}
		if p.Arch == "*" {
			if p.OS == "android" || p.OS == "ios" {
				p.Arch = "arm64"
			} else {
				p.Arch = runtime.GOARCH
			}
			c.Build.Target[i] = p
		}

		switch p.OS {
		case "android", "ios":
			err := Build(c)
			if err != nil {
				return fmt.Errorf("error building: %w", err)
			}
			// we only want this target for install
			ot := c.Build.Target
			c.Build.Target = []config.Platform{p}
			err = mobile.Install(c)
			c.Build.Target = ot
			if err != nil {
				return fmt.Errorf("install: %w", err)
			}
		case "wasm", "js":
			return fmt.Errorf("can not install on platform %s; use build or run instead", p.OS)
		case "darwin":
			c.Pack.DMG = false
			err := Pack(c)
			if err != nil {
				return err
			}
			app := filepath.Join("/Applications", c.Name+".app")
			if err := removeExecutable(appExecutable(c, app)); err != nil {
				return err
			}
			err = exec.Run("cp", "-a", filepath.Join(c.Build.Output, c.Name+".app"), "/Applications")
			if err != nil {
				return err
			}
			if c.AppGoSymlink {
				if err := goSymlinkApp(c, app); err != nil {
					return err
				}
			}
			if len(c.Opens) == 0 {
				return nil
			}
			// so that the system knows the types of files that it opens now
			return exec.Run(lsregister, "-f", app)
		default:
			if err := generate.ContentCode(".", &c.ContentCode); err != nil {
				logx.PrintlnWarn("warning:", err)
			}
			return exec.Major().SetEnv("GOOS", p.OS).SetEnv("GOARCH", runtime.GOARCH).Run("go", "install")
		}
	}
	return nil
}

// appExecutable returns the path of the executable of the given macOS app
// of the given config, as packed by [Pack].
func appExecutable(c *config.Config, app string) string {
	return filepath.Join(app, "Contents", "MacOS", c.Name+".app")
}

// removeExecutable removes the given executable of an installed app, if
// it exists, before a new one is copied there: cp writes through a symbolic
// link of AppGoSymlink, and it writes over a file in place, which macOS
// then kills when it starts, with an invalid code signature, as it keeps
// the signature of the file that was there.
func removeExecutable(file string) error {
	if err := os.Remove(file); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

// goSymlinkApp installs the app of the given config with go install, and
// makes the executable of the given macOS app a symbolic link to the one
// that it installs (see [config.Config.AppGoSymlink]).
func goSymlinkApp(c *config.Config, app string) error {
	if err := exec.Major().Run("go", "install"); err != nil {
		return err
	}
	target, err := exec.Output("go", "list", "-f", "{{.Target}}")
	if err != nil {
		return err
	}
	target = strings.TrimSpace(target)
	if target == "" {
		return fmt.Errorf("AppGoSymlink: go install does not install an executable here")
	}
	exe := appExecutable(c, app)
	if err := os.Remove(exe); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Symlink(target, exe)
}
