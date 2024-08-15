// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"path/filepath"
	"runtime"

	"cogentcore.org/core/base/exec"
	"cogentcore.org/core/base/logx"
	"cogentcore.org/core/cmd/core/config"
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
		_, err := exec.LookPath("apt-get")
		if err == nil {
			err := vc.Run("sudo", "apt-get", "update")
			if err != nil {
				return err
			}
			return vc.Run("sudo", "apt-get", "install", "-f", "-y", "libgl1-mesa-dev", "libegl1-mesa-dev", "mesa-vulkan-drivers", "xorg-dev")
		}
		_, err = exec.LookPath("dnf")
		if err == nil {
			return vc.Run("sudo", "dnf", "install", "libX11-devel", "libXcursor-devel", "libXrandr-devel", "libXinerama-devel", "mesa-libGL-devel", "libXi-devel", "libXxf86vm-devel")
		}
		return fmt.Errorf("unknown Linux distro (apt-get and dnf not found); file an issue at https://github.com/cogentcore/core/issues")
	case "windows":
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
			WindowsRegistryAddPath("C:\\w64devkit\\bin")
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
