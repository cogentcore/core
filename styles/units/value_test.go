// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package units

import (
	"reflect"
	"strings"
	"testing"

	"cogentcore.org/core/math32"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUnitFuncs tests that the generated constructor, setter, and context
// functions of each unit agree with [New] and [Context.ToDots].
func TestUnitFuncs(t *testing.T) {
	ctors := map[Units]func(float32) Value{
		UnitDp: Dp, UnitPx: Px, UnitEw: Ew, UnitEh: Eh, UnitPw: Pw, UnitPh: Ph,
		UnitRem: Rem, UnitEm: Em, UnitEx: Ex, UnitCh: Ch, UnitVw: Vw, UnitVh: Vh,
		UnitVmin: Vmin, UnitVmax: Vmax, UnitCm: Cm, UnitMm: Mm, UnitQ: Q,
		UnitIn: In, UnitPc: Pc, UnitPt: Pt,
	}
	uc := NewContext()
	uc.SetSizes(1000, 500, 200, 100, 400, 300)
	for un, ctor := range ctors {
		assert.Equal(t, New(3, un), ctor(3), un)

		name := strings.ToUpper(un.String()[:1]) + un.String()[1:]
		v := Value{Custom: func(uc *Context) float32 { return 1 }}
		reflect.ValueOf(&v).MethodByName(name).Call([]reflect.Value{reflect.ValueOf(float32(3))})
		assert.Equal(t, New(3, un), v, "the setter clears Custom: %v", un)

		m := reflect.ValueOf(uc).MethodByName(name)
		require.True(t, m.IsValid(), name)
		got := m.Call([]reflect.Value{reflect.ValueOf(float32(3))})[0].Interface().(float32)
		assert.Equal(t, uc.ToDots(3, un), got, un)
	}
}

func TestContext(t *testing.T) {
	uc := NewContext()
	assert.Equal(t, float32(DpPerInch), uc.DPI)
	assert.False(t, uc.SetSizes(0, 0, 0, 0, 0, 0), "zeros are ignored")
	assert.False(t, uc.SetSizes(800, 600, 800, 600, 800, 600), "the same as the defaults")
	assert.True(t, uc.SetSizes(1000, 0, 0, 0, 0, 0))
	for i := range 6 {
		sz := make([]float32, 6)
		sz[i] = 77
		assert.True(t, uc.SetSizes(sz[0], sz[1], sz[2], sz[3], sz[4], sz[5]), "size %d", i)
	}
	assert.Equal(t, float32(77), uc.Vpw)
	assert.Equal(t, float32(77), uc.Pah)

	uc.SetFont(0)
	assert.Equal(t, float32(16), uc.FontEm, "zero is the default size")
	uc.SetFont(20)
	assert.Equal(t, float32(20), uc.FontEm)
	assert.Equal(t, float32(11), uc.FontEx)
	assert.Equal(t, float32(9), uc.FontCh)
	assert.Equal(t, float32(16), uc.FontRem)

	assert.Equal(t, float32(5), uc.FromDots(uc.ToDots(5, UnitPt), UnitPt))
	assert.InDelta(t, 2.54, uc.Convert(1, UnitIn, UnitCm), 1e-5)
	assert.Equal(t, uc.ToDots(12, UnitPx), uc.PxToDots(12))
	assert.InDelta(t, 12, uc.DotsToPx(uc.PxToDots(12)), 1e-5)
	assert.Contains(t, uc.String(), `"FontEm": 20`)

	var zero Context
	assert.Equal(t, float32(1), zero.Dots(UnitDp), "an uninitialized context uses the defaults")
	assert.Equal(t, zero.DPI, zero.Dots(Units(1000)), "an unknown unit is the DPI")
}

func TestValueMethods(t *testing.T) {
	uc := NewContext()
	v := Dp(5)
	v.Zero()
	assert.Equal(t, Value{Unit: UnitDot}, v)
	assert.Equal(t, Value{Unit: UnitDot}, Zero())

	v.Dot(7)
	assert.Equal(t, Value{Value: 7, Unit: UnitDot, Dots: 7}, v, "dots are set at once")
	assert.Equal(t, v, Dot(7))
	assert.Equal(t, float32(7), v.ToDots(uc))

	c := Custom(func(uc *Context) float32 { return uc.Vpw / 2 })
	assert.Equal(t, float32(400), c.ToDots(uc))
	assert.Equal(t, float32(400), c.Dots)
	v.SetCustom(func(uc *Context) float32 { return 9 })
	assert.Equal(t, float32(9), v.ToDots(uc))

	in := In(1)
	assert.Equal(t, int64(in.ToDots(uc)), int64(in.ToDotsFixed(uc)))

	pt := Pt(72)
	conv := pt.Convert(UnitIn, uc)
	assert.InDelta(t, 1, conv.Value, 1e-5)
	assert.Equal(t, UnitIn, conv.Unit)
}

func TestValueStringCSSDp(t *testing.T) {
	tests := []struct {
		v    Value
		want string
	}{
		{Px(96), "160px"},
		{In(1), "160px"},
		{Pt(72), "160px"},
		{Pc(6), "160px"},
		{Cm(2.54), "160px"},
		{Mm(25.4), "160px"},
		{Q(101.6), "160px"},
		{Dp(10), "10px"},
		{Em(2), "2em"},
		{Pw(50), "50%"},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.want, tc.v.StringCSSDp(), "%v", tc.v)
	}
}

