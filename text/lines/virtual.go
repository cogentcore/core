// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import (
	"slices"

	"cogentcore.org/core/events"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/runes"
	"cogentcore.org/core/text/textpos"
)

// VirtualText is display-only text that is shown in the views of [Lines]
// at a source position, such as the composition of an input method,
// language server inlay hints, or suggested completions. It is not part
// of the text: it is not saved, edited, undone, searched, or sent to
// language servers, and the cursor cannot be positioned within it.
//
// Text typed at the position goes before the virtual text, which stays
// attached to the source text after it, and the cursor at the position
// is shown before it.
type VirtualText struct {

	// Pos is the source position that the text is shown at.
	Pos textpos.Pos

	// Text is the text, which can contain newlines, in which case
	// the following lines are shown as separate lines in the view,
	// with the source text after Pos following the last line.
	Text []rune

	// Style is the style of the text. If nil, the style of the
	// source text at Pos is used.
	Style *rich.Style
}

// virtualText is a [VirtualText] as stored for its source line.
type virtualText struct {

	// char is the source char position in the line.
	char int

	// lines are the lines of the text, split at newlines.
	lines [][]rune

	// style is the style of the text, or nil to use the source text style.
	style *rich.Style

	// layer is the layer of the text (see [Lines.SetVirtualText]).
	layer string

	// view is the view that the text is shown in, or -1 for all views.
	view int
}

// len returns the number of display chars of the text,
// which does not include newlines.
func (vt *virtualText) len() int {
	n := 0
	for _, l := range vt.lines {
		n += len(l)
	}
	return n
}

// shownIn returns whether the text is shown in the given view.
func (vt *virtualText) shownIn(vw *view) bool {
	return vt.view < 0 || vt.view == vw.id
}

// SetVirtualText replaces all of the virtual text in the given layer for
// the given view (or all views if vid < 0) with the given texts, such as
// "ime", "inlay" or "suggestion". Texts at the same position are shown
// in order. The views are laid out again and sent a virtual text event
// (see [Lines.OnVirtualText]), but not a Change or Input event, as the
// text has not changed.
func (ls *Lines) SetVirtualText(layer string, vid int, vts ...VirtualText) {
	ls.Lock()
	ls.setVirtualText(layer, vid, vts...)
	ls.Unlock()
	ls.sendVirtualText()
}

// ClearVirtualText removes all of the virtual text in the given
// layer for the given view (or all views if vid < 0).
func (ls *Lines) ClearVirtualText(layer string, vid int) {
	ls.SetVirtualText(layer, vid)
}

// VirtualText returns the virtual text in the given layer for the
// given view (or all views if vid < 0), with positions updated for any
// edits since it was set, in order.
func (ls *Lines) VirtualText(layer string, vid int) []VirtualText {
	ls.Lock()
	defer ls.Unlock()
	var vts []VirtualText
	for ln, lvt := range ls.virtual {
		for i := range lvt {
			vt := &lvt[i]
			if vt.layer != layer || vt.view != vid {
				continue
			}
			vts = append(vts, VirtualText{Pos: textpos.Pos{Line: ln, Char: vt.char}, Text: runes.Join(vt.lines, []rune("\n")), Style: vt.style})
		}
	}
	return vts
}

// VirtualTextView returns the view position, in view lines and display
// chars, of the char at the given offset within the virtual text in the
// given layer at the given source position, as shown in the given view.
// Newlines in the text are not counted in the offset. For example, an
// input method uses the position of its composition cursor to place its
// candidate window. It returns false if there is no such virtual text.
func (ls *Lines) VirtualTextView(vid int, layer string, pos textpos.Pos, offset int) (textpos.Pos, bool) {
	ls.Lock()
	defer ls.Unlock()
	vw := ls.view(vid)
	if vw == nil {
		return textpos.Pos{}, false
	}
	d := ls.srcToDisplay(vw, pos.Line, pos.Char)
	for i := range ls.virtualAt(pos.Line) {
		vt := &ls.virtual[pos.Line][i]
		if vt.char < pos.Char || !vt.shownIn(vw) {
			continue
		}
		if vt.char > pos.Char {
			break
		}
		if vt.layer == layer {
			offset = max(0, min(offset, vt.len()))
			return ls.displayToView(vw, textpos.Pos{Line: pos.Line, Char: d + offset}), true
		}
		d += vt.len()
	}
	return textpos.Pos{}, false
}

// OnVirtualText adds a listener for when the virtual text shown in
// the given view changes. It is an [events.Custom] event.
func (ls *Lines) OnVirtualText(vid int, fun func(e events.Event)) {
	ls.Lock()
	defer ls.Unlock()
	vw := ls.view(vid)
	if vw != nil {
		vw.listeners.Add(events.Custom, fun)
	}
}

