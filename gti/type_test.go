// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gti

import "testing"

func TestFormatDoc(t *testing.T) {
	type test struct {
		doc   string
		name  string
		label string
		want  string
	}
	tests := []test{
		{"UpdateVersion updates the version of the project by one patch version", "UpdateVersion", "Update version", "Update version updates the version of the project by one patch version"},
		{"ToggleSelectionMode toggles the editor between selection mode or not.", "ToggleSelectionMode", "Select element", "Select element toggles the editor between selection mode or not"},
	}
	for _, test := range tests {
		have := FormatDoc(test.doc, test.name, test.label)
		if have != test.want {
			t.Errorf("expected %q but got %q for (%q, %q, %q)", test.want, have, test.doc, test.name, test.label)
		}
	}
}
