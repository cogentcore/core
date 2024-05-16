// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/styles"
)

func TestNewValue(t *testing.T) {
	type test struct {
		name  string
		value any
	}
	values := []test{
		{"bool", true},
		{"int", 42},
		{"float", 3.14},
		{"string", "hello"},
		{"enum", styles.Center},
	}
	for _, value := range values {
		b := NewBody()
		NewValue(value.value, b)
		b.AssertRender(t, "valuer/"+value.name)
	}
}