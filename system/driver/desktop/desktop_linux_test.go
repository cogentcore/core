// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build linux && !android

package desktop

import (
	"os"
	"os/user"
	"path/filepath"
	"testing"

	"cogentcore.org/core/system"
	"github.com/godbus/dbus/v5"
	"github.com/stretchr/testify/assert"
)

// displayAvailable returns whether there is an X11 or Wayland display, as
// with xvfb-run on the CI.
func displayAvailable() bool {
	return os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
}

func TestPlatformLinux(t *testing.T) {
	assert.Equal(t, system.Linux, TheApp.Platform())
	usr, err := user.Current()
	if assert.NoError(t, err) {
		assert.Equal(t, filepath.Join(usr.HomeDir, ".config"), TheApp.DataDir())
	}
}

func TestDarkScheme(t *testing.T) {
	assert.True(t, isDarkScheme(1), "prefer dark")
	assert.False(t, isDarkScheme(2), "prefer light")
	t.Setenv("GTK_THEME", "Adwaita:dark")
	assert.True(t, isDarkScheme(0), "no preference: the GTK theme")
	t.Setenv("GTK_THEME", "Adwaita")
	assert.NotPanics(t, func() { isDarkScheme(0) })

	v, ok := variantUint32(dbus.MakeVariant(uint32(1)))
	assert.True(t, ok)
	assert.Equal(t, uint32(1), v)
	v, ok = variantUint32(dbus.MakeVariant(dbus.MakeVariant(uint32(2))))
	assert.True(t, ok, "a nested variant, as some portals send")
	assert.Equal(t, uint32(2), v)
	_, ok = variantUint32(dbus.MakeVariant("dark"))
	assert.False(t, ok)
}
