// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

import (
	"strings"
	"unicode"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/fileinfo"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/pi/lex"
	"cogentcore.org/core/pi/token"
	"cogentcore.org/core/texteditor/textbuf"
)

///////////////////////////////////////////////////////////////////////////////
//    Complete and Spell

// OfferComplete pops up a menu of possible completions
func (ed *Editor) OfferComplete() {
	if ed.Buffer.Complete == nil || ed.ISearch.On || ed.QReplace.On || ed.IsDisabled() {
		return
	}
	ed.Buffer.Complete.Cancel()
	if !ed.Buffer.Opts.Completion {
		return
	}
	if ed.Buffer.InComment(ed.CursorPos) || ed.Buffer.InLitString(ed.CursorPos) {
		return
	}

	ed.Buffer.Complete.SrcLn = ed.CursorPos.Ln
	ed.Buffer.Complete.SrcCh = ed.CursorPos.Ch
	st := lex.Pos{ed.CursorPos.Ln, 0}
	en := lex.Pos{ed.CursorPos.Ln, ed.CursorPos.Ch}
	tbe := ed.Buffer.Region(st, en)
	var s string
	if tbe != nil {
		s = string(tbe.ToBytes())
		s = strings.TrimLeft(s, " \t") // trim ' ' and '\t'
	}

	//	count := ed.Buf.ByteOffs[ed.CursorPos.Ln] + ed.CursorPos.Ch
	cpos := ed.CharStartPos(ed.CursorPos).ToPoint() // physical location
	cpos.X += 5
	cpos.Y += 10
	ed.Buffer.SetByteOffs() // make sure the pos offset is updated!!
	ed.Buffer.CurView = ed
	ed.Buffer.Complete.SrcLn = ed.CursorPos.Ln
	ed.Buffer.Complete.SrcCh = ed.CursorPos.Ch
	ed.Buffer.Complete.Show(ed, cpos, s)
}

// CancelComplete cancels any pending completion.
// Call this when new events have moved beyond any prior completion scenario.
func (ed *Editor) CancelComplete() {
	if ed.Buffer == nil {
		return
	}
	if ed.Buffer.Complete == nil {
		return
	}
	if ed.Buffer.Complete.Cancel() {
		ed.Buffer.CurView = nil
	}
}

// Lookup attempts to lookup symbol at current location, popping up a window
// if something is found.
func (ed *Editor) Lookup() { //gti:add
	if ed.Buffer.Complete == nil || ed.ISearch.On || ed.QReplace.On || ed.IsDisabled() {
		return
	}

	var ln int
	var ch int
	if ed.HasSelection() {
		ln = ed.SelectRegion.Start.Ln
		if ed.SelectRegion.End.Ln != ln {
			return // no multiline selections for lookup
		}
		ch = ed.SelectRegion.End.Ch
	} else {
		ln = ed.CursorPos.Ln
		if ed.IsWordEnd(ed.CursorPos) {
			ch = ed.CursorPos.Ch
		} else {
			ch = ed.WordAt().End.Ch
		}
	}
	ed.Buffer.Complete.SrcLn = ln
	ed.Buffer.Complete.SrcCh = ch
	st := lex.Pos{ed.CursorPos.Ln, 0}
	en := lex.Pos{ed.CursorPos.Ln, ch}

	tbe := ed.Buffer.Region(st, en)
	var s string
	if tbe != nil {
		s = string(tbe.ToBytes())
		s = strings.TrimLeft(s, " \t") // trim ' ' and '\t'
	}

	//	count := ed.Buf.ByteOffs[ed.CursorPos.Ln] + ed.CursorPos.Ch
	cpos := ed.CharStartPos(ed.CursorPos).ToPoint() // physical location
	cpos.X += 5
	cpos.Y += 10
	ed.Buffer.SetByteOffs() // make sure the pos offset is updated!!
	ed.Buffer.CurView = ed
	ed.Buffer.Complete.Lookup(s, ed.CursorPos.Ln, ed.CursorPos.Ch, ed.Scene, cpos)
}

