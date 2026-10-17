// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shapedgt

import (
	"math"
	"slices"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/shaped"
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

	// bounds are the most recent lines with bounds of the wrapped lines.
	bounds []boundsEntry
}

// maxBounds is the number of lines with bounds that a [wrapEntry] holds.
const maxBounds = 2

// boundsKey has the values other than those of the [shapeEntry] and the
// [wrapEntry] that [Shaper.LinesBounds] depends on, except for the colors,
// which are set on each copy.
type boundsKey struct {
	width      float32 // the exact width, of which the wrap size is the integer part
	defSize    float32 // the size of the default style
	lineHeight float32
	whiteSpace text.WhiteSpaces
	align      text.Aligns
}

// boundsEntry is the lines with bounds of wrapped lines for a [boundsKey].
type boundsEntry struct {
	key   boundsKey
	lines *shaped.Lines
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
	lines, truncated, _, cached := sh.wrapCached(tx, defSty, tsty, size, adjust)
	if cached {
		lines = copyLines(lines)
	}
	return lines, truncated
}

// wrapCached is [Shaper.WrapLinesCached], also returning the cache entry of
// the wrapping, which is nil when it is not cached, and whether the lines are
// those of the entry, which were wrapped before and are not the caller's to change.
func (sh *Shaper) wrapCached(tx rich.Text, defSty *rich.Style, tsty *text.Style, size math32.Vector2, adjust func(outs []shaping.Output)) ([]shaping.Line, int, *wrapEntry, bool) {
	cfg, maxSize := wrapConfig(defSty, tsty, size)

	// Texts that cannot be cached are shaped and wrapped every time.
	if !cacheable(tx) {
		txt := tx.Join()
		outs := sh.ShapeTextOutput(tx, tsty, txt)
		if adjust != nil {
			adjust(outs)
		}
		lines, truncated := sh.wrapOutputs(outs, txt, cfg, maxSize, adjust)
		return lines, truncated, nil, false
	}

	// The text is shaped unless it is cached, and its runes are only needed to wrap it.
	var txt []rune
	se, h := sh.shapes.get(tx, tsty)
	if se == nil {
		txt = tx.Join()
		outs := sh.ShapeTextOutput(tx, tsty, txt)
		if adjust != nil {
			adjust(outs)
		}
		se = sh.shapes.add(tx, tsty, h, outs)
	}

	// The text is wrapped unless it is cached wrapped to this size.
	if we := se.wrapEntry(cfg, maxSize); we != nil {
		return we.lines, we.truncated, we, true
	}
	if txt == nil {
		txt = tx.Join()
	}
	lines, truncated := sh.wrapOutputs(copyOutputs(se.outs), txt, cfg, maxSize, adjust)
	return lines, truncated, se.addWrap(cfg, maxSize, lines, truncated), false
}

// WrapLinesBoundsCached returns the lines with bounds of the text wrapped to
// the given size, as [Shaper.LinesBounds] makes them from the lines of
// [Shaper.WrapLinesCached], using the cache of recent texts, which also
// keeps the lines with bounds of their recent wrappings. The adjust function
// is as for [Shaper.WrapLinesCached]. The lines are the caller's to change.
// This should already have the mutex lock, and is used by shapedjs but is
// not an end-user call.
func (sh *Shaper) WrapLinesBoundsCached(tx rich.Text, defSty *rich.Style, tsty *text.Style, size math32.Vector2, adjust func(outs []shaping.Output)) *shaped.Lines {
	// Lines of wrappings that are not cached are made every time.
	lines, truncated, we, cached := sh.wrapCached(tx, defSty, tsty, size, adjust)
	if we == nil {
		return sh.LinesBounds(lines, truncated, tx, defSty, tsty, size)
	}

	// Cached lines with bounds are copied.
	key := boundsKey{width: size.X, defSize: defSty.Size, lineHeight: tsty.LineHeight, whiteSpace: tsty.WhiteSpace, align: tsty.Align}
	for i := range we.bounds {
		if be := &we.bounds[i]; be.key == key {
			return copyBounds(be.lines, tx, tsty)
		}
	}

	// New bounds are kept as a copy, replacing the oldest ones when there are maxBounds.
	if cached {
		lines = copyLines(lines)
	}
	lns := sh.LinesBounds(lines, truncated, tx, defSty, tsty, size)
	be := boundsEntry{key: key, lines: copyBounds(lns, nil, tsty)}
	if len(we.bounds) < maxBounds {
		we.bounds = append(we.bounds, be)
	} else {
		copy(we.bounds, we.bounds[1:])
		we.bounds[maxBounds-1] = be
	}
	return lns
}

// copyBounds returns a copy of the lines with bounds, with their own runs
// and spans, for the given source text and the colors of the given style.
// The runs share the glyphs, which [Run.SetGlyphXAdvance] copies before
// changing them, the only change made to the glyphs of shaped lines.
func copyBounds(lns *shaped.Lines, tx rich.Text, tsty *text.Style) *shaped.Lines {
	cp := *lns
	cp.Source = tx
	cp.Color, cp.SelectionColor, cp.HighlightColor = tsty.Color, tsty.SelectColor, tsty.HighlightColor
	cp.Lines = slices.Clone(lns.Lines)
	for li := range cp.Lines {
		ln := &cp.Lines[li]
		ln.Source = slices.Clone(ln.Source)
		for si := range ln.Source {
			ln.Source[si] = slices.Clone(ln.Source[si])
		}
		ln.Runs = slices.Clone(ln.Runs)
		for ri, r := range ln.Runs {
			run := *r.(*Run)
			ln.Runs[ri] = &run
		}
	}
	return &cp
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

// wrapEntry returns the cached wrapping of the entry with the given
// configuration and size, or nil if it is not cached.
func (se *shapeEntry) wrapEntry(cfg shaping.WrapConfig, maxSize int) *wrapEntry {
	for i := range se.wraps {
		if we := &se.wraps[i]; we.matches(cfg, maxSize) {
			return we
		}
	}
	return nil
}

// addWrap adds a copy of the lines of the entry wrapped with the given
// configuration and size, replacing the oldest one when it has maxWraps,
// and returns it, or nil when wrappings with the configuration are not cached.
func (se *shapeEntry) addWrap(cfg shaping.WrapConfig, maxSize int, lines []shaping.Line, truncated int) *wrapEntry {
	we := wrapEntry{maxSize: maxSize, direction: cfg.Direction, truncateAfter: cfg.TruncateAfterLines, policy: cfg.BreakPolicy, noTrim: cfg.DisableTrailingWhitespaceTrim, lines: copyLines(lines), truncated: truncated}
	if !we.matches(cfg, maxSize) {
		return nil
	}
	if len(se.wraps) < maxWraps {
		se.wraps = append(se.wraps, we)
		return &se.wraps[len(se.wraps)-1]
	}
	copy(se.wraps, se.wraps[1:])
	se.wraps[maxWraps-1] = we
	return &se.wraps[maxWraps-1]
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
