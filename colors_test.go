// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colors

import (
	"fmt"
	"image/color"
)

func ExampleIsNil_true() {
	fmt.Println(IsNil(color.RGBA{}))
	// Output: true
}

func ExampleIsNil_false() {
	fmt.Println(IsNil(Blue))
	// Output: false
}

func ExampleFromRGB() {
	fmt.Println(FromRGB(50, 100, 150))
	// Output: {50 100 150 255}
}

func ExampleFromNRGBA() {
	fmt.Println(FromNRGBA(50, 100, 150, 200))
	// Output: {39 78 118 200}
}

func ExampleAsRGBA() {
	fmt.Println(AsRGBA(color.Gray{50}))
	// Output: {50 50 50 255}
}

func ExampleAsString() {
	fmt.Println(AsString(Orange))
	// Output: rgba(255, 165, 0, 255)
}

func ExampleFromName() {
	fmt.Println(FromName("yellowgreen"))
	// Output: {154 205 50 255} <nil>
}

func ExampleFromName_fail() {
	fmt.Println(FromName("invalidcolor"))
	// Output: {0 0 0 0} colors.FromName: name not found: invalidcolor
}

func ExampleFromString_rgba() {
	fmt.Println(FromString("rgba(188, 12, 71, 201)", White))
	// Output: {188 12 71 201} <nil>
}

func ExampleFromString_hsl() {
	fmt.Println(FromString("hsl(12, 62, 50)", White))
	// Output: {207 80 48 255} <nil>
}

func ExampleFromString_hcta() {
	fmt.Println(FromString("hcta(83, 91, 48, 233)", White))
	// Output: {148 107 0 18} <nil>
}
