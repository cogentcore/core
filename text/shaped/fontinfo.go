// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shaped

import "cogentcore.org/core/text/rich"

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
