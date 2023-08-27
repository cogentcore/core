// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package laser

import (
	"testing"
)

type TstStruct struct {
	Field1 string
}

// test slice type functions
func TestSliceType(t *testing.T) {
	var sl []string

	ts := SliceElType(sl).String()
	if ts != "string" {
		t.Errorf("slice el type should be string, not: %v\n", ts)
	}

	ts = SliceElType(&sl).String()
	if ts != "string" {
		t.Errorf("slice el type should be string, not: %v\n", ts)
	}

	var slp []*string

	ts = SliceElType(slp).String()
	if ts != "*string" {
		t.Errorf("slice el type should be *string, not: %v\n", ts)
	}

	ts = SliceElType(&slp).String()
	if ts != "*string" {
		t.Errorf("slice el type should be *string, not: %v\n", ts)
	}

	var slsl [][]string

	ts = SliceElType(slsl).String()
	if ts != "[]string" {
		t.Errorf("slice el type should be []string, not: %v\n", ts)
	}

	ts = SliceElType(&slsl).String()
	if ts != "[]string" {
		t.Errorf("slice el type should be []string, not: %v\n", ts)
	}

	// fmt.Printf("slsl kind: %v\n", SliceElType(slsl).Kind())

}
