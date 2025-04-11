// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fonts

import (
	"bytes"
	"cmp"
	"slices"

	"cogentcore.org/core/text/rich"
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/fontscan"
)

// FontInfo contains basic font information for aviailable fonts.
type FontInfo struct {

	// Family name.
	Family string

	// Weight: normal, bold, etc
	Weight rich.Weights

	// Slant: normal or italic
	Slant rich.Slants

	// Stretch: normal, expanded, condensed, etc
	Stretch rich.Stretch

	// Font contains info about
	Font fontscan.Footprint `display:"-"`
}

// FontFamily is used for selecting a font family in a font chooser.
type FontFamily struct {

	// Family name.
	Family string

	// example text, styled according to font family in chooser.
	Example string
}

// FontInfoExample is example text to demonstrate fonts.
var FontInfoExample = "AaBbCcIiPpQq12369$€¢?.:/()àáâãäåæç日本中国⇧⌘"

// Label satisfies the Labeler interface
func (fi FontInfo) Label() string {
	return fi.Family
}

// Label satisfies the Labeler interface
func (fi FontFamily) Label() string {
	return fi.Family
}

// FontFamilies returns a list of FontFamily with one representative per family.
func FontFamilies(fi []FontInfo) []FontFamily {
	slices.SortFunc(fi, func(a, b FontInfo) int {
		return cmp.Compare(a.Family, b.Family)
	})
	n := len(fi)
	ff := make([]FontFamily, 0, n)
	for i := 0; i < n; i++ {
		cur := fi[i].Family
		ff = append(ff, FontFamily{Family: cur, Example: FontInfoExample})
		for i < n-1 {
			if fi[i+1].Family != cur {
				break
			}
			i++
		}
	}
	return ff
}

// FontData contains font information for embedded font data.
type FontData struct {

	// Family name.
	Family string

	// Weight: normal, bold, etc.
	Weight rich.Weights

	// Slant: normal or italic.
	Slant rich.Slants

	// Stretch: normal, expanded, condensed, etc.
	Stretch rich.Stretch

	// Data contains the font data.
	Data []byte `display:"-"`

	// Font contains the loaded font face(s).
	Fonts []*font.Face
}

// Load loads the data, setting the Font.
func (fd *FontData) Load() error {
	faces, err := font.ParseTTC(bytes.NewReader(fd.Data))
	if err != nil {
		return err
	}
	fd.Fonts = faces
	return nil
}
