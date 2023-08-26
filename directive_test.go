// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package directive

import (
	"reflect"
	"testing"
)

type test struct {
	Dir    Directive // the expected/input directive (also used to get source when used as expected value)
	Has    bool      // whether it is expected to contain a directive when parsing
	String string    // the expected string representation
}

var tests = []test{
	{
		Dir: Directive{
			Source:    "//tool:directive arg0 key0=value0 arg1 key1=value1",
			Tool:      "tool",
			Directive: "directive",
			Args:      []string{"arg0", "arg1"},
			NameValue: map[string]string{"key0": "value0", "key1": "value1"},
		},
		Has:    true,
		String: "//tool:directive arg0 arg1 key0=value0 key1=value1",
	},
	{
		Dir: Directive{
			Source:    "//enums:enum trimprefix=Button",
			Tool:      "enums",
			Directive: "enum",
			Args:      []string{},
			NameValue: map[string]string{"trimprefix": "Button"},
		},
		Has:    true,
		String: "//enums:enum trimprefix=Button",
	},
	{
		Dir: Directive{
			Source:    "//enums:structflag field=Flag NodeFlags",
			Tool:      "enums",
			Directive: "structflag",
			Args:      []string{"NodeFlags"},
			NameValue: map[string]string{"field": "Flag"},
		},
		Has:    true,
		String: "//enums:structflag NodeFlags field=Flag",
	},
	{
		Dir: Directive{
			Source:    "//goki:ki",
			Tool:      "goki",
			Directive: "ki",
			Args:      []string{},
			NameValue: map[string]string{},
		},
		Has:    true,
		String: "//goki:ki",
	},
	{
		Dir: Directive{
			Source:    "//goki:ki noNew",
			Tool:      "goki",
			Directive: "ki",
			Args:      []string{"noNew"},
			NameValue: map[string]string{},
		},
		Has:    true,
		String: "//goki:ki noNew",
	},
	{
		Dir: Directive{
			Source:    "goki:ki embeds=false",
			Tool:      "goki",
			Directive: "ki",
			Args:      []string{},
			NameValue: map[string]string{"embeds": "false"},
		},
		Has:    true,
		String: "//goki:ki embeds=false",
	},
	{
		Dir:    Directive{},
		Has:    false,
		String: "(invalid directive)",
	},
}

func TestParse(t *testing.T) {
	for _, test := range tests {
		have, has := Parse(test.Dir.Source)
		if has != test.Has {
			t.Errorf("expected comment string %q to have a has value of %v, but Parse returned %v", test.Dir.Source, test.Has, has)
		}
		if !reflect.DeepEqual(have, test.Dir) {
			t.Errorf("expected directive for \n%q \n\tto be \n%#v \n\tbut got \n%#v \n\tinstead", test.Dir.Source, test.Dir, have)
		}
	}
}

func TestString(t *testing.T) {
	for _, test := range tests {
		str := test.Dir.String()
		if str != test.String {
			t.Errorf("expected formatted string for \n%#v \n\tto be\n%q \n\tbut got \n%q \n\tinstead", test.Dir, test.String, str)
		}
	}
}
