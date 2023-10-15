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
func (ed *Editor) OfferComplete() {
	if ed.Buf.Complete == nil || ed.ISearch.On || ed.QReplace.On || ed.IsDisabled() {
		return
	}
	ed.Buf.Complete.Cancel()
	if !ed.Buf.Opts.Completion && !ed.ForceComplete {
		return
	}
	if ed.Buf.InComment(ed.CursorPos) || ed.Buf.InLitString(ed.CursorPos) {
		return
	}

	ed.Buf.Complete.SrcLn = ed.CursorPos.Ln
	ed.Buf.Complete.SrcCh = ed.CursorPos.Ch
	st := lex.Pos{ed.CursorPos.Ln, 0}
	en := lex.Pos{ed.CursorPos.Ln, ed.CursorPos.Ch}
	tbe := ed.Buf.Region(st, en)
	var s string
	if tbe != nil {
		s = string(tbe.ToBytes())
		s = strings.TrimLeft(s, " \t") // trim ' ' and '\t'
	}

	//	count := ed.Buf.ByteOffs[ed.CursorPos.Ln] + ed.CursorPos.Ch
	cpos := ed.CharStartPos(ed.CursorPos).ToPoint() // physical location
	cpos.X += 5
	cpos.Y += 10
	ed.Buf.SetByteOffs() // make sure the pos offset is updated!!
	ed.Buf.CurView = ed
	ed.Buf.Complete.Show(s, ed.CursorPos.Ln, ed.CursorPos.Ch, ed.Sc, cpos, ed.ForceComplete)
}

// CancelComplete cancels any pending completion -- call this when new events
// have moved beyond any prior completion scenario
func (ed *Editor) CancelComplete() {
	ed.ForceComplete = false
	if ed.Buf == nil {
		return
	}
	if ed.Buf.Complete == nil {
		return
	}
	if ed.Buf.Complete.Cancel() {
		ed.Buf.CurView = nil
	}
}

// Lookup attempts to lookup symbol at current location, popping up a window
// if something is found
func (ed *Editor) Lookup() {
	if ed.Buf.Complete == nil || ed.ISearch.On || ed.QReplace.On || ed.IsDisabled() {
		return
	}

	var ln int
	var ch int
	if ed.HasSelection() {
		ln = ed.SelectReg.Start.Ln
		if ed.SelectReg.End.Ln != ln {
			return // no multiline selections for lookup
		}
		ch = ed.SelectReg.End.Ch
	} else {
		ln = ed.CursorPos.Ln
		if ed.IsWordEnd(ed.CursorPos) {
			ch = ed.CursorPos.Ch
		} else {
			ch = ed.WordAt().End.Ch
		}
	}
	ed.Buf.Complete.SrcLn = ln
	ed.Buf.Complete.SrcCh = ch
	st := lex.Pos{ed.CursorPos.Ln, 0}
	en := lex.Pos{ed.CursorPos.Ln, ch}

	tbe := ed.Buf.Region(st, en)
	var s string
	if tbe != nil {
		s = string(tbe.ToBytes())
		s = strings.TrimLeft(s, " \t") // trim ' ' and '\t'
	}

	//	count := ed.Buf.ByteOffs[ed.CursorPos.Ln] + ed.CursorPos.Ch
	cpos := ed.CharStartPos(ed.CursorPos).ToPoint() // physical location
	cpos.X += 5
	cpos.Y += 10
	ed.Buf.SetByteOffs() // make sure the pos offset is updated!!
	ed.Buf.CurView = ed
	ed.Buf.Complete.Lookup(s, ed.CursorPos.Ln, ed.CursorPos.Ch, ed.Sc, cpos, ed.ForceComplete)
}

