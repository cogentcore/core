// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package math32

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTruncate(t *testing.T) {
	tests := []struct {
		x    float32
		prec int
		cor  float32
	}{
		{x: Pi, prec: 1, cor: 3},
		{x: Pi, prec: 2, cor: 3.1},
		{x: Pi, prec: 3, cor: 3.14},
		{x: Pi, prec: 4, cor: 3.142},
		{x: Pi, prec: 5, cor: 3.1416},
		{x: Pi, prec: 6, cor: 3.14159},
		{x: Pi, prec: 7, cor: 3.141593},
	}
	for _, tt := range tests {
		got := Truncate(tt.x, tt.prec)
		assert.Equal(t, tt.cor, got)
	}
}

func TestTruncate64(t *testing.T) {
	tests := []struct {
		x    float64
		prec int
		cor  float64
	}{
		{x: Pi, prec: 1, cor: 3},
		{x: Pi, prec: 2, cor: 3.1},
		{x: Pi, prec: 3, cor: 3.14},
		{x: Pi, prec: 4, cor: 3.142},
		{x: Pi, prec: 5, cor: 3.1416},
		{x: Pi, prec: 6, cor: 3.14159},
		{x: Pi, prec: 7, cor: 3.141593},
	}
	for _, tt := range tests {
		got := Truncate64(tt.x, tt.prec)
		assert.Equal(t, tt.cor, got)
	}
}

func TestIntMultiple(t *testing.T) {
	tests := []struct {
		x, mod, cor float32
	}{
		{x: 2, mod: 1, cor: 2},
		{x: 2.1, mod: 1, cor: 2},
		{x: 2.4, mod: 1, cor: 2},
		{x: 2.5, mod: 1, cor: 3},
		{x: 2.9, mod: 1, cor: 3},
		{x: 1000, mod: 50, cor: 1000},
		{x: 1005, mod: 50, cor: 1000},
		{x: 1020, mod: 50, cor: 1000},
		{x: 1025, mod: 50, cor: 1050},
		{x: 1049, mod: 50, cor: 1050},
	}
	for _, tt := range tests {
		got := IntMultiple(tt.x, tt.mod)
		assert.Equal(t, tt.cor, got)
	}
}

func TestIntMultipleGE(t *testing.T) {
	tests := []struct {
		x, mod, cor float32
	}{
		{x: 2, mod: 1, cor: 2},
		{x: 2.1, mod: 1, cor: 3},
		{x: 2.4, mod: 1, cor: 3},
		{x: 2.5, mod: 1, cor: 3},
		{x: 2.9, mod: 1, cor: 3},
		{x: 1000, mod: 50, cor: 1000},
		{x: 1005, mod: 50, cor: 1050},
		{x: 1020, mod: 50, cor: 1050},
		{x: 1025, mod: 50, cor: 1050},
		{x: 1049, mod: 50, cor: 1050},
	}
	for _, tt := range tests {
		got := IntMultipleGE(tt.x, tt.mod)
		assert.Equal(t, tt.cor, got)
	}
}

func TestWrapMax(t *testing.T) {
	tests := []struct {
		x, mx, cor float32
	}{
		{x: 2, mx: 1, cor: 0},
		{x: 2.5, mx: 2, cor: 0.5},
		{x: 10002.5, mx: 2, cor: 0.5},
		{x: -2.5, mx: 2, cor: 1.5},
		{x: -200.5, mx: 2, cor: 1.5},
		{x: 3.14, mx: 3.1, cor: 0.04},
	}
	for _, tt := range tests {
		got := WrapMax(tt.x, tt.mx)
		assert.InDelta(t, tt.cor, got, 1e-5)
	}
}

func TestWrapMinMax(t *testing.T) {
	tests := []struct {
		x, mn, mx, cor float32
	}{
		{x: 2, mn: -1, mx: 1, cor: 0},
		{x: 2.5, mn: -2, mx: 2, cor: -1.5},
		{x: 10002.5, mn: -2, mx: 2, cor: -1.5},
		{x: -2.5, mn: -2, mx: 2, cor: 1.5},
		{x: -200.5, mn: -2, mx: 2, cor: -0.5},
		{x: 3.14, mn: -3.1, mx: 3.1, cor: -3.06},
	}
	for _, tt := range tests {
		got := WrapMinMax(tt.x, tt.mn, tt.mx)
		assert.InDelta(t, tt.cor, got, 1e-5)
	}
}

func TestWrapPi(t *testing.T) {
	tests := []struct {
		x, cor float32
	}{
		{x: 0, cor: 0},
		{x: Pi, cor: -Pi},
		{x: -Pi, cor: -Pi},
		{x: 3 * Pi, cor: -Pi},
		{x: -3 * Pi, cor: -Pi},
		{x: 4 * Pi, cor: 0},
		{x: 0.5 * Pi, cor: 0.5 * Pi},
		{x: -0.5 * Pi, cor: -0.5 * Pi},
		{x: 2 * Pi, cor: 0},
	}
	for _, tt := range tests {
		got := WrapPi(tt.x)
		assert.InDelta(t, tt.cor, got, 1e-5)
	}
}

func TestMinAngleDiff(t *testing.T) {
	tests := []struct {
		a, b, cor float32
	}{
		{a: 0, b: Pi, cor: Pi},
		{a: Pi, b: -Pi - 0.5, cor: 0.5},
		{a: -Pi, b: Pi + 0.5, cor: -0.5},
		{a: Pi, b: -Pi, cor: 0},
		{a: Pi, b: -Pi + 0.1, cor: -0.1},
	}
	for _, tt := range tests {
		got := MinAngleDiff(tt.a, tt.b)
		assert.InDelta(t, tt.cor, got, 1e-5)
	}
}
