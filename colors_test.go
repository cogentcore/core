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

func ExampleFromName_error() {
	fmt.Println(FromName("invalidcolor"))
	// Output: {0 0 0 0} colors.FromName: name not found: invalidcolor
}

func ExampleFromString_name() {
	fmt.Println(FromString("violet", Gray))
	// Output: {238 130 238 255} <nil>
}

func ExampleFromString_hex() {
	fmt.Println(FromString("#2af", Yellow))
	// Output: {34 170 255 255} <nil>
}

func ExampleFromString_rgb() {
	fmt.Println(FromString("rgb(202, 38, 16, 112)", White))
	// Output: {89 16 7 112} <nil>
}

func ExampleFromString_rgba() {
	fmt.Println(FromString("rgba(188, 12, 71, 201)", Black))
	// Output: {148 9 56 201} <nil>
}

func ExampleFromString_hsl() {
	fmt.Println(FromString("hsl(12, 62, 50, 189)", Blue))
	// Output: {154 59 35 189} <nil>
}

func ExampleFromString_hsla() {
	fmt.Println(FromString("hsla(12, 62, 50)", Rebeccapurple))
	// Output: {207 80 48 255} <nil>
}

func ExampleFromString_hct() {
	fmt.Println(FromString("hct(240, 56, 66)", Tan))
	// Output: {7 171 240 255} <nil>
}

func ExampleFromString_hcta() {
	fmt.Println(FromString("hcta(83, 91, 48, 233)", Lightcoral))
	// Output: {135 98 0 233} <nil>
}

func ExampleFromString_darken() {
	fmt.Println(FromString("darken-10", Green))
	// Output: {1 100 0 255} <nil>
}

func ExampleFromString_blend() {
	fmt.Println(FromString("blend-40-#fff", Black))
	// Output: {153 153 153 255} <nil>
}

func ExampleFromString_error() {
	fmt.Println(FromString("lighten-something", Rosybrown))
	// Output: {0 0 0 0} colors.FromString: error getting numeric value from "something": strconv.ParseFloat: parsing "something": invalid syntax
}

func ExampleFromAny() {
	fmt.Println(FromAny("rgb(12, 18, 92)", Lawngreen))
	// Output: {12 18 92 255} <nil>
}

func ExampleFromAny_error() {
	fmt.Println(FromAny([]float32{}, Yellowgreen))
	// Output: {0 0 0 0} colors.FromAny: could not get color from value [] of type []float32
}

func ExampleFromHex() {
	fmt.Println(FromHex("#FF00FF"))
	// Output: {255 0 255 255} <nil>
}

func ExampleFromHex_lower() {
	fmt.Println(FromHex("#1abc2e"))
	// Output: {26 188 46 255} <nil>
}

func ExampleFromHex_short() {
	fmt.Println(FromHex("F3A"))
	// Output: {255 51 170 255} <nil>
}

func ExampleFromHex_shortLower() {
	fmt.Println(FromHex("#bb6"))
	// Output: {187 187 102 255} <nil>
}

func ExampleFromHex_error() {
	fmt.Println(FromHex("#ef"))
	// Output: {0 0 0 0} colors.FromHex: could not process "ef"
}

func ExampleAsHex() {
	fmt.Println(AsHex(Blueviolet))
	// Output: #8A2BE2
}

func ExampleAsHex_withA() {
	fmt.Println(AsHex(WithA(Red, 100)))
	// Output: #FF000064
}

func ExampleWithR() {
	fmt.Println(WithR(Blue, 50))
	// Output: {50 0 255 255}
}

func ExampleWithG() {
	fmt.Println(WithG(Orange, 11))
	// Output: {255 11 0 255}
}

func ExampleWithB() {
	fmt.Println(WithB(Pink, 100))
	// Output: {255 192 100 255}
}

func ExampleWithA() {
	fmt.Println(WithA(Royalblue, 215))
	// Output: {55 88 190 215}
}

func ExampleWithAF32() {
	fmt.Println(WithAF32(Olive, 0.6))
	// Output: {77 77 0 153}
}

func ExampleApplyOpacity() {
	fmt.Println(ApplyOpacity(WithAF32(Gold, 0.4), 0.3))
	// Output: {30 25 0 30}
}

func ExampleClearer() {
	fmt.Println(Clearer(Gainsboro, 20))
	// Output: {176 176 176 204}
}

func ExampleOpaquer() {
	fmt.Println(Opaquer(WithAF32(Green, 0.15), 30))
	// Output: {0 57 0 114}
}

func ExampleBlend() {
	fmt.Println(Blend(30, Lightblue, Darkblue))
	// Output: {52 65 166 255}
}

func ExampleAlphaBlend() {
	fmt.Println(AlphaBlend(Wheat, WithAF32(Blue, 0.5)))
	// Output: {123 111 217 255}
}

func ExampleInverse() {
	fmt.Println(Inverse(Limegreen))
	// Output: {205 50 205 255}
}

func ExampleAdd() {
	fmt.Println(Add(Red, Blue))
	// Output: {255 0 255 255}
}

func ExampleSub() {
	fmt.Println(Sub(Purple, Blue))
	// Output: {128 0 0 0}
}
