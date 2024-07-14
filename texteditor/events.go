// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

import (
	"fmt"
	"image"
	"unicode"

	"cogentcore.org/core/base/indent"
	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/parse"
	"cogentcore.org/core/parse/lexer"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/system"
	"cogentcore.org/core/texteditor/textbuf"
)

func (ed *Editor) handleFocus() {
	ed.OnFocusLost(func(e events.Event) {
		if ed.IsReadOnly() {
			ed.clearCursor()
			return
		}
		if ed.AbilityIs(abilities.Focusable) {
			ed.editDone()
			ed.SetState(false, states.Focused)
		}
	})
}

func (ed *Editor) handleKeyChord() {
	ed.OnKeyChord(func(e events.Event) {
		ed.keyInput(e)
	})
}

// shiftSelect sets the selection start if the shift key is down but wasn't on the last key move.
// If the shift key has been released the select region is set to textbuf.RegionNil
func (ed *Editor) shiftSelect(kt events.Event) {
	hasShift := kt.HasAnyModifier(key.Shift)
	if hasShift {
		if ed.SelectRegion == textbuf.RegionNil {
			ed.selectStart = ed.CursorPos
		}
	} else {
		ed.SelectRegion = textbuf.RegionNil
	}
}

// shiftSelectExtend updates the select region if the shift key is down and renders the selected text.
// If the shift key is not down the previously selected text is rerendered to clear the highlight
func (ed *Editor) shiftSelectExtend(kt events.Event) {
	hasShift := kt.HasAnyModifier(key.Shift)
	if hasShift {
		ed.selectRegionUpdate(ed.CursorPos)
	}
}

