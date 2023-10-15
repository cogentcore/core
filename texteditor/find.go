// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textview

import (
	"unicode"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/textview/textbuf"
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
func (tv *View) FindMatches(find string, useCase, lexItems bool) ([]textbuf.Match, bool) {
	fsz := len(find)
	if fsz == 0 {
		tv.Highlights = nil
		return nil, false
	}
	_, matches := tv.Buf.Search([]byte(find), !useCase, lexItems)
	if len(matches) == 0 {
		tv.Highlights = nil
		return matches, false
	}
	hi := make([]textbuf.Region, len(matches))
	for i, m := range matches {
		hi[i] = m.Reg
		if i > ViewMaxFindHighlights {
			break
		}
	}
	tv.Highlights = hi
	return matches, true
}

// MatchFromPos finds the match at or after the given text position -- returns 0, false if none
func (tv *View) MatchFromPos(matches []textbuf.Match, cpos lex.Pos) (int, bool) {
	for i, m := range matches {
		reg := tv.Buf.AdjustReg(m.Reg)
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
func (tv *View) ISearchMatches() bool {
	got := false
	tv.ISearch.Matches, got = tv.FindMatches(tv.ISearch.Find, tv.ISearch.UseCase, false)
	return got
}

// ISearchNextMatch finds next match after given cursor position, and highlights
// it, etc
func (tv *View) ISearchNextMatch(cpos lex.Pos) bool {
	if len(tv.ISearch.Matches) == 0 {
		tv.ISearchSig()
		return false
	}
	tv.ISearch.Pos, _ = tv.MatchFromPos(tv.ISearch.Matches, cpos)
	tv.ISearchSelectMatch(tv.ISearch.Pos)
	return true
}

// ISearchSelectMatch selects match at given match index (e.g., tv.ISearch.Pos)
func (tv *View) ISearchSelectMatch(midx int) {
	nm := len(tv.ISearch.Matches)
	if midx >= nm {
		tv.ISearchSig()
		return
	}
	m := tv.ISearch.Matches[midx]
	reg := tv.Buf.AdjustReg(m.Reg)
	pos := reg.Start
	tv.SelectReg = reg
	tv.SetCursor(pos)
	tv.SavePosHistory(tv.CursorPos)
	tv.ScrollCursorToCenterIfHidden()
	tv.ISearchSig()
}

// ISearchSig sends the signal that ISearch is updated
func (tv *View) ISearchSig() {
	// tv.ViewSig.Emit(tv.This(), int64(ViewISearch), tv.CursorPos)
}

// ISearchStart is an emacs-style interactive search mode -- this is called when
// the search command itself is entered
func (tv *View) ISearchStart() {
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)
	if tv.ISearch.On {
		if tv.ISearch.Find != "" { // already searching -- find next
			sz := len(tv.ISearch.Matches)
			if sz > 0 {
				if tv.ISearch.Pos < sz-1 {
					tv.ISearch.Pos++
				} else {
					tv.ISearch.Pos = 0
				}
				tv.ISearchSelectMatch(tv.ISearch.Pos)
			}
		} else { // restore prev
			if PrevISearchString != "" {
				tv.ISearch.Find = PrevISearchString
				tv.ISearch.UseCase = lex.HasUpperCase(tv.ISearch.Find)
				tv.ISearchMatches()
				tv.ISearchNextMatch(tv.CursorPos)
				tv.ISearch.StartPos = tv.CursorPos
			}
			// nothing..
		}
	} else {
		tv.ISearch.On = true
		tv.ISearch.Find = ""
		tv.ISearch.StartPos = tv.CursorPos
		tv.ISearch.UseCase = false
		tv.ISearch.Matches = nil
		tv.SelectReset()
		tv.ISearch.Pos = -1
		tv.ISearchSig()
	}
}

// ISearchKeyInput is an emacs-style interactive search mode -- this is called
// when keys are typed while in search mode
func (tv *View) ISearchKeyInput(kt events.Event) {
	r := kt.KeyRune()
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)
	// if tv.ISearch.Find == PrevISearchString { // undo starting point
	// 	tv.ISearch.Find = ""
	// }
	if unicode.IsUpper(r) { // todo: more complex
		tv.ISearch.UseCase = true
	}
	tv.ISearch.Find += string(r)
	tv.ISearchMatches()
	sz := len(tv.ISearch.Matches)
	if sz == 0 {
		tv.ISearch.Pos = -1
		tv.ISearchSig()
		return
	}
	tv.ISearchNextMatch(tv.CursorPos)
}

// ISearchBackspace gets rid of one item in search string
func (tv *View) ISearchBackspace() {
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)
	if tv.ISearch.Find == PrevISearchString { // undo starting point
		tv.ISearch.Find = ""
		tv.ISearch.UseCase = false
		tv.ISearch.Matches = nil
		tv.SelectReset()
		tv.ISearch.Pos = -1
		tv.ISearchSig()
		return
	}
	if len(tv.ISearch.Find) <= 1 {
		tv.SelectReset()
		tv.ISearch.Find = ""
		tv.ISearch.UseCase = false
		return
	}
	tv.ISearch.Find = tv.ISearch.Find[:len(tv.ISearch.Find)-1]
	tv.ISearchMatches()
	sz := len(tv.ISearch.Matches)
	if sz == 0 {
		tv.ISearch.Pos = -1
		tv.ISearchSig()
		return
	}
	tv.ISearchNextMatch(tv.CursorPos)
}

