// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package shapedjs

import (
	"strings"
	"sync"
	"syscall/js"

	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
)

// FontsChanged, if non-nil, is called when web fonts have finished loading
// that change the measurements of texts that were measured before (with a
// fallback font), whose measurements the shapers have discarded, so that
// the texts can be laid out again.
var FontsChanged func()

var (
	// shapers are the shapers made by [NewShaper], whose cached texts are
	// discarded when the fonts that they were measured with have loaded.
	shapers []*Shaper

	// shapersMu protects shapers.
	shapersMu sync.Mutex
)

// listenFontLoads adds the listener for web fonts that finish loading.
func listenFontLoads() {
	doc := js.Global().Get("document")
	if !doc.Truthy() {
		return
	}
	fonts := doc.Get("fonts")
	if !fonts.Truthy() {
		return
	}
	fonts.Call("addEventListener", "loadingdone", js.FuncOf(func(this js.Value, args []js.Value) any {
		var faces []webFace
		if len(args) > 0 {
			faces = webFaces(args[0].Get("fontfaces"))
		}
		go fontsLoaded(faces)
		return nil
	}))
}

// webFaces returns the faces of the given FontFace objects.
func webFaces(ffs js.Value) []webFace {
	if !ffs.Truthy() {
		return nil
	}
	n := ffs.Length()
	faces := make([]webFace, 0, n)
	for i := range n {
		ff := ffs.Index(i)
		style := ff.Get("style").String()
		faces = append(faces, webFace{
			family: strings.Trim(ff.Get("family").String(), `"' `),
			italic: strings.HasPrefix(style, "italic") || strings.HasPrefix(style, "oblique"),
			ranges: parseUnicodeRange(ff.Get("unicodeRange").String()),
		})
	}
	return faces
}

// fontsLoaded discards the measurements of the texts and glyphs that the
// given faces, which have finished loading, change, as they were measured
// with a fallback font, and calls [FontsChanged] if there were any.
func fontsLoaded(faces []webFace) {
	if len(faces) == 0 {
		return
	}

	// The cached texts of each shaper with runes of the faces are discarded.
	shapersMu.Lock()
	shs := shapers
	shapersMu.Unlock()
	changed := 0
	for _, sh := range shs {
		sh.Lock()
		changed += sh.ForgetTexts(func(tx rich.Text, customFont rich.FontName) bool {
			return textUsesFaces(tx, customFont, faces)
		})
		sh.Unlock()
	}

	// So are the measured glyphs of the fonts of the faces.
	changed += theGlyphCache.forget(func(fn *text.Font) bool {
		var sty rich.Style
		rich.RuneToStyle(&sty, fn.StyleRune)
		families := fn.Family
		if families == "" {
			families = sty.FontFamily(&rich.Settings)
		}
		for i := range faces {
			if faces[i].usedBy(families, sty.Slant == rich.Italic) {
				return true
			}
		}
		return false
	})
	if changed > 0 && FontsChanged != nil {
		FontsChanged()
	}
}
