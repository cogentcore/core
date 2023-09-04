// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xmls

import "testing"

type testStruct struct {
	A string
	B float32
}

func TestXML(t *testing.T) {
	s := &testStruct{A: "aaa", B: 3.14}
	err := Save(s, "testdata/s.xml")
	if err != nil {
		t.Error(err)
	}
	b, err := WriteBytes(s)
	if err != nil {
		t.Error(err)
	}

	a := &testStruct{}
	err = Open(a, "testdata/s.xml")
	if err != nil {
		t.Error(err)
	}
	if *a != *s {
		t.Errorf("Open failed to read same data as saved: wanted %v != got %v", s, a)
	}

	c := &testStruct{}
	err = ReadBytes(c, b)
	if err != nil {
		t.Error(err)
	}
	if *c != *s {
		t.Errorf("ReadBytes or WriteBytes failed to read same data as saved: wanted %v != got %v", s, c)
	}

	err = SaveIndent(s, "testdata/s_indent.xml")
	if err != nil {
		t.Error(err)
	}
}
