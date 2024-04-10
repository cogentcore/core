// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

import (
	"fmt"
	"image"
	"unicode"

	"cogentcore.org/core/abilities"
	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/goosi"
	"cogentcore.org/core/keyfun"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/pi"
	"cogentcore.org/core/pi/lex"
	"cogentcore.org/core/states"
	"cogentcore.org/core/texteditor/textbuf"
	"cogentcore.org/core/xgo/indent"
)

func (ed *Editor) HandleEvents() {
	ed.Layout.HandleEvents()
	ed.HandleKeyChord()
	ed.HandleMouse()
	ed.HandleLinkCursor()
	ed.HandleFocus()
	ed.AddContextMenu(ed.ContextMenu)
}

func (ed *Editor) OnAdd() {
	ed.Layout.OnAdd()
	ed.HandleClose()
}

func (ed *Editor) HandleClose() {
	ed.OnClose(func(e events.Event) {
		ed.EditDone()
	})
}

func (ed *Editor) HandleFocus() {
	ed.OnFocusLost(func(e events.Event) {
		if ed.IsReadOnly() {
			ed.ClearCursor()
			return
		}
		if ed.AbilityIs(abilities.Focusable) {
			ed.EditDone()
			ed.SetState(false, states.Focused)
		}
	})
}

///////////////////////////////////////////////////////////////////////////////
//    KeyInput handling

func (ed *Editor) HandleKeyChord() {
	ed.OnKeyChord(func(e events.Event) {
		ed.KeyInput(e)
	})
}

// ShiftSelect sets the selection start if the shift key is down but wasn't on the last key move.
// If the shift key has been released the select region is set to textbuf.RegionNil
func (ed *Editor) ShiftSelect(kt events.Event) {
	hasShift := kt.HasAnyModifier(key.Shift)
	if hasShift {
		if ed.SelectRegion == textbuf.RegionNil {
			ed.SelectStart = ed.CursorPos
		}
	} else {
		ed.SelectRegion = textbuf.RegionNil
	}
}

// ShiftSelectExtend updates the select region if the shift key is down and renders the selected text.
// If the shift key is not down the previously selected text is rerendered to clear the highlight
func (ed *Editor) ShiftSelectExtend(kt events.Event) {
	hasShift := kt.HasAnyModifier(key.Shift)
	if hasShift {
		ed.SelectRegUpdate(ed.CursorPos)
	}
}