// ISpellKeyInput locates the word to spell check based on cursor position and
// the key input, then passes the text region to SpellCheck
func (ed *Editor) ISpellKeyInput(kt events.Event) {
	if !ed.Buffer.IsSpellEnabled(ed.CursorPos) {
		return
	}

	isDoc := ed.Buffer.Info.Cat == fileinfo.Doc
	tp := ed.CursorPos

	kf := keymap.Of(kt.KeyChord())
	switch kf {
	case keymap.MoveUp:
		if isDoc {
			ed.Buffer.SpellCheckLineTag(tp.Ln)
		}
	case keymap.MoveDown:
		if isDoc {
			ed.Buffer.SpellCheckLineTag(tp.Ln)
		}
	case keymap.MoveRight:
		if ed.IsWordEnd(tp) {
			reg := ed.WordBefore(tp)
			ed.SpellCheck(reg)
			break
		}
		if tp.Ch == 0 { // end of line
			tp.Ln--
			if isDoc {
				ed.Buffer.SpellCheckLineTag(tp.Ln) // redo prior line
			}
			tp.Ch = ed.Buffer.LineLen(tp.Ln)
			reg := ed.WordBefore(tp)
			ed.SpellCheck(reg)
			break
		}
		txt := ed.Buffer.Line(tp.Ln)
		var r rune
		atend := false
		if tp.Ch >= len(txt) {
			atend = true
			tp.Ch++
		} else {
			r = txt[tp.Ch]
		}
		if atend || core.IsWordBreak(r, rune(-1)) {
			tp.Ch-- // we are one past the end of word
			reg := ed.WordBefore(tp)
			ed.SpellCheck(reg)
		}
	case keymap.Enter:
		tp.Ln--
		if isDoc {
			ed.Buffer.SpellCheckLineTag(tp.Ln) // redo prior line
		}
		tp.Ch = ed.Buffer.LineLen(tp.Ln)
		reg := ed.WordBefore(tp)
		ed.SpellCheck(reg)
	case keymap.FocusNext:
		tp.Ch-- // we are one past the end of word
		reg := ed.WordBefore(tp)
		ed.SpellCheck(reg)
	case keymap.Backspace, keymap.Delete:
		if ed.IsWordMiddle(ed.CursorPos) {
			reg := ed.WordAt()
			ed.SpellCheck(ed.Buffer.Region(reg.Start, reg.End))
		} else {
			reg := ed.WordBefore(tp)
			ed.SpellCheck(reg)
		}
	case keymap.None:
		if unicode.IsSpace(kt.KeyRune()) || unicode.IsPunct(kt.KeyRune()) && kt.KeyRune() != '\'' { // contractions!
			tp.Ch-- // we are one past the end of word
			reg := ed.WordBefore(tp)
			ed.SpellCheck(reg)
		} else {
			if ed.IsWordMiddle(ed.CursorPos) {
				reg := ed.WordAt()
				ed.SpellCheck(ed.Buffer.Region(reg.Start, reg.End))
			}
		}
	}
}

// SpellCheck offers spelling corrections if we are at a word break or other word termination
// and the word before the break is unknown -- returns true if misspelled word found
func (ed *Editor) SpellCheck(reg *textbuf.Edit) bool {
	if ed.Buffer.Spell == nil {
		return false
	}
	wb := string(reg.ToBytes())
	lwb := lex.FirstWordApostrophe(wb) // only lookup words
	if len(lwb) <= 2 {
		return false
	}
	widx := strings.Index(wb, lwb) // adjust region for actual part looking up
	ld := len(wb) - len(lwb)
	reg.Reg.Start.Ch += widx
	reg.Reg.End.Ch += widx - ld

	sugs, knwn := ed.Buffer.Spell.CheckWord(lwb)
	if knwn {
		ed.Buffer.RemoveTag(reg.Reg.Start, token.TextSpellErr)
		return false
	}
	// fmt.Printf("spell err: %s\n", wb)
	ed.Buffer.Spell.SetWord(wb, sugs, reg.Reg.Start.Ln, reg.Reg.Start.Ch)
	ed.Buffer.RemoveTag(reg.Reg.Start, token.TextSpellErr)
	ed.Buffer.AddTagEdit(reg, token.TextSpellErr)
	return true
}

// OfferCorrect pops up a menu of possible spelling corrections for word at
// current CursorPos. If no misspelling there or not in spellcorrect mode
// returns false
func (ed *Editor) OfferCorrect() bool {
	if ed.Buffer.Spell == nil || ed.ISearch.On || ed.QReplace.On || ed.IsDisabled() {
		return false
	}
	sel := ed.SelectRegion
	if !ed.SelectWord() {
		ed.SelectRegion = sel
		return false
	}
	tbe := ed.Selection()
	if tbe == nil {
		ed.SelectRegion = sel
		return false
	}
	ed.SelectRegion = sel
	wb := string(tbe.ToBytes())
	wbn := strings.TrimLeft(wb, " \t")
	if len(wb) != len(wbn) {
		return false // SelectWord captures leading whitespace - don't offer if there is leading whitespace
	}
	sugs, knwn := ed.Buffer.Spell.CheckWord(wb)
	if knwn && !ed.Buffer.Spell.IsLastLearned(wb) {
		return false
	}
	ed.Buffer.Spell.SetWord(wb, sugs, tbe.Reg.Start.Ln, tbe.Reg.Start.Ch)

	cpos := ed.CharStartPos(ed.CursorPos).ToPoint() // physical location
	cpos.X += 5
	cpos.Y += 10
	ed.Buffer.CurView = ed
	ed.Buffer.Spell.Show(wb, ed.Scene, cpos)
	return true
}

// CancelCorrect cancels any pending spell correction.
// Call this when new events have moved beyond any prior correction scenario.
func (ed *Editor) CancelCorrect() {
	if ed.Buffer.Spell == nil || ed.ISearch.On || ed.QReplace.On {
		return
	}
	if !ed.Buffer.Opts.SpellCorrect {
		return
	}
	ed.Buffer.CurView = nil
	ed.Buffer.Spell.Cancel()
}
