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
		NewValue(value.value, "", b)
		b.AssertRender(t, "valuer/"+value.name)
	}
}

/* TODO(config)
func TestAddValue(t *testing.T) {
	type myType int
	type myValue struct {
		StringValue
	}
	AddValue(myType(0), func() Value {
		return &myValue{}
	})
	assert.Equal(t, &myValue{}, ToValue(myType(0), ""))
}

func TestToValue(t *testing.T) {
	assert.Equal(t, &NilValue{}, ToValue(nil, ""))
}
*/
