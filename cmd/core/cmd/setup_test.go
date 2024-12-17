// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

const expectedlinuxDistroString = `Debian/Ubuntu: sudo apt install golang gcc libgl1-mesa-dev libegl1-mesa-dev mesa-vulkan-drivers xorg-dev
Fedora: sudo dnf install golang golang-misc gcc libX11-devel libXcursor-devel libXrandr-devel libXinerama-devel mesa-libGL-devel libXi-devel libXxf86vm-devel
Arch: sudo pacman -S go xorg-server-devel libxcursor libxrandr libxinerama libxi vulkan-swrast
Solus: sudo eopkg it -c system.devel golang mesalib-devel libxrandr-devel libxcursor-devel libxi-devel libxinerama-devel
openSUSE: sudo zypper install go gcc libXcursor-devel libXrandr-devel Mesa-libGL-devel libXi-devel libXinerama-devel libXxf86vm-devel
Void: sudo xbps-install -S go base-devel xorg-server-devel libXrandr-devel libXcursor-devel libXinerama-devel
Alpine: sudo apk add go gcc libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev linux-headers mesa-dev
NixOS: nix-shell -p libGL pkg-config xorg.libX11.dev xorg.libXcursor xorg.libXi xorg.libXinerama xorg.libXrandr xorg.libXxf86vm
`

func TestLinuxDistroString(t *testing.T) {
	str := ""
	for _, ld := range linuxDistros {
		str += ld.String() + "\n"
	}
	assert.Equal(t, expectedlinuxDistroString, str)
	fmt.Println("Current linux distro string:\n\n" + str)
}