// ISearchCancel cancels ISearch mode
func (tv *View) ISearchCancel() {
	if !tv.ISearch.On {
		return
	}
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)
	if tv.ISearch.Find != "" {
		PrevISearchString = tv.ISearch.Find
	}
	tv.ISearch.PrevPos = tv.ISearch.Pos
	tv.ISearch.Find = ""
	tv.ISearch.UseCase = false
	tv.ISearch.On = false
	tv.ISearch.Pos = -1
	tv.ISearch.Matches = nil
	tv.Highlights = nil
	tv.SavePosHistory(tv.CursorPos)
	tv.SelectReset()
	tv.ISearchSig()
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
func (tv *View) QReplaceSig() {
	// tv.ViewSig.Emit(tv.This(), int64(ViewQReplace), tv.CursorPos)
}

// QReplaceDialog prompts the user for a query-replace items, with choosers with history
func QReplaceDialog(ctx gi.Widget, opts gi.DlgOpts, find string, lexitems bool, fun func(dlg *gi.Dialog)) *gi.Dialog {
	dlg := gi.NewStdDialog(ctx, opts, fun)

	sc := dlg.Stage.Scene
	prIdx := dlg.PromptWidgetIdx()

	tff := sc.InsertNewChild(gi.ChooserType, prIdx+1, "find").(*gi.Chooser)
	tff.Editable = true
	tff.SetStretchMaxWidth()
	tff.SetMinPrefWidth(units.Ch(60))
	tff.ConfigParts(sc)
	tff.ItemsFromStringList(PrevQReplaceFinds, true, 0)
	if find != "" {
		tff.SetCurVal(find)
	}

	tfr := sc.InsertNewChild(gi.ChooserType, prIdx+2, "repl").(*gi.Chooser)
	tfr.Editable = true
	tfr.SetStretchMaxWidth()
	tfr.SetMinPrefWidth(units.Ch(60))
	tfr.ConfigParts(sc)
	tfr.ItemsFromStringList(PrevQReplaceRepls, true, 0)

	lb := sc.InsertNewChild(gi.SwitchType, prIdx+3, "lexb").(*gi.Switch)
	lb.SetText("Lexical Items")
	lb.SetState(lexitems, states.Checked)
	lb.Tooltip = "search matches entire lexically tagged items -- good for finding local variable names like 'i' and not matching everything"

	return dlg
}

// QReplaceDialogValues gets the string values
func QReplaceDialogValues(dlg *gi.Dialog) (find, repl string, lexItems bool) {
	sc := dlg.Stage.Scene
	tff := sc.ChildByName("find", 1).(*gi.Chooser)
	if tf, found := tff.TextField(); found {
		find = tf.Text()
	}
	tfr := sc.ChildByName("repl", 2).(*gi.Chooser)
	if tf, found := tfr.TextField(); found {
		repl = tf.Text()
	}
	lb := sc.ChildByName("lexb", 3).(*gi.Switch)
	lexItems = lb.StateIs(states.Checked)
	return
}

// QReplacePrompt is an emacs-style query-replace mode -- this starts the process, prompting
// user for items to search etc
func (tv *View) QReplacePrompt() {
	find := ""
	if tv.HasSelection() {
		find = string(tv.Selection().ToBytes())
	}
	QReplaceDialog(tv, gi.DlgOpts{Title: "Query-Replace", Prompt: "Enter strings for find and replace, then select Ok -- with dialog dismissed press <b>y</b> to replace current match, <b>n</b> to skip, <b>Enter</b> or <b>q</b> to quit, <b>!</b> to replace-all remaining"}, find, tv.QReplace.LexItems, func(dlg *gi.Dialog) {
		if dlg.Accepted {
			find, repl, lexItems := QReplaceDialogValues(dlg)
			tv.QReplaceStart(find, repl, lexItems)
		}
	})
}

