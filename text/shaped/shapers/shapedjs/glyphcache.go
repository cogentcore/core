// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package shapedjs

import (
	"sync"
	"syscall/js"

	"cogentcore.org/core/text/text"
)

var theGlyphCache glyphCache

func init() {
	theGlyphCache.init()
}

// glyphCache caches the measurements of glyph clusters by font and text.
// They are cached by text, not by glyph id, as the runes of a font can be
// shaped with glyphs of other (fallback) faces, whose glyph ids overlap.
type glyphCache struct {
	fonts map[text.Font]*fontGlyphs

	sync.Mutex
}

// fontGlyphs are the glyph measurements of one font: those of clusters of
// one rune by rune, and of the others by their text, and the widths of words.
type fontGlyphs struct {
	runes    map[rune]*Metrics
	clusters map[string]*Metrics
	words    map[string]float32
}

func (gc *glyphCache) init() {
	gc.fonts = make(map[text.Font]*fontGlyphs)
}

// forget forgets the measurements of the fonts for which forget returns
// true, as when their fonts have loaded, and returns how many fonts it forgot.
func (gc *glyphCache) forget(forget func(fn *text.Font) bool) int {
	gc.Lock()
	defer gc.Unlock()
	n := 0
	for fn := range gc.fonts {
		if forget(&fn) {
			delete(gc.fonts, fn)
			n++
		}
	}
	return n
}

// font returns the glyph measurements of the given font, which are only to
// be used with the lock.
func (gc *glyphCache) font(fn *text.Font) *fontGlyphs {
	fg := gc.fonts[*fn]
	if fg == nil {
		fg = &fontGlyphs{runes: make(map[rune]*Metrics)}
		gc.fonts[*fn] = fg
	}
	return fg
}

// glyph returns the measurements of the given glyph cluster text, measuring
// it with the font set on the context if it is not cached.
func (fg *fontGlyphs) glyph(ctx js.Value, tx []rune) *Metrics {
	if len(tx) == 1 {
		if g, ok := fg.runes[tx[0]]; ok {
			return g
		}
		g := MeasureText(ctx, string(tx))
		fg.runes[tx[0]] = g
		return g
	}
	s := string(tx)
	if g, ok := fg.clusters[s]; ok {
		return g
	}
	if fg.clusters == nil {
		fg.clusters = make(map[string]*Metrics)
	}
	g := MeasureText(ctx, s)
	fg.clusters[s] = g
	return g
}

// word returns the width of the given word, measuring it with the font set
// on the context if it is not cached.
func (fg *fontGlyphs) word(ctx js.Value, word []rune) float32 {
	s := string(word)
	if w, ok := fg.words[s]; ok {
		return w
	}
	if fg.words == nil || len(fg.words) >= maxWords {
		fg.words = make(map[string]float32)
	}
	w := float32(ctx.Call("measureText", s).Get("width").Float())
	fg.words[s] = w
	return w
}

// maxWords is the number of word widths that a font keeps before it starts over.
const maxWords = 16384
