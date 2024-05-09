// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"testing"

	"cogentcore.org/core/base/option"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/tree"
)

func TestValues(t *testing.T) {
	type test struct {
		Name  string
		Value any
		Tags  string
	}
	values := []test{
		{"bool", true, ""},
		{"int", 3, ""},
		{"float", 6.7, ""},
		{"slider", 0.4, `view:"slider"`},
		{"enum", core.ButtonElevated, ""},
		{"bitflag", core.WidgetFlags(0), ""},
		{"type", core.ButtonType, ""},
		{"byte-slice", []byte("hello"), ""},
		{"rune-slice", []rune("hello"), ""},
		{"nil", (*int)(nil), ""},
		{"icon", icons.Add, ""},
		{"icon-show-name", icons.Add, `view:"show-name"`},
		{"font", core.AppearanceSettings.Font, ""},
		{"file", core.Filename("README.md"), ""},
		{"func", SettingsWindow, ""},
		{"option", option.New("an option"), ""},
		{"colormap", ColorMapName("ColdHot"), ""},
		{"color", colors.Orange, ""},
		{"keychord", key.CodeReturnEnter, ""},
		{"keymap", keymap.AvailableMaps[0], ""},
		{"tree", core.NewButton(tree.NewRoot[*core.Frame]("frame")), ""},

		{"map", map[string]int{"Go": 1, "C++": 3, "Python": 5}, ""},
		{"map-inline", map[string]int{"Go": 1, "C++": 3}, ""},
	}
	for _, value := range values {
		b := core.NewBody()
		NewValue(b, value.Value, value.Tags)
		b.AssertRender(t, "values/"+value.Name)
	}
}