func TestValueSetString(t *testing.T) {
	tests := []struct {
		in   string
		want Value
	}{
		{"5", Px(5)},
		{"12px", Px(12)},
		{"2rem", Rem(2)},
		{"2em", Em(2)},
		{" 3.5dp ", Dp(3.5)},
		{"10vmin", Vmin(10)},
		{"10vmax", Vmax(10)},
		{"1in", In(1)},
		{"4q", Q(4)},
		{"3dot", New(3, UnitDot)},
		{"7", Px(7)},
		{"7PT", Pt(7)},
	}
	for _, tc := range tests {
		var v Value
		assert.NoError(t, v.SetString(tc.in), tc.in)
		assert.Equal(t, tc.want, v, tc.in)
	}
	var v Value
	assert.Error(t, v.SetString("x"))
	assert.Error(t, v.SetString("abcpx"))
}

func TestValueSetAny(t *testing.T) {
	var v Value
	assert.NoError(t, v.SetAny("3em", "width"))
	assert.Equal(t, Em(3), v)
	assert.NoError(t, v.SetAny(Pt(4), "width"))
	assert.Equal(t, Pt(4), v)
	p := Vw(5)
	assert.NoError(t, v.SetAny(&p, "width"))
	assert.Equal(t, Vw(5), v)
	assert.NoError(t, v.SetAny(8, "width"))
	assert.Equal(t, Dp(8), v, "a number is in dp")
	assert.Error(t, v.SetAny(struct{}{}, "width"))
}

func TestXY(t *testing.T) {
	uc := NewContext()
	var xy XY
	xy.Set(Dp(2))
	assert.Equal(t, Dp(2), xy.X)
	assert.Equal(t, Dp(2), xy.Y)
	xy.Set(Dp(2), Px(3))
	assert.Equal(t, "(2dp, 3px)", xy.String())
	xy.ToDots(uc)
	assert.Equal(t, math32.Vec2(2, uc.PxToDots(3)), xy.Dots())
	assert.Equal(t, float32(3), xy.Dim(math32.Y).Value)
	xy.SetDim(math32.X, Em(1))
	assert.Equal(t, Em(1), xy.Dim(math32.X))
	xy.SetDim(math32.Y, Em(2))
	assert.Equal(t, UnitEm, xy.Y.Unit)
	xy.Zero()
	assert.Equal(t, float32(0), xy.X.Value)
	xy.Set()
	assert.Equal(t, XY{}, xy)
	assert.Panics(t, func() { xy.Dim(math32.Z) })
	assert.Panics(t, func() { xy.SetDim(math32.Z, Dp(1)) })
}

// TestDotZeroClearCustom tests that setting a value in dots, or to zero,
// clears any custom function, like the other setters.
func TestDotZeroClearCustom(t *testing.T) {
	uc := &Context{}
	uc.Defaults()
	v := Custom(func(uc *Context) float32 { return 9 })
	v.Dot(5)
	assert.Equal(t, float32(5), v.ToDots(uc))
	v = Custom(func(uc *Context) float32 { return 9 })
	v.Zero()
	assert.Equal(t, float32(0), v.ToDots(uc))
}
