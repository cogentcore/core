// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package packman

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"goki.dev/goki/config"
	"goki.dev/goki/mobile"
	"goki.dev/xe"
)

// Build builds an executable for the package
// at the config path for the config platforms.
//
//gti:add
func Build(c *config.Config) error {
	if len(c.Build.Target) == 0 {
		return errors.New("build: expected at least 1 platform")
	}
	err := os.MkdirAll(filepath.Join(".", "bin", "build"), 0700)
	if err != nil {
		return fmt.Errorf("build: failed to create bin/build directory: %w", err)
	}
	androidArchs := []string{}
	for _, platform := range c.Build.Target {
		err := config.OSSupported(platform.OS)
		if err != nil {
			return err
		}
		if platform.Arch != "all" {
			err := config.ArchSupported(platform.Arch)
			if err != nil {
				return err
			}
		}
		if platform.OS == "android" {
			androidArchs = append(androidArchs, platform.Arch)
			continue
		}
		if platform.OS == "ios" {
			// TODO: implement ios
			continue
		}
		if platform.OS == "js" {
			// TODO: implement js
			continue
		}
		err = BuildDesktop(c.Build.Package, platform)
		if err != nil {
			return fmt.Errorf("build: %w", err)
		}
	}
	if len(androidArchs) != 0 {
		return mobile.Build(c)
	}
	return nil
}

// BuildDesktop builds an executable for the package at the given path for the given desktop platform.
// BuildDesktop does not check whether platforms are valid, so it should be called through Build in almost all cases.
func BuildDesktop(pkgPath string, platform config.Platform) error {
	m := xe.Major()
	m.Env["GOOS"] = platform.OS
	m.Env["GOARCH"] = platform.Arch
	err := xe.Run(m, "go", "build", "-o", BuildPath(pkgPath), pkgPath)
	if err != nil {
		return fmt.Errorf("error building for platform %s/%s: %w", platform.OS, platform.Arch, err)
	}
	return nil
}
