// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shapedgt

import (
	"math"
	"slices"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
	"github.com/go-text/typesetting/di"
	"github.com/go-text/typesetting/shaping"
)

// maxShapeCache is the number of texts that the shape cache holds before
// it starts over.
const maxShapeCache = 8192

// shapeCache caches the shaped outputs of texts. The layout of a text wraps
// it several times (sizing up and then down, in more than one iteration), at
// different sizes, and pages repeat texts (such as the cells of a table), but
// the shaping of a text does not depend on the size it is wrapped to. Shaping
// is the main cost of wrapping, so the cache saves most of it.
type shapeCache struct {

	// entries are the cached texts by hash, with hash collisions chained.
	entries map[uint32]*shapeEntry

	// n is the number of entries.
	n int

	// settings are the [rich.Settings] that the entries were shaped with.
	settings rich.SettingsData
}

// shapeEntry is the shaped outputs of one text in a [shapeCache].
type shapeEntry struct {

	// tx is a copy of the source text.
	tx rich.Text

	// fontSize, direction and customFont are the [text.Style] values that
	// shaping depends on.
	fontSize   float32
	direction  rich.Directions
	customFont rich.FontName

	// outs are the shaped outputs, which callers get copies of.
	outs []shaping.Output

	// wraps are the most recent wrappings of the outputs.
	wraps []wrapEntry

	// next is the next entry with the same hash.
	next *shapeEntry
}

// maxWraps is the number of wrappings of a text that a [shapeEntry] holds.
const maxWraps = 4

// wrapEntry is the lines of a text wrapped with a configuration and size.
type wrapEntry struct {
	maxSize       int
	direction     di.Direction
	truncateAfter int
	policy        shaping.LineBreakPolicy
	noTrim        bool
	lines         []shaping.Line
	truncated     int
}

// ResetCache empties the cache of shaped and wrapped texts, as when the
// fonts that they are measured with change.
func (sh *Shaper) ResetCache() {
	sh.shapes.reset()
}

// reset empties the cache.
func (sc *shapeCache) reset() {
	sc.entries = nil
	sc.n = 0
}

// hashText returns the hash of the text and the style values that shaping
// depends on (FNV-1a).
func hashText(tx rich.Text, tsty *text.Style) uint32 {
	h := uint32(2166136261)
	for _, s := range tx {
		h = (h ^ uint32(len(s))) * 16777619
		for _, r := range s {
			h = (h ^ uint32(r)) * 16777619
		}
	}
	h = (h ^ math.Float32bits(tsty.FontSize.Dots)) * 16777619
	return (h ^ uint32(tsty.Direction)) * 16777619
}

// matches returns whether the entry is for the given text and style.
func (se *shapeEntry) matches(tx rich.Text, tsty *text.Style) bool {
	if se.fontSize != tsty.FontSize.Dots || se.direction != tsty.Direction || se.customFont != tsty.CustomFont || len(se.tx) != len(tx) {
		return false
	}
	for i, s := range tx {
		if !slices.Equal(se.tx[i], s) {
			return false
		}
	}
	return true
}

// get returns the cache entry for the given text and style, with its hash,
// or nil if it is not cached.
func (sc *shapeCache) get(tx rich.Text, tsty *text.Style) (*shapeEntry, uint32) {
	// Entries shaped with other global settings no longer apply.
	if sc.settings != rich.Settings {
		sc.reset()
		sc.settings = rich.Settings
	}

	// The entry for the text, if any, is in the chain of its hash.
	h := hashText(tx, tsty)
	for se := sc.entries[h]; se != nil; se = se.next {
		if se.matches(tx, tsty) {
			return se, h
		}
	}
	return nil, h
}

// add adds an entry with a copy of the outputs of the given text and style,
// with the hash that get returned.
func (sc *shapeCache) add(tx rich.Text, tsty *text.Style, h uint32, outs []shaping.Output) *shapeEntry {
	// A full cache starts over.
	if sc.entries == nil || sc.n >= maxShapeCache {
		sc.entries = make(map[uint32]*shapeEntry)
		sc.n = 0
	}

	// The entry has its own copies of the text and the outputs.
	se := &shapeEntry{tx: make(rich.Text, len(tx)), fontSize: tsty.FontSize.Dots, direction: tsty.Direction, customFont: tsty.CustomFont}
	for i, s := range tx {
		se.tx[i] = slices.Clone(s)
	}
	se.outs = copyOutputs(outs)
	se.next = sc.entries[h]
	sc.entries[h] = se
	sc.n++
	return se
}

// copyOutputs returns a copy of the outputs with their own glyphs, as line
// wrapping changes the glyphs at the ends of lines.
func copyOutputs(outs []shaping.Output) []shaping.Output {
	cp := slices.Clone(outs)
	for i := range cp {
		cp[i].Glyphs = slices.Clone(cp[i].Glyphs)
	}
	return cp
}