func (ls *Lines) sendVirtualText() {
	e := &events.Base{Typ: events.Custom}
	e.Init()
	for _, vw := range ls.views {
		vw.listeners.Call(e)
	}
}

// setVirtualText does [Lines.SetVirtualText] without locking or sending events.
func (ls *Lines) setVirtualText(layer string, vid int, vts ...VirtualText) {
	vid = max(vid, -1)
	ls.virtual = ls.virtualLines()
	changed := false
	for ln := range ls.virtual {
		n := len(ls.virtual[ln])
		ls.virtual[ln] = slices.DeleteFunc(ls.virtual[ln], func(vt virtualText) bool {
			return vt.layer == layer && vt.view == vid
		})
		changed = changed || len(ls.virtual[ln]) != n
	}
	for _, vt := range vts {
		pos, ok := ls.validCharPos(vt.Pos)
		if !ok || len(vt.Text) == 0 {
			continue
		}
		lvt := ls.virtual[pos.Line]
		i := len(lvt)
		for i > 0 && lvt[i-1].char > pos.Char {
			i--
		}
		ls.virtual[pos.Line] = slices.Insert(lvt, i, virtualText{char: pos.Char, lines: runes.Split(vt.Text, []rune("\n")), style: vt.Style, layer: layer, view: vid})
		changed = true
	}
	if changed {
		ls.layoutViews()
	}
}

// virtualLines returns ls.virtual with one entry per line.
func (ls *Lines) virtualLines() [][]virtualText {
	return ls.virtualLinesFor(ls.numLines())
}

// virtualLinesFor returns ls.virtual with n entries.
func (ls *Lines) virtualLinesFor(n int) [][]virtualText {
	if len(ls.virtual) == n {
		return ls.virtual
	}
	if len(ls.virtual) > n {
		return ls.virtual[:n]
	}
	return append(ls.virtual, make([][]virtualText, n-len(ls.virtual))...)
}

// virtualAt returns the virtual text for the given source line.
func (ls *Lines) virtualAt(ln int) []virtualText {
	if ln < 0 || ln >= len(ls.virtual) {
		return nil
	}
	return ls.virtual[ln]
}

// hasVirtual returns whether any virtual text is shown in the
// given view on the given source line.
func (ls *Lines) hasVirtual(vw *view, ln int) bool {
	for i := range ls.virtualAt(ln) {
		if ls.virtual[ln][i].shownIn(vw) {
			return true
		}
	}
	return false
}

// displayLineLen returns the number of display chars
// of the given source line in the given view.
func (ls *Lines) displayLineLen(vw *view, ln int) int {
	n := len(ls.lines[ln])
	for i := range ls.virtualAt(ln) {
		vt := &ls.virtual[ln][i]
		if vt.shownIn(vw) {
			n += vt.len()
		}
	}
	return n
}

// srcToDisplay returns the display char position in the given view
// for the given source char position in the given source line.
// The cursor is shown before any virtual text at its position.
func (ls *Lines) srcToDisplay(vw *view, ln, ch int) int {
	d := ch
	for i := range ls.virtualAt(ln) {
		vt := &ls.virtual[ln][i]
		if vt.char >= ch {
			break
		}
		if vt.shownIn(vw) {
			d += vt.len()
		}
	}
	return d
}

// displayToSrc returns the source char position in the given source line
// for the given display char position in the given view. Display positions
// within virtual text return its source position.
func (ls *Lines) displayToSrc(vw *view, ln, d int) int {
	src, disp := 0, 0
	for i := range ls.virtualAt(ln) {
		vt := &ls.virtual[ln][i]
		if !vt.shownIn(vw) {
			continue
		}
		vst := disp + vt.char - src // display start of the virtual text
		if d < vst {
			return src + d - disp
		}
		if d < vst+vt.len() {
			return vt.char
		}
		src, disp = vt.char, vst+vt.len()
	}
	return src + d - disp
}

// isVirtualDisplay returns whether the given display char position
// in the given source line is within virtual text in the given view.
func (ls *Lines) isVirtualDisplay(vw *view, ln, d int) bool {
	src, disp := 0, 0
	for i := range ls.virtualAt(ln) {
		vt := &ls.virtual[ln][i]
		if !vt.shownIn(vw) {
			continue
		}
		vst := disp + vt.char - src
		if d < vst {
			return false
		}
		if d < vst+vt.len() {
			return true
		}
		src, disp = vt.char, vst+vt.len()
	}
	return false
}

