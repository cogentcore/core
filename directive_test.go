// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package directive

import (
	"reflect"
	"testing"
)

type test struct {
	Dir    *Directive // the expected/input directive
	Source string     // the input source string
	String string     // the expected output string representation (defaults to [test.Source] if unset)
}

var tests = []test{
	{
		Dir: &Directive{
			Tool:      "tool",
			Directive: "directive",
			Args:      []string{"arg0", "-key0=value0", "arg1", "-key1", "value1"},
		},
		Source: "//tool:directive arg0 -key0=value0 arg1 -key1 value1",
	},
	{
		Dir: &Directive{
			Tool:      "enums",
			Directive: "enum",
			Args:      []string{"-trimprefix=Button"},
		},
		Source: "//enums:enum -trimprefix=Button",
	},
	{
		Dir: &Directive{
			Tool:      "enums",
			Directive: "structflag",
			Args:      []string{"-field", "Flag", "NodeFlags"},
		},
		Source: "//enums:structflag -field Flag NodeFlags",
	},
	{
		Dir: &Directive{
			Tool:      "goki",
			Directive: "ki",
			Args:      []string{},
		},
		Source: "//goki:ki",
	},
	{
		Dir: &Directive{
			Tool:      "goki",
			Directive: "ki",
			Args:      []string{"-noNew"},
		},
		Source: "//goki:ki -noNew",
	},
	{
		Dir: &Directive{
			Tool:      "goki",
			Directive: "ki",
			Args:      []string{"-embeds=false"},
		},
		Source: "goki:ki -embeds=false",
		String: "//goki:ki -embeds=false",
	},
	{
		Dir:    nil,
		Source: "",
		String: "<nil>",
	},
	{
		Dir:    nil,
		Source: "//goki",
		String: "<nil>",
	},
}

func TestParse(t *testing.T) {
	for _, test := range tests {
		have, err := Parse(test.Source)
		if err != nil {
			t.Errorf("error parsing directive %q: %v", test.Source, err)
		}
		if !reflect.DeepEqual(have, test.Dir) {
			t.Errorf("expected directive for \n%q \n\tto be \n%#v \n\tbut got \n%#v \n\tinstead", test.Source, test.Dir, have)
		}
	}
}

func TestString(t *testing.T) {
	for _, test := range tests {
		if test.String == "" {
			test.String = test.Source
		}
		str := test.Dir.String()
		if str != test.String {
			t.Errorf("expected formatted string for \n%#v \n\tto be\n%q \n\tbut got \n%q \n\tinstead", test.Dir, test.String, str)
		}
	}
}