// ISpellKeyInput locates the word to spell check based on cursor position and
// the key input, then passes the text region to SpellCheck
func (ed *Editor) ISpellKeyInput(kt events.Event) {
	if !ed.Buf.IsSpellEnabled(ed.CursorPos) {
		return
	}

	isDoc := ed.Buf.Info.Cat == filecat.Doc
	tp := ed.CursorPos

	kf := gi.KeyFun(kt.KeyChord())
	switch kf {
	case gi.KeyFunMoveUp:
		if isDoc {
			ed.Buf.SpellCheckLineTag(tp.Ln)
		}
	case gi.KeyFunMoveDown:
		if isDoc {
			ed.Buf.SpellCheckLineTag(tp.Ln)
		}
	case gi.KeyFunMoveRight:
		if ed.IsWordEnd(tp) {
			reg := ed.WordBefore(tp)
			ed.SpellCheck(reg)
			break
		}
		if tp.Ch == 0 { // end of line
			tp.Ln--
			if isDoc {
				ed.Buf.SpellCheckLineTag(tp.Ln) // redo prior line
			}
			tp.Ch = ed.Buf.LineLen(tp.Ln)
			reg := ed.WordBefore(tp)
			ed.SpellCheck(reg)
			break
		}
		txt := ed.Buf.Line(tp.Ln)
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
			reg := ed.WordBefore(tp)
			ed.SpellCheck(reg)
		}
	case gi.KeyFunEnter:
		tp.Ln--
		if isDoc {
			ed.Buf.SpellCheckLineTag(tp.Ln) // redo prior line
		}
		tp.Ch = ed.Buf.LineLen(tp.Ln)
		reg := ed.WordBefore(tp)
		ed.SpellCheck(reg)
	case gi.KeyFunFocusNext:
		tp.Ch-- // we are one past the end of word
		reg := ed.WordBefore(tp)
		ed.SpellCheck(reg)
	case gi.KeyFunBackspace, gi.KeyFunDelete:
		if ed.IsWordMiddle(ed.CursorPos) {
			reg := ed.WordAt()
			ed.SpellCheck(ed.Buf.Region(reg.Start, reg.End))
		} else {
			reg := ed.WordBefore(tp)
			ed.SpellCheck(reg)
		}
	case gi.KeyFunNil:
		if unicode.IsSpace(kt.KeyRune()) || unicode.IsPunct(kt.KeyRune()) && kt.KeyRune() != '\'' { // contractions!
			tp.Ch-- // we are one past the end of word
			reg := ed.WordBefore(tp)
			ed.SpellCheck(reg)
		} else {
			if ed.IsWordMiddle(ed.CursorPos) {
				reg := ed.WordAt()
				ed.SpellCheck(ed.Buf.Region(reg.Start, reg.End))
			}
		}
	}
}

// SpellCheck offers spelling corrections if we are at a word break or other word termination
// and the word before the break is unknown -- returns true if misspelled word found
func (ed *Editor) SpellCheck(reg *textbuf.Edit) bool {
	if ed.Buf.Spell == nil {
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

	sugs, knwn := ed.Buf.Spell.CheckWord(lwb)
	if knwn {
		ed.Buf.RemoveTag(reg.Reg.Start, token.TextSpellErr)
		return false
	}
	// fmt.Printf("spell err: %s\n", wb)
	ed.Buf.Spell.SetWord(wb, sugs, reg.Reg.Start.Ln, reg.Reg.Start.Ch)
	ed.Buf.RemoveTag(reg.Reg.Start, token.TextSpellErr)
	ed.Buf.AddTagEdit(reg, token.TextSpellErr)
	return true
}

// OfferCorrect pops up a menu of possible spelling corrections for word at
// current CursorPos -- if no misspelling there or not in spellcorrect mode
// returns false
func (ed *Editor) OfferCorrect() bool {
	if ed.Buf.Spell == nil || ed.ISearch.On || ed.QReplace.On || ed.IsDisabled() {
		return false
	}
	sel := ed.SelectReg
	if !ed.SelectWord() {
		ed.SelectReg = sel
		return false
	}
	tbe := ed.Selection()
	if tbe == nil {
		ed.SelectReg = sel
		return false
	}
	ed.SelectReg = sel
	wb := string(tbe.ToBytes())
	wbn := strings.TrimLeft(wb, " \t")
	if len(wb) != len(wbn) {
		return false // SelectWord captures leading whitespace - don't offer if there is leading whitespace
	}
	sugs, knwn := ed.Buf.Spell.CheckWord(wb)
	if knwn && !ed.Buf.Spell.IsLastLearned(wb) {
		return false
	}
	ed.Buf.Spell.SetWord(wb, sugs, tbe.Reg.Start.Ln, tbe.Reg.Start.Ch)

	cpos := ed.CharStartPos(ed.CursorPos).ToPoint() // physical location
	cpos.X += 5
	cpos.Y += 10
	ed.Buf.CurView = ed
	ed.Buf.Spell.Show(wb, ed.Sc, cpos)
	return true
}

// CancelCorrect cancels any pending spell correction -- call this when new events
// have moved beyond any prior correction scenario
func (ed *Editor) CancelCorrect() {
	if ed.Buf.Spell == nil || ed.ISearch.On || ed.QReplace.On {
		return
	}
	if !ed.Buf.Opts.SpellCorrect {
		return
	}
	ed.Buf.CurView = nil
	ed.Buf.Spell.Cancel()
}
