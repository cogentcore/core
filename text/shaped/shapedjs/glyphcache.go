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

// glyphCache caches glyph sizing data by font and rune
type glyphCache struct {
	glyphs map[Font]map[rune]*Glyph
	sync.Mutex
}

func (gc *glyphCache) init() {
	gc.glyphs = make(map[Font]map[rune]*Glyph)
}

// Glyph returns the metric data for given rune in given font.
func (gc *glyphCache) Glyph(ctx js.Value, fn *Font, tsty *text.Style, rn rune) *Glyph {
	gc.Lock()
	defer gc.Unlock()

	fc, hasfc := gc.glyphs[*fn]
	if hasfc {
		g, ok := fc[rn]
		if ok {
			return g
		}
	} else {
		fc = make(map[rune]*Glyph)
	}
	g := gc.measureGlyph(ctx, fn, tsty, rn)
	fc[rn] = g
	gc.glyphs[*fn] = fc
	return g
}

func (gc *glyphCache) measureGlyph(ctx js.Value, fn *Font, tsty *text.Style, rn rune) *Glyph {
	SetFontStyle(ctx, fn, tsty, 0)
	m := MeasureText(ctx, string([]rune{rn}))
	g := &Glyph{}
	g.Width = m.Width
	g.Height = m.ActualBoundingBoxAscent + m.ActualBoundingBoxDescent
	// todo: set values from meas
	return g
}
