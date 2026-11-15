// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package styleprops

import (
	"testing"

	"cogentcore.org/core/enums"
	"cogentcore.org/core/styles/sides"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/assert"
)

// testStyle is a style with fields of each kind of style function.
type testStyle struct {
	I  int
	F  float32
	FP float32
	B  bool
	U  units.Value
	E  sides.Indexes
}

var testFuncs = map[string]Func{
	"i":  Int(int(3), func(s *testStyle) *int { return &s.I }),
	"f":  Float(float32(1.5), func(s *testStyle) *float32 { return &s.F }),
	"fp": FloatProportion(float32(1), func(s *testStyle) *float32 { return &s.FP }),
	"b":  Bool(true, func(s *testStyle) *bool { return &s.B }),
	"u":  Units(units.Dp(1), func(s *testStyle) *units.Value { return &s.U }),
	"e":  Enum(sides.Top, func(s *testStyle) enums.EnumSetter { return &s.E }),
}

// set sets the property of the given key of a style.
func set(s *testStyle, key string, val any, parent *testStyle) {
	var par any
	if parent != nil {
		par = parent
	}
	testFuncs[key](s, key, val, par, nil)
}

// TestStyleFuncs tests setting the fields of a style from properties,
// and inheriting and initializing them.
func TestStyleFuncs(t *testing.T) {
	s := &testStyle{}
	set(s, "i", "7", nil)
	set(s, "f", "2.5%", nil)
	set(s, "fp", "50%", nil)
	set(s, "b", "true", nil)
	set(s, "u", "4px", nil)
	set(s, "e", "left", nil)
	assert.Equal(t, testStyle{I: 7, F: 2.5, FP: 0.5, B: true, U: units.Px(4), E: sides.Left}, *s)
	set(s, "fp", 0.25, nil)
	assert.Equal(t, float32(0.25), s.FP, "a number is not a percent")
	set(s, "fp", "1.5", nil)
	assert.Equal(t, float32(1.5), s.FP, "nor is a string without %")
	set(s, "e", sides.Right, nil)
	assert.Equal(t, sides.Right, s.E)
	set(s, "e", 2, nil)
	assert.Equal(t, sides.Indexes(2), s.E)
	set(s, "i", "bad", nil)
	assert.Equal(t, 7, s.I, "not changed by a bad value")

	p := &testStyle{I: 1, F: 2, FP: 3, B: false, U: units.Em(2), E: sides.Bottom}
	for k := range testFuncs {
		set(s, k, "inherit", p)
	}
	assert.Equal(t, *p, *s)
	for k := range testFuncs {
		set(s, k, "initial", p)
	}
	assert.Equal(t, testStyle{I: 3, F: 1.5, FP: 1, B: true, U: units.Dp(1), E: sides.Top}, *s)

	inh, init := InhInit("inherit", nil)
	assert.False(t, inh, "nothing to inherit from")
	assert.False(t, init)
	SetError("key", 1, assert.AnError)
}

// TestXMLString tests converting a style string to and from properties.
func TestXMLString(t *testing.T) {
	props := map[string]any{}
	FromXMLString("fill: red; Stroke-Width:2 ;xml:space:preserve;bad", props)
	assert.Equal(t, map[string]any{"fill": "red", "stroke-width": "2", "xml:space": "preserve"}, props)
	assert.Equal(t, "fill:red;", ToXMLString(map[string]any{"fill": "red", "transform": "x"}))
}
