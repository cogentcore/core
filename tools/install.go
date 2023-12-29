// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tools

import (
	"fmt"
	"runtime"

	"goki.dev/goki/config"
	"goki.dev/goki/mobile"
	"goki.dev/xe"
)

// Install installs the package on the local system.
// It uses the same config info as build.
func Install(c *config.Config) error { //gti:add
	for i, p := range c.Build.Target {
		err := config.OSSupported(p.OS)
		if err != nil {
			return fmt.Errorf("install: %w", err)
		}
		// if no arch is specified, we can assume it is the current arch,
		// as the user is running it (it could be a different arch when testing
		// on an external mobile device, but it is up to the user to specify
		// that arch in that case)
		if p.Arch == "*" {
			p.Arch = runtime.GOARCH
			c.Build.Target[i] = p
		}
		if p.OS == "android" || p.OS == "ios" {
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
			continue
		}
		if p.OS == "js" {
			// TODO: implement js
			continue
		}
		err = InstallDesktop(c.Build.Package, p.OS)
		if err != nil {
			return fmt.Errorf("install: %w", err)
		}
	}
	return nil
}

// InstallDesktop builds and installs an executable for the package at the given path for the given desktop platform.
// InstallDesktop does not check whether operating systems are valid, so it should be called through Install in almost all cases.
func InstallDesktop(pkgPath string, osName string) error {
	xc := xe.Major()
	xc.Env["GOOS"] = osName
	xc.Env["GOARCH"] = runtime.GOARCH
	err := xc.Run("go", "install", pkgPath)
	if err != nil {
		return fmt.Errorf("error installing on platform %s/%s: %w", osName, runtime.GOARCH, err)
	}
	return nil
}
