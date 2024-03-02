// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"testing"

	"cogentcore.org/core/gi"
	"cogentcore.org/core/ki"
)

func TestValues(t *testing.T) {
	type test struct {
		Name  string
		Value any
		Tags  string
	}
	values := []test{
		{"ki", gi.NewButton(ki.NewRoot[*gi.Frame]("frame")), ""},
		{"bool", true, ""},
		{"int", 3, ""},
		{"float", 6.7, ""},
		{"slider", 0.4, `view:"slider"`},
	}
	for _, value := range values {
		b := gi.NewBody()
		NewValue(b, value.Value, value.Tags)
		b.AssertRender(t, "values/"+value.Name)
	}
}
