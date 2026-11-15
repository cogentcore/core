// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package minmax

import (
	"math"
	"testing"

	"cogentcore.org/core/math32"
	"github.com/stretchr/testify/assert"
)

func TestF32(t *testing.T) {
	var mr F32
	mr.SetInfinity()
	assert.False(t, mr.IsValid(), "infinity is the empty range")
	for _, v := range []float32{3, -1, 5} {
		mr.FitValInRange(v)
	}
	assert.Equal(t, F32{Min: -1, Max: 5}, mr)
	assert.False(t, mr.FitValInRange(2), "already in the range")
	assert.True(t, mr.IsValid())
	assert.Equal(t, "{-1 5}", mr.String())
	assert.Equal(t, float32(6), mr.Range())
	assert.Equal(t, float32(2), mr.Midpoint())
	assert.InDelta(t, 1.0/6, mr.Scale(), 1e-6)

	assert.True(t, mr.InRange(-1))
	assert.True(t, mr.InRange(5))
	assert.False(t, mr.InRange(5.5))
	assert.True(t, mr.IsLow(-2))
	assert.False(t, mr.IsLow(0))

	assert.Equal(t, float32(-1), mr.ClampValue(-10))
	assert.Equal(t, float32(5), mr.ClampValue(10))
	assert.Equal(t, float32(1), mr.ClampValue(1))
	assert.Equal(t, float32(0.5), mr.NormValue(2))
	assert.Equal(t, float32(1), mr.NormValue(100), "clamped")
	assert.Equal(t, float32(0), mr.ClipNormValue(-5))
	assert.Equal(t, float32(1), mr.ClipNormValue(50))
	assert.Equal(t, float32(0.5), mr.ClipNormValue(2))
	assert.Equal(t, float32(2), mr.ProjValue(0.5))

	assert.True(t, mr.FitInRange(F32{Min: -3, Max: 1}))
	assert.True(t, mr.FitInRange(F32{Min: 0, Max: 7}))
	assert.False(t, mr.FitInRange(F32{Min: 0, Max: 1}))
	assert.Equal(t, F32{Min: -3, Max: 7}, mr)

	mr.Set(4, 4)
	assert.Equal(t, float32(0), mr.Scale(), "an empty range has no scale")
}

func TestF32Sanitize(t *testing.T) {
	tests := []struct{ in, want F32 }{
		{F32{Min: math32.Inf(1), Max: math32.Inf(-1)}, F32{Min: -1, Max: 1}},
		{F32{Min: 5, Max: 2}, F32{Min: 2, Max: 5}},
		{F32{Min: 3, Max: 3}, F32{Min: 2, Max: 4}},
		{F32{Min: 1, Max: 2}, F32{Min: 1, Max: 2}},
	}
	for _, tc := range tests {
		mr := tc.in
		mr.Sanitize()
		assert.Equal(t, tc.want, mr, "%v", tc.in)
	}
}

func TestF64(t *testing.T) {
	var mr F64
	mr.SetInfinity()
	assert.False(t, mr.IsValid())
	for _, v := range []float64{3, -1, 5} {
		mr.FitValInRange(v)
	}
	assert.Equal(t, F64{Min: -1, Max: 5}, mr)
	assert.Equal(t, 6.0, mr.Range())
	assert.Equal(t, 2.0, mr.Midpoint())
	assert.InDelta(t, 1.0/6, mr.Scale(), 1e-12)
	assert.True(t, mr.InRange(0))
	assert.True(t, mr.IsLow(-2))
	assert.Equal(t, -1.0, mr.ClampValue(-10))
	assert.Equal(t, 5.0, mr.ClampValue(10))
	assert.Equal(t, 0.5, mr.NormValue(2))
	assert.Equal(t, 0.0, mr.ClipNormValue(-5))
	assert.Equal(t, 1.0, mr.ClipNormValue(50))
	assert.Equal(t, 2.0, mr.ProjValue(0.5))
	assert.True(t, mr.FitInRange(F64{Min: -3, Max: 7}))
	assert.Equal(t, F64{Min: -3, Max: 7}, mr)

	mr.Set(math.Inf(1), math.Inf(-1))
	mr.Sanitize()
	assert.Equal(t, F64{Min: -1, Max: 1}, mr)
	mr.Set(5, 2)
	mr.Sanitize()
	assert.Equal(t, F64{Min: 2, Max: 5}, mr)
	mr.Set(4, 4)
	assert.Equal(t, 0.0, mr.Scale())
}

