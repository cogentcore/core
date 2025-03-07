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
	sync.Mutex
}

func (gc *glyphCache) init() {
	gc.glyphs = make(map[text.Font]map[font.GID]*Metrics)
}

// Glyph returns the metric data for given GID in given font.
func (gc *glyphCache) Glyph(ctx js.Value, fn *text.Font, tsty *text.Style, tx []rune, gid font.GID) *Metrics {
	gc.Lock()
	defer gc.Unlock()

	fc, hasfc := gc.glyphs[*fn]
	if hasfc {
		g, ok := fc[gid]
		if ok {
			return g
		}
	} else {
		fc = make(map[font.GID]*Metrics)
	}
	g := gc.measureGlyph(ctx, fn, tsty, tx)
	fc[gid] = g
	gc.glyphs[*fn] = fc
	return g
}

func (gc *glyphCache) measureGlyph(ctx js.Value, fn *text.Font, tsty *text.Style, tx []rune) *Metrics {
	SetFontStyle(ctx, fn, tsty, 0)
	return MeasureText(ctx, string(tx))
}
