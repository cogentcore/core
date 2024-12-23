// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package cmd provides utilities for managing
// apps and packages that use the Cogent Core framework.
package cmd

//go:generate core generate

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"cogentcore.org/core/base/exec"
	"cogentcore.org/core/cmd/core/config"
	"cogentcore.org/core/cmd/core/mobile"
	"cogentcore.org/core/cmd/core/web"
)

// Build builds an executable for the package
// at the config path for the config platforms.
func Build(c *config.Config) error { //types:add
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
			return mobile.Build(c)
		}
		if platform.OS == "web" {
			err := os.MkdirAll(c.Build.Output, 0777)
			if err != nil {
				return err
			}
			return web.Build(c)
		}
		err = buildDesktop(c, platform)
		if err != nil {
			return fmt.Errorf("build: %w", err)
		}
	}
	return nil
}

// buildDesktop builds an executable for the config package for the given desktop platform.
func buildDesktop(c *config.Config, platform config.Platform) error {
	xc := exec.Major()
	xc.Env["GOOS"] = platform.OS
	xc.Env["GOARCH"] = platform.Arch

	args := []string{"build"}
	if c.Build.Debug {
		args = append(args, "-tags", "debug")
	}
	if c.Build.Trimpath {
		args = append(args, "-trimpath")
	}
	ldflags := ""
	output := filepath.Base(c.Build.Output)
	if platform.OS == "windows" {
		output += ".exe"
		// see https://stackoverflow.com/questions/23250505/how-do-i-create-an-executable-from-golang-that-doesnt-open-a-console-window-whe
		if c.Build.Windowsgui {
			ldflags += " -H=windowsgui"
		}
	}
	ldflags += " " + config.LinkerFlags(c)

	args = append(args, "-ldflags", ldflags, "-o", filepath.Join(c.Build.Output, output))

	err = xc.Run("go", args...)
	if err != nil {
		return fmt.Errorf("error building for platform %s/%s: %w", platform.OS, platform.Arch, err)
	}

	return nil
}