// keyInput handles keyboard input into the text field and from the completion menu
func (ed *Editor) keyInput(e events.Event) {
	if core.DebugSettings.KeyEventTrace {
		fmt.Printf("View KeyInput: %v\n", ed.Path())
	}
	kf := keymap.Of(e.KeyChord())

	if e.IsHandled() {
		return
	}
	if ed.Buffer == nil || ed.Buffer.numLines() == 0 {
		return
	}

	// cancelAll cancels search, completer, and..
	cancelAll := func() {
		ed.CancelComplete()
		ed.CancelCorrect()
		ed.iSearchCancel()
		ed.qReplaceCancel()
		ed.lastAutoInsert = 0
	}

	if kf != keymap.Recenter { // always start at centering
		ed.lastRecenter = 0
	}

	if kf != keymap.Undo && ed.lastWasUndo {
		ed.Buffer.emacsUndoSave()
		ed.lastWasUndo = false
	}

	gotTabAI := false // got auto-indent tab this time

	// first all the keys that work for both inactive and active
	switch kf {
	case keymap.MoveRight:
		cancelAll()
		e.SetHandled()
		ed.shiftSelect(e)
		ed.cursorForward(1)
		ed.shiftSelectExtend(e)
		ed.ISpellKeyInput(e)
	case keymap.WordRight:
		cancelAll()
		e.SetHandled()
		ed.shiftSelect(e)
		ed.cursorForwardWord(1)
		ed.shiftSelectExtend(e)
	case keymap.MoveLeft:
		cancelAll()
		e.SetHandled()
		ed.shiftSelect(e)
		ed.cursorBackward(1)
		ed.shiftSelectExtend(e)
	case keymap.WordLeft:
		cancelAll()
		e.SetHandled()
		ed.shiftSelect(e)
		ed.cursorBackwardWord(1)
		ed.shiftSelectExtend(e)
	case keymap.MoveUp:
		cancelAll()
		e.SetHandled()
		ed.shiftSelect(e)
		ed.cursorUp(1)
		ed.shiftSelectExtend(e)
		ed.ISpellKeyInput(e)
	case keymap.MoveDown:
		cancelAll()
		e.SetHandled()
		ed.shiftSelect(e)
		ed.cursorDown(1)
		ed.shiftSelectExtend(e)
		ed.ISpellKeyInput(e)
	case keymap.PageUp:
		cancelAll()
		e.SetHandled()
		ed.shiftSelect(e)
		ed.cursorPageUp(1)
		ed.shiftSelectExtend(e)
	case keymap.PageDown:
		cancelAll()
		e.SetHandled()
		ed.shiftSelect(e)
		ed.cursorPageDown(1)
		ed.shiftSelectExtend(e)
	case keymap.Home:
		cancelAll()
		e.SetHandled()
		ed.shiftSelect(e)
		ed.cursorStartLine()
		ed.shiftSelectExtend(e)
	case keymap.End:
		cancelAll()
		e.SetHandled()
		ed.shiftSelect(e)
		ed.cursorEndLine()
		ed.shiftSelectExtend(e)
	case keymap.DocHome:
		cancelAll()
		e.SetHandled()
		ed.shiftSelect(e)
		ed.CursorStartDoc()
		ed.shiftSelectExtend(e)
	case keymap.DocEnd:
		cancelAll()
		e.SetHandled()
		ed.shiftSelect(e)
		ed.cursorEndDoc()
		ed.shiftSelectExtend(e)
	case keymap.Recenter:
		cancelAll()
		e.SetHandled()
		ed.reMarkup()
		ed.cursorRecenter()
	case keymap.SelectMode:
		cancelAll()
		e.SetHandled()
		ed.selectModeToggle()
	case keymap.CancelSelect:
		ed.CancelComplete()
		e.SetHandled()
		ed.escPressed() // generic cancel
	case keymap.SelectAll:
		cancelAll()
		e.SetHandled()
		ed.selectAll()
	case keymap.Copy:
		cancelAll()
		e.SetHandled()
		ed.Copy(true) // reset
	case keymap.Search:
		e.SetHandled()
		ed.qReplaceCancel()
		ed.CancelComplete()
		ed.iSearchStart()
	case keymap.Abort:
		cancelAll()
		e.SetHandled()
		ed.escPressed()
	case keymap.Jump:
		cancelAll()
		e.SetHandled()
		ed.JumpToLinePrompt()
	case keymap.HistPrev:
		cancelAll()
		e.SetHandled()
		ed.CursorToHistoryPrev()
	case keymap.HistNext:
		cancelAll()
		e.SetHandled()
		ed.CursorToHistoryNext()
	case keymap.Lookup:
		cancelAll()
		e.SetHandled()
		ed.Lookup()
	}
	if ed.IsReadOnly() {
		switch {
		case kf == keymap.FocusNext: // tab
			e.SetHandled()
			ed.CursorNextLink(true)
		case kf == keymap.FocusPrev: // tab
			e.SetHandled()
			ed.CursorPrevLink(true)
		case kf == keymap.None && ed.ISearch.On:
			if unicode.IsPrint(e.KeyRune()) && !e.HasAnyModifier(key.Control, key.Meta) {
				ed.iSearchKeyInput(e)
			}
		case e.KeyRune() == ' ' || kf == keymap.Accept || kf == keymap.Enter:
			e.SetHandled()
			ed.CursorPos.Ch--
			ed.CursorNextLink(true) // todo: cursorcurlink
			ed.OpenLinkAt(ed.CursorPos)
		}
		return
	}
	if e.IsHandled() {
		ed.lastWasTabAI = gotTabAI
		return
	}
	switch kf {
	case keymap.Replace:
		e.SetHandled()
		ed.CancelComplete()
		ed.iSearchCancel()
		ed.QReplacePrompt()
	case keymap.Backspace:
		// todo: previous item in qreplace
		if ed.ISearch.On {
			ed.iSearchBackspace()
		} else {
			e.SetHandled()
			ed.cursorBackspace(1)
			ed.ISpellKeyInput(e)
			ed.OfferComplete()
		}
	case keymap.Kill:
		cancelAll()
		e.SetHandled()
		ed.cursorKill()
	case keymap.Delete:
		cancelAll()
		e.SetHandled()
		ed.cursorDelete(1)
		ed.ISpellKeyInput(e)
	case keymap.BackspaceWord:
		cancelAll()
		e.SetHandled()
		ed.cursorBackspaceWord(1)
	case keymap.DeleteWord:
		cancelAll()
		e.SetHandled()
		ed.cursorDeleteWord(1)
	case keymap.Cut:
		cancelAll()
		e.SetHandled()
		ed.Cut()
	case keymap.Paste:
		cancelAll()
		e.SetHandled()
		ed.Paste()
	case keymap.Transpose:
		cancelAll()
		e.SetHandled()
		ed.cursorTranspose()
	case keymap.TransposeWord:
		cancelAll()
		e.SetHandled()
		ed.cursorTransposeWord()
	case keymap.PasteHist:
		cancelAll()
		e.SetHandled()
		ed.pasteHistory()
	case keymap.Accept:
		cancelAll()
		e.SetHandled()
		ed.editDone()
	case keymap.Undo:
		cancelAll()
		e.SetHandled()
		ed.undo()
		ed.lastWasUndo = true
	case keymap.Redo:
		cancelAll()
		e.SetHandled()
		ed.redo()
	case keymap.Complete:
		ed.iSearchCancel()
		e.SetHandled()
		if ed.Buffer.isSpellEnabled(ed.CursorPos) {
			ed.OfferCorrect()
		} else {
			ed.OfferComplete()
		}
	case keymap.Enter:
		cancelAll()
		if !e.HasAnyModifier(key.Control, key.Meta) {
			e.SetHandled()
			if ed.Buffer.Options.AutoIndent {
				lp, _ := parse.LanguageSupport.Properties(ed.Buffer.ParseState.Sup)
				if lp != nil && lp.Lang != nil && lp.HasFlag(parse.ReAutoIndent) {
					// only re-indent current line for supported types
					tbe, _, _ := ed.Buffer.autoIndent(ed.CursorPos.Ln) // reindent current line
					if tbe != nil {
						// go back to end of line!
						npos := lexer.Pos{Ln: ed.CursorPos.Ln, Ch: ed.Buffer.lineLen(ed.CursorPos.Ln)}
						ed.setCursor(npos)
					}
				}
				ed.InsertAtCursor([]byte("\n"))
				tbe, _, cpos := ed.Buffer.autoIndent(ed.CursorPos.Ln)
				if tbe != nil {
					ed.SetCursorShow(lexer.Pos{Ln: tbe.Reg.End.Ln, Ch: cpos})
				}
			} else {
				ed.InsertAtCursor([]byte("\n"))
			}
			ed.ISpellKeyInput(e)
		}
		// todo: KeFunFocusPrev -- unindent
	case keymap.FocusNext: // tab
		cancelAll()
		if !e.HasAnyModifier(key.Control, key.Meta) {
			e.SetHandled()
			lasttab := ed.lastWasTabAI
			if !lasttab && ed.CursorPos.Ch == 0 && ed.Buffer.Options.AutoIndent {
				_, _, cpos := ed.Buffer.autoIndent(ed.CursorPos.Ln)
				ed.CursorPos.Ch = cpos
				ed.renderCursor(true)
				gotTabAI = true
			} else {
				ed.InsertAtCursor(indent.Bytes(ed.Buffer.Options.IndentChar(), 1, ed.Styles.Text.TabSize))
			}
			ed.NeedsRender()
			ed.ISpellKeyInput(e)
		}
	case keymap.FocusPrev: // shift-tab
		cancelAll()
		if !e.HasAnyModifier(key.Control, key.Meta) {
			e.SetHandled()
			if ed.CursorPos.Ch > 0 {
				ind, _ := lexer.LineIndent(ed.Buffer.line(ed.CursorPos.Ln), ed.Styles.Text.TabSize)
				if ind > 0 {
					ed.Buffer.indentLine(ed.CursorPos.Ln, ind-1)
					intxt := indent.Bytes(ed.Buffer.Options.IndentChar(), ind-1, ed.Styles.Text.TabSize)
					npos := lexer.Pos{Ln: ed.CursorPos.Ln, Ch: len(intxt)}
					ed.SetCursorShow(npos)
				}
			}
			ed.ISpellKeyInput(e)
		}
	case keymap.None:
		if unicode.IsPrint(e.KeyRune()) {
			if !e.HasAnyModifier(key.Control, key.Meta) {
				ed.keyInputInsertRune(e)
			}
		}
		ed.ISpellKeyInput(e)
	}
	ed.lastWasTabAI = gotTabAI
}

