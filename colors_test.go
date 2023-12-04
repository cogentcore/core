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
