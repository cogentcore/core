// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

import (
	"strings"
	"unicode"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/texteditor/textbuf"
	"goki.dev/goosi/events"
	"goki.dev/pi/v2/filecat"
	"goki.dev/pi/v2/lex"
	"goki.dev/pi/v2/token"
)

///////////////////////////////////////////////////////////////////////////////
//    Complete and Spell

// OfferComplete pops up a menu of possible completions
func (tv *View) OfferComplete() {
	if tv.Buf.Complete == nil || tv.ISearch.On || tv.QReplace.On || tv.IsDisabled() {
		return
	}
	tv.Buf.Complete.Cancel()
	if !tv.Buf.Opts.Completion && !tv.ForceComplete {
		return
	}
	if tv.Buf.InComment(tv.CursorPos) || tv.Buf.InLitString(tv.CursorPos) {
		return
	}

	tv.Buf.Complete.SrcLn = tv.CursorPos.Ln
	tv.Buf.Complete.SrcCh = tv.CursorPos.Ch
	st := lex.Pos{tv.CursorPos.Ln, 0}
	en := lex.Pos{tv.CursorPos.Ln, tv.CursorPos.Ch}
	tbe := tv.Buf.Region(st, en)
	var s string
	if tbe != nil {
		s = string(tbe.ToBytes())
		s = strings.TrimLeft(s, " \t") // trim ' ' and '\t'
	}

	//	count := tv.Buf.ByteOffs[tv.CursorPos.Ln] + tv.CursorPos.Ch
	cpos := tv.CharStartPos(tv.CursorPos).ToPoint() // physical location
	cpos.X += 5
	cpos.Y += 10
	tv.Buf.SetByteOffs() // make sure the pos offset is updated!!
	tv.Buf.CurView = tv
	tv.Buf.Complete.Show(s, tv.CursorPos.Ln, tv.CursorPos.Ch, tv.Sc, cpos, tv.ForceComplete)
}

// CancelComplete cancels any pending completion -- call this when new events
// have moved beyond any prior completion scenario
func (tv *View) CancelComplete() {
	tv.ForceComplete = false
	if tv.Buf == nil {
		return
	}
	if tv.Buf.Complete == nil {
		return
	}
	if tv.Buf.Complete.Cancel() {
		tv.Buf.CurView = nil
	}
}

// Lookup attempts to lookup symbol at current location, popping up a window
// if something is found
func (tv *View) Lookup() {
	if tv.Buf.Complete == nil || tv.ISearch.On || tv.QReplace.On || tv.IsDisabled() {
		return
	}

	var ln int
	var ch int
	if tv.HasSelection() {
		ln = tv.SelectReg.Start.Ln
		if tv.SelectReg.End.Ln != ln {
			return // no multiline selections for lookup
		}
		ch = tv.SelectReg.End.Ch
	} else {
		ln = tv.CursorPos.Ln
		if tv.IsWordEnd(tv.CursorPos) {
			ch = tv.CursorPos.Ch
		} else {
			ch = tv.WordAt().End.Ch
		}
	}
	tv.Buf.Complete.SrcLn = ln
	tv.Buf.Complete.SrcCh = ch
	st := lex.Pos{tv.CursorPos.Ln, 0}
	en := lex.Pos{tv.CursorPos.Ln, ch}

	tbe := tv.Buf.Region(st, en)
	var s string
	if tbe != nil {
		s = string(tbe.ToBytes())
		s = strings.TrimLeft(s, " \t") // trim ' ' and '\t'
	}

	//	count := tv.Buf.ByteOffs[tv.CursorPos.Ln] + tv.CursorPos.Ch
	cpos := tv.CharStartPos(tv.CursorPos).ToPoint() // physical location
	cpos.X += 5
	cpos.Y += 10
	tv.Buf.SetByteOffs() // make sure the pos offset is updated!!
	tv.Buf.CurView = tv
	tv.Buf.Complete.Lookup(s, tv.CursorPos.Ln, tv.CursorPos.Ch, tv.Sc, cpos, tv.ForceComplete)
}

