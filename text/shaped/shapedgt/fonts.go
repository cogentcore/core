// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shapedgt

import (
	"os"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/text/rich"
	"github.com/go-text/typesetting/fontscan"
)

// FontInfo contains basic font information for choosing a given font.
// displayed in the font chooser dialog.
type FontInfo struct {

	// official regularized name of font
	Name string

	// weight: normal, bold, etc
	Weight rich.Weights

	// slant: normal or italic
	Slant rich.Slants

	// stretch: normal, expanded, condensed, etc
	Stretch rich.Stretch

	// example text -- styled according to font params in chooser
	Example string
}

// FontInfoExample is example text to demonstrate fonts -- from Inkscape plus extra
var FontInfoExample = "AaBbCcIiPpQq12369$€¢?.:/()àáâãäåæç日本中国⇧⌘"

// Label satisfies the Labeler interface
func (fi FontInfo) Label() string {
	return fi.Name
}

// FontList returns the list of fonts that have been loaded.
func FontList() []FontInfo {
	str := errors.Log1(os.UserCacheDir())
	ft := errors.Log1(fontscan.SystemFonts(nil, str))
	fi := make([]FontInfo, len(ft))
	for i := range ft {
		fi[i].Name = ft[i].Family
		as := ft[i].Aspect
		fi[i].Weight = rich.Weights(int(as.Weight / 100.0))
		fi[i].Slant = rich.Slants(as.Style - 1)
		// fi[i].Stretch = rich.Stretch()
		fi[i].Example = FontInfoExample
	}
	return fi
}