func TestInt(t *testing.T) {
	var mr Int
	mr.SetInfinity()
	assert.False(t, mr.IsValid())
	for _, v := range []int{3, -1, 5} {
		mr.FitValInRange(v)
	}
	assert.Equal(t, Int{Min: -1, Max: 5}, mr)
	assert.False(t, mr.FitValInRange(0))
	assert.Equal(t, 6, mr.Range())
	assert.Equal(t, float32(2), mr.Midpoint())
	assert.InDelta(t, 1.0/6, mr.Scale(), 1e-6)
	assert.True(t, mr.InRange(5))
	assert.False(t, mr.InRange(6))
	assert.True(t, mr.IsLow(-2))
	assert.Equal(t, -1, mr.Clamp(-10))
	assert.Equal(t, 5, mr.Clamp(10))
	assert.Equal(t, float32(0.5), mr.NormValue(2))
	assert.Equal(t, float32(0), mr.ClipNormValue(-5))
	assert.Equal(t, float32(1), mr.ClipNormValue(50))
	assert.Equal(t, float32(2), mr.ProjValue(0.5))
	assert.True(t, mr.FitInRange(Int{Min: -3, Max: 7}))
	assert.False(t, mr.FitInRange(Int{Min: 0, Max: 1}))
	assert.Equal(t, Int{Min: -3, Max: 7}, mr)
	mr.Set(2, 2)
	assert.Equal(t, float32(0), mr.Scale())
}

func TestAvgMax32(t *testing.T) {
	var am AvgMax32
	am.Init()
	am.CalcAvg()
	assert.Equal(t, float32(0), am.Max, "no values: the max is not -MaxFloat32")
	am.Init()
	for i, v := range []float32{1, 4, 2} {
		am.UpdateValue(v, int32(i))
	}
	am.CalcAvg()
	assert.Equal(t, float32(7.0/3), am.Avg)
	assert.Equal(t, float32(4), am.Max)
	assert.Equal(t, int32(1), am.MaxIndex)
	assert.Equal(t, "{Avg: 2.3333333, Max: 4, Sum: 7, MaxIndex: 1, N: 3}", am.String())

	am.UpdateFromOther(3, 10, 1, 7)
	am.CalcAvg()
	assert.Equal(t, float32(2.5), am.Avg)
	assert.Equal(t, float32(10), am.Max)
	assert.Equal(t, int32(7), am.MaxIndex)

	var cp AvgMax32
	cp.CopyFrom(&am)
	assert.Equal(t, am, cp)
}

func TestAvgMax64(t *testing.T) {
	var am AvgMax64
	am.Init()
	am.CalcAvg()
	assert.Equal(t, 0.0, am.Max)
	am.Init()
	for i, v := range []float64{1, 4, 2} {
		am.UpdateValue(v, i)
	}
	am.CalcAvg()
	assert.InDelta(t, 7.0/3, am.Avg, 1e-12)
	assert.Equal(t, 4.0, am.Max)
	assert.Equal(t, int32(1), am.MaxIndex)

	oth := AvgMax64{Sum: 5, N: 1, Max: 9, MaxIndex: 3}
	am.UpdateFrom(&oth)
	am.CalcAvg()
	assert.Equal(t, 3.0, am.Avg)
	assert.Equal(t, 9.0, am.Max)
	assert.Equal(t, int32(3), am.MaxIndex)

	var cp AvgMax64
	cp.CopyFrom(&am)
	assert.Equal(t, am, cp)
}

func TestRange(t *testing.T) {
	var r32 Range32
	mn, mx := r32.Clamp(-2, 3)
	assert.Equal(t, []float32{-2, 3}, []float32{mn, mx}, "nothing fixed")
	r32.SetMin(-5).SetMax(10)
	assert.True(t, r32.FixMin && r32.FixMax)
	assert.Equal(t, float32(15), r32.Range())
	mn, mx = r32.Clamp(-2, 3)
	assert.Equal(t, []float32{-5, 10}, []float32{mn, mx})

	var r64 Range64
	r64.SetMin(-5).SetMax(10)
	assert.Equal(t, 15.0, r64.Range())
	m1, m2 := r64.Clamp(-2, 3)
	assert.Equal(t, []float64{-5, 10}, []float64{m1, m2})
}

// TestIsLowHigh tests that values are low below the minimum and high
// above the maximum.
func TestIsLowHigh(t *testing.T) {
	f32 := F32{Min: 1, Max: 5}
	assert.True(t, f32.IsHigh(6))
	assert.False(t, f32.IsHigh(3))
	assert.True(t, f32.IsLow(0))
	f64 := F64{Min: 1, Max: 5}
	assert.True(t, f64.IsHigh(6))
	assert.False(t, f64.IsHigh(3))
	in := Int{Min: 1, Max: 5}
	assert.True(t, in.IsHigh(6))
	assert.False(t, in.IsHigh(3))
}

// TestAvgMax32UpdateFrom tests that the values of another set are
// counted once.
func TestAvgMax32UpdateFrom(t *testing.T) {
	am := AvgMax32{}
	am.Init()
	am.UpdateValue(1, 0)
	oth := AvgMax32{Sum: 5, N: 1, Max: 5, MaxIndex: 3}
	am.UpdateFrom(&oth)
	am.CalcAvg()
	assert.Equal(t, float32(6), am.Sum)
	assert.Equal(t, int32(2), am.N)
	assert.Equal(t, float32(3), am.Avg)
	assert.Equal(t, int32(3), am.MaxIndex)
}
