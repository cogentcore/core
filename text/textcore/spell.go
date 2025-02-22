// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"strings"
	"unicode"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/events"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/parse/lexer"
	"cogentcore.org/core/text/spell"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/text/token"
)

// iSpellKeyInput locates the word to spell check based on cursor position and
// the key input, then passes the text region to SpellCheck
func (ed *Editor) iSpellKeyInput(kt events.Event) {
	if !ed.isSpellEnabled(ed.CursorPos) {
		return
	}
	isDoc := ed.Lines.FileInfo().Cat == fileinfo.Doc
	tp := ed.CursorPos
	kf := keymap.Of(kt.KeyChord())
	switch kf {
	case keymap.MoveUp:
		if isDoc {
			ed.spellCheckLineTag(tp.Line)
		}
	case keymap.MoveDown:
		if isDoc {
			ed.spellCheckLineTag(tp.Line)
		}
	case keymap.MoveRight:
		if ed.Lines.IsWordEnd(tp) {
			reg := ed.Lines.WordBefore(tp)
			ed.spellCheck(reg)
			break
		}
		if tp.Char == 0 { // end of line
			tp.Line--
			if isDoc {
				ed.spellCheckLineTag(tp.Line) // redo prior line
			}
			tp.Char = ed.Lines.LineLen(tp.Line)
			reg := ed.Lines.WordBefore(tp)
			ed.spellCheck(reg)
			break
		}
		txt := ed.Lines.Line(tp.Line)
		var r rune
		atend := false
		if tp.Char >= len(txt) {
			atend = true
			tp.Char++
		} else {
			r = txt[tp.Char]
		}
		if atend || textpos.IsWordBreak(r, rune(-1)) {
			tp.Char-- // we are one past the end of word
			reg := ed.Lines.WordBefore(tp)
			ed.spellCheck(reg)
		}
	case keymap.Enter:
		tp.Line--
		if isDoc {
			ed.spellCheckLineTag(tp.Line) // redo prior line
		}
		tp.Char = ed.Lines.LineLen(tp.Line)
		reg := ed.Lines.WordBefore(tp)
		ed.spellCheck(reg)
	case keymap.FocusNext:
		tp.Char-- // we are one past the end of word
		reg := ed.Lines.WordBefore(tp)
		ed.spellCheck(reg)
	case keymap.Backspace, keymap.Delete:
		if ed.Lines.IsWordMiddle(ed.CursorPos) {
			reg := ed.Lines.WordAt(ed.CursorPos)
			ed.spellCheck(ed.Lines.Region(reg.Start, reg.End))
		} else {
			reg := ed.Lines.WordBefore(tp)
			ed.spellCheck(reg)
		}
	case keymap.None:
		if unicode.IsSpace(kt.KeyRune()) || unicode.IsPunct(kt.KeyRune()) && kt.KeyRune() != '\'' { // contractions!
			tp.Char-- // we are one past the end of word
			reg := ed.Lines.WordBefore(tp)
			ed.spellCheck(reg)
		} else {
			if ed.Lines.IsWordMiddle(ed.CursorPos) {
				reg := ed.Lines.WordAt(ed.CursorPos)
				ed.spellCheck(ed.Lines.Region(reg.Start, reg.End))
			}
		}
	}
}

// spellCheck offers spelling corrections if we are at a word break or other word termination
// and the word before the break is unknown -- returns true if misspelled word found
func (ed *Editor) spellCheck(reg *textpos.Edit) bool {
	if ed.spell == nil {
		return false
	}
	wb := string(reg.ToBytes())
	lwb := lexer.FirstWordApostrophe(wb) // only lookup words
	if len(lwb) <= 2 {
		return false
	}
	widx := strings.Index(wb, lwb) // adjust region for actual part looking up
	ld := len(wb) - len(lwb)
	reg.Region.Start.Char += widx
	reg.Region.End.Char += widx - ld

	sugs, knwn := ed.spell.checkWord(lwb)
	if knwn {
		ed.Lines.RemoveTag(reg.Region.Start, token.TextSpellErr)
		return false
	}
	// fmt.Printf("spell err: %s\n", wb)
	ed.spell.setWord(wb, sugs, reg.Region.Start.Line, reg.Region.Start.Char)
	ed.Lines.RemoveTag(reg.Region.Start, token.TextSpellErr)
	ed.Lines.AddTagEdit(reg, token.TextSpellErr)
	return true
}