// keyInputInsertBracket handle input of opening bracket-like entity
// (paren, brace, bracket)
func (ed *Editor) keyInputInsertBracket(kt events.Event) {
	pos := ed.CursorPos
	match := true
	newLine := false
	curLn := ed.Buffer.line(pos.Ln)
	lnLen := len(curLn)
	lp, _ := parse.LanguageSupport.Properties(ed.Buffer.ParseState.Sup)
	if lp != nil && lp.Lang != nil {
		match, newLine = lp.Lang.AutoBracket(&ed.Buffer.ParseState, kt.KeyRune(), pos, curLn)
	} else {
		if kt.KeyRune() == '{' {
			if pos.Ch == lnLen {
				if lnLen == 0 || unicode.IsSpace(curLn[pos.Ch-1]) {
					newLine = true
				}
				match = true
			} else {
				match = unicode.IsSpace(curLn[pos.Ch])
			}
		} else {
			match = pos.Ch == lnLen || unicode.IsSpace(curLn[pos.Ch]) // at end or if space after
		}
	}
	if match {
		ket, _ := lexer.BracePair(kt.KeyRune())
		if newLine && ed.Buffer.Options.AutoIndent {
			ed.InsertAtCursor([]byte(string(kt.KeyRune()) + "\n"))
			tbe, _, cpos := ed.Buffer.autoIndent(ed.CursorPos.Ln)
			if tbe != nil {
				pos = lexer.Pos{Ln: tbe.Reg.End.Ln, Ch: cpos}
				ed.SetCursorShow(pos)
			}
			ed.InsertAtCursor([]byte("\n" + string(ket)))
			ed.Buffer.autoIndent(ed.CursorPos.Ln)
		} else {
			ed.InsertAtCursor([]byte(string(kt.KeyRune()) + string(ket)))
			pos.Ch++
		}
		ed.lastAutoInsert = ket
	} else {
		ed.InsertAtCursor([]byte(string(kt.KeyRune())))
		pos.Ch++
	}
	ed.SetCursorShow(pos)
	ed.setCursorColumn(ed.CursorPos)
}

