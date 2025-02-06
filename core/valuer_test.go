// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"reflect"
	"testing"

	"cogentcore.org/core/base/option"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles/states"
	"github.com/stretchr/testify/assert"
)

func TestNewValue(t *testing.T) {
	type test struct {
		name  string
		value any
		tags  reflect.StructTag
	}
	values := []test{
		{"bool", true, ""},
		{"checkbox", true, `display:"checkbox"`},
		{"int", 3, ""},
		{"float", 6.7, ""},
		{"slider", 0.4, `display:"slider"`},
		{"enum", ButtonElevated, ""},
		{"bitflag", states.States(0), ""},
		// {"type", types.For[Button](), ""},
		{"byte-slice", []byte("hello"), ""},
		{"rune-slice", []rune("hello"), ""},
		{"nil", (*int)(nil), ""},
		{"icon", icons.Add, ""},
		// {"font", AppearanceSettings.Font, ""}, // TODO(text):
		{"file", Filename("README.md"), ""},
		{"func", SettingsWindow, ""},
		{"option", option.New("an option"), ""},
		{"colormap", ColorMapName("ColdHot"), ""},
		{"color", colors.Orange, ""},
		{"keychord", key.CodeReturnEnter, ""},
		{"keymap", keymap.AvailableMaps[0], ""},
		{"tree", NewButton(NewFrame()), ""},
		{"tree-nil", (*Button)(nil), ""},

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
		NewValue(value.value, value.tags, b)
		b.AssertRender(t, "valuer/"+value.name)
	}
}

func TestAddValueType(t *testing.T) {
	type myType int
	type myValue struct {
		Text
	}
	AddValueType[myType, myValue]()
	assert.IsType(t, &myValue{}, toValue(myType(0), ""))
}