// offerCorrect pops up a menu of possible spelling corrections for word at
// current CursorPos. If no misspelling there or not in spellcorrect mode
// returns false
func (ed *Editor) offerCorrect() bool {
	if ed.spell == nil || ed.ISearch.On || ed.QReplace.On || ed.IsDisabled() {
		return false
	}
	sel := ed.SelectRegion
	if !ed.selectWord() {
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
	sugs, knwn := ed.spell.checkWord(wb)
	if knwn && !ed.spell.isLastLearned(wb) {
		return false
	}
	ed.spell.setWord(wb, sugs, tbe.Region.Start.Line, tbe.Region.Start.Char)

	cpos := ed.charStartPos(ed.CursorPos).ToPoint() // physical location
	cpos.X += 5
	cpos.Y += 10
	ed.spell.show(wb, ed.Scene, cpos)
	return true
}

// cancelCorrect cancels any pending spell correction.
// Call this when new events have moved beyond any prior correction scenario.
func (ed *Editor) cancelCorrect() {
	if ed.spell == nil || ed.ISearch.On || ed.QReplace.On {
		return
	}
	if !ed.Lines.Settings.SpellCorrect {
		return
	}
	ed.spell.cancel()
}

// isSpellEnabled returns true if spelling correction is enabled,
// taking into account given position in text if it is relevant for cases
// where it is only conditionally enabled
func (ed *Editor) isSpellEnabled(pos textpos.Pos) bool {
	if ed.spell == nil || !ed.Lines.Settings.SpellCorrect {
		return false
	}
	switch ed.Lines.FileInfo().Cat {
	case fileinfo.Doc: // not in code!
		return !ed.Lines.InTokenCode(pos)
	case fileinfo.Code:
		return ed.Lines.InComment(pos) || ed.Lines.InLitString(pos)
	default:
		return false
	}
}

// setSpell sets spell correct functions so that spell correct will
// automatically be offered as the user types
func (ed *Editor) setSpell() {
	if ed.spell != nil {
		return
	}
	initSpell()
	ed.spell = newSpell()
	ed.spell.onSelect(func(e events.Event) {
		ed.correctText(ed.spell.correction)
	})
}

// correctText edits the text using the string chosen from the correction menu
func (ed *Editor) correctText(s string) {
	st := textpos.Pos{ed.spell.srcLn, ed.spell.srcCh} // start of word
	ed.Lines.RemoveTag(st, token.TextSpellErr)
	oend := st
	oend.Char += len(ed.spell.word)
	ed.Lines.ReplaceText(st, oend, st, s, lines.ReplaceNoMatchCase)
	ep := st
	ep.Char += len(s)
	ed.SetCursorShow(ep)
}

// SpellCheckLineErrors runs spell check on given line, and returns Lex tags
// with token.TextSpellErr for any misspelled words
func (ed *Editor) SpellCheckLineErrors(ln int) lexer.Line {
	if !ed.Lines.IsValidLine(ln) {
		return nil
	}
	return spell.CheckLexLine(ed.Lines.Line(ln), ed.Lines.HiTags(ln))
}

// spellCheckLineTag runs spell check on given line, and sets Tags for any
// misspelled words and updates markup for that line.
func (ed *Editor) spellCheckLineTag(ln int) {
	if !ed.Lines.IsValidLine(ln) {
		return
	}
	ser := ed.SpellCheckLineErrors(ln)
	ntgs := ed.Lines.AdjustedTags(ln)
	ntgs.DeleteToken(token.TextSpellErr)
	for _, t := range ser {
		ntgs.AddSort(t)
	}
	ed.Lines.SetTags(ln, ntgs)
	ed.Lines.MarkupLines(ln, ln)
	ed.Lines.StartDelayedReMarkup()
}
