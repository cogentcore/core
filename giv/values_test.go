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
		Tags  map[string]string
	}
	values := []test{
		{"ki", gi.NewButton(ki.NewRoot[*gi.Frame]("frame")), nil},
		{"bool", true, nil},
		{"int", 3, nil},
		{"float", 6.7, nil},
		{"slider", 0.4, map[string]string{"view": "slider"}},
	}
	for _, value := range values {
		b := gi.NewBody()
		NewValue(b, value.Value).SetTags(value.Tags)
		b.AssertRender(t, "values/"+value.Name)
	}
}
