// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"cogentcore.org/core/base/exec"
	"cogentcore.org/core/base/logx"
	"cogentcore.org/core/cmd/core/config"
	"github.com/mitchellh/go-homedir"
)

// Setup installs platform-specific dependencies for the current platform.
// It only needs to be called once per system.
func Setup(c *config.Config) error { //types:add
	vc := exec.Verbose().SetBuffer(false)
	switch runtime.GOOS {
	case "darwin":
		p, err := exec.Output("xcode-select", "-p")
		if err != nil || p == "" {
			err := vc.Run("xcode-select", "--install")
			if err != nil {
				return err
			}
		} else {
			logx.PrintlnWarn("xcode tools already installed")
		}
		return nil
	case "linux":
		for _, ld := range linuxDistros {
			_, err := exec.LookPath(ld.tool)
			if err != nil {
				continue // package manager not found
			}
			cmd, args := ld.cmd()
			err = vc.Run(cmd, args...)
			if err != nil {
				return err // package installation failed
			}
			return nil // success
		}
		return errors.New("unknown Linux distro; please file a bug report at https://github.com/cogentcore/core/issues")
	case "windows":
		// We must be in the home directory to avoid permission issues with file downloading.
		hd, err := homedir.Dir()
		if err != nil {
			return err
		}
		err = os.Chdir(hd)
		if err != nil {
			return err
		}
		if _, err := exec.LookPath("gcc"); err != nil {
			err := vc.Run("curl", "-OL", "https://github.com/skeeto/w64devkit/releases/download/v2.0.0/w64devkit-x64-2.0.0.exe")
			if err != nil {
				return err
			}
			path, err := filepath.Abs("w64devkit-x64-2.0.0.exe")
			if err != nil {
				return err
			}
			err = vc.Run(path, "x", "-oC:", "-aoa")
			if err != nil {
				return err
			}
			err = windowsRegistryAddPath(`C:\w64devkit\bin`)
			if err != nil {
				return err
			}
		} else {
			logx.PrintlnWarn("gcc already installed")
		}
		if _, err := exec.LookPath("git"); err != nil {
			err := vc.Run("curl", "-OL", "https://github.com/git-for-windows/git/releases/download/v2.45.2.windows.1/Git-2.45.2-64-bit.exe")
			if err != nil {
				return err
			}
			path, err := filepath.Abs("Git-2.45.2-64-bit.exe")
			if err != nil {
				return err
			}
			err = vc.Run(path)
			if err != nil {
				return err
			}
		} else {
			logx.PrintlnWarn("git already installed")
		}
		return nil
	}
	return fmt.Errorf("platform %q not supported for core setup", runtime.GOOS)
}

// linuxDistro represents the data needed to install dependencies for a specific Linux
// distribution family with the same installation steps.
type linuxDistro struct {

	// name contains the user-friendly name(s) of the Linux distribution(s).
	name string

	// sudo is whether the package manager requires sudo.
	sudo bool

	// tool is the name of the package manager used for installation.
	tool string

	// command contains the subcommand(s) in the package manager used to install packages.
	command []string

	// packages are the packages that need to be installed.
	packages []string
}

// cmd returns the command and arguments to install the packages for the Linux distribution.
func (ld *linuxDistro) cmd() (cmd string, args []string) {
	if ld.sudo {
		cmd = "sudo"
		args = append(args, ld.tool)
	} else {
		cmd = ld.tool
	}
	args = append(args, ld.command...)
	args = append(args, ld.packages...)
	return
}

func (ld *linuxDistro) String() string {
	cmd, args := ld.cmd()
	return fmt.Sprintf("%-15s %s %s", ld.name+":", cmd, strings.Join(args, " "))
}

// linuxDistros contains the supported Linux distributions,
// based on https://docs.fyne.io/started.
var linuxDistros = []*linuxDistro{
	{name: "Debian/Ubuntu", sudo: true, tool: "apt", command: []string{"install"}, packages: []string{
		"golang", "gcc", "libgl1-mesa-dev", "libegl1-mesa-dev", "mesa-vulkan-drivers", "xorg-dev",
	}},
	{name: "Fedora", sudo: true, tool: "dnf", command: []string{"install"}, packages: []string{
		"golang", "golang-misc", "gcc", "libX11-devel", "libXcursor-devel", "libXrandr-devel", "libXinerama-devel", "mesa-libGL-devel", "libXi-devel", "libXxf86vm-devel", "mesa-vulkan-drivers",
	}},
	{name: "Arch", sudo: true, tool: "pacman", command: []string{"-S"}, packages: []string{
		"go", "xorg-server-devel", "libxcursor", "libxrandr", "libxinerama", "libxi", "vulkan-swrast",
	}},
	{name: "Solus", sudo: true, tool: "eopkg", command: []string{"it", "-c"}, packages: []string{
		"system.devel", "golang", "mesalib-devel", "libxrandr-devel", "libxcursor-devel", "libxi-devel", "libxinerama-devel", "vulkan",
	}},
	{name: "openSUSE", sudo: true, tool: "zypper", command: []string{"install"}, packages: []string{
		"go", "gcc", "libXcursor-devel", "libXrandr-devel", "Mesa-libGL-devel", "libXi-devel", "libXinerama-devel", "libXxf86vm-devel", "libvulkan1",
	}},
	{name: "Void", sudo: true, tool: "xbps-install", command: []string{"-S"}, packages: []string{
		"go", "base-devel", "xorg-server-devel", "libXrandr-devel", "libXcursor-devel", "libXinerama-devel", "vulkan-loader",
	}},
	{name: "Alpine", sudo: true, tool: "apk", command: []string{"add"}, packages: []string{
		"go", "gcc", "libxcursor-dev", "libxrandr-dev", "libxinerama-dev", "libxi-dev", "linux-headers", "mesa-dev", "vulkan-loader",
	}},
	{name: "NixOS", sudo: false, tool: "nix-shell", command: []string{"-p"}, packages: []string{
		"libGL", "pkg-config", "xorg.libX11.dev", "xorg.libXcursor", "xorg.libXi", "xorg.libXinerama", "xorg.libXrandr", "xorg.libXxf86vm", "mesa.drivers", "vulkan-loader",
	}},
}
