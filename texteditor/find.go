// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

import (
	"unicode"

	"cogentcore.org/core/base/stringsx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/parse/lexer"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/texteditor/text"
)

// findMatches finds the matches with given search string (literal, not regex)
// and case sensitivity, updates highlights for all.  returns false if none
// found
func (ed *Editor) findMatches(find string, useCase, lexItems bool) ([]text.Match, bool) {
	fsz := len(find)
	if fsz == 0 {
		ed.Highlights = nil
		return nil, false
	}
	_, matches := ed.Buffer.Search([]byte(find), !useCase, lexItems)
	if len(matches) == 0 {
		ed.Highlights = nil
		return matches, false
	}
	hi := make([]text.Region, len(matches))
	for i, m := range matches {
		hi[i] = m.Reg
		if i > viewMaxFindHighlights {
			break
		}
	}
	ed.Highlights = hi
	return matches, true
}

// matchFromPos finds the match at or after the given text position -- returns 0, false if none
func (ed *Editor) matchFromPos(matches []text.Match, cpos lexer.Pos) (int, bool) {
	for i, m := range matches {
		reg := ed.Buffer.AdjustRegion(m.Reg)
		if reg.Start == cpos || cpos.IsLess(reg.Start) {
			return i, true
		}
	}
	return 0, false
}

// ISearch holds all the interactive search data
type ISearch struct {

	// if true, in interactive search mode
	On bool `json:"-" xml:"-"`

	// current interactive search string
	Find string `json:"-" xml:"-"`

	// pay attention to case in isearch -- triggered by typing an upper-case letter
	useCase bool

	// current search matches
	Matches []text.Match `json:"-" xml:"-"`

	// position within isearch matches
	pos int

	// position in search list from previous search
	prevPos int

	// starting position for search -- returns there after on cancel
	startPos lexer.Pos
}

// viewMaxFindHighlights is the maximum number of regions to highlight on find
var viewMaxFindHighlights = 1000

// PrevISearchString is the previous ISearch string
var PrevISearchString string

// iSearchMatches finds ISearch matches -- returns true if there are any
func (ed *Editor) iSearchMatches() bool {
	got := false
	ed.ISearch.Matches, got = ed.findMatches(ed.ISearch.Find, ed.ISearch.useCase, false)
	return got
}

// iSearchNextMatch finds next match after given cursor position, and highlights
// it, etc
func (ed *Editor) iSearchNextMatch(cpos lexer.Pos) bool {
	if len(ed.ISearch.Matches) == 0 {
		ed.iSearchEvent()
		return false
	}
	ed.ISearch.pos, _ = ed.matchFromPos(ed.ISearch.Matches, cpos)
	ed.iSearchSelectMatch(ed.ISearch.pos)
	return true
}

// iSearchSelectMatch selects match at given match index (e.g., ed.ISearch.Pos)
func (ed *Editor) iSearchSelectMatch(midx int) {
	nm := len(ed.ISearch.Matches)
	if midx >= nm {
		ed.iSearchEvent()
		return
	}
	m := ed.ISearch.Matches[midx]
	reg := ed.Buffer.AdjustRegion(m.Reg)
	pos := reg.Start
	ed.SelectRegion = reg
	ed.setCursor(pos)
	ed.savePosHistory(ed.CursorPos)
	ed.scrollCursorToCenterIfHidden()
	ed.iSearchEvent()
}

// iSearchEvent sends the signal that ISearch is updated
func (ed *Editor) iSearchEvent() {
	ed.Send(events.Input)
}

