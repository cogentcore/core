// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package girl

import (
	"io/ioutil"
	"log"
	"math"
	"path/filepath"
	"strings"

	"github.com/goki/freetype/truetype"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
)

// OpenFont loads the font specified by the font style from the font library.
// This is the primary method to use for loading fonts, as it uses a robust
// fallback method to finding an appropriate font, and falls back on the
// builtin Go font as a last resort.  The Face field will have the resulting
// font.  The font size is always rounded to nearest integer, to produce
// better-looking results (presumably).  The current metrics and given
// unit.Context are updated based on the properties of the font.
func OpenFont(fs *gist.Font, ctxt *units.Context) {
	facenm := FontFaceName(fs.Family, fs.Stretch, fs.Weight, fs.Style)
	if fs.Size.Dots == 0 {
		fs.Size.ToDots(ctxt)
	}
	intDots := int(math.Round(float64(fs.Size.Dots)))
	if intDots == 0 {
		// fmt.Printf("FontStyle Error: bad font size: %v or units context: %v\n", fs.Size, *ctxt)
		intDots = 12
	}
	face, err := FontLibrary.Font(facenm, intDots)
	if err != nil {
		log.Printf("%v\n", err)
		if fs.Face == nil {
			face, err = FontLibrary.Font("Go", intDots) // guaranteed to exist
			fs.Face = face
		}
	} else {
		fs.Face = face
	}
	fs.Rem = ctxt.ToDots(12, units.Pt)
	fs.SetUnitContext(ctxt)
}

// OpenFontFace loads a font file at given path, with given raw size in
// display dots, and if strokeWidth is > 0, the font is drawn in outline form
// (stroked) instead of filled (supported in SVG).
// loadFontMu must be locked prior to calling
func OpenFontFace(name, path string, size int, strokeWidth int) (*gist.FontFace, error) {
	if strings.HasPrefix(path, "gofont") {
		return OpenGoFont(name, path, size, strokeWidth)
	}
	fontBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".otf" {
		// note: this compiles but otf fonts are NOT yet supported apparently
		f, err := sfnt.Parse(fontBytes)
		if err != nil {
			return nil, err
		}
		face, err := opentype.NewFace(f, &opentype.FaceOptions{
			Size: float64(size),
			// Hinting: font.HintingFull,
		})
		ff := gist.NewFontFace(name, size, face)
		return ff, err
	} else {
		f, err := truetype.Parse(fontBytes)
		if err != nil {
			return nil, err
		}
		face := truetype.NewFace(f, &truetype.Options{
			Size:   float64(size),
			Stroke: strokeWidth,
			// Hinting: font.HintingFull,
			// GlyphCacheEntries: 1024, // default is 512 -- todo benchmark
		})
		ff := gist.NewFontFace(name, size, face)
		return ff, nil
	}
}

// FontStyleCSS looks for "tag" name props in cssAgg props, and applies those to
// style if found, and returns true -- false if no such tag found
func FontStyleCSS(fs *gist.Font, tag string, cssAgg ki.Props, unit *units.Context, ctxt gist.Context) bool {
	if cssAgg == nil {
		return false
	}
	tp, ok := cssAgg[tag]
	if !ok {
		return false
	}
	pmap, ok := tp.(ki.Props) // must be a props map
	if !ok {
		return false
	}
	fs.SetStyleProps(nil, pmap, ctxt)
	OpenFont(fs, unit)
	return true
}
