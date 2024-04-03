// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/glop/option"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keyfun"
	"cogentcore.org/core/ki"
)

func TestValues(t *testing.T) {
	type test struct {
		Name  string
		Value any
		Tags  string
	}
	values := []test{
		{"ki", gi.NewButton(ki.NewRoot[*gi.Frame]("frame")), ""},
		{"bool", true, ""},
		{"int", 3, ""},
		{"float", 6.7, ""},
		{"slider", 0.4, `view:"slider"`},
		{"enum", gi.ButtonElevated, ""},
		{"bitflag", gi.WidgetFlags(0), ""},
		{"type", gi.ButtonType, ""},
		{"byte-slice", []byte("hello"), ""},
		{"rune-slice", []rune("hello"), ""},
		{"nil", (*int)(nil), ""},
		{"icon", icons.Add, ""},
		{"icon-show-name", icons.Add, `view:"show-name"`},
		{"font", gi.AppearanceSettings.FontFamily, ""},
		{"file", gi.Filename("README.md"), ""},
		{"func", SettingsWindow, ""},
		{"option", option.New("an option"), ""},
		{"colormap", ColorMapName("ColdHot"), ""},
		{"color", colors.Orange, ""},
		{"keychord", key.CodeReturnEnter, ""},
		{"keymap", keyfun.AvailableMaps[0], ""},
	}
	for _, value := range values {
		b := gi.NewBody()
		NewValue(b, value.Value, value.Tags)
		b.AssertRender(t, "values/"+value.Name)
	}
}
