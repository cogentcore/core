// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tools

import (
	"fmt"
	"os"
	"path/filepath"

	"goki.dev/goki/config"
	"goki.dev/xe"
)

// Setup does platform-specific setup that ensures that development can be done
// for the config platform, mostly by installing necessary tools.
//
//gti:add
func Setup(c *config.Config) error {
	switch c.Setup.Platform.OS {
	// TODO: support more platforms in setup
	case "ios":
		return SetupIOS(c)
	}
	return nil
}

// SetupIOS is the implementation of [Setup] for iOS.
func SetupIOS(c *config.Config) error {
	tmp, err := os.MkdirTemp("", "goki-setup-ios-vulkan")
	if err != nil {
		return err
	}
	err = xe.Major().SetDir(tmp).Run("git", "clone", "https://github.com/goki/vulkan_mac_deps")
	if err != nil {
		return err
	}
	hdir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("error getting user home directory: %w", err)
	}
	gdir := filepath.Join(hdir, "Library", "goki")
	err = xe.MkdirAll(gdir, 0750)
	if err != nil {
		return err
	}
	err = xe.Run("cp", "-r", filepath.Join(tmp, "vulkan_mac_deps", "sdk", "ios", "MoltenVK.framework"), gdir)
	if err != nil {
		return err
	}
	return xe.RemoveAll(tmp)
}