// iSearchStart is an emacs-style interactive search mode -- this is called when
// the search command itself is entered
func (ed *Editor) iSearchStart() {
	if ed.ISearch.On {
		if ed.ISearch.Find != "" { // already searching -- find next
			sz := len(ed.ISearch.Matches)
			if sz > 0 {
				if ed.ISearch.pos < sz-1 {
					ed.ISearch.pos++
				} else {
					ed.ISearch.pos = 0
				}
				ed.iSearchSelectMatch(ed.ISearch.pos)
			}
		} else { // restore prev
			if PrevISearchString != "" {
				ed.ISearch.Find = PrevISearchString
				ed.ISearch.useCase = lexer.HasUpperCase(ed.ISearch.Find)
				ed.iSearchMatches()
				ed.iSearchNextMatch(ed.CursorPos)
				ed.ISearch.startPos = ed.CursorPos
			}
			// nothing..
		}
	} else {
		ed.ISearch.On = true
		ed.ISearch.Find = ""
		ed.ISearch.startPos = ed.CursorPos
		ed.ISearch.useCase = false
		ed.ISearch.Matches = nil
		ed.SelectReset()
		ed.ISearch.pos = -1
		ed.iSearchEvent()
	}
	ed.NeedsRender()
}

// iSearchKeyInput is an emacs-style interactive search mode -- this is called
// when keys are typed while in search mode
func (ed *Editor) iSearchKeyInput(kt events.Event) {
	kt.SetHandled()
	r := kt.KeyRune()
	// if ed.ISearch.Find == PrevISearchString { // undo starting point
	// 	ed.ISearch.Find = ""
	// }
	if unicode.IsUpper(r) { // todo: more complex
		ed.ISearch.useCase = true
	}
	ed.ISearch.Find += string(r)
	ed.iSearchMatches()
	sz := len(ed.ISearch.Matches)
	if sz == 0 {
		ed.ISearch.pos = -1
		ed.iSearchEvent()
		return
	}
	ed.iSearchNextMatch(ed.CursorPos)
	ed.NeedsRender()
}

// iSearchBackspace gets rid of one item in search string
func (ed *Editor) iSearchBackspace() {
	if ed.ISearch.Find == PrevISearchString { // undo starting point
		ed.ISearch.Find = ""
		ed.ISearch.useCase = false
		ed.ISearch.Matches = nil
		ed.SelectReset()
		ed.ISearch.pos = -1
		ed.iSearchEvent()
		return
	}
	if len(ed.ISearch.Find) <= 1 {
		ed.SelectReset()
		ed.ISearch.Find = ""
		ed.ISearch.useCase = false
		return
	}
	ed.ISearch.Find = ed.ISearch.Find[:len(ed.ISearch.Find)-1]
	ed.iSearchMatches()
	sz := len(ed.ISearch.Matches)
	if sz == 0 {
		ed.ISearch.pos = -1
		ed.iSearchEvent()
		return
	}
	ed.iSearchNextMatch(ed.CursorPos)
	ed.NeedsRender()
}

// iSearchCancel cancels ISearch mode
func (ed *Editor) iSearchCancel() {
	if !ed.ISearch.On {
		return
	}
	if ed.ISearch.Find != "" {
		PrevISearchString = ed.ISearch.Find
	}
	ed.ISearch.prevPos = ed.ISearch.pos
	ed.ISearch.Find = ""
	ed.ISearch.useCase = false
	ed.ISearch.On = false
	ed.ISearch.pos = -1
	ed.ISearch.Matches = nil
	ed.Highlights = nil
	ed.savePosHistory(ed.CursorPos)
	ed.SelectReset()
	ed.iSearchEvent()
	ed.NeedsRender()
}

// QReplace holds all the query-replace data
type QReplace struct {

	// if true, in interactive search mode
	On bool `json:"-" xml:"-"`

	// current interactive search string
	Find string `json:"-" xml:"-"`

	// current interactive search string
	Replace string `json:"-" xml:"-"`

	// pay attention to case in isearch -- triggered by typing an upper-case letter
	useCase bool

	// search only as entire lexically tagged item boundaries -- key for replacing short local variables like i
	lexItems bool

	// current search matches
	Matches []text.Match `json:"-" xml:"-"`

	// position within isearch matches
	pos int `json:"-" xml:"-"`

	// starting position for search -- returns there after on cancel
	startPos lexer.Pos
}

