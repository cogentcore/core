// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fonts

import (
	"bytes"
	"strings"
	"sync"

	"github.com/cogentcore/typesetting/font/opentype"
)

// WebFont is a font that is added at run time, with the family name that
// it is used with (as the web fonts of the @font-face rules of HTML pages).
type WebFont struct {
	// Family is the family name that the font is used with.
	Family string

	// Data is the data of the font, as TrueType, OpenType or WOFF.
	Data []byte
}

// webFonts are the fonts that are added with [AddWebFont].
var webFonts struct {
	sync.Mutex
	fonts    []WebFont
	families map[string]bool
}

// AddWebFont adds the given font data (TrueType, OpenType, or WOFF; see
// also [WebFont]) with the given family name, which is used for the given
// family by the text shapers (which add it when they next shape text),
// for all of the faces of the data. It returns an error if the data is
// not a font.
func AddWebFont(family string, data []byte) error {
	if _, err := opentype.NewLoaders(bytes.NewReader(data)); err != nil {
		return err
	}
	webFonts.Lock()
	defer webFonts.Unlock()
	if webFonts.families == nil {
		webFonts.families = map[string]bool{}
	}
	webFonts.families[strings.ToLower(family)] = true
	webFonts.fonts = append(webFonts.fonts, WebFont{Family: family, Data: data})
	return nil
}

// HasWebFont returns whether a font has been added for the given family
// with [AddWebFont] (ignoring case).
func HasWebFont(family string) bool {
	webFonts.Lock()
	defer webFonts.Unlock()
	return webFonts.families[strings.ToLower(family)]
}

// WebFonts returns the fonts added with [AddWebFont] after the first n, and
// the number of fonts added in all, with which a shaper adds the fonts that
// it does not have yet.
func WebFonts(n int) ([]WebFont, int) {
	webFonts.Lock()
	defer webFonts.Unlock()
	if n >= len(webFonts.fonts) {
		return nil, len(webFonts.fonts)
	}
	return webFonts.fonts[n:], len(webFonts.fonts)
}
