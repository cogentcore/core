// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gist

import (
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
