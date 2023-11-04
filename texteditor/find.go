// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

import (
	"unicode"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/texteditor/textbuf"
	"goki.dev/girl/states"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/pi/v2/lex"
)

///////////////////////////////////////////////////////////////////////////////
//    Search / Find

// FindMatches finds the matches with given search string (literal, not regex)
// and case sensitivity, updates highlights for all.  returns false if none
// found
func (ed *Editor) FindMatches(find string, useCase, lexItems bool) ([]textbuf.Match, bool) {
	fsz := len(find)
	if fsz == 0 {
		ed.Highlights = nil
		return nil, false
	}
	_, matches := ed.Buf.Search([]byte(find), !useCase, lexItems)
	if len(matches) == 0 {
		ed.Highlights = nil
		return matches, false
	}
	hi := make([]textbuf.Region, len(matches))
	for i, m := range matches {
		hi[i] = m.Reg
		if i > ViewMaxFindHighlights {
			break
		}
	}
	ed.Highlights = hi
	return matches, true
}

// MatchFromPos finds the match at or after the given text position -- returns 0, false if none
func (ed *Editor) MatchFromPos(matches []textbuf.Match, cpos lex.Pos) (int, bool) {
	for i, m := range matches {
		reg := ed.Buf.AdjustReg(m.Reg)
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
	UseCase bool `json:"-" xml:"-"`

	// current search matches
	Matches []textbuf.Match `json:"-" xml:"-"`

	// position within isearch matches
	Pos int `json:"-" xml:"-"`

	// position in search list from previous search
	PrevPos int `json:"-" xml:"-"`

	// starting position for search -- returns there after on cancel
	StartPos lex.Pos `json:"-" xml:"-"`
}

// ViewMaxFindHighlights is the maximum number of regions to highlight on find
var ViewMaxFindHighlights = 1000

// PrevISearchString is the previous ISearch string
var PrevISearchString string

// ISearchMatches finds ISearch matches -- returns true if there are any
func (ed *Editor) ISearchMatches() bool {
	got := false
	ed.ISearch.Matches, got = ed.FindMatches(ed.ISearch.Find, ed.ISearch.UseCase, false)
	return got
}

// ISearchNextMatch finds next match after given cursor position, and highlights
// it, etc
func (ed *Editor) ISearchNextMatch(cpos lex.Pos) bool {
	if len(ed.ISearch.Matches) == 0 {
		ed.ISearchSig()
		return false
	}
	ed.ISearch.Pos, _ = ed.MatchFromPos(ed.ISearch.Matches, cpos)
	ed.ISearchSelectMatch(ed.ISearch.Pos)
	return true
}

// ISearchSelectMatch selects match at given match index (e.g., ed.ISearch.Pos)
func (ed *Editor) ISearchSelectMatch(midx int) {
	nm := len(ed.ISearch.Matches)
	if midx >= nm {
		ed.ISearchSig()
		return
	}
	m := ed.ISearch.Matches[midx]
	reg := ed.Buf.AdjustReg(m.Reg)
	pos := reg.Start
	ed.SelectReg = reg
	ed.SetCursor(pos)
	ed.SavePosHistory(ed.CursorPos)
	ed.ScrollCursorToCenterIfHidden()
	ed.ISearchSig()
}

// ISearchSig sends the signal that ISearch is updated
func (ed *Editor) ISearchSig() {
	// ed.ViewSig.Emit(ed.This(), int64(ViewISearch), ed.CursorPos)
}

// ISearchStart is an emacs-style interactive search mode -- this is called when
// the search command itself is entered
func (ed *Editor) ISearchStart() {
	updt := ed.UpdateStart()
	defer ed.UpdateEndRender(updt)
	if ed.ISearch.On {
		if ed.ISearch.Find != "" { // already searching -- find next
			sz := len(ed.ISearch.Matches)
			if sz > 0 {
				if ed.ISearch.Pos < sz-1 {
					ed.ISearch.Pos++
				} else {
					ed.ISearch.Pos = 0
				}
				ed.ISearchSelectMatch(ed.ISearch.Pos)
			}
		} else { // restore prev
			if PrevISearchString != "" {
				ed.ISearch.Find = PrevISearchString
				ed.ISearch.UseCase = lex.HasUpperCase(ed.ISearch.Find)
				ed.ISearchMatches()
				ed.ISearchNextMatch(ed.CursorPos)
				ed.ISearch.StartPos = ed.CursorPos
			}
			// nothing..
		}
	} else {
		ed.ISearch.On = true
		ed.ISearch.Find = ""
		ed.ISearch.StartPos = ed.CursorPos
		ed.ISearch.UseCase = false
		ed.ISearch.Matches = nil
		ed.SelectReset()
		ed.ISearch.Pos = -1
		ed.ISearchSig()
	}
}

// ISearchKeyInput is an emacs-style interactive search mode -- this is called
// when keys are typed while in search mode
func (ed *Editor) ISearchKeyInput(kt events.Event) {
	r := kt.KeyRune()
	updt := ed.UpdateStart()
	defer ed.UpdateEndRender(updt)
	// if ed.ISearch.Find == PrevISearchString { // undo starting point
	// 	ed.ISearch.Find = ""
	// }
	if unicode.IsUpper(r) { // todo: more complex
		ed.ISearch.UseCase = true
	}
	ed.ISearch.Find += string(r)
	ed.ISearchMatches()
	sz := len(ed.ISearch.Matches)
	if sz == 0 {
		ed.ISearch.Pos = -1
		ed.ISearchSig()
		return
	}
	ed.ISearchNextMatch(ed.CursorPos)
}

// ISearchBackspace gets rid of one item in search string
func (ed *Editor) ISearchBackspace() {
	updt := ed.UpdateStart()
	defer ed.UpdateEndRender(updt)
	if ed.ISearch.Find == PrevISearchString { // undo starting point
		ed.ISearch.Find = ""
		ed.ISearch.UseCase = false
		ed.ISearch.Matches = nil
		ed.SelectReset()
		ed.ISearch.Pos = -1
		ed.ISearchSig()
		return
	}
	if len(ed.ISearch.Find) <= 1 {
		ed.SelectReset()
		ed.ISearch.Find = ""
		ed.ISearch.UseCase = false
		return
	}
	ed.ISearch.Find = ed.ISearch.Find[:len(ed.ISearch.Find)-1]
	ed.ISearchMatches()
	sz := len(ed.ISearch.Matches)
	if sz == 0 {
		ed.ISearch.Pos = -1
		ed.ISearchSig()
		return
	}
	ed.ISearchNextMatch(ed.CursorPos)
}

// ISearchCancel cancels ISearch mode
func (ed *Editor) ISearchCancel() {
	if !ed.ISearch.On {
		return
	}
	updt := ed.UpdateStart()
	defer ed.UpdateEndRender(updt)
	if ed.ISearch.Find != "" {
		PrevISearchString = ed.ISearch.Find
	}
	ed.ISearch.PrevPos = ed.ISearch.Pos
	ed.ISearch.Find = ""
	ed.ISearch.UseCase = false
	ed.ISearch.On = false
	ed.ISearch.Pos = -1
	ed.ISearch.Matches = nil
	ed.Highlights = nil
	ed.SavePosHistory(ed.CursorPos)
	ed.SelectReset()
	ed.ISearchSig()
}

///////////////////////////////////////////////////////////////////////////////
//    Query-Replace

// QReplace holds all the query-replace data
type QReplace struct {

	// if true, in interactive search mode
	On bool `json:"-" xml:"-"`

	// current interactive search string
	Find string `json:"-" xml:"-"`

	// current interactive search string
	Replace string `json:"-" xml:"-"`

	// pay attention to case in isearch -- triggered by typing an upper-case letter
	UseCase bool `json:"-" xml:"-"`

	// search only as entire lexically-tagged item boundaries -- key for replacing short local variables like i
	LexItems bool `json:"-" xml:"-"`

	// current search matches
	Matches []textbuf.Match `json:"-" xml:"-"`

	// position within isearch matches
	Pos int `json:"-" xml:"-"`

	// position in search list from previous search
	PrevPos int `json:"-" xml:"-"`

	// starting position for search -- returns there after on cancel
	StartPos lex.Pos `json:"-" xml:"-"`
}

// PrevQReplaceFinds are the previous QReplace strings
var PrevQReplaceFinds []string

// PrevQReplaceRepls are the previous QReplace strings
var PrevQReplaceRepls []string

// QReplaceSig sends the signal that QReplace is updated
func (ed *Editor) QReplaceSig() {
	// ed.ViewSig.Emit(ed.This(), int64(ViewQReplace), ed.CursorPos)
}

// QReplaceDialog adds to the given dialog a display prompting the user for
// query-replace items, with choosers with history
func QReplaceDialog(d *gi.Dialog, find string, lexitems bool) *gi.Dialog {
	tff := gi.NewChooser(d, "find")
	tff.Editable = true
	tff.SetStretchMaxWidth()
	tff.SetMinPrefWidth(units.Ch(60))
	tff.SetStrings(PrevQReplaceFinds, true, 0)
	if find != "" {
		tff.SetCurVal(find)
	}

	tfr := gi.NewChooser(d, "repl")
	tfr.Editable = true
	tfr.SetStretchMaxWidth()
	tfr.SetMinPrefWidth(units.Ch(60))
	tfr.SetStrings(PrevQReplaceRepls, true, 0)

	lb := gi.NewSwitch(d, "lexb")
	lb.SetText("Lexical Items")
	lb.SetState(lexitems, states.Checked)
	lb.Tooltip = "search matches entire lexically tagged items -- good for finding local variable names like 'i' and not matching everything"

	return d
}

// QReplaceDialogValues gets the string values
func QReplaceDialogValues(d *gi.Dialog) (find, repl string, lexItems bool) {
	tff := d.ChildByName("find", 1).(*gi.Chooser)
	if tf, found := tff.TextField(); found {
		find = tf.Text()
	}
	tfr := d.ChildByName("repl", 2).(*gi.Chooser)
	if tf, found := tfr.TextField(); found {
		repl = tf.Text()
	}
	lb := d.ChildByName("lexb", 3).(*gi.Switch)
	lexItems = lb.StateIs(states.Checked)
	return
}

// QReplacePrompt is an emacs-style query-replace mode -- this starts the process, prompting
// user for items to search etc
func (ed *Editor) QReplacePrompt() {
	find := ""
	if ed.HasSelection() {
		find = string(ed.Selection().ToBytes())
	}
	d := QReplaceDialog(gi.NewDialog(ed).Title("Query-Replace").
		Prompt("Enter strings for find and replace, then select Query-Replace -- with dialog dismissed press <b>y</b> to replace current match, <b>n</b> to skip, <b>Enter</b> or <b>q</b> to quit, <b>!</b> to replace-all remaining"),
		find, ed.QReplace.LexItems)
	d.OnAccept(func(e events.Event) {
		find, repl, lexItems := QReplaceDialogValues(d)
		ed.QReplaceStart(find, repl, lexItems)
	}).Cancel().Ok("Query-Replace").Run()
}

// QReplaceStart starts query-replace using given find, replace strings
func (ed *Editor) QReplaceStart(find, repl string, lexItems bool) {
	ed.QReplace.On = true
	ed.QReplace.Find = find
	ed.QReplace.Replace = repl
	ed.QReplace.LexItems = lexItems
	ed.QReplace.StartPos = ed.CursorPos
	ed.QReplace.UseCase = lex.HasUpperCase(find)
	ed.QReplace.Matches = nil
	ed.QReplace.Pos = -1

	gi.StringsInsertFirstUnique(&PrevQReplaceFinds, find, gi.Prefs.Params.SavedPathsMax)
	gi.StringsInsertFirstUnique(&PrevQReplaceRepls, repl, gi.Prefs.Params.SavedPathsMax)

	ed.QReplaceMatches()
	ed.QReplace.Pos, _ = ed.MatchFromPos(ed.QReplace.Matches, ed.CursorPos)
	ed.QReplaceSelectMatch(ed.QReplace.Pos)
	ed.QReplaceSig()
}

// QReplaceMatches finds QReplace matches -- returns true if there are any
func (ed *Editor) QReplaceMatches() bool {
	got := false
	ed.QReplace.Matches, got = ed.FindMatches(ed.QReplace.Find, ed.QReplace.UseCase, ed.QReplace.LexItems)
	return got
}

// QReplaceNextMatch finds next match using, QReplace.Pos and highlights it, etc
func (ed *Editor) QReplaceNextMatch() bool {
	nm := len(ed.QReplace.Matches)
	if nm == 0 {
		return false
	}
	ed.QReplace.Pos++
	if ed.QReplace.Pos >= nm {
		return false
	}
	ed.QReplaceSelectMatch(ed.QReplace.Pos)
	return true
}

// QReplaceSelectMatch selects match at given match index (e.g., ed.QReplace.Pos)
func (ed *Editor) QReplaceSelectMatch(midx int) {
	nm := len(ed.QReplace.Matches)
	if midx >= nm {
		return
	}
	m := ed.QReplace.Matches[midx]
	reg := ed.Buf.AdjustReg(m.Reg)
	pos := reg.Start
	ed.SelectReg = reg
	ed.SetCursor(pos)
	ed.SavePosHistory(ed.CursorPos)
	ed.ScrollCursorToCenterIfHidden()
	ed.QReplaceSig()
}

// QReplaceReplace replaces at given match index (e.g., ed.QReplace.Pos)
func (ed *Editor) QReplaceReplace(midx int) {
	nm := len(ed.QReplace.Matches)
	if midx >= nm {
		return
	}
	m := ed.QReplace.Matches[midx]
	rep := ed.QReplace.Replace
	reg := ed.Buf.AdjustReg(m.Reg)
	pos := reg.Start
	// last arg is matchCase, only if not using case to match and rep is also lower case
	matchCase := !ed.QReplace.UseCase && !lex.HasUpperCase(rep)
	ed.Buf.ReplaceText(reg.Start, reg.End, pos, rep, EditSignal, matchCase)
	ed.Highlights[midx] = textbuf.RegionNil
	ed.SetCursor(pos)
	ed.SavePosHistory(ed.CursorPos)
	ed.ScrollCursorToCenterIfHidden()
	ed.QReplaceSig()
}

// QReplaceReplaceAll replaces all remaining from index
func (ed *Editor) QReplaceReplaceAll(midx int) {
	nm := len(ed.QReplace.Matches)
	if midx >= nm {
		return
	}
	for mi := midx; mi < nm; mi++ {
		ed.QReplaceReplace(mi)
	}
}

// QReplaceKeyInput is an emacs-style interactive search mode -- this is called
// when keys are typed while in search mode
func (ed *Editor) QReplaceKeyInput(kt events.Event) {
	updt := ed.UpdateStart()
	defer ed.UpdateEndRender(updt)

	switch {
	case kt.KeyRune() == 'y':
		ed.QReplaceReplace(ed.QReplace.Pos)
		if !ed.QReplaceNextMatch() {
			ed.QReplaceCancel()
		}
	case kt.KeyRune() == 'n':
		if !ed.QReplaceNextMatch() {
			ed.QReplaceCancel()
		}
	case kt.KeyRune() == 'q' || kt.KeyChord() == "ReturnEnter":
		ed.QReplaceCancel()
	case kt.KeyRune() == '!':
		ed.QReplaceReplaceAll(ed.QReplace.Pos)
		ed.QReplaceCancel()
	}
}

// QReplaceCancel cancels QReplace mode
func (ed *Editor) QReplaceCancel() {
	if !ed.QReplace.On {
		return
	}
	updt := ed.UpdateStart()
	defer ed.UpdateEndRender(updt)
	ed.QReplace.On = false
	ed.QReplace.Pos = -1
	ed.QReplace.Matches = nil
	ed.Highlights = nil
	ed.SavePosHistory(ed.CursorPos)
	ed.SelectReset()
	ed.QReplaceSig()
}

// EscPressed emitted for keyfun.Abort or keyfun.CancelSelect -- effect depends on state..
func (ed *Editor) EscPressed() {
	updt := ed.UpdateStart()
	defer ed.UpdateEndRender(updt)
	switch {
	case ed.ISearch.On:
		ed.ISearchCancel()
		ed.SetCursorShow(ed.ISearch.StartPos)
	case ed.QReplace.On:
		ed.QReplaceCancel()
		ed.SetCursorShow(ed.ISearch.StartPos)
	case ed.HasSelection():
		ed.SelectReset()
	default:
		ed.Highlights = nil
	}
}