var (
	// prevQReplaceFinds are the previous QReplace strings
	prevQReplaceFinds []string

	// prevQReplaceRepls are the previous QReplace strings
	prevQReplaceRepls []string
)

// qReplaceEvent sends the event that QReplace is updated
func (ed *Editor) qReplaceEvent() {
	ed.Send(events.Input)
}

// QReplacePrompt is an emacs-style query-replace mode -- this starts the process, prompting
// user for items to search etc
func (ed *Editor) QReplacePrompt() {
	find := ""
	if ed.HasSelection() {
		find = string(ed.Selection().ToBytes())
	}
	d := core.NewBody("Query-Replace")
	core.NewText(d).SetType(core.TextSupporting).SetText("Enter strings for find and replace, then select Query-Replace; with dialog dismissed press <b>y</b> to replace current match, <b>n</b> to skip, <b>Enter</b> or <b>q</b> to quit, <b>!</b> to replace-all remaining")
	fc := core.NewChooser(d).SetEditable(true).SetDefaultNew(true)
	fc.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 0)
		s.Min.X.Ch(80)
	})
	fc.SetStrings(prevQReplaceFinds...).SetCurrentIndex(0)
	if find != "" {
		fc.SetCurrentValue(find)
	}

	rc := core.NewChooser(d).SetEditable(true).SetDefaultNew(true)
	rc.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 0)
		s.Min.X.Ch(80)
	})
	rc.SetStrings(prevQReplaceRepls...).SetCurrentIndex(0)

	lexitems := ed.QReplace.lexItems
	lxi := core.NewSwitch(d).SetText("Lexical Items").SetChecked(lexitems)
	lxi.SetTooltip("search matches entire lexically tagged items -- good for finding local variable names like 'i' and not matching everything")

	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Query-Replace").OnClick(func(e events.Event) {
			var find, repl string
			if s, ok := fc.CurrentItem.Value.(string); ok {
				find = s
			}
			if s, ok := rc.CurrentItem.Value.(string); ok {
				repl = s
			}
			lexItems := lxi.IsChecked()
			ed.QReplaceStart(find, repl, lexItems)
		})
	})
	d.RunDialog(ed)
}

// QReplaceStart starts query-replace using given find, replace strings
func (ed *Editor) QReplaceStart(find, repl string, lexItems bool) {
	ed.QReplace.On = true
	ed.QReplace.Find = find
	ed.QReplace.Replace = repl
	ed.QReplace.lexItems = lexItems
	ed.QReplace.startPos = ed.CursorPos
	ed.QReplace.useCase = lexer.HasUpperCase(find)
	ed.QReplace.Matches = nil
	ed.QReplace.pos = -1

	stringsx.InsertFirstUnique(&prevQReplaceFinds, find, core.SystemSettings.SavedPathsMax)
	stringsx.InsertFirstUnique(&prevQReplaceRepls, repl, core.SystemSettings.SavedPathsMax)

	ed.qReplaceMatches()
	ed.QReplace.pos, _ = ed.matchFromPos(ed.QReplace.Matches, ed.CursorPos)
	ed.qReplaceSelectMatch(ed.QReplace.pos)
	ed.qReplaceEvent()
}

// qReplaceMatches finds QReplace matches -- returns true if there are any
func (ed *Editor) qReplaceMatches() bool {
	got := false
	ed.QReplace.Matches, got = ed.findMatches(ed.QReplace.Find, ed.QReplace.useCase, ed.QReplace.lexItems)
	return got
}

// qReplaceNextMatch finds next match using, QReplace.Pos and highlights it, etc
func (ed *Editor) qReplaceNextMatch() bool {
	nm := len(ed.QReplace.Matches)
	if nm == 0 {
		return false
	}
	ed.QReplace.pos++
	if ed.QReplace.pos >= nm {
		return false
	}
	ed.qReplaceSelectMatch(ed.QReplace.pos)
	return true
}