// QReplaceStart starts query-replace using given find, replace strings
func (tv *View) QReplaceStart(find, repl string, lexItems bool) {
	tv.QReplace.On = true
	tv.QReplace.Find = find
	tv.QReplace.Replace = repl
	tv.QReplace.LexItems = lexItems
	tv.QReplace.StartPos = tv.CursorPos
	tv.QReplace.UseCase = lex.HasUpperCase(find)
	tv.QReplace.Matches = nil
	tv.QReplace.Pos = -1

	gi.StringsInsertFirstUnique(&PrevQReplaceFinds, find, gi.Prefs.Params.SavedPathsMax)
	gi.StringsInsertFirstUnique(&PrevQReplaceRepls, repl, gi.Prefs.Params.SavedPathsMax)

	tv.QReplaceMatches()
	tv.QReplace.Pos, _ = tv.MatchFromPos(tv.QReplace.Matches, tv.CursorPos)
	tv.QReplaceSelectMatch(tv.QReplace.Pos)
	tv.QReplaceSig()
}

// QReplaceMatches finds QReplace matches -- returns true if there are any
func (tv *View) QReplaceMatches() bool {
	got := false
	tv.QReplace.Matches, got = tv.FindMatches(tv.QReplace.Find, tv.QReplace.UseCase, tv.QReplace.LexItems)
	return got
}

// QReplaceNextMatch finds next match using, QReplace.Pos and highlights it, etc
func (tv *View) QReplaceNextMatch() bool {
	nm := len(tv.QReplace.Matches)
	if nm == 0 {
		return false
	}
	tv.QReplace.Pos++
	if tv.QReplace.Pos >= nm {
		return false
	}
	tv.QReplaceSelectMatch(tv.QReplace.Pos)
	return true
}

// QReplaceSelectMatch selects match at given match index (e.g., tv.QReplace.Pos)
func (tv *View) QReplaceSelectMatch(midx int) {
	nm := len(tv.QReplace.Matches)
	if midx >= nm {
		return
	}
	m := tv.QReplace.Matches[midx]
	reg := tv.Buf.AdjustReg(m.Reg)
	pos := reg.Start
	tv.SelectReg = reg
	tv.SetCursor(pos)
	tv.SavePosHistory(tv.CursorPos)
	tv.ScrollCursorToCenterIfHidden()
	tv.QReplaceSig()
}

// QReplaceReplace replaces at given match index (e.g., tv.QReplace.Pos)
func (tv *View) QReplaceReplace(midx int) {
	nm := len(tv.QReplace.Matches)
	if midx >= nm {
		return
	}
	m := tv.QReplace.Matches[midx]
	rep := tv.QReplace.Replace
	reg := tv.Buf.AdjustReg(m.Reg)
	pos := reg.Start
	// last arg is matchCase, only if not using case to match and rep is also lower case
	matchCase := !tv.QReplace.UseCase && !lex.HasUpperCase(rep)
	tv.Buf.ReplaceText(reg.Start, reg.End, pos, rep, EditSignal, matchCase)
	tv.Highlights[midx] = textbuf.RegionNil
	tv.SetCursor(pos)
	tv.SavePosHistory(tv.CursorPos)
	tv.ScrollCursorToCenterIfHidden()
	tv.QReplaceSig()
}

// QReplaceReplaceAll replaces all remaining from index
func (tv *View) QReplaceReplaceAll(midx int) {
	nm := len(tv.QReplace.Matches)
	if midx >= nm {
		return
	}
	for mi := midx; mi < nm; mi++ {
		tv.QReplaceReplace(mi)
	}
}

// QReplaceKeyInput is an emacs-style interactive search mode -- this is called
// when keys are typed while in search mode
func (tv *View) QReplaceKeyInput(kt events.Event) {
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)

	switch {
	case kt.KeyRune() == 'y':
		tv.QReplaceReplace(tv.QReplace.Pos)
		if !tv.QReplaceNextMatch() {
			tv.QReplaceCancel()
		}
	case kt.KeyRune() == 'n':
		if !tv.QReplaceNextMatch() {
			tv.QReplaceCancel()
		}
	case kt.KeyRune() == 'q' || kt.KeyChord() == "ReturnEnter":
		tv.QReplaceCancel()
	case kt.KeyRune() == '!':
		tv.QReplaceReplaceAll(tv.QReplace.Pos)
		tv.QReplaceCancel()
	}
}

// QReplaceCancel cancels QReplace mode
func (tv *View) QReplaceCancel() {
	if !tv.QReplace.On {
		return
	}
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)
	tv.QReplace.On = false
	tv.QReplace.Pos = -1
	tv.QReplace.Matches = nil
	tv.Highlights = nil
	tv.SavePosHistory(tv.CursorPos)
	tv.SelectReset()
	tv.QReplaceSig()
}

// EscPressed emitted for KeyFunAbort or KeyFunCancelSelect -- effect depends on state..
func (tv *View) EscPressed() {
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)
	switch {
	case tv.ISearch.On:
		tv.ISearchCancel()
		tv.SetCursorShow(tv.ISearch.StartPos)
	case tv.QReplace.On:
		tv.QReplaceCancel()
		tv.SetCursorShow(tv.ISearch.StartPos)
	case tv.HasSelection():
		tv.SelectReset()
	default:
		tv.Highlights = nil
	}
}
