// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

const expectedlinuxDistroString = `Debian/Ubuntu:  sudo apt install gcc libgl1-mesa-dev libegl1-mesa-dev mesa-vulkan-drivers xorg-dev libwayland-dev libxkbcommon-dev
Fedora:         sudo dnf install gcc libX11-devel libXcursor-devel libXrandr-devel libXinerama-devel mesa-libGL-devel libXi-devel libXxf86vm-devel mesa-vulkan-drivers wayland-devel libxkbcommon-devel
Arch:           sudo pacman -S xorg-server-devel libxcursor libxrandr libxinerama libxi vulkan-swrast wayland libxkbcommon
Solus:          sudo eopkg it -c system.devel mesalib-devel libxrandr-devel libxcursor-devel libxi-devel libxinerama-devel vulkan wayland-devel libxkbcommon-devel
openSUSE:       sudo zypper install gcc libXcursor-devel libXrandr-devel libxkbcommon Mesa-libGL-devel libXi-devel libXinerama-devel libXxf86vm-devel libvulkan1 wayland-devel
Void:           sudo xbps-install -S base-devel xorg-server-devel libXrandr-devel libXcursor-devel libXinerama-devel vulkan-loader wayland-devel libxkbcommon-devel
Alpine:         sudo apk add gcc libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev linux-headers mesa-dev vulkan-loader wayland-dev libxkbcommon-dev
NixOS:          nix-shell -p libGL pkg-config xorg.libX11.dev xorg.libXcursor xorg.libXi xorg.libXinerama xorg.libXrandr xorg.libXxf86vm mesa.drivers vulkan-loader wayland libxkbcommon
`

func TestLinuxDistroString(t *testing.T) {
	str := ""
	for _, ld := range linuxDistros {
		str += ld.String() + "\n"
	}
	assert.Equal(t, expectedlinuxDistroString, str)
	fmt.Println("Current linux distro string:\n\n" + str)
}