// ISpellKeyInput locates the word to spell check based on cursor position and
// the key input, then passes the text region to SpellCheck
func (tv *View) ISpellKeyInput(kt events.Event) {
	if !tv.Buf.IsSpellEnabled(tv.CursorPos) {
		return
	}

	isDoc := tv.Buf.Info.Cat == filecat.Doc
	tp := tv.CursorPos

	kf := gi.KeyFun(kt.KeyChord())
	switch kf {
	case gi.KeyFunMoveUp:
		if isDoc {
			tv.Buf.SpellCheckLineTag(tp.Ln)
		}
	case gi.KeyFunMoveDown:
		if isDoc {
			tv.Buf.SpellCheckLineTag(tp.Ln)
		}
	case gi.KeyFunMoveRight:
		if tv.IsWordEnd(tp) {
			reg := tv.WordBefore(tp)
			tv.SpellCheck(reg)
			break
		}
		if tp.Ch == 0 { // end of line
			tp.Ln--
			if isDoc {
				tv.Buf.SpellCheckLineTag(tp.Ln) // redo prior line
			}
			tp.Ch = tv.Buf.LineLen(tp.Ln)
			reg := tv.WordBefore(tp)
			tv.SpellCheck(reg)
			break
		}
		txt := tv.Buf.Line(tp.Ln)
		var r rune
		atend := false
		if tp.Ch >= len(txt) {
			atend = true
			tp.Ch++
		} else {
			r = txt[tp.Ch]
		}
		if atend || lex.IsWordBreak(r, rune(-1)) {
			tp.Ch-- // we are one past the end of word
			reg := tv.WordBefore(tp)
			tv.SpellCheck(reg)
		}
	case gi.KeyFunEnter:
		tp.Ln--
		if isDoc {
			tv.Buf.SpellCheckLineTag(tp.Ln) // redo prior line
		}
		tp.Ch = tv.Buf.LineLen(tp.Ln)
		reg := tv.WordBefore(tp)
		tv.SpellCheck(reg)
	case gi.KeyFunFocusNext:
		tp.Ch-- // we are one past the end of word
		reg := tv.WordBefore(tp)
		tv.SpellCheck(reg)
	case gi.KeyFunBackspace, gi.KeyFunDelete:
		if tv.IsWordMiddle(tv.CursorPos) {
			reg := tv.WordAt()
			tv.SpellCheck(tv.Buf.Region(reg.Start, reg.End))
		} else {
			reg := tv.WordBefore(tp)
			tv.SpellCheck(reg)
		}
	case gi.KeyFunNil:
		if unicode.IsSpace(kt.KeyRune()) || unicode.IsPunct(kt.KeyRune()) && kt.KeyRune() != '\'' { // contractions!
			tp.Ch-- // we are one past the end of word
			reg := tv.WordBefore(tp)
			tv.SpellCheck(reg)
		} else {
			if tv.IsWordMiddle(tv.CursorPos) {
				reg := tv.WordAt()
				tv.SpellCheck(tv.Buf.Region(reg.Start, reg.End))
			}
		}
	}
}

// SpellCheck offers spelling corrections if we are at a word break or other word termination
// and the word before the break is unknown -- returns true if misspelled word found
func (tv *View) SpellCheck(reg *textbuf.Edit) bool {
	if tv.Buf.Spell == nil {
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

	sugs, knwn := tv.Buf.Spell.CheckWord(lwb)
	if knwn {
		tv.Buf.RemoveTag(reg.Reg.Start, token.TextSpellErr)
		return false
	}
	// fmt.Printf("spell err: %s\n", wb)
	tv.Buf.Spell.SetWord(wb, sugs, reg.Reg.Start.Ln, reg.Reg.Start.Ch)
	tv.Buf.RemoveTag(reg.Reg.Start, token.TextSpellErr)
	tv.Buf.AddTagEdit(reg, token.TextSpellErr)
	return true
}

// OfferCorrect pops up a menu of possible spelling corrections for word at
// current CursorPos -- if no misspelling there or not in spellcorrect mode
// returns false
func (tv *View) OfferCorrect() bool {
	if tv.Buf.Spell == nil || tv.ISearch.On || tv.QReplace.On || tv.IsDisabled() {
		return false
	}
	sel := tv.SelectReg
	if !tv.SelectWord() {
		tv.SelectReg = sel
		return false
	}
	tbe := tv.Selection()
	if tbe == nil {
		tv.SelectReg = sel
		return false
	}
	tv.SelectReg = sel
	wb := string(tbe.ToBytes())
	wbn := strings.TrimLeft(wb, " \t")
	if len(wb) != len(wbn) {
		return false // SelectWord captures leading whitespace - don't offer if there is leading whitespace
	}
	sugs, knwn := tv.Buf.Spell.CheckWord(wb)
	if knwn && !tv.Buf.Spell.IsLastLearned(wb) {
		return false
	}
	tv.Buf.Spell.SetWord(wb, sugs, tbe.Reg.Start.Ln, tbe.Reg.Start.Ch)

	cpos := tv.CharStartPos(tv.CursorPos).ToPoint() // physical location
	cpos.X += 5
	cpos.Y += 10
	tv.Buf.CurView = tv
	tv.Buf.Spell.Show(wb, tv.Sc, cpos)
	return true
}

// CancelCorrect cancels any pending spell correction -- call this when new events
// have moved beyond any prior correction scenario
func (tv *View) CancelCorrect() {
	if tv.Buf.Spell == nil || tv.ISearch.On || tv.QReplace.On {
		return
	}
	if !tv.Buf.Opts.SpellCorrect {
		return
	}
	tv.Buf.CurView = nil
	tv.Buf.Spell.Cancel()
}
