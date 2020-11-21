// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package girl

import (
	"fmt"
	"testing"

	"github.com/goki/gi/gist"
)

type testFontSpec struct {
	fn  string
	cor string
	str gist.FontStretch
	wt  gist.FontWeights
	sty gist.FontStyles
}

var testFontNames = []testFontSpec{
	{"NotoSansBlack", "NotoSans Black", gist.FontStrNormal, gist.WeightBlack, gist.FontNormal},
	{"NotoSansBlackItalic", "NotoSans Black Italic", gist.FontStrNormal, gist.WeightBlack, gist.FontItalic},
	{"NotoSansBold", "NotoSans Bold", gist.FontStrNormal, gist.WeightBold, gist.FontNormal},
	{"NotoSansCondensed", "NotoSans Condensed", gist.FontStrCondensed, gist.WeightNormal, gist.FontNormal},
	{"NotoSansCondensedBlack", "NotoSans Condensed Black", gist.FontStrCondensed, gist.WeightBlack, gist.FontNormal},
	{"NotoSansCondensedBlackItalic", "NotoSans Condensed Black Italic", gist.FontStrCondensed, gist.WeightBlack, gist.FontItalic},
	{"NotoSansCondensedExtraBold", "NotoSans Condensed ExtraBold", gist.FontStrCondensed, gist.WeightExtraBold, gist.FontNormal},
	{"NotoSansCondensedExtraBoldItalic", "NotoSans Condensed ExtraBold Italic", gist.FontStrCondensed, gist.WeightExtraBold, gist.FontItalic},
	{"NotoSansExtraBold", "NotoSans ExtraBold", gist.FontStrNormal, gist.WeightExtraBold, gist.FontNormal},
	{"NotoSansExtraBoldItalic", "NotoSans ExtraBold Italic", gist.FontStrNormal, gist.WeightExtraBold, gist.FontItalic},
	{"NotoSansRegular", "NotoSans", gist.FontStrNormal, gist.WeightNormal, gist.FontNormal},
	{"NotoSansNormal", "NotoSans", gist.FontStrNormal, gist.WeightNormal, gist.FontNormal},
}

func TestFontMods(t *testing.T) {
	for _, ft := range testFontNames {
		fo := FixFontMods(ft.fn)
		if fo != ft.cor {
			t.Errorf("FixFontMods output: %v != correct: %v for font: %v\n", fo, ft.cor, ft.fn)
		}

		base, str, wt, sty := gist.FontNameToMods(fo)
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

		frc := gist.FontNameFromMods(base, str, wt, sty)
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

var testStrs = []gist.FontStretch{gist.FontStrNormal, gist.FontStrCondensed, gist.FontStrExpanded}
var testWts = []gist.FontWeights{gist.WeightNormal, gist.WeightLight, gist.WeightBold, gist.WeightBlack}
var testStys = []gist.FontStyles{gist.FontNormal, gist.FontItalic, gist.FontOblique}
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
