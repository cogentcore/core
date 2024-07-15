// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"path/filepath"
	"runtime"

	"cogentcore.org/core/base/exec"
	"cogentcore.org/core/cmd/core/config"
	"github.com/mitchellh/go-homedir"
)

const vulkanVersion = "1.3.283.0"

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
		}
		err = vc.Run("curl", "-O", "https://sdk.lunarg.com/sdk/download/"+vulkanVersion+"/mac/vulkansdk-macos-"+vulkanVersion+".dmg")
		if err != nil {
			return err
		}
		err = exec.Run("sudo", "hdiutil", "attach", "vulkansdk-macos-"+vulkanVersion+".dmg")
		if err != nil {
			return err
		}
		home, err := homedir.Dir()
		if err != nil {
			return err
		}
		root := filepath.Join(home, "VulkanSDK", vulkanVersion)
		err = vc.Run("sudo", "/Volumes/vulkansdk-macos-"+vulkanVersion+"/InstallVulkan.app/Contents/MacOS/InstallVulkan", "--root", root, "--accept-licenses", "--default-answer", "--confirm-command", "install", "com.lunarg.vulkan.core", "com.lunarg.vulkan.usr", "com.lunarg.vulkan.sdl2", "com.lunarg.vulkan.glm", "com.lunarg.vulkan.volk", "com.lunarg.vulkan.vma")
		if err != nil {
			return err
		}
		return nil
	case "linux":
		_, err := exec.LookPath("apt-get")
		if err == nil {
			err := vc.Run("sudo", "apt-get", "update")
			if err != nil {
				return err
			}
			return vc.Run("sudo", "apt-get", "install", "libgl1-mesa-dev", "xorg-dev")
		}
		_, err = exec.LookPath("dnf")
		if err == nil {
			return vc.Run("sudo", "dnf", "install", "libX11-devel", "libXcursor-devel", "libXrandr-devel", "libXinerama-devel", "mesa-libGL-devel", "libXi-devel", "libXxf86vm-devel")
		}
		return fmt.Errorf("unknown Linux distro (apt-get and dnf not found); file an issue at https://github.com/cogentcore/core/issues")
	case "windows":
		err := vc.Run("curl", "-O", "https://github.com/jmeubank/tdm-gcc/releases/download/v10.3.0-tdm64-2/tdm64-gcc-10.3.0-2.exe")
		if err != nil {
			return err
		}
		err = vc.Run("tdm64-gcc-10.3.0-2.exe")
		if err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("platform %q not supported for core setup", runtime.GOOS)
}
