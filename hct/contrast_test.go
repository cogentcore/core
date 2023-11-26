// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hct

import (
	"image/color"
	"testing"

	"goki.dev/mat32/v2"
)

func TestContrastRatio(t *testing.T) {
	type data struct {
		a    color.Color
		b    color.Color
		want float32
	}
	tests := []data{
		{color.White, color.Black, 21},
		{color.Black, color.White, 21},
		{color.RGBA{100, 100, 100, 255}, color.RGBA{100, 100, 100, 255}, 1},
		{color.RGBA{0, 0, 255, 255}, color.RGBA{255, 255, 255, 255}, 8.59},
	}
	for i, test := range tests {
		res := ContrastRatio(test.a, test.b)
		if mat32.Abs(res-test.want) > 0.1 {
			t.Errorf("%d: expected %g but got %g", i, test.want, res)
		}
	}
}

func TestToneContrastRatio(t *testing.T) {
	type data struct {
		a    float32
		b    float32
		want float32
	}
	tests := []data{
		{0, 100, 21},
		{100, 0, 21},
		{50, 50, 1},
		{100, 32.302586, 8.59},
	}
	for i, test := range tests {
		res := ToneContrastRatio(test.a, test.b)
		if mat32.Abs(res-test.want) > 0.1 {
			t.Errorf("%d: expected %g but got %g", i, test.want, res)
		}
	}
}

func TestContrastColor(t *testing.T) {
	type data struct {
		color color.Color
		ratio float32
		want  color.Color
	}
	tests := []data{
		{color.RGBA{0, 0, 0, 255}, 21, color.RGBA{255, 255, 255, 255}},
		{color.RGBA{255, 255, 255, 255}, 21, color.RGBA{0, 0, 0, 255}},
		{color.RGBA{100, 100, 100, 255}, 1, color.RGBA{100, 100, 100, 255}},
		{color.RGBA{0, 0, 255, 255}, 8.59, color.RGBA{255, 255, 255, 255}},
	}
	for i, test := range tests {
		res := ContrastColor(test.color, test.ratio)
		if res != test.want {
			t.Errorf("%d: expected %v but got %v", i, test.want, res)
		}
	}
}

func TestContrastColorTry(t *testing.T) {
	type data struct {
		color color.Color
		ratio float32
		want  color.Color
		ok    bool
	}
	tests := []data{
		{color.RGBA{0, 0, 0, 255}, 21, color.RGBA{255, 255, 255, 255}, true},
		{color.RGBA{150, 200, 255, 255}, 21, color.RGBA{}, false},
		{color.RGBA{100, 100, 100, 255}, 1, color.RGBA{100, 100, 100, 255}, true},
		{color.RGBA{0, 0, 255, 255}, 8.59, color.RGBA{255, 255, 255, 255}, true},
	}
	for i, test := range tests {
		res, ok := ContrastColorTry(test.color, test.ratio)
		if res != test.want {
			t.Errorf("%d: expected %v, %v but got %v, %v", i, test.want, test.ok, res, ok)
		}
	}
}

func TestContrastTone(t *testing.T) {
	type data struct {
		tone  float32
		ratio float32
		want  float32
	}
	tests := []data{
		{0, 21, 100},
		{100, 21, 0},
		{50, 1, 50},
		{32.302586, 8.59, 100},
	}
	for i, test := range tests {
		res := ContrastTone(test.tone, test.ratio)
		if mat32.Abs(res-test.want) > 0.5 {
			t.Errorf("%d: expected %g but got %g", i, test.want, res)
		}
	}
}

func TestContrastToneTry(t *testing.T) {
	type data struct {
		tone  float32
		ratio float32
		want  float32
		ok    bool
	}
	tests := []data{
		{0, 21, 100, true},
		{60, 18, -1, false},
		{50, 1, 50, true},
		{32.302586, 8.59, 100, true},
	}
	for i, test := range tests {
		res, ok := ContrastToneTry(test.tone, test.ratio)
		if ok != test.ok || mat32.Abs(res-test.want) > 0.5 {
			t.Errorf("%d: expected %g, %v but got %g, %v", i, test.want, test.ok, res, ok)
		}
	}
}

func TestContrastToneLighter(t *testing.T) {
	type data struct {
		tone  float32
		ratio float32
		want  float32
	}
	tests := []data{
		{0, 21, 100},
		{100, 21, 100},
		{50, 1, 50},
		{32.302586, 8.59, 100},
	}
	for i, test := range tests {
		res := ContrastToneLighter(test.tone, test.ratio)
		if mat32.Abs(res-test.want) > 0.1 {
			t.Errorf("%d: expected %g but got %g", i, test.want, res)
		}
	}
}

func TestContrastToneLighterTry(t *testing.T) {
	type data struct {
		tone  float32
		ratio float32
		want  float32
		ok    bool
	}
	tests := []data{
		{0, 21, 100, true},
		{100, 21, -1, false},
		{50, 1, 50, true},
		{32.302586, 8.59, 100, true},
	}
	for i, test := range tests {
		res, ok := ContrastToneLighterTry(test.tone, test.ratio)
		if ok != test.ok || mat32.Abs(res-test.want) > 0.1 {
			t.Errorf("%d: expected %g, %v but got %g, %v", i, test.want, test.ok, res, ok)
		}
	}
}

func TestContrastToneDarker(t *testing.T) {
	type data struct {
		tone  float32
		ratio float32
		want  float32
	}
	tests := []data{
		{100, 21, 0},
		{0, 21, 0},
		{50, 1, 50},
		{100, 8.59, 32.302586},
	}
	for i, test := range tests {
		res := ContrastToneDarker(test.tone, test.ratio)
		if mat32.Abs(res-test.want) > 0.1 {
			t.Errorf("%d: expected %g but got %g", i, test.want, res)
		}
	}
}

func TestContrastToneDarkerTry(t *testing.T) {
	type data struct {
		tone  float32
		ratio float32
		want  float32
		ok    bool
	}
	tests := []data{
		{100, 21, 0, true},
		{0, 21, -1, false},
		{50, 1, 50, true},
		{100, 8.59, 32.302586, true},
	}
	for i, test := range tests {
		res, ok := ContrastToneDarkerTry(test.tone, test.ratio)
		if ok != test.ok || mat32.Abs(res-test.want) > 0.1 {
			t.Errorf("%d: expected %g, %v but got %g, %v", i, test.want, test.ok, res, ok)
		}
	}
}