// keyInputInsertRune handles the insertion of a typed character
func (ed *Editor) keyInputInsertRune(kt events.Event) {
	kt.SetHandled()
	if ed.ISearch.On {
		ed.CancelComplete()
		ed.iSearchKeyInput(kt)
	} else if ed.QReplace.On {
		ed.CancelComplete()
		ed.qReplaceKeyInput(kt)
	} else {
		if kt.KeyRune() == '{' || kt.KeyRune() == '(' || kt.KeyRune() == '[' {
			ed.keyInputInsertBracket(kt)
		} else if kt.KeyRune() == '}' && ed.Buffer.Options.AutoIndent && ed.CursorPos.Ch == ed.Buffer.lineLen(ed.CursorPos.Ln) {
			ed.CancelComplete()
			ed.lastAutoInsert = 0
			ed.InsertAtCursor([]byte(string(kt.KeyRune())))
			tbe, _, cpos := ed.Buffer.autoIndent(ed.CursorPos.Ln)
			if tbe != nil {
				ed.SetCursorShow(lexer.Pos{Ln: tbe.Reg.End.Ln, Ch: cpos})
			}
		} else if ed.lastAutoInsert == kt.KeyRune() { // if we type what we just inserted, just move past
			ed.CursorPos.Ch++
			ed.SetCursorShow(ed.CursorPos)
			ed.lastAutoInsert = 0
		} else {
			ed.lastAutoInsert = 0
			ed.InsertAtCursor([]byte(string(kt.KeyRune())))
			if kt.KeyRune() == ' ' {
				ed.CancelComplete()
			} else {
				ed.OfferComplete()
			}
		}
		if kt.KeyRune() == '}' || kt.KeyRune() == ')' || kt.KeyRune() == ']' {
			cp := ed.CursorPos
			np := cp
			np.Ch--
			tp, found := ed.Buffer.braceMatch(kt.KeyRune(), np)
			if found {
				ed.scopelights = append(ed.scopelights, textbuf.NewRegionPos(tp, lexer.Pos{tp.Ln, tp.Ch + 1}))
				ed.scopelights = append(ed.scopelights, textbuf.NewRegionPos(np, lexer.Pos{cp.Ln, cp.Ch}))
			}
		}
	}
}

// openLink opens given link, either by sending LinkSig signal if there are
// receivers, or by calling the TextLinkHandler if non-nil, or URLHandler if
// non-nil (which by default opens user's default browser via
// system/App.OpenURL())
func (ed *Editor) openLink(tl *paint.TextLink) {
	if ed.LinkHandler != nil {
		ed.LinkHandler(tl)
	} else {
		system.TheApp.OpenURL(tl.URL)
	}
}

