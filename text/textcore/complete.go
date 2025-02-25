// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"fmt"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/parse"
	"cogentcore.org/core/text/parse/complete"
	"cogentcore.org/core/text/parse/parser"
	"cogentcore.org/core/text/textpos"
)

// setCompleter sets completion functions so that completions will
// automatically be offered as the user types
func (ed *Editor) setCompleter(data any, matchFun complete.MatchFunc, editFun complete.EditFunc,
	lookupFun complete.LookupFunc) {
	if ed.Complete != nil {
		if ed.Complete.Context == data {
			ed.Complete.MatchFunc = matchFun
			ed.Complete.EditFunc = editFun
			ed.Complete.LookupFunc = lookupFun
			return
		}
		ed.deleteCompleter()
	}
	ed.Complete = core.NewComplete().SetContext(data).SetMatchFunc(matchFun).
		SetEditFunc(editFun).SetLookupFunc(lookupFun)
	ed.Complete.OnSelect(func(e events.Event) {
		ed.completeText(ed.Complete.Completion)
	})
	// todo: what about CompleteExtend event type?
	// TODO(kai/complete): clean this up and figure out what to do about Extend and only connecting once
	// note: only need to connect once..
	// tb.Complete.CompleteSig.ConnectOnly(func(dlg *core.Dialog) {
	// 	tbf, _ := recv.Embed(TypeBuf).(*Buf)
	// 	if sig == int64(core.CompleteSelect) {
	// 		tbf.CompleteText(data.(string)) // always use data
	// 	} else if sig == int64(core.CompleteExtend) {
	// 		tbf.CompleteExtend(data.(string)) // always use data
	// 	}
	// })
}

func (ed *Editor) deleteCompleter() {
	if ed.Complete == nil {
		return
	}
	ed.Complete.Cancel()
	ed.Complete = nil
}

// completeText edits the text using the string chosen from the completion menu
func (ed *Editor) completeText(s string) {
	if s == "" {
		return
	}
	// give the completer a chance to edit the completion before insert,
	// also it return a number of runes past the cursor to delete
	st := textpos.Pos{ed.Complete.SrcLn, 0}
	en := textpos.Pos{ed.Complete.SrcLn, ed.Lines.LineLen(ed.Complete.SrcLn)}
	var tbes string
	tbe := ed.Lines.Region(st, en)
	if tbe != nil {
		tbes = string(tbe.ToBytes())
	}
	c := ed.Complete.GetCompletion(s)
	pos := textpos.Pos{ed.Complete.SrcLn, ed.Complete.SrcCh}
	ced := ed.Complete.EditFunc(ed.Complete.Context, tbes, ed.Complete.SrcCh, c, ed.Complete.Seed)
	if ced.ForwardDelete > 0 {
		delEn := textpos.Pos{ed.Complete.SrcLn, ed.Complete.SrcCh + ced.ForwardDelete}
		ed.Lines.DeleteText(pos, delEn)
	}
	// now the normal completion insertion
	st = pos
	st.Char -= len(ed.Complete.Seed)
	ed.Lines.ReplaceText(st, pos, st, ced.NewText, lines.ReplaceNoMatchCase)
	ep := st
	ep.Char += len(ced.NewText) + ced.CursorAdjust
	ed.SetCursorShow(ep)
}

// offerComplete pops up a menu of possible completions
func (ed *Editor) offerComplete() {
	if ed.Complete == nil || ed.ISearch.On || ed.QReplace.On || ed.IsDisabled() {
		return
	}
	ed.Complete.Cancel()
	if !ed.Lines.Settings.Completion {
		return
	}
	if ed.Lines.InComment(ed.CursorPos) || ed.Lines.InLitString(ed.CursorPos) {
		return
	}

	ed.Complete.SrcLn = ed.CursorPos.Line
	ed.Complete.SrcCh = ed.CursorPos.Char
	st := textpos.Pos{ed.CursorPos.Line, 0}
	en := textpos.Pos{ed.CursorPos.Line, ed.CursorPos.Char}
	tbe := ed.Lines.Region(st, en)
	var s string
	if tbe != nil {
		s = string(tbe.ToBytes())
		s = strings.TrimLeft(s, " \t") // trim ' ' and '\t'
	}

	//	count := ed.Buf.ByteOffs[ed.CursorPos.Line] + ed.CursorPos.Char
	cpos := ed.charStartPos(ed.CursorPos).ToPoint() // physical location
	cpos.X += 5
	cpos.Y += 10
	ed.Complete.SrcLn = ed.CursorPos.Line
	ed.Complete.SrcCh = ed.CursorPos.Char
	ed.Complete.Show(ed, cpos, s)
}

// CancelComplete cancels any pending completion.
// Call this when new events have moved beyond any prior completion scenario.
func (ed *Editor) CancelComplete() {
	if ed.Lines == nil {
		return
	}
	if ed.Complete == nil {
		return
	}
	ed.Complete.Cancel()
}

