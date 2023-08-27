// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package laser

import (
	"testing"
)

// test map type functions
func TestMapType(t *testing.T) {
	var mp map[string]int

	ts := MapValueType(mp).String()
	if ts != "int" {
		t.Errorf("map val type should be int, not: %v\n", ts)
	}

	ts = MapValueType(&mp).String()
	if ts != "int" {
		t.Errorf("map val type should be int, not: %v\n", ts)
	}

	ts = MapKeyType(mp).String()
	if ts != "string" {
		t.Errorf("map key type should be string, not: %v\n", ts)
	}

	ts = MapKeyType(&mp).String()
	if ts != "string" {
		t.Errorf("map key type should be string, not: %v\n", ts)
	}

}
