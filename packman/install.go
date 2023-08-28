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
	"runtime"
	"strings"

	"goki.dev/goki/config"
)

// Install installs the package the config ID
// by looking for it in the list of supported packages.
// If the config ID is a filepath, it calls [InstallLocal] instead.
func Install(c *config.Config) error {
	if c.Install.Package == "." || c.Install.Package == ".." || strings.Contains(c.Install.Package, "/") {
		return InstallLocal(c)
	}
	packages, err := LoadPackages()
	if err != nil {
		return fmt.Errorf("error loading packages: %w", err)
	}
	for _, pkg := range packages {
		if pkg.ID == c.Install.Package {
			return InstallPackage(pkg)
		}
	}
	return fmt.Errorf("error: could not find package %s", c.Install.Package)
}

// InstallPackage installs the given package object.
func InstallPackage(pkg Package) error {
	fmt.Println("Installing", pkg.Name)
	commands, ok := pkg.InstallCommands[runtime.GOOS]
	if !ok {
		return fmt.Errorf("error: the requested package (%s) does not support your operating system (%s)", pkg.Name, runtime.GOOS)
	}
	for _, command := range commands {
		cmd := exec.Command(command.Name, command.Args...)
		b, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("error installing %s: %w", pkg.Name, err)
		}
		fmt.Println(string(b))
	}
	fmt.Println("Successfully installed", pkg.Name)
	return nil
}

// InstallLocal installs a local package from the filesystem
// on the user's device for the config target operating systems.
func InstallLocal(c *config.Config) error {
	for _, os := range c.Install.Target {
		err := config.OSSupported(os)
		if err != nil {
			return fmt.Errorf("install: %w", err)
		}
		if os == "android" || os == "ios" {
			err := installLocalMobile(c, os)
			if err != nil {
				return fmt.Errorf("install: %w", err)
			}
			continue
		}
		if os == "js" {
			// TODO: implement js
			continue
		}
		err = installLocalDesktop(c.Install.Package, os)
		if err != nil {
			return fmt.Errorf("install: %w", err)
		}
	}
	return nil
}

// installLocalDesktop builds and installs an executable for the package at the given path for the given desktop platform.
// installLocalDesktop does not check whether operating systems are valid, so it should be called through Install in almost all cases.
func installLocalDesktop(pkgPath string, osName string) error {
	cmd := exec.Command("go", "install", pkgPath)
	cmd.Env = append(os.Environ(), "GOOS="+osName, "GOARCH="+runtime.GOARCH)
	fmt.Println(CmdString(cmd))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error installing on platform %s/%s: %w, %s", osName, runtime.GOARCH, err, string(output))
	}
	fmt.Println(string(output))
	return nil
}

// buildMobile builds and installs an executable for the package at the given path for the given mobile operating system.
// buildMobile does not check whether operating systems are valid, so it should be called through Install in almost all cases.
func installLocalMobile(c *config.Config, osName string) error {
	if osName == "ios" {
		return errors.New("ios is not yet supported")
	}
	c.Build.Platform = []config.Platform{{osName, runtime.GOARCH}}
	err := Build(c)
	if err != nil {
		return fmt.Errorf("install: %w", err)
	}
	cmd := exec.Command("adb", "install", filepath.Join(BuildPath(c.Install.Package), AppName(c.Install.Package)+".apk"))
	fmt.Println(CmdString(cmd))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error installing on platform %s: %w, %s", osName, err, string(output))
	}
	fmt.Println(string(output))
	return nil
}