// linkAt returns link at given cursor position, if one exists there --
// returns true and the link if there is a link, and false otherwise
func (ed *Editor) linkAt(pos lexer.Pos) (*paint.TextLink, bool) {
	if !(pos.Ln < len(ed.renders) && len(ed.renders[pos.Ln].Links) > 0) {
		return nil, false
	}
	cpos := ed.charStartPos(pos).ToPointCeil()
	cpos.Y += 2
	cpos.X += 2
	lpos := ed.charStartPos(lexer.Pos{Ln: pos.Ln})
	rend := &ed.renders[pos.Ln]
	for ti := range rend.Links {
		tl := &rend.Links[ti]
		tlb := tl.Bounds(rend, lpos)
		if cpos.In(tlb) {
			return tl, true
		}
	}
	return nil, false
}

// OpenLinkAt opens a link at given cursor position, if one exists there --
// returns true and the link if there is a link, and false otherwise -- highlights selected link
func (ed *Editor) OpenLinkAt(pos lexer.Pos) (*paint.TextLink, bool) {
	tl, ok := ed.linkAt(pos)
	if ok {
		rend := &ed.renders[pos.Ln]
		st, _ := rend.SpanPosToRuneIndex(tl.StartSpan, tl.StartIndex)
		end, _ := rend.SpanPosToRuneIndex(tl.EndSpan, tl.EndIndex)
		reg := textbuf.NewRegion(pos.Ln, st, pos.Ln, end)
		_ = reg
		ed.HighlightRegion(reg)
		ed.SetCursorTarget(pos)
		ed.savePosHistory(ed.CursorPos)
		ed.openLink(tl)
	}
	return tl, ok
}

// handleMouse handles mouse events
func (ed *Editor) handleMouse() {
	ed.On(events.MouseDown, func(e events.Event) { // note: usual is Click..
		if !ed.StateIs(states.Focused) {
			ed.SetFocusEvent()
		}
		pt := ed.PointToRelPos(e.Pos())
		newPos := ed.PixelToCursor(pt)
		switch e.MouseButton() {
		case events.Left:
			ed.SetState(true, states.Focused)
			ed.setCursorFromMouse(pt, newPos, e.SelectMode())
			ed.savePosHistory(ed.CursorPos)
		case events.Middle:
			if !ed.IsReadOnly() {
				ed.setCursorFromMouse(pt, newPos, e.SelectMode())
				ed.savePosHistory(ed.CursorPos)
			}
		}
	})
	ed.On(events.MouseUp, func(e events.Event) { // note: usual is Click..
		pt := ed.PointToRelPos(e.Pos())
		newPos := ed.PixelToCursor(pt)
		switch e.MouseButton() {
		case events.Left:
			ed.OpenLinkAt(newPos)
		case events.Middle:
			if !ed.IsReadOnly() {
				ed.Paste()
			}
		}
	})
	ed.OnDoubleClick(func(e events.Event) {
		if !ed.StateIs(states.Focused) {
			ed.SetFocusEvent()
			ed.Send(events.Focus, e) // sets focused flag
		}
		e.SetHandled()
		if ed.selectWord() {
			ed.CursorPos = ed.SelectRegion.Start
		}
		ed.NeedsRender()
	})
	ed.On(events.TripleClick, func(e events.Event) {
		if !ed.StateIs(states.Focused) {
			ed.SetFocusEvent()
			ed.Send(events.Focus, e) // sets focused flag
		}
		e.SetHandled()
		sz := ed.Buffer.lineLen(ed.CursorPos.Ln)
		if sz > 0 {
			ed.SelectRegion.Start.Ln = ed.CursorPos.Ln
			ed.SelectRegion.Start.Ch = 0
			ed.SelectRegion.End.Ln = ed.CursorPos.Ln
			ed.SelectRegion.End.Ch = sz
		}
		ed.NeedsRender()
	})
	ed.On(events.SlideMove, func(e events.Event) {
		e.SetHandled()
		if !ed.selectMode {
			ed.selectModeToggle()
		}
		pt := ed.PointToRelPos(e.Pos())
		newPos := ed.PixelToCursor(pt)
		ed.setCursorFromMouse(pt, newPos, events.SelectOne)
	})
}

func (ed *Editor) handleLinkCursor() {
	ed.On(events.MouseMove, func(e events.Event) {
		if !ed.hasLinks {
			return
		}
		pt := ed.PointToRelPos(e.Pos())
		mpos := ed.PixelToCursor(pt)
		if mpos.Ln >= ed.NumLines {
			return
		}
		pos := ed.renderStartPos()
		pos.Y += ed.offsets[mpos.Ln]
		pos.X += ed.LineNumberOffset
		rend := &ed.renders[mpos.Ln]
		inLink := false
		for _, tl := range rend.Links {
			tlb := tl.Bounds(rend, pos)
			if e.Pos().In(tlb) {
				inLink = true
				break
			}
		}
		if inLink {
			ed.Styles.Cursor = cursors.Pointer
		} else {
			ed.Styles.Cursor = cursors.Text
		}
	})
}

