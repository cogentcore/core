// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command pelletier writes the settings files in testdata/settings for
// the settings compatibility tests, using pelletier/go-toml/v2 with the
// same encoder settings as the previous version of tomlx, from the
// non-default fields of the settings as [core.SaveSettings] does.
// Run it from this directory with:
//
//	go run . ..
package main

import (
	"bytes"
	"fmt"
	"image/color"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"github.com/pelletier/go-toml/v2"
)

func main() {
	dir := os.Args[1]

	as := &core.AppearanceSettingsData{}
	defaults(as)
	as.Theme = core.ThemeDark
	as.Color = color.RGBA{200, 50, 100, 255}
	as.Zoom = 120
	as.Screens = map[string]core.ScreenSettings{"Built-in Retina Display": {Zoom: 150}, "DELL U2720Q": {Zoom: 90}}
	as.Highlighting = "monokai"
	as.Text.SansSerif = "Arial"
	as.MenuMax = 40
	as.InlineLengths.Map = 3
	save(as, filepath.Join(dir, "appearance.toml"))

	ss := &core.SystemSettingsData{}
	defaults(ss)
	ss.KeyMap = "MacEmacs"
	ss.KeyMaps.Valid = true
	ss.KeyMaps.Value = keymap.StandardMaps[:2]
	ss.Editor.TabSize = 8
	ss.Clock24 = true
	ss.User = core.User{User: user.User{Uid: "501", Gid: "20", Username: "test", Name: "Test User", HomeDir: "/Users/test"}, Email: "test@example.com"}
	ss.FavPaths = append(ss.FavPaths, ss.FavPaths[0])
	ss.FavPaths[len(ss.FavPaths)-1].Icon = icons.Folder
	ss.FavPaths[len(ss.FavPaths)-1].Name = "work"
	ss.FavPaths[len(ss.FavPaths)-1].Path = "~/work"
	save(ss, filepath.Join(dir, "system.toml"))

	ts := &core.TimingSettingsData{}
	defaults(ts)
	ts.SnackbarTimeout = 10 * time.Second
	ts.DoubleClickInterval = 400 * time.Millisecond
	ts.CursorBlinkTime = 0
	ts.LongHoverTime = 1500 * time.Millisecond
	ts.ScrollWheelSpeed = 2.5
	save(ts, filepath.Join(dir, "timing.toml"))
}

// defaults sets the default values of the given settings, as core does.
func defaults(se core.Settings) {
	reflectx.SetFromDefaultTags(se)
	se.Defaults()
}

// save writes the non-default fields of the given settings to the given
// file as the previous version of core.SaveSettings did.
func save(se core.Settings, filename string) {
	var b bytes.Buffer
	err := toml.NewEncoder(&b).SetIndentTables(true).SetArraysMultiline(true).Encode(reflectx.NonDefaultFields(se))
	if err == nil {
		err = os.WriteFile(filename, b.Bytes(), 0666)
	}
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