// Lookup attempts to lookup symbol at current location, popping up a window
// if something is found.
func (ed *Editor) Lookup() { //types:add
	if ed.Complete == nil || ed.ISearch.On || ed.QReplace.On || ed.IsDisabled() {
		return
	}

	var ln int
	var ch int
	if ed.HasSelection() {
		ln = ed.SelectRegion.Start.Line
		if ed.SelectRegion.End.Line != ln {
			return // no multiline selections for lookup
		}
		ch = ed.SelectRegion.End.Char
	} else {
		ln = ed.CursorPos.Line
		if ed.Lines.IsWordEnd(ed.CursorPos) {
			ch = ed.CursorPos.Char
		} else {
			ch = ed.Lines.WordAt(ed.CursorPos).End.Char
		}
	}
	ed.Complete.SrcLn = ln
	ed.Complete.SrcCh = ch
	st := textpos.Pos{ed.CursorPos.Line, 0}
	en := textpos.Pos{ed.CursorPos.Line, ch}

	tbe := ed.Lines.Region(st, en)
	var s string
	if tbe != nil {
		s = string(tbe.ToBytes())
		s = strings.TrimLeft(s, " \t") // trim ' ' and '\t'
	}

	//	count := ed.Buf.ByteOffs[ed.CursorPos.Line] + ed.CursorPos.Char
	cpos := ed.charStartPos(ed.CursorPos).ToPoint() // physical location
	cpos.X += 5
	cpos.Y += 10
	ed.Complete.Lookup(s, ed.CursorPos.Line, ed.CursorPos.Char, ed.Scene, cpos)
}

// completeParse uses [parse] symbols and language; the string is a line of text
// up to point where user has typed.
// The data must be the *FileState from which the language type is obtained.
func completeParse(data any, text string, posLine, posChar int) (md complete.Matches) {
	sfs := data.(*parse.FileStates)
	if sfs == nil {
		// log.Printf("CompletePi: data is nil not FileStates or is nil - can't complete\n")
		return md
	}
	lp, err := parse.LanguageSupport.Properties(sfs.Known)
	if err != nil {
		// log.Printf("CompletePi: %v\n", err)
		return md
	}
	if lp.Lang == nil {
		return md
	}

	// note: must have this set to ture to allow viewing of AST
	// must set it in pi/parse directly -- so it is changed in the fileparse too
	parser.GUIActive = true // note: this is key for debugging -- runs slower but makes the tree unique

	md = lp.Lang.CompleteLine(sfs, text, textpos.Pos{posLine, posChar})
	return md
}

// completeEditParse uses the selected completion to edit the text.
func completeEditParse(data any, text string, cursorPos int, comp complete.Completion, seed string) (ed complete.Edit) {
	sfs := data.(*parse.FileStates)
	if sfs == nil {
		// log.Printf("CompleteEditPi: data is nil not FileStates or is nil - can't complete\n")
		return ed
	}
	lp, err := parse.LanguageSupport.Properties(sfs.Known)
	if err != nil {
		// log.Printf("CompleteEditPi: %v\n", err)
		return ed
	}
	if lp.Lang == nil {
		return ed
	}
	return lp.Lang.CompleteEdit(sfs, text, cursorPos, comp, seed)
}

// lookupParse uses [parse] symbols and language; the string is a line of text
// up to point where user has typed.
// The data must be the *FileState from which the language type is obtained.
func lookupParse(data any, txt string, posLine, posChar int) (ld complete.Lookup) {
	sfs := data.(*parse.FileStates)
	if sfs == nil {
		// log.Printf("LookupPi: data is nil not FileStates or is nil - can't lookup\n")
		return ld
	}
	lp, err := parse.LanguageSupport.Properties(sfs.Known)
	if err != nil {
		// log.Printf("LookupPi: %v\n", err)
		return ld
	}
	if lp.Lang == nil {
		return ld
	}

	// note: must have this set to ture to allow viewing of AST
	// must set it in pi/parse directly -- so it is changed in the fileparse too
	parser.GUIActive = true // note: this is key for debugging -- runs slower but makes the tree unique

	ld = lp.Lang.Lookup(sfs, txt, textpos.Pos{posLine, posChar})
	if len(ld.Text) > 0 {
		// todo:
		// TextDialog(nil, "Lookup: "+txt, string(ld.Text))
		return ld
	}
	if ld.Filename != "" {
		tx := lines.FileRegionBytes(ld.Filename, ld.StLine, ld.EdLine, true, 10) // comments, 10 lines back max
		prmpt := fmt.Sprintf("%v [%d:%d]", ld.Filename, ld.StLine, ld.EdLine)
		_ = tx
		_ = prmpt
		// todo:
		// TextDialog(nil, "Lookup: "+txt+": "+prmpt, string(tx))
		return ld
	}

	return ld
}