// KeyInput handles keyboard input into the text field and from the completion menu
func (ed *Editor) KeyInput(kt events.Event) {
	if core.DebugSettings.KeyEventTrace {
		fmt.Printf("View KeyInput: %v\n", ed.Path())
	}
	kf := keyfun.Of(kt.KeyChord())

	if kt.IsHandled() {
		return
	}
	if ed.Buffer == nil || ed.Buffer.NumLines() == 0 {
		return
	}

	// cancelAll cancels search, completer, and..
	cancelAll := func() {
		ed.CancelComplete()
		ed.CancelCorrect()
		ed.ISearchCancel()
		ed.QReplaceCancel()
		ed.lastAutoInsert = 0
	}

	if kf != keyfun.Recenter { // always start at centering
		ed.lastRecenter = 0
	}

	if kf != keyfun.Undo && ed.Is(EditorLastWasUndo) {
		ed.Buffer.EmacsUndoSave()
		ed.SetFlag(false, EditorLastWasUndo)
	}

	gotTabAI := false // got auto-indent tab this time

	// first all the keys that work for both inactive and active
	switch kf {
	case keyfun.MoveRight:
		cancelAll()
		kt.SetHandled()
		ed.ShiftSelect(kt)
		ed.CursorForward(1)
		ed.ShiftSelectExtend(kt)
		ed.ISpellKeyInput(kt)
	case keyfun.WordRight:
		cancelAll()
		kt.SetHandled()
		ed.ShiftSelect(kt)
		ed.CursorForwardWord(1)
		ed.ShiftSelectExtend(kt)
	case keyfun.MoveLeft:
		cancelAll()
		kt.SetHandled()
		ed.ShiftSelect(kt)
		ed.CursorBackward(1)
		ed.ShiftSelectExtend(kt)
	case keyfun.WordLeft:
		cancelAll()
		kt.SetHandled()
		ed.ShiftSelect(kt)
		ed.CursorBackwardWord(1)
		ed.ShiftSelectExtend(kt)
	case keyfun.MoveUp:
		cancelAll()
		kt.SetHandled()
		ed.ShiftSelect(kt)
		ed.CursorUp(1)
		ed.ShiftSelectExtend(kt)
		ed.ISpellKeyInput(kt)
	case keyfun.MoveDown:
		cancelAll()
		kt.SetHandled()
		ed.ShiftSelect(kt)
		ed.CursorDown(1)
		ed.ShiftSelectExtend(kt)
		ed.ISpellKeyInput(kt)
	case keyfun.PageUp:
		cancelAll()
		kt.SetHandled()
		ed.ShiftSelect(kt)
		ed.CursorPageUp(1)
		ed.ShiftSelectExtend(kt)
	case keyfun.PageDown:
		cancelAll()
		kt.SetHandled()
		ed.ShiftSelect(kt)
		ed.CursorPageDown(1)
		ed.ShiftSelectExtend(kt)
	case keyfun.Home:
		cancelAll()
		kt.SetHandled()
		ed.ShiftSelect(kt)
		ed.CursorStartLine()
		ed.ShiftSelectExtend(kt)
	case keyfun.End:
		cancelAll()
		kt.SetHandled()
		ed.ShiftSelect(kt)
		ed.CursorEndLine()
		ed.ShiftSelectExtend(kt)
	case keyfun.DocHome:
		cancelAll()
		kt.SetHandled()
		ed.ShiftSelect(kt)
		ed.CursorStartDoc()
		ed.ShiftSelectExtend(kt)
	case keyfun.DocEnd:
		cancelAll()
		kt.SetHandled()
		ed.ShiftSelect(kt)
		ed.CursorEndDoc()
		ed.ShiftSelectExtend(kt)
	case keyfun.Recenter:
		cancelAll()
		kt.SetHandled()
		ed.ReMarkup()
		ed.CursorRecenter()
	case keyfun.SelectMode:
		cancelAll()
		kt.SetHandled()
		ed.SelectModeToggle()
	case keyfun.CancelSelect:
		ed.CancelComplete()
		kt.SetHandled()
		ed.EscPressed() // generic cancel
	case keyfun.SelectAll:
		cancelAll()
		kt.SetHandled()
		ed.SelectAll()
	case keyfun.Copy:
		cancelAll()
		kt.SetHandled()
		ed.Copy(true) // reset
	case keyfun.Search:
		kt.SetHandled()
		ed.QReplaceCancel()
		ed.CancelComplete()
		ed.ISearchStart()
	case keyfun.Abort:
		cancelAll()
		kt.SetHandled()
		ed.EscPressed()
	case keyfun.Jump:
		cancelAll()
		kt.SetHandled()
		ed.JumpToLinePrompt()
	case keyfun.HistPrev:
		cancelAll()
		kt.SetHandled()
		ed.CursorToHistPrev()
	case keyfun.HistNext:
		cancelAll()
		kt.SetHandled()
		ed.CursorToHistNext()
	case keyfun.Lookup:
		cancelAll()
		kt.SetHandled()
		ed.Lookup()
	}
	if ed.IsReadOnly() {
		switch {
		case kf == keyfun.FocusNext: // tab
			kt.SetHandled()
			ed.CursorNextLink(true)
		case kf == keyfun.FocusPrev: // tab
			kt.SetHandled()
			ed.CursorPrevLink(true)
		case kf == keyfun.Nil && ed.ISearch.On:
			if unicode.IsPrint(kt.KeyRune()) && !kt.HasAnyModifier(key.Control, key.Meta) {
				ed.ISearchKeyInput(kt)
			}
		case kt.KeyRune() == ' ' || kf == keyfun.Accept || kf == keyfun.Enter:
			kt.SetHandled()
			ed.CursorPos.Ch--
			ed.CursorNextLink(true) // todo: cursorcurlink
			ed.OpenLinkAt(ed.CursorPos)
		}
		return
	}
	if kt.IsHandled() {
		ed.SetFlag(gotTabAI, EditorLastWasTabAI)
		return
	}
	switch kf {
	case keyfun.Replace:
		kt.SetHandled()
		ed.CancelComplete()
		ed.ISearchCancel()
		ed.QReplacePrompt()
	case keyfun.Backspace:
		// todo: previous item in qreplace
		if ed.ISearch.On {
			ed.ISearchBackspace()
		} else {
			kt.SetHandled()
			ed.CursorBackspace(1)
			ed.ISpellKeyInput(kt)
			ed.OfferComplete()
		}
	case keyfun.Kill:
		cancelAll()
		kt.SetHandled()
		ed.CursorKill()
	case keyfun.Delete:
		cancelAll()
		kt.SetHandled()
		ed.CursorDelete(1)
		ed.ISpellKeyInput(kt)
	case keyfun.BackspaceWord:
		cancelAll()
		kt.SetHandled()
		ed.CursorBackspaceWord(1)
	case keyfun.DeleteWord:
		cancelAll()
		kt.SetHandled()
		ed.CursorDeleteWord(1)
	case keyfun.Cut:
		cancelAll()
		kt.SetHandled()
		ed.Cut()
	case keyfun.Paste:
		cancelAll()
		kt.SetHandled()
		ed.Paste()
	case keyfun.Transpose:
		cancelAll()
		kt.SetHandled()
		ed.CursorTranspose()
	case keyfun.TransposeWord:
		cancelAll()
		kt.SetHandled()
		ed.CursorTransposeWord()
	case keyfun.PasteHist:
		cancelAll()
		kt.SetHandled()
		ed.PasteHist()
	case keyfun.Accept:
		cancelAll()
		kt.SetHandled()
		ed.EditDone()
	case keyfun.Undo:
		cancelAll()
		kt.SetHandled()
		ed.Undo()
		ed.SetFlag(true, EditorLastWasUndo)
	case keyfun.Redo:
		cancelAll()
		kt.SetHandled()
		ed.Redo()
	case keyfun.Complete:
		ed.ISearchCancel()
		kt.SetHandled()
		if ed.Buffer.IsSpellEnabled(ed.CursorPos) {
			ed.OfferCorrect()
		} else {
			ed.OfferComplete()
		}
	case keyfun.Enter:
		cancelAll()
		if !kt.HasAnyModifier(key.Control, key.Meta) {
			kt.SetHandled()
			if ed.Buffer.Opts.AutoIndent {
				lp, _ := pi.LangSupport.Properties(ed.Buffer.PiState.Sup)
				if lp != nil && lp.Lang != nil && lp.HasFlag(pi.ReAutoIndent) {
					// only re-indent current line for supported types
					tbe, _, _ := ed.Buffer.AutoIndent(ed.CursorPos.Ln) // reindent current line
					if tbe != nil {
						// go back to end of line!
						npos := lex.Pos{Ln: ed.CursorPos.Ln, Ch: ed.Buffer.LineLen(ed.CursorPos.Ln)}
						ed.SetCursor(npos)
					}
				}
				ed.InsertAtCursor([]byte("\n"))
				tbe, _, cpos := ed.Buffer.AutoIndent(ed.CursorPos.Ln)
				if tbe != nil {
					ed.SetCursorShow(lex.Pos{Ln: tbe.Reg.End.Ln, Ch: cpos})
				}
			} else {
				ed.InsertAtCursor([]byte("\n"))
			}
			ed.ISpellKeyInput(kt)
		}
		// todo: KeFunFocusPrev -- unindent
	case keyfun.FocusNext: // tab
		cancelAll()
		if !kt.HasAnyModifier(key.Control, key.Meta) {
			kt.SetHandled()
			lasttab := ed.Is(EditorLastWasTabAI)
			if !lasttab && ed.CursorPos.Ch == 0 && ed.Buffer.Opts.AutoIndent {
				_, _, cpos := ed.Buffer.AutoIndent(ed.CursorPos.Ln)
				ed.CursorPos.Ch = cpos
				ed.RenderCursor(true)
				gotTabAI = true
			} else {
				ed.InsertAtCursor(indent.Bytes(ed.Buffer.Opts.IndentChar(), 1, ed.Styles.Text.TabSize))
			}
			ed.NeedsRender()
			ed.ISpellKeyInput(kt)
		}
	case keyfun.FocusPrev: // shift-tab
		cancelAll()
		if !kt.HasAnyModifier(key.Control, key.Meta) {
			kt.SetHandled()
			if ed.CursorPos.Ch > 0 {
				ind, _ := lex.LineIndent(ed.Buffer.Line(ed.CursorPos.Ln), ed.Styles.Text.TabSize)
				if ind > 0 {
					ed.Buffer.IndentLine(ed.CursorPos.Ln, ind-1)
					intxt := indent.Bytes(ed.Buffer.Opts.IndentChar(), ind-1, ed.Styles.Text.TabSize)
					npos := lex.Pos{Ln: ed.CursorPos.Ln, Ch: len(intxt)}
					ed.SetCursorShow(npos)
				}
			}
			ed.ISpellKeyInput(kt)
		}
	case keyfun.Nil:
		if unicode.IsPrint(kt.KeyRune()) {
			if !kt.HasAnyModifier(key.Control, key.Meta) {
				ed.KeyInputInsertRune(kt)
			}
		}
		ed.ISpellKeyInput(kt)
	}
	ed.SetFlag(gotTabAI, EditorLastWasTabAI)
}

