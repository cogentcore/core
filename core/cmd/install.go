// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"path/filepath"
	"runtime"

	"cogentcore.org/core/core/config"
	"cogentcore.org/core/core/mobile"
	"cogentcore.org/core/xe"
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
		case "web":
			return fmt.Errorf("can not install on platform web; use build or run instead")
		case "darwin":
			c.Pack.DMG = false
			err := Pack(c)
			if err != nil {
				return err
			}
			return xe.Run("cp", "-a", filepath.Join(".core", "bin", "darwin", c.Name+".app"), "/Applications")
		default:
			return xe.Major().SetEnv("GOOS", p.OS).SetEnv("GOARCH", runtime.GOARCH).Run("go", "install", c.Build.Package)
		}
	}
	return nil
}