// setCursorFromMouse sets cursor position from mouse mouse action -- handles
// the selection updating etc.
func (ed *Editor) setCursorFromMouse(pt image.Point, newPos lexer.Pos, selMode events.SelectModes) {
	oldPos := ed.CursorPos
	if newPos == oldPos {
		return
	}
	//	fmt.Printf("set cursor fm mouse: %v\n", newPos)
	defer ed.NeedsRender()

	if !ed.selectMode && selMode == events.ExtendContinuous {
		if ed.SelectRegion == textbuf.RegionNil {
			ed.selectStart = ed.CursorPos
		}
		ed.setCursor(newPos)
		ed.selectRegionUpdate(ed.CursorPos)
		ed.renderCursor(true)
		return
	}

	ed.setCursor(newPos)
	if ed.selectMode || selMode != events.SelectOne {
		if !ed.selectMode && selMode != events.SelectOne {
			ed.selectMode = true
			ed.selectStart = newPos
			ed.selectRegionUpdate(ed.CursorPos)
		}
		if !ed.StateIs(states.Sliding) && selMode == events.SelectOne {
			ln := ed.CursorPos.Ln
			ch := ed.CursorPos.Ch
			if ln != ed.SelectRegion.Start.Ln || ch < ed.SelectRegion.Start.Ch || ch > ed.SelectRegion.End.Ch {
				ed.SelectReset()
			}
		} else {
			ed.selectRegionUpdate(ed.CursorPos)
		}
		if ed.StateIs(states.Sliding) {
			ed.AutoScroll(math32.Vector2FromPoint(pt).Sub(ed.Geom.Scroll))
		} else {
			ed.scrollCursorToCenterIfHidden()
		}
	} else if ed.HasSelection() {
		ln := ed.CursorPos.Ln
		ch := ed.CursorPos.Ch
		if ln != ed.SelectRegion.Start.Ln || ch < ed.SelectRegion.Start.Ch || ch > ed.SelectRegion.End.Ch {
			ed.SelectReset()
		}
	}
}

///////////////////////////////////////////////////////////
//  Context Menu

// ShowContextMenu displays the context menu with options dependent on situation
func (ed *Editor) ShowContextMenu(e events.Event) {
	if ed.Buffer.spell != nil && !ed.HasSelection() && ed.Buffer.isSpellEnabled(ed.CursorPos) {
		if ed.Buffer.spell != nil {
			if ed.OfferCorrect() {
				return
			}
		}
	}
	ed.WidgetBase.ShowContextMenu(e)
}

// contextMenu builds the text editor context menu
func (ed *Editor) contextMenu(m *core.Scene) {
	core.NewButton(m).SetText("Copy").SetIcon(icons.ContentCopy).
		SetKey(keymap.Copy).SetState(!ed.HasSelection(), states.Disabled).
		OnClick(func(e events.Event) {
			ed.Copy(true)
		})
	if !ed.IsReadOnly() {
		core.NewButton(m).SetText("Cut").SetIcon(icons.ContentCopy).
			SetKey(keymap.Cut).SetState(!ed.HasSelection(), states.Disabled).
			OnClick(func(e events.Event) {
				ed.Cut()
			})
		core.NewButton(m).SetText("Paste").SetIcon(icons.ContentPaste).
			SetKey(keymap.Paste).SetState(ed.Clipboard().IsEmpty(), states.Disabled).
			OnClick(func(e events.Event) {
				ed.Paste()
			})
		core.NewSeparator(m)
		core.NewFuncButton(m).SetFunc(ed.Buffer.Save).SetIcon(icons.Save)
		core.NewFuncButton(m).SetFunc(ed.Buffer.SaveAs).SetIcon(icons.SaveAs)
		core.NewFuncButton(m).SetFunc(ed.Buffer.Open).SetIcon(icons.Open)
		core.NewFuncButton(m).SetFunc(ed.Buffer.Revert).SetIcon(icons.Reset)
	} else {
		core.NewButton(m).SetText("Clear").SetIcon(icons.ClearAll).
			OnClick(func(e events.Event) {
				ed.Clear()
			})
	}
}
