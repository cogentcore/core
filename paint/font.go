// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

import (
	"log"
	"math"
	"path/filepath"
	"strings"

	"github.com/goki/freetype/truetype"
	"goki.dev/colors"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/grr"
	"goki.dev/ki/v2"
	"golang.org/x/image/font/opentype"
)

// OpenFont loads the font specified by the font style from the font library.
// This is the primary method to use for loading fonts, as it uses a robust
// fallback method to finding an appropriate font, and falls back on the
// builtin Go font as a last resort.  It returns the font
// style object with Face set to the resulting font.
// The font size is always rounded to nearest integer, to produce
// better-looking results (presumably).  The current metrics and given
// unit.Context are updated based on the properties of the font.
func OpenFont(fs *styles.FontRender, ctxt *units.Context) styles.Font {
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
			face = grr.Log(FontLibrary.Font("Roboto", intDots)) // guaranteed to exist
			fs.Face = face
		}
	} else {
		fs.Face = face
	}
	fs.SetUnitContext(ctxt)
	return fs.Font
}

// OpenFontFace loads a font face from the given font file bytes, with the given
// name and path for context, with given raw size in display dots, and if
// strokeWidth is > 0, the font is drawn in outline form (stroked) instead of
// filled (supported in SVG). loadFontMu must be locked prior to calling.
func OpenFontFace(bytes []byte, name, path string, size int, strokeWidth int) (*styles.FontFace, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".otf" {
		// note: this compiles but otf fonts are NOT yet supported apparently
		f, err := opentype.Parse(bytes)
		if err != nil {
			return nil, err
		}
		face, err := opentype.NewFace(f, &opentype.FaceOptions{
			Size: float64(size),
			DPI:  72,
			// Hinting: font.HintingFull,
		})
		ff := styles.NewFontFace(name, size, face)
		return ff, err
	} else {
		f, err := truetype.Parse(bytes)
		if err != nil {
			return nil, err
		}
		face := truetype.NewFace(f, &truetype.Options{
			Size:   float64(size),
			Stroke: strokeWidth,
			// Hinting: font.HintingFull,
			// GlyphCacheEntries: 1024, // default is 512 -- todo benchmark
		})
		ff := styles.NewFontFace(name, size, face)
		return ff, nil
	}
}

// FontStyleCSS looks for "tag" name props in cssAgg props, and applies those to
// style if found, and returns true -- false if no such tag found
func FontStyleCSS(fs *styles.FontRender, tag string, cssAgg map[string]any, unit *units.Context, ctxt colors.Context) bool {
	if cssAgg == nil {
		return false
	}
	tp, ok := cssAgg[tag]
	if !ok {
		return false
	}
	pmap, ok := tp.(map[string]any) // must be a props map
	if ok {
		fs.SetStyleProps(nil, pmap, ctxt)
		OpenFont(fs, unit)
		return true
	}
	kmap, ok := tp.(ki.Props) // must be a props map
	if ok {
		fs.SetStyleProps(nil, kmap, ctxt)
		OpenFont(fs, unit)
		return true
	}
	return false
}
