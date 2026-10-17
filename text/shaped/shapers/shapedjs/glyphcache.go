// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package shapedjs

import (
	"sync"
	"syscall/js"

	"cogentcore.org/core/text/text"
	"github.com/go-text/typesetting/font"
)

var theGlyphCache glyphCache

func init() {
	theGlyphCache.init()
}

// glyphCache caches glyph sizing data by font and rune
type glyphCache struct {
	glyphs map[text.Font]map[font.GID]*Metrics

	// last is the font of the last lookup and lastGlyphs its glyphs, as
	// consecutive lookups are nearly always in the same font, which saves
	// looking up the font in glyphs.
	last       text.Font
	lastGlyphs map[font.GID]*Metrics

	sync.Mutex
}

func (gc *glyphCache) init() {
	gc.glyphs = make(map[text.Font]map[font.GID]*Metrics)
}

// Glyph returns the metric data for given GID in given font.
func (gc *glyphCache) Glyph(ctx js.Value, fn *text.Font, tsty *text.Style, tx []rune, gid font.GID) *Metrics {
	gc.Lock()
	defer gc.Unlock()

	// The glyphs of the font are those of the last lookup, or looked up.
	fc := gc.lastGlyphs
	if fc == nil || gc.last != *fn {
		fc = gc.glyphs[*fn]
		if fc == nil {
			fc = make(map[font.GID]*Metrics)
			gc.glyphs[*fn] = fc
		}
		gc.last, gc.lastGlyphs = *fn, fc
	}

	// A glyph not measured yet is measured.
	if g, ok := fc[gid]; ok {
		return g
	}
	g := gc.measureGlyph(ctx, fn, tsty, tx)
	fc[gid] = g
	return g
}

func (gc *glyphCache) measureGlyph(ctx js.Value, fn *text.Font, tsty *text.Style, tx []rune) *Metrics {
	SetFontStyle(ctx, fn, tsty, 0)
	return MeasureText(ctx, string(tx))
}
