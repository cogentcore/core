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

// Info contains basic font information for aviailable fonts.
// This is used for a chooser for example.
type Info struct {

	// Family name.
	Family string

	// Weight: normal, bold, etc
	Weight rich.Weights

	// Slant: normal or italic
	Slant rich.Slants

	// Stretch: normal, expanded, condensed, etc
	Stretch rich.Stretch

	// Font contains info about the location, family, etc of the font file.
	Font fontscan.Footprint `display:"-"`
}

// Family is used for selecting a font family in a font chooser.
type Family struct {

	// Family name.
	Family string

	// example text, styled according to font family in chooser.
	Example string
}

// InfoExample is example text to demonstrate fonts.
var InfoExample = "AaBbCcIiPpQq12369$€¢?.:/()àáâãäåæç日本中国⇧⌘"

// Label satisfies the Labeler interface
func (fi Info) Label() string {
	return fi.Family
}

// Label satisfies the Labeler interface
func (fi Family) Label() string {
	return fi.Family
}

// Families returns a list of [Family] with one representative per family.
func Families(fi []Info) []Family {
	slices.SortFunc(fi, func(a, b Info) int {
		return cmp.Compare(a.Family, b.Family)
	})
	n := len(fi)
	ff := make([]Family, 0, n)
	for i := 0; i < n; i++ {
		cur := fi[i].Family
		ff = append(ff, Family{Family: cur, Example: InfoExample})
		for i < n-1 {
			if fi[i+1].Family != cur {
				break
			}
			i++
		}
	}
	return ff
}

// Data contains font information for embedded font data.
type Data struct {

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
func (fd *Data) Load() error {
	faces, err := font.ParseTTC(bytes.NewReader(fd.Data))
	if err != nil {
		return err
	}
	fd.Fonts = faces
	return nil
}
