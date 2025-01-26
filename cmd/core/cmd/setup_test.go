// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

const expectedlinuxDistroString = `Debian/Ubuntu:  sudo apt install gcc libgl1-mesa-dev libegl1-mesa-dev mesa-vulkan-drivers xorg-dev
Fedora:         sudo dnf install gcc libX11-devel libXcursor-devel libXrandr-devel libXinerama-devel mesa-libGL-devel libXi-devel libXxf86vm-devel mesa-vulkan-drivers
Arch:           sudo pacman -S xorg-server-devel libxcursor libxrandr libxinerama libxi vulkan-swrast
Solus:          sudo eopkg it -c system.devel mesalib-devel libxrandr-devel libxcursor-devel libxi-devel libxinerama-devel vulkan
openSUSE:       sudo zypper install gcc libXcursor-devel libXrandr-devel Mesa-libGL-devel libXi-devel libXinerama-devel libXxf86vm-devel libvulkan1
Void:           sudo xbps-install -S base-devel xorg-server-devel libXrandr-devel libXcursor-devel libXinerama-devel vulkan-loader
Alpine:         sudo apk add gcc libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev linux-headers mesa-dev vulkan-loader
NixOS:          nix-shell -p libGL pkg-config xorg.libX11.dev xorg.libXcursor xorg.libXi xorg.libXinerama xorg.libXrandr xorg.libXxf86vm mesa.drivers vulkan-loader
`

func TestLinuxDistroString(t *testing.T) {
	str := ""
	for _, ld := range linuxDistros {
		str += ld.String() + "\n"
	}
	assert.Equal(t, expectedlinuxDistroString, str)
	fmt.Println("Current linux distro string:\n\n" + str)
}
