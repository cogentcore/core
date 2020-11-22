// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package girl

import (
	"fmt"
	"testing"

	"github.com/goki/gi/gist"
)

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

func TestFontFaceName(t *testing.T) {
	return // skip as very verbose
	for _, nm := range testNms {
		for _, str := range testStrs {
			for _, wt := range testWts {
				for _, sty := range testStys {
					fn := FontFaceName(nm, str, wt, sty)
					fmt.Printf("FontName: nm:\t%v\t str:\t%v\t wt:\t%v\t sty:\t%v\t res:\t%v\n", nm, str, wt, sty, fn)
				}
			}
		}
	}
}
