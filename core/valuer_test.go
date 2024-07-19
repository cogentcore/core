// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/styles"
	"github.com/stretchr/testify/assert"
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
		{"tree", NewWidgetBase()},
		{"tree-nil", (*Frame)(nil)},
	}
	for _, value := range values {
		b := NewBody()
		NewValue(value.value, "", b)
		b.AssertRender(t, "valuer/"+value.name)
	}
}

func TestAddValueType(t *testing.T) {
	type myType int
	type myValue struct {
		Text
	}
	AddValueType[myType, myValue]()
	assert.IsType(t, &myValue{}, toValue(myType(0), ""))
}
