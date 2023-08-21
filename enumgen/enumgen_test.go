// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package enumgen

import (
	"os"
	"testing"
)

func TestGenerate(t *testing.T) {
	c := Config{}
	c.Defaults()
	c.Dir = "./testdata"
	c.Output = "./testdata/enumgen.go"
	err := Generate(c)
	if err != nil {
		t.Errorf("error while generating: %v", err)
	}
	have, err := os.ReadFile("testdata/enumgen.go")
	if err != nil {
		t.Errorf("error while reading generated file: %v", err)
	}
	want, err := os.ReadFile("testdata/enumgen.golden")
	if err != nil {
		t.Errorf("error while reading golden file: %v", err)
	}
	if string(have) != string(want) {
		t.Errorf("expected generated file and expected file to be the same, but they are not (compare ./testdata/enumgen.go and ./testdata/enumgen.golden to see the difference)")
	}
}
