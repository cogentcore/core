// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package packman

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"goki.dev/goki/config"
)

// Build builds an executable for the package
// at the config path for the config platforms.
func Build(c *config.Config) error {
	if len(c.Build.Platform) == 0 {
		return errors.New("build: expected at least 1 platform")
	}
	err := os.MkdirAll(filepath.Join(".", "bin", "build"), 0700)
	if err != nil {
		return fmt.Errorf("build: failed to create bin/build directory: %w", err)
	}
	androidArchs := []string{}
	for _, platform := range c.Build.Platform {
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
		err = buildDesktop(c.Build.Package, platform)
		if err != nil {
			return fmt.Errorf("build: %w", err)
		}
	}
	if len(androidArchs) != 0 {
		return buildMobile(c.Build.Package, "android", androidArchs)
	}
	return nil
}

// buildDesktop builds an executable for the package at the given path for the given desktop platform.
// buildDesktop does not check whether platforms are valid, so it should be called through Build in almost all cases.
func buildDesktop(pkgPath string, platform config.Platform) error {
	cmd := exec.Command("go", "build", "-o", BuildPath(pkgPath), pkgPath)
	cmd.Env = append(os.Environ(), "GOOS="+platform.OS, "GOARCH="+platform.Arch)
	fmt.Println(CmdString(cmd))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error building for platform %s/%s: %w, %s", platform.OS, platform.Arch, err, string(output))
	}
	fmt.Println(string(output))
	return nil
}

// buildMobile builds an executable for the package at the given path for the given mobile operating system and architectures.
// buildMobile does not check whether operating systems and architectures are valid, so it should be called through Build in almost all cases.
func buildMobile(pkgPath string, osName string, archs []string) error {
	target := ""
	for i, arch := range archs {
		target += osName + "/" + arch
		if i != len(archs)-1 {
			target += ","
		}
	}
	cmd := exec.Command("gomobile", "build", "-o", filepath.Join(BuildPath(pkgPath), AppName(pkgPath)+".apk"), "-target", target, pkgPath)
	fmt.Println(CmdString(cmd))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error building for platform %s/%v: %w, %s", osName, archs, err, string(output))
	}
	fmt.Println(string(output))
	return nil
}