// qReplaceSelectMatch selects match at given match index (e.g., ed.QReplace.Pos)
func (ed *Editor) qReplaceSelectMatch(midx int) {
	nm := len(ed.QReplace.Matches)
	if midx >= nm {
		return
	}
	m := ed.QReplace.Matches[midx]
	reg := ed.Buffer.AdjustRegion(m.Reg)
	pos := reg.Start
	ed.SelectRegion = reg
	ed.setCursor(pos)
	ed.savePosHistory(ed.CursorPos)
	ed.scrollCursorToCenterIfHidden()
	ed.qReplaceEvent()
}

// qReplaceReplace replaces at given match index (e.g., ed.QReplace.Pos)
func (ed *Editor) qReplaceReplace(midx int) {
	nm := len(ed.QReplace.Matches)
	if midx >= nm {
		return
	}
	m := ed.QReplace.Matches[midx]
	rep := ed.QReplace.Replace
	reg := ed.Buffer.AdjustRegion(m.Reg)
	pos := reg.Start
	// last arg is matchCase, only if not using case to match and rep is also lower case
	matchCase := !ed.QReplace.useCase && !lexer.HasUpperCase(rep)
	ed.Buffer.ReplaceText(reg.Start, reg.End, pos, rep, EditSignal, matchCase)
	ed.Highlights[midx] = text.RegionNil
	ed.setCursor(pos)
	ed.savePosHistory(ed.CursorPos)
	ed.scrollCursorToCenterIfHidden()
	ed.qReplaceEvent()
}

// QReplaceReplaceAll replaces all remaining from index
func (ed *Editor) QReplaceReplaceAll(midx int) {
	nm := len(ed.QReplace.Matches)
	if midx >= nm {
		return
	}
	for mi := midx; mi < nm; mi++ {
		ed.qReplaceReplace(mi)
	}
}

// qReplaceKeyInput is an emacs-style interactive search mode -- this is called
// when keys are typed while in search mode
func (ed *Editor) qReplaceKeyInput(kt events.Event) {
	kt.SetHandled()
	switch {
	case kt.KeyRune() == 'y':
		ed.qReplaceReplace(ed.QReplace.pos)
		if !ed.qReplaceNextMatch() {
			ed.qReplaceCancel()
		}
	case kt.KeyRune() == 'n':
		if !ed.qReplaceNextMatch() {
			ed.qReplaceCancel()
		}
	case kt.KeyRune() == 'q' || kt.KeyChord() == "ReturnEnter":
		ed.qReplaceCancel()
	case kt.KeyRune() == '!':
		ed.QReplaceReplaceAll(ed.QReplace.pos)
		ed.qReplaceCancel()
	}
	ed.NeedsRender()
}

// qReplaceCancel cancels QReplace mode
func (ed *Editor) qReplaceCancel() {
	if !ed.QReplace.On {
		return
	}
	ed.QReplace.On = false
	ed.QReplace.pos = -1
	ed.QReplace.Matches = nil
	ed.Highlights = nil
	ed.savePosHistory(ed.CursorPos)
	ed.SelectReset()
	ed.qReplaceEvent()
	ed.NeedsRender()
}

// escPressed emitted for [keymap.Abort] or [keymap.CancelSelect];
// effect depends on state.
func (ed *Editor) escPressed() {
	switch {
	case ed.ISearch.On:
		ed.iSearchCancel()
		ed.SetCursorShow(ed.ISearch.startPos)
	case ed.QReplace.On:
		ed.qReplaceCancel()
		ed.SetCursorShow(ed.ISearch.startPos)
	case ed.HasSelection():
		ed.SelectReset()
	default:
		ed.Highlights = nil
	}
	ed.NeedsRender()
}
