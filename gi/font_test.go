// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"testing"
)

type testFontSpec struct {
	fn  string
	cor string
	str FontStretch
	wt  FontWeights
	sty FontStyles
}

var testFontNames = []testFontSpec{
	{"NotoSansBlack", "NotoSans Black", FontStrNormal, WeightBlack, FontNormal},
	{"NotoSansBlackItalic", "NotoSans Black Italic", FontStrNormal, WeightBlack, FontItalic},
	{"NotoSansBold", "NotoSans Bold", FontStrNormal, WeightBold, FontNormal},
	{"NotoSansCondensed", "NotoSans Condensed", FontStrCondensed, WeightNormal, FontNormal},
	{"NotoSansCondensedBlack", "NotoSans Condensed Black", FontStrCondensed, WeightBlack, FontNormal},
	{"NotoSansCondensedBlackItalic", "NotoSans Condensed Black Italic", FontStrCondensed, WeightBlack, FontItalic},
	{"NotoSansCondensedExtraBold", "NotoSans Condensed ExtraBold", FontStrCondensed, WeightExtraBold, FontNormal},
	{"NotoSansCondensedExtraBoldItalic", "NotoSans Condensed ExtraBold Italic", FontStrCondensed, WeightExtraBold, FontItalic},
	{"NotoSansExtraBold", "NotoSans ExtraBold", FontStrNormal, WeightExtraBold, FontNormal},
	{"NotoSansExtraBoldItalic", "NotoSans ExtraBold Italic", FontStrNormal, WeightExtraBold, FontItalic},
	{"NotoSansRegular", "NotoSans", FontStrNormal, WeightNormal, FontNormal},
	{"NotoSansNormal", "NotoSans", FontStrNormal, WeightNormal, FontNormal},
}

func TestFontMods(t *testing.T) {
	for _, ft := range testFontNames {
		fo := FixFontMods(ft.fn)
		if fo != ft.cor {
			t.Errorf("FixFontMods output: %v != correct: %v for font: %v\n", fo, ft.cor, ft.fn)
		}

		base, str, wt, sty := FontNameToMods(fo)
		if base != "NotoSans" {
			t.Errorf("FontNameToMods base: %v != correct: %v for font: %v\n", base, "NotoSans", fo)
		}
		if str != ft.str {
			t.Errorf("FontNameToMods strength: %v != correct: %v for font: %v\n", str, ft.str, fo)
		}
		if wt != ft.wt {
			t.Errorf("FontNameToMods weight: %v != correct: %v for font: %v\n", wt, ft.wt, fo)
		}
		if sty != ft.sty {
			t.Errorf("FontNameToMods style: %v != correct: %v for font: %v\n", sty, ft.sty, fo)
		}

		frc := FontNameFromMods(base, str, wt, sty)
		if frc != fo {
			t.Errorf("FontNameFromMods reconstructed font name: %v != correct: %v\n", frc, fo)
		}
	}
}

// note: the responses to the following two tests depend on what is installed on the system

func TestFontAlts(t *testing.T) {
	fa, serif, mono := FontAlts("serif")
	fmt.Printf("FontAlts: serif: %v  serif: %v, mono: %v\n", fa, serif, mono)

	fa, serif, mono = FontAlts("sans-serif")
	fmt.Printf("FontAlts: sans-serif: %v  serif: %v, mono: %v\n", fa, serif, mono)

	fa, serif, mono = FontAlts("monospace")
	fmt.Printf("FontAlts: monospace: %v  serif: %v, mono: %v\n", fa, serif, mono)

	fa, serif, mono = FontAlts("cursive")
	fmt.Printf("FontAlts: cursive: %v  serif: %v, mono: %v\n", fa, serif, mono)

	fa, serif, mono = FontAlts("fantasy")
	fmt.Printf("FontAlts: fantasy: %v  serif: %v, mono: %v\n", fa, serif, mono)
}

var testStrs = []FontStretch{FontStrNormal, FontStrCondensed, FontStrExpanded}
var testWts = []FontWeights{WeightNormal, WeightLight, WeightBold, WeightBlack}
var testStys = []FontStyles{FontNormal, FontItalic, FontOblique}
var testNms = []string{"serif", "sans-serif", "monospace", "courier", "cursive", "fantasy"}

// func TestFontFaceName(t *testing.T) {
// 	for _, nm := range testNms {
// 		for _, str := range testStrs {
// 			for _, wt := range testWts {
// 				for _, sty := range testStys {
// 					fn := FontFaceName(nm, str, wt, sty)
// 					fmt.Printf("FontName: nm:\t%v\t str:\t%v\t wt:\t%v\t sty:\t%v\t res:\t%v\n", nm, str, wt, sty, fn)
// 				}
// 			}
// 		}
// 	}
// }
