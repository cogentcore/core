// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package goki provides utilities for managing
// apps and packages that use the Goki framework.
package goki

//go:generate goki generate

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"goki.dev/goki/config"
	"goki.dev/goki/mobile"
	"goki.dev/goki/web"
	"goki.dev/xe"
)

// Build builds an executable for the package
// at the config path for the config platforms.
func Build(c *config.Config) error { //gti:add
	if len(c.Build.Target) == 0 {
		return errors.New("build: expected at least 1 platform")
	}
	for _, platform := range c.Build.Target {
		err := config.OSSupported(platform.OS)
		if err != nil {
			return err
		}
		if platform.Arch != "*" {
			err := config.ArchSupported(platform.Arch)
			if err != nil {
				return err
			}
		}
		if platform.OS == "android" || platform.OS == "ios" {
			if platform.Arch == "*" {
				archs := config.ArchsForOS[platform.OS]
				c.Build.Target = make([]config.Platform, len(archs))
				for i, arch := range archs {
					c.Build.Target[i] = config.Platform{OS: platform.OS, Arch: arch}
				}
			}
			return mobile.Build(c)
		}
		if platform.OS == "js" {
			// need to get real output location so that commands work
			if c.Build.Output == "" {
				c.Build.Output = filepath.Join(".goki", "bin", "js", "app.wasm")
			}
			// we must end with a wasm file
			if !strings.Contains(filepath.Base(c.Build.Output), ".wasm") {
				c.Build.Output = filepath.Join(c.Build.Output, "app.wasm")
			}
			err := os.MkdirAll(filepath.Dir(c.Build.Output), 0777)
			if err != nil {
				return err
			}
			return web.Build(c)
		}
		err = BuildDesktop(c, platform)
		if err != nil {
			return fmt.Errorf("build: %w", err)
		}
	}
	return nil
}

// BuildDesktop builds an executable for the config package for the given desktop platform.
// BuildDesktop does not check whether platforms are valid, so it should be called through Build in almost all cases.
func BuildDesktop(c *config.Config, platform config.Platform) error {
	xc := xe.Major()
	xc.Env["GOOS"] = platform.OS
	xc.Env["GOARCH"] = platform.Arch

	// need to get real output location so that install commands work later
	if c.Build.Output == "" {
		c.Build.Output = filepath.Join(".goki", "bin", platform.OS, c.Name)
	}
	err := os.MkdirAll(filepath.Dir(c.Build.Output), 0777)
	if err != nil {
		return err
	}
	tags := []string{"build"}
	if c.Build.Debug {
		tags = append(tags, "-tags", "debug")
	}
	if platform.OS == "windows" {
		c.Build.Output += ".exe"
		// see https://stackoverflow.com/questions/23250505/how-do-i-create-an-executable-from-golang-that-doesnt-open-a-console-window-whe
		// tags = append(tags, "-ldflags", "-H=windowsgui")
	}
	tags = append(tags, "-o", c.Build.Output, c.Build.Package)

	err = xc.Run("go", tags...)
	if err != nil {
		return fmt.Errorf("error building for platform %s/%s: %w", platform.OS, platform.Arch, err)
	}

	return nil
}
