// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package reflectx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ptr returns a pointer to the given value.
func ptr[T any](v T) *T { return &v }

// numbers returns the value 3 of every basic numeric type, and pointers to
// them, and the nil pointers of those types.
func numbers() (vals, nils []any) {
	vals = []any{
		int(3), int8(3), int16(3), int32(3), int64(3),
		uint(3), uint8(3), uint16(3), uint32(3), uint64(3), uintptr(3),
		float32(3), float64(3),
		ptr(int(3)), ptr(int8(3)), ptr(int16(3)), ptr(int32(3)), ptr(int64(3)),
		ptr(uint(3)), ptr(uint8(3)), ptr(uint16(3)), ptr(uint32(3)), ptr(uint64(3)), ptr(uintptr(3)),
		ptr(float32(3)), ptr(float64(3)),
	}
	nils = []any{
		(*int)(nil), (*int8)(nil), (*int16)(nil), (*int32)(nil), (*int64)(nil),
		(*uint)(nil), (*uint8)(nil), (*uint16)(nil), (*uint32)(nil), (*uint64)(nil), (*uintptr)(nil),
		(*float32)(nil), (*float64)(nil), (*bool)(nil), (*string)(nil),
	}
	return
}

// myInt is a named int type, which takes the reflection paths.
type myInt int

// myString is a named string type.
type myString string

// booler implements bools.Booler.
type booler struct{ b bool }

func (b booler) Bool() bool { return b.b }

func TestToNumbers(t *testing.T) {
	vals, nils := numbers()
	for _, v := range vals {
		i, err := ToInt(v)
		assert.NoError(t, err, "%T", v)
		assert.Equal(t, int64(3), i, "%T", v)
		f, err := ToFloat(v)
		assert.NoError(t, err, "%T", v)
		assert.Equal(t, 3.0, f, "%T", v)
		f32, err := ToFloat32(v)
		assert.NoError(t, err, "%T", v)
		assert.Equal(t, float32(3), f32, "%T", v)
		b, err := ToBool(v)
		assert.NoError(t, err, "%T", v)
		assert.True(t, b, "%T", v)
	}
	for _, v := range nils {
		_, err := ToInt(v)
		assert.Error(t, err, "%T", v)
		_, err = ToFloat(v)
		assert.Error(t, err, "%T", v)
		_, err = ToFloat32(v)
		assert.Error(t, err, "%T", v)
		_, err = ToBool(v)
		assert.Error(t, err, "%T", v)
		assert.Equal(t, "nil", ToString(v), "%T", v)
		assert.Equal(t, "nil", ToStringPrec(v, 3), "%T", v)
	}
}

func TestToNumbersOther(t *testing.T) {
	tests := []struct {
		v any
		i int64
		f float64
		b bool
	}{
		{true, 1, 1, true},
		{false, 0, 0, false},
		{ptr(true), 1, 1, true},
		{"12", 12, 12, true},
		{ptr("12"), 12, 12, true},
		{myInt(5), 5, 5, true},
		{ptr(myInt(5)), 5, 5, true},
		{myString("7"), 7, 7, true},
		{complex(2, 1), 2, 2, true},
		{ptr(complex64(complex(2, 1))), 2, 2, true},
		{3.7, 3, 3.7, true},
	}
	for _, tt := range tests {
		i, err := ToInt(tt.v)
		if assert.NoError(t, err, "%T %v", tt.v, tt.v) {
			assert.Equal(t, tt.i, i, "%T %v", tt.v, tt.v)
		}
		f, err := ToFloat(tt.v)
		if assert.NoError(t, err, "%T %v", tt.v, tt.v) {
			assert.Equal(t, tt.f, f, "%T %v", tt.v, tt.v)
		}
		f32, err := ToFloat32(tt.v)
		if assert.NoError(t, err, "%T %v", tt.v, tt.v) {
			assert.Equal(t, float32(tt.f), f32, "%T %v", tt.v, tt.v)
		}
	}

	b, err := ToBool("true")
	assert.NoError(t, err)
	assert.True(t, b)
	b, err = ToBool(booler{true})
	assert.NoError(t, err)
	assert.True(t, b)
	b, err = ToBool(myInt(0))
	assert.NoError(t, err)
	assert.False(t, b)

	for _, bad := range []any{"x", struct{}{}, []int{1}} {
		_, err := ToInt(bad)
		assert.Error(t, err, "%T", bad)
		_, err = ToFloat(bad)
		assert.Error(t, err, "%T", bad)
		_, err = ToFloat32(bad)
		assert.Error(t, err, "%T", bad)
		_, err = ToBool(bad)
		assert.Error(t, err, "%T", bad)
	}
	i, err := ToInt("0x10")
	assert.NoError(t, err)
	assert.Equal(t, int64(16), i, "base prefixes are parsed")
}
