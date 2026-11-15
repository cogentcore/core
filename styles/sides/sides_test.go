// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sides

import (
	"image/color"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/assert"
)

// TestSet tests setting sides from 0 to 5 values, as in CSS.
func TestSet(t *testing.T) {
	assert.Equal(t, Sides[int]{}, *NewSides[int]())
	assert.Equal(t, Sides[int]{1, 1, 1, 1}, *NewSides(1))
	assert.Equal(t, Sides[int]{1, 2, 1, 2}, *NewSides(1, 2))
	assert.Equal(t, Sides[int]{1, 2, 3, 2}, *NewSides(1, 2, 3))
	assert.Equal(t, Sides[int]{1, 2, 3, 4}, *NewSides(1, 2, 3, 4))
	assert.Equal(t, Sides[int]{1, 2, 3, 4}, *NewSides(1, 2, 3, 4, 5), "extra values are ignored")
	s := NewSides(1, 2, 3, 4)
	assert.Equal(t, Sides[int]{}, *s.Zero())
	s.SetTop(5).SetRight(6).SetBottom(7).SetLeft(8)
	assert.Equal(t, Sides[int]{5, 6, 7, 8}, *s)
	assert.False(t, AreSame(*s))
	assert.False(t, AreZero(*s))
	assert.True(t, AreSame(*NewSides(3)))
	assert.True(t, AreZero(Sides[int]{}))
}

// TestSetAny tests setting sides from values of different types.
func TestSetAny(t *testing.T) {
	var s Sides[units.Value]
	v := units.Dp(2)
	vals := []units.Value{units.Dp(1), units.Dp(2)}
	assert.NoError(t, s.SetAny(v))
	assert.Equal(t, *NewSides(v), s)
	assert.NoError(t, s.SetAny(&v))
	assert.NoError(t, s.SetAny(vals))
	assert.Equal(t, *NewSides(vals...), s)
	assert.NoError(t, s.SetAny(&vals))
	o := *NewSides(units.Px(3))
	assert.NoError(t, s.SetAny(o))
	assert.Equal(t, o, s)
	assert.NoError(t, s.SetAny(&o))
	assert.NoError(t, s.SetAny("1dp 2px 3em"))
	assert.Equal(t, *NewSides(units.Dp(1), units.Px(2), units.Em(3)), s)
	assert.NoError(t, s.SetAny(4), "formatted as a string")
	assert.Equal(t, *NewSides(units.Px(4)), s, "a bare number is in px")
	assert.Error(t, s.SetAny("1dp bad"))

	var f Sides[float32]
	assert.Error(t, f.SetString("1 2"), "float32 has no SetString")
}

// TestValuesFloats tests converting values to dots, and the math of floats.
func TestValuesFloats(t *testing.T) {
	uc := &units.Context{}
	uc.Defaults()
	sv := NewValues(units.Dot(1), units.Dot(2), units.Dot(3), units.Dot(4))
	f := sv.ToDots(uc)
	assert.Equal(t, NewFloats(1, 2, 3, 4), f)
	assert.Equal(t, f, sv.Dots())
	assert.Equal(t, sv, f.ToValues())

	g := NewFloats(4, 3, 2, 1)
	assert.Equal(t, NewFloats(5), f.Add(g))
	assert.Equal(t, NewFloats(-3, -1, 1, 3), f.Sub(g))
	assert.Equal(t, NewFloats(2, 4, 6, 8), f.MulScalar(2))
	assert.Equal(t, NewFloats(1, 2, 2, 1), f.Min(g))
	assert.Equal(t, NewFloats(4, 3, 3, 4), f.Max(g))
	assert.Equal(t, NewFloats(1, 2, 3, 4), NewFloats(1.2, 1.6, 3.4, 3.5).Round())
	assert.Equal(t, math32.Vec2(4, 1), f.Pos())
	assert.Equal(t, math32.Vec2(6, 4), f.Size())
}

// TestColors tests setting the colors of sides.
func TestColors(t *testing.T) {
	var c Colors
	red := colors.Red
	reds := []color.RGBA{colors.Red, colors.Blue}
	assert.NoError(t, c.SetAny(red, nil))
	assert.Equal(t, NewColors(red), c)
	assert.NoError(t, c.SetAny(&red, nil))
	assert.NoError(t, c.SetAny(reds, nil))
	assert.Equal(t, NewColors(reds...), c)
	assert.NoError(t, c.SetAny(&reds, nil))
	o := NewColors(colors.Green)
	assert.NoError(t, c.SetAny(o.Sides, nil))
	assert.Equal(t, o, c)
	assert.NoError(t, c.SetAny(&o.Sides, nil))
	assert.NoError(t, c.SetAny("red blue", nil))
	assert.Equal(t, NewColors(colors.Red, colors.Blue), c)
	assert.Error(t, c.SetAny("red notacolor", nil))
	assert.Error(t, c.SetAny(12, nil), "not a color")
}
