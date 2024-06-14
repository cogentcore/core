// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

/* TODO(config)
import (
	"testing"

	"cogentcore.org/core/base/option"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
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
		{"slider", 0.4, `display:"slider"`},
		{"enum", ButtonElevated, ""},
		{"bitflag", WidgetFlags(0), ""},
		{"type", ButtonType, ""},
		{"byte-slice", []byte("hello"), ""},
		{"rune-slice", []rune("hello"), ""},
		{"nil", (*int)(nil), ""},
		{"icon", icons.Add, ""},
		{"icon-show-name", icons.Add, `display:"show-name"`},
		{"font", AppearanceSettings.Font, ""},
		{"file", Filename("README.md"), ""},
		{"func", SettingsWindow, ""},
		{"option", option.New("an option"), ""},
		{"colormap", ColorMapName("ColdHot"), ""},
		{"color", colors.Orange, ""},
		{"keychord", key.CodeReturnEnter, ""},
		{"keymap", keymap.AvailableMaps[0], ""},
		{"tree", NewButton(NewFrame()), ""},

		{"map", map[string]int{"Go": 1, "C++": 3, "Python": 5}, ""},
		{"map-inline", map[string]int{"Go": 1, "C++": 3}, ""},
		{"slice", []int{1, 3, 5, 7, 9}, ""},
		{"slice-inline", []int{1, 3, 5}, ""},
		{"struct", &morePerson{Name: "Go", Age: 35, Job: "Programmer", LikesGo: true}, ""},
		{"struct-inline", &person{Name: "Go", Age: 35}, ""},
		{"table", &[]language{{"Go", 10}, {"Python", 5}}, ""},
	}
	for _, value := range values {
		b := NewBody()
		NewValue(b, value.Value, value.Tags)
		b.AssertRender(t, "values/"+value.Name)
	}
}
*/