// cacheable returns whether the shaped outputs of the text can be cached,
// which they cannot when the text is empty or has math, as shaping math
// also makes the math paths.
func cacheable(tx rich.Text) bool {
	if tx.Len() == 0 {
		return false
	}
	for si := range tx {
		if sty, _ := tx.Span(si); sty.IsMath() {
			return false
		}
	}
	return true
}

// WrapLinesCached returns the lines of the text wrapped to the given size,
// as [Shaper.WrapLinesOutput] wraps the outputs of [Shaper.ShapeTextOutput],
// using a cache of the shaped outputs and wrapped lines of recent texts.
// The adjust function, if non-nil, adjusts the outputs of newly shaped texts
// and of newly wrapped lines, as shapedjs does with the measurements of the
// browser. As it adjusts the outputs of wrapped lines anew, the outputs of
// texts that fit on one line keep their adjustments, without trimming their
// trailing whitespace. The lines are the caller's to change.
// This should already have the mutex lock, and is used by shapedjs but is
// not an end-user call.
func (sh *Shaper) WrapLinesCached(tx rich.Text, defSty *rich.Style, tsty *text.Style, size math32.Vector2, adjust func(outs []shaping.Output)) ([]shaping.Line, int) {
	txt := tx.Join()
	cfg, maxSize := wrapConfig(defSty, tsty, size)

	// Texts that cannot be cached are shaped and wrapped every time.
	if !cacheable(tx) {
		outs := sh.ShapeTextOutput(tx, tsty, txt)
		if adjust != nil {
			adjust(outs)
		}
		return sh.wrapOutputs(outs, txt, cfg, maxSize, adjust)
	}

	// The text is shaped unless it is cached.
	se, h := sh.shapes.get(tx, tsty)
	if se == nil {
		outs := sh.ShapeTextOutput(tx, tsty, txt)
		if adjust != nil {
			adjust(outs)
		}
		se = sh.shapes.add(tx, tsty, h, outs)
	}

	// The text is wrapped unless it is cached wrapped to this size.
	if lines, truncated, ok := se.wrapped(cfg, maxSize); ok {
		return lines, truncated
	}
	lines, truncated := sh.wrapOutputs(copyOutputs(se.outs), txt, cfg, maxSize, adjust)
	se.addWrap(cfg, maxSize, lines, truncated)
	return lines, truncated
}

// wrapOutputs wraps the outputs into lines, adjusting the outputs of the
// lines with adjust if non-nil (see [Shaper.WrapLinesCached]). The lines
// are only valid until the next call.
func (sh *Shaper) wrapOutputs(outs []shaping.Output, txt []rune, cfg shaping.WrapConfig, maxSize int, adjust func(outs []shaping.Output)) ([]shaping.Line, int) {
	// Text that fits on one line does not need the line wrapper.
	if lines, ok := singleLine(outs, txt, cfg, maxSize, adjust == nil && !cfg.DisableTrailingWhitespaceTrim); ok {
		return lines, 0
	}

	// Wrapping can cut outputs, which are adjusted again.
	lines, truncated := sh.wrapParagraph(cfg, maxSize, txt, outs)
	if adjust != nil {
		for _, ln := range lines {
			adjust(ln)
		}
	}
	return lines, truncated
}

// wrapped returns a copy of the cached lines of the entry wrapped with the
// given configuration and size, and whether they are cached.
func (se *shapeEntry) wrapped(cfg shaping.WrapConfig, maxSize int) ([]shaping.Line, int, bool) {
	for i := range se.wraps {
		if we := &se.wraps[i]; we.matches(cfg, maxSize) {
			return copyLines(we.lines), we.truncated, true
		}
	}
	return nil, 0, false
}

// addWrap adds a copy of the lines of the entry wrapped with the given
// configuration and size, replacing the oldest one when it has maxWraps.
func (se *shapeEntry) addWrap(cfg shaping.WrapConfig, maxSize int, lines []shaping.Line, truncated int) {
	we := wrapEntry{maxSize: maxSize, direction: cfg.Direction, truncateAfter: cfg.TruncateAfterLines, policy: cfg.BreakPolicy, noTrim: cfg.DisableTrailingWhitespaceTrim, lines: copyLines(lines), truncated: truncated}
	if len(se.wraps) < maxWraps {
		se.wraps = append(se.wraps, we)
		return
	}
	copy(se.wraps, se.wraps[1:])
	se.wraps[maxWraps-1] = we
}

// matches returns whether the wrapped lines are for the given configuration and size.
func (we *wrapEntry) matches(cfg shaping.WrapConfig, maxSize int) bool {
	return we.maxSize == maxSize && we.direction == cfg.Direction && we.truncateAfter == cfg.TruncateAfterLines && we.policy == cfg.BreakPolicy && we.noTrim == cfg.DisableTrailingWhitespaceTrim && !cfg.TextContinues && len(cfg.Truncator.Glyphs) == 0
}

// copyLines returns a copy of the lines with their own outputs and glyphs.
func copyLines(lines []shaping.Line) []shaping.Line {
	cp := make([]shaping.Line, len(lines))
	for i, ln := range lines {
		cp[i] = copyOutputs(ln)
	}
	return cp
}
