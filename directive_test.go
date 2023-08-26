// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package directive

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	dirs := []Directive{
		{
			Source:    "//tool:directive arg0 key0=value0 arg1 key1=value1",
			Tool:      "tool",
			Directive: "directive",
			Args:      []string{"arg0", "arg1"},
			Props:     map[string]string{"key0": "value0", "key1": "value1"},
		},
		{
			Source:    "//enums:enum trimprefix=Button",
			Tool:      "enums",
			Directive: "enum",
			Args:      []string{},
			Props:     map[string]string{"trimprefix": "Button"},
		},
		{
			Source:    "//enums:structflag NodeFlags field=Flag",
			Tool:      "enums",
			Directive: "structflag",
			Args:      []string{"NodeFlags"},
			Props:     map[string]string{"field": "Flag"},
		},
		{
			Source:    "//goki:ki",
			Tool:      "goki",
			Directive: "ki",
			Args:      []string{},
			Props:     map[string]string{},
		},
		{
			Source:    "//goki:ki noNew",
			Tool:      "goki",
			Directive: "ki",
			Args:      []string{"noNew"},
			Props:     map[string]string{},
		},
	}
	for _, dir := range dirs {
		have, has := Parse(dir.Source)
		if !has {
			t.Errorf("expected comment string %q to have a directive, but Parse returned false", dir.Source)
		}
		if !reflect.DeepEqual(have, dir) {
			t.Errorf("expected directive for \n%q \n\tto be \n%v \n\tbut got \n%v \n\tinstead", dir.Source, dir, have)
		}
	}
}
