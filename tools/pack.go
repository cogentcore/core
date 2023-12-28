// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tools

import (
	"os"
	"path/filepath"

	"goki.dev/goki/config"
	"goki.dev/goki/packman"
	"goki.dev/xe"
)

// Pack builds and packages the app for the target platform.
// For android, ios, and js, it is equivalent to build.
func Pack(c *config.Config) error { //gti:add
	err := packman.Build(c)
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Join(".goki", "bin", "pack"), 0777)
	if err != nil {
		return err
	}
	for _, platform := range c.Build.Target {
		switch platform.OS {
		case "android", "ios", "js": // build already packages
			continue
		case "darwin":
			err := PackDarwin(c)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// PackDarwin packages the app for macOS.
func PackDarwin(c *config.Config) error {
	apath := filepath.Join(".goki", "bin", "pack", c.Name+".app")
	cpath := filepath.Join(apath, "Contents")
	mpath := filepath.Join(cpath, "MacOS")
	// rpath := filepath.Join(cpath, "Resources")

	err := os.MkdirAll(mpath, 0777)
	if err != nil {
		return err
	}

	err = xe.Run("cp", filepath.Join(".goki", "bin", "build", c.Name), mpath)
	if err != nil {
		return err
	}
	err = xe.Run("chmod", "+x", mpath)
	if err != nil {
		return err
	}
	err = xe.Run("chmod", "+x", filepath.Join(mpath, c.Name))
	if err != nil {
		return err
	}
	return nil
}