// isVirtualViewLine returns whether the given view line
// only contains virtual text, so the cursor skips it.
func (ls *Lines) isVirtualViewLine(vw *view, vl int) bool {
	if vl < 0 || vl >= len(vw.vlineStarts) {
		return false
	}
	vs := vw.vlineStarts[vl]
	if !ls.hasVirtual(vw, vs.Line) {
		return false
	}
	n := ls.viewLineLen(vw, vl)
	if vl+1 >= len(vw.vlineStarts) || vw.vlineStarts[vl+1].Line != vs.Line {
		n-- // the end of the source line, which is not virtual
	}
	if n <= 0 {
		return false
	}
	for d := vs.Char; d < vs.Char+n; d++ {
		if !ls.isVirtualDisplay(vw, vs.Line, d) {
			return false
		}
	}
	return true
}

// displayLine returns the display text and markup for the given source
// line in the given view, with any virtual text spliced into the source
// text and markup, and the display positions where the following lines
// of multi-line virtual text start, which must start new view lines.
// If there is no virtual text, the source text and markup are returned.
func (ls *Lines) displayLine(vw *view, ln int, txt []rune, mu rich.Text) ([]rune, rich.Text, []int) {
	if !ls.hasVirtual(vw, ln) {
		return txt, mu, nil
	}
	dtxt := make([]rune, 0, ls.displayLineLen(vw, ln))
	dmu := mu.Clone()
	var breaks []int
	src := 0
	for i := range ls.virtual[ln] {
		vt := &ls.virtual[ln][i]
		if !vt.shownIn(vw) {
			continue
		}
		dtxt = append(dtxt, txt[src:vt.char]...)
		src = vt.char
		sty := vt.style
		if sty == nil {
			sty = styleAt(dmu, len(dtxt))
		}
		for li, l := range vt.lines {
			if li > 0 {
				breaks = append(breaks, len(dtxt))
			}
			insertSpan(&dmu, len(dtxt), sty, l)
			dtxt = append(dtxt, l...)
		}
	}
	dtxt = append(dtxt, txt[src:]...)
	return dtxt, dmu, breaks
}

// styleAt returns the style of the markup at the given char position,
// or the last style if it is at the end.
func styleAt(mu rich.Text, ch int) *rich.Style {
	if len(mu) == 0 {
		return &rich.Style{}
	}
	si, _, _ := mu.Index(ch)
	if si < 0 {
		si = len(mu) - 1
	}
	sty, _ := mu.Span(si)
	return sty
}

// insertSpan inserts a span with the given style and text into the markup
// at the given char position, which can be its end.
func insertSpan(mu *rich.Text, ch int, sty *rich.Style, txt []rune) {
	if len(txt) == 0 {
		return
	}
	if ch >= mu.Len() {
		mu.AddSpan(sty, txt)
		return
	}
	si := mu.SplitSpan(ch)
	if si < 0 {
		mu.AddSpan(sty, txt)
		return
	}
	mu.InsertSpan(si, sty, txt)
}

// virtualInserted updates the virtual text for text inserted from st to ed,
// which must be called after the lines are inserted.
func (ls *Lines) virtualInserted(st, ed textpos.Pos) {
	if st.Line >= len(ls.virtual) {
		return
	}
	lvt := ls.virtual[st.Line]
	i := 0
	for i < len(lvt) && lvt[i].char < st.Char {
		i++
	}
	moved := lvt[i:]
	for j := range moved {
		moved[j].char += ed.Char - st.Char
	}
	if ed.Line == st.Line {
		return
	}
	ls.virtual[st.Line] = lvt[:i:i]
	if ed.Line < len(ls.virtual) {
		ls.virtual[ed.Line] = append(ls.virtual[ed.Line], moved...)
	}
}

// virtualDeleted updates the virtual text for text deleted from st to ed,
// which must be called before the lines are deleted. Virtual text within
// the deleted text is removed, and that after it is moved to st.
func (ls *Lines) virtualDeleted(st, ed textpos.Pos) {
	if st.Line >= len(ls.virtual) {
		return
	}
	var kept []virtualText
	for _, vt := range ls.virtual[st.Line] {
		if vt.char <= st.Char {
			kept = append(kept, vt)
		} else if st.Line == ed.Line && vt.char >= ed.Char {
			vt.char -= ed.Char - st.Char
			kept = append(kept, vt)
		}
	}
	if ed.Line > st.Line && ed.Line < len(ls.virtual) {
		for _, vt := range ls.virtual[ed.Line] {
			if vt.char >= ed.Char {
				vt.char += st.Char - ed.Char
				kept = append(kept, vt)
			}
		}
		// The deleted lines are removed by linesDeleted, which keeps
		// the slot of the end line for the joined line.
		ls.virtual[ed.Line] = kept
		return
	}
	ls.virtual[st.Line] = kept
}
