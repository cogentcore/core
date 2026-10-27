// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build linux

package desktop

import (
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/godbus/dbus/v5"
)

// TODO(kai): implement SetTitleBarIsDark on linux

const (
	portalDest      = "org.freedesktop.portal.Desktop"
	portalPath      = "/org/freedesktop/portal/desktop"
	portalSettings  = "org.freedesktop.portal.Settings"
	appearanceNS    = "org.freedesktop.appearance"
	colorSchemeName = "color-scheme"
)

// watchTheme sets the initial system color theme and starts watching it for
// changes, using the color-scheme setting of the XDG desktop portal, which
// is supported by GNOME, KDE, and other desktops. If the portal does not
// have a preference, it falls back on whether the GTK theme name has "dark"
// in it, without watching for changes.
func (a *App) watchTheme() {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		slog.Debug("system color theme: no session bus: " + err.Error())
		a.dark.Store(isDarkGTK())
		return
	}
	obj := conn.Object(portalDest, portalPath)
	scheme, err := readColorScheme(obj)
	if err != nil {
		slog.Debug("system color theme: no desktop portal color-scheme: " + err.Error())
		conn.Close()
		a.dark.Store(isDarkGTK())
		return
	}
	a.dark.Store(isDarkScheme(scheme))

	err = conn.AddMatchSignal(
		dbus.WithMatchObjectPath(portalPath),
		dbus.WithMatchInterface(portalSettings),
		dbus.WithMatchMember("SettingChanged"),
		dbus.WithMatchArg(0, appearanceNS),
		dbus.WithMatchArg(1, colorSchemeName),
	)
	if err != nil {
		slog.Error("error watching system color theme: " + err.Error())
		conn.Close()
		return
	}
	signals := make(chan *dbus.Signal, 10)
	conn.Signal(signals)
	go func() {
		for sig := range signals {
			if len(sig.Body) < 3 {
				continue
			}
			ns, _ := sig.Body[0].(string)
			key, _ := sig.Body[1].(string)
			if ns != appearanceNS || key != colorSchemeName {
				continue
			}
			if v, ok := sig.Body[2].(dbus.Variant); ok {
				if scheme, ok := variantUint32(v); ok {
					a.setDark(isDarkScheme(scheme))
				}
			}
		}
	}()
}

// readColorScheme returns the color-scheme setting of the desktop portal:
// 0 for no preference, 1 for dark, and 2 for light.
func readColorScheme(obj dbus.BusObject) (uint32, error) {
	var v dbus.Variant
	err := obj.Call(portalSettings+".ReadOne", 0, appearanceNS, colorSchemeName).Store(&v)
	if err != nil {
		// ReadOne is only in version 2 of the portal; Read wraps
		// the value in another variant.
		err = obj.Call(portalSettings+".Read", 0, appearanceNS, colorSchemeName).Store(&v)
		if err != nil {
			return 0, err
		}
	}
	scheme, _ := variantUint32(v)
	return scheme, nil
}

// variantUint32 returns the uint32 value of the given variant,
// unwrapping any variants nested in it.
func variantUint32(v dbus.Variant) (uint32, bool) {
	for {
		switch x := v.Value().(type) {
		case dbus.Variant:
			v = x
		case uint32:
			return x, true
		default:
			return 0, false
		}
	}
}

// isDarkScheme returns whether the given desktop portal color-scheme
// value is dark, falling back on [isDarkGTK] when there is no preference.
func isDarkScheme(scheme uint32) bool {
	switch scheme {
	case 1:
		return true
	case 2:
		return false
	}
	return isDarkGTK()
}

// isDarkGTK returns whether the GTK theme is dark, based on whether its name
// has "dark" in it, from the GTK_THEME environment variable or the GNOME
// gtk-theme setting.
func isDarkGTK() bool {
	theme := os.Getenv("GTK_THEME")
	if theme == "" {
		out, err := exec.Command("gsettings", "get", "org.gnome.desktop.interface", "gtk-theme").Output()
		if err != nil {
			return false
		}
		theme = string(out)
	}
	return strings.Contains(strings.ToLower(theme), "dark")
}