// KeyInputInsertBra handle input of opening bracket-like entity (paren, brace, bracket)
func (ed *Editor) KeyInputInsertBra(kt events.Event) {
	pos := ed.CursorPos
	match := true
	newLine := false
	curLn := ed.Buffer.Line(pos.Ln)
	lnLen := len(curLn)
	lp, _ := pi.LangSupport.Properties(ed.Buffer.PiState.Sup)
	if lp != nil && lp.Lang != nil {
		match, newLine = lp.Lang.AutoBracket(&ed.Buffer.PiState, kt.KeyRune(), pos, curLn)
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
		ket, _ := lex.BracePair(kt.KeyRune())
		if newLine && ed.Buffer.Opts.AutoIndent {
			ed.InsertAtCursor([]byte(string(kt.KeyRune()) + "\n"))
			tbe, _, cpos := ed.Buffer.AutoIndent(ed.CursorPos.Ln)
			if tbe != nil {
				pos = lex.Pos{Ln: tbe.Reg.End.Ln, Ch: cpos}
				ed.SetCursorShow(pos)
			}
			ed.InsertAtCursor([]byte("\n" + string(ket)))
			ed.Buffer.AutoIndent(ed.CursorPos.Ln)
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
	ed.SetCursorCol(ed.CursorPos)
}

// KeyInputInsertRune handles the insertion of a typed character
func (ed *Editor) KeyInputInsertRune(kt events.Event) {
	kt.SetHandled()
	if ed.ISearch.On {
		ed.CancelComplete()
		ed.ISearchKeyInput(kt)
	} else if ed.QReplace.On {
		ed.CancelComplete()
		ed.QReplaceKeyInput(kt)
	} else {
		if kt.KeyRune() == '{' || kt.KeyRune() == '(' || kt.KeyRune() == '[' {
			ed.KeyInputInsertBra(kt)
		} else if kt.KeyRune() == '}' && ed.Buffer.Opts.AutoIndent && ed.CursorPos.Ch == ed.Buffer.LineLen(ed.CursorPos.Ln) {
			ed.CancelComplete()
			ed.lastAutoInsert = 0
			ed.InsertAtCursor([]byte(string(kt.KeyRune())))
			tbe, _, cpos := ed.Buffer.AutoIndent(ed.CursorPos.Ln)
			if tbe != nil {
				ed.SetCursorShow(lex.Pos{Ln: tbe.Reg.End.Ln, Ch: cpos})
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
			tp, found := ed.Buffer.BraceMatch(kt.KeyRune(), np)
			if found {
				ed.Scopelights = append(ed.Scopelights, textbuf.NewRegionPos(tp, lex.Pos{tp.Ln, tp.Ch + 1}))
				ed.Scopelights = append(ed.Scopelights, textbuf.NewRegionPos(np, lex.Pos{cp.Ln, cp.Ch}))
			}
		}
	}
}

// OpenLink opens given link, either by sending LinkSig signal if there are
// receivers, or by calling the TextLinkHandler if non-nil, or URLHandler if
// non-nil (which by default opens user's default browser via
// goosi/App.OpenURL())
func (ed *Editor) OpenLink(tl *paint.TextLink) {
	if ed.LinkHandler != nil {
		ed.LinkHandler(tl)
	} else {
		goosi.TheApp.OpenURL(tl.URL)
	}
}

// LinkAt returns link at given cursor position, if one exists there --
// returns true and the link if there is a link, and false otherwise
func (ed *Editor) LinkAt(pos lex.Pos) (*paint.TextLink, bool) {
	if !(pos.Ln < len(ed.Renders) && len(ed.Renders[pos.Ln].Links) > 0) {
		return nil, false
	}
	cpos := ed.CharStartPos(pos).ToPointCeil()
	cpos.Y += 2
	cpos.X += 2
	lpos := ed.CharStartPos(lex.Pos{Ln: pos.Ln})
	rend := &ed.Renders[pos.Ln]
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
func (ed *Editor) OpenLinkAt(pos lex.Pos) (*paint.TextLink, bool) {
	tl, ok := ed.LinkAt(pos)
	if ok {
		rend := &ed.Renders[pos.Ln]
		st, _ := rend.SpanPosToRuneIndex(tl.StartSpan, tl.StartIndex)
		end, _ := rend.SpanPosToRuneIndex(tl.EndSpan, tl.EndIndex)
		reg := textbuf.NewRegion(pos.Ln, st, pos.Ln, end)
		_ = reg
		ed.HighlightRegion(reg)
		ed.SetCursorTarget(pos)
		ed.SavePosHistory(ed.CursorPos)
		ed.OpenLink(tl)
	}
	return tl, ok
}

// HandleMouse handles mouse events.Event
func (ed *Editor) HandleMouse() {
	ed.On(events.MouseDown, func(e events.Event) { // note: usual is Click..
		if !ed.StateIs(states.Focused) {
			ed.SetFocusEvent()
		}
		pt := ed.PointToRelPos(e.Pos())
		newPos := ed.PixelToCursor(pt)
		switch e.MouseButton() {
		case events.Left:
			ed.SetState(true, states.Focused)
			ed.SetCursorFromMouse(pt, newPos, e.SelectMode())
			ed.SavePosHistory(ed.CursorPos)
		case events.Middle:
			if !ed.IsReadOnly() {
				ed.SetCursorFromMouse(pt, newPos, e.SelectMode())
				ed.SavePosHistory(ed.CursorPos)
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
		if ed.SelectWord() {
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
		sz := ed.Buffer.LineLen(ed.CursorPos.Ln)
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
		if !ed.SelectMode {
			ed.SelectModeToggle()
		}
		pt := ed.PointToRelPos(e.Pos())
		newPos := ed.PixelToCursor(pt)
		ed.SetCursorFromMouse(pt, newPos, events.SelectOne)
	})
}

func (ed *Editor) HandleLinkCursor() {
	ed.On(events.MouseMove, func(e events.Event) {
		if !ed.HasLinks {

		}
		pt := ed.PointToRelPos(e.Pos())
		mpos := ed.PixelToCursor(pt)
		if mpos.Ln >= ed.NLines {
			return
		}
		pos := ed.RenderStartPos()
		pos.Y += ed.Offs[mpos.Ln]
		pos.X += ed.LineNoOff
		rend := &ed.Renders[mpos.Ln]
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

// SetCursorFromMouse sets cursor position from mouse mouse action -- handles
// the selection updating etc.
func (ed *Editor) SetCursorFromMouse(pt image.Point, newPos lex.Pos, selMode events.SelectModes) {
	oldPos := ed.CursorPos
	if newPos == oldPos {
		return
	}
	//	fmt.Printf("set cursor fm mouse: %v\n", newPos)
	defer ed.NeedsRender()

	if !ed.SelectMode && selMode == events.ExtendContinuous {
		if ed.SelectRegion == textbuf.RegionNil {
			ed.SelectStart = ed.CursorPos
		}
		ed.SetCursor(newPos)
		ed.SelectRegUpdate(ed.CursorPos)
		ed.RenderCursor(true)
		return
	}

	ed.SetCursor(newPos)
	if ed.SelectMode || selMode != events.SelectOne {
		if !ed.SelectMode && selMode != events.SelectOne {
			ed.SelectMode = true
			ed.SelectStart = newPos
			ed.SelectRegUpdate(ed.CursorPos)
		}
		if !ed.StateIs(states.Sliding) && selMode == events.SelectOne {
			ln := ed.CursorPos.Ln
			ch := ed.CursorPos.Ch
			if ln != ed.SelectRegion.Start.Ln || ch < ed.SelectRegion.Start.Ch || ch > ed.SelectRegion.End.Ch {
				ed.SelectReset()
			}
		} else {
			ed.SelectRegUpdate(ed.CursorPos)
		}
		if ed.StateIs(states.Sliding) {
			ed.AutoScroll(mat32.V2FromPoint(pt).Sub(ed.Geom.Scroll))
		} else {
			ed.ScrollCursorToCenterIfHidden()
		}
	} else if ed.HasSelection() {
		ln := ed.CursorPos.Ln
		ch := ed.CursorPos.Ch
		if ln != ed.SelectRegion.Start.Ln || ch < ed.SelectRegion.Start.Ch || ch > ed.SelectRegion.End.Ch {
			ed.SelectReset()
		}
	}
}
