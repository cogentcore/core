// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

import (
	"fmt"
	"image"
	"unicode"

	"goki.dev/cursors"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/keyfun"
	"goki.dev/gi/v2/texteditor/textbuf"
	"goki.dev/girl/abilities"
	"goki.dev/girl/paint"
	"goki.dev/girl/states"
	"goki.dev/glop/indent"
	"goki.dev/goosi"
	"goki.dev/goosi/events"
	"goki.dev/goosi/events/key"
	"goki.dev/pi/v2/lex"
	"goki.dev/pi/v2/pi"
)

// ViewEvents sets connections between mouse and key events and actions
func (ed *Editor) HandleEditorEvents() {
	ed.HandleWidgetEvents()
	ed.HandleLayoutEvents()
	ed.HandleEditorKeyChord()
	ed.HandleEditorMouse()
	ed.HandleEditorLinkCursor()
	ed.HandleEditorFocus()
}

func (ed *Editor) OnAdd() {
	ed.Layout.OnAdd()
	ed.HandleEditorClose()
}

func (ed *Editor) HandleEditorClose() {
	ed.OnClose(func(e events.Event) {
		ed.EditDone()
	})
}

func (ed *Editor) HandleEditorFocus() {
	ed.OnFocusLost(func(e events.Event) {
		if ed.IsReadOnly() {
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

func (ed *Editor) HandleEditorKeyChord() {
	ed.OnKeyChord(func(e events.Event) {
		ed.KeyInput(e)
	})
}

// ShiftSelect sets the selection start if the shift key is down but wasn't on the last key move.
// If the shift key has been released the select region is set to textbuf.RegionNil
func (ed *Editor) ShiftSelect(kt events.Event) {
	hasShift := kt.HasAnyModifier(key.Shift)
	if hasShift {
		if ed.SelectReg == textbuf.RegionNil {
			ed.SelectStart = ed.CursorPos
		}
	} else {
		ed.SelectReg = textbuf.RegionNil
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
	if gi.KeyEventTrace {
		fmt.Printf("View KeyInput: %v\n", ed.Path())
	}
	kf := keyfun.Of(kt.KeyChord())

	// todo:
	// cpop := win.CurPopup()
	// if gi.PopupIsCompleter(cpop) {
	// 	setprocessed := ed.Buf.Complete.KeyInput(kf)
	// 	if setprocessed {
	// 		kt.SetHandled()
	// 	}
	// }
	//
	// if gi.PopupIsCorrector(cpop) {
	// 	setprocessed := ed.Buf.Spell.KeyInput(kf)
	// 	if setprocessed {
	// 		kt.SetHandled()
	// 	}
	// }

	if kt.IsHandled() {
		return
	}
	if ed.Buf == nil || ed.Buf.NumLines() == 0 {
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
		ed.Buf.EmacsUndoSave()
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
		ed.JumpToLineAddText()
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
		ed.QReplaceAddText()
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
		if ed.Buf.IsSpellEnabled(ed.CursorPos) {
			ed.OfferCorrect()
		} else {
			ed.ForceComplete = true
			ed.OfferComplete()
			ed.ForceComplete = false
		}
	case keyfun.Enter:
		cancelAll()
		if !kt.HasAnyModifier(key.Control, key.Meta) {
			kt.SetHandled()
			if ed.Buf.Opts.AutoIndent {
				lp, _ := pi.LangSupport.Props(ed.Buf.PiState.Sup)
				if lp != nil && lp.Lang != nil && lp.HasFlag(pi.ReAutoIndent) {
					// only re-indent current line for supported types
					tbe, _, _ := ed.Buf.AutoIndent(ed.CursorPos.Ln) // reindent current line
					if tbe != nil {
						// go back to end of line!
						npos := lex.Pos{Ln: ed.CursorPos.Ln, Ch: ed.Buf.LineLen(ed.CursorPos.Ln)}
						ed.SetCursor(npos)
					}
				}
				ed.InsertAtCursor([]byte("\n"))
				tbe, _, cpos := ed.Buf.AutoIndent(ed.CursorPos.Ln)
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
			updt := ed.UpdateStart()
			lasttab := ed.Is(EditorLastWasTabAI)
			if !lasttab && ed.CursorPos.Ch == 0 && ed.Buf.Opts.AutoIndent {
				_, _, cpos := ed.Buf.AutoIndent(ed.CursorPos.Ln)
				ed.CursorPos.Ch = cpos
				ed.RenderCursor(true)
				gotTabAI = true
			} else {
				ed.InsertAtCursor(indent.Bytes(ed.Buf.Opts.IndentChar(), 1, ed.Styles.Text.TabSize))
			}
			ed.UpdateEndRender(updt)
			ed.ISpellKeyInput(kt)
		}
	case keyfun.FocusPrev: // shift-tab
		cancelAll()
		if !kt.HasAnyModifier(key.Control, key.Meta) {
			kt.SetHandled()
			if ed.CursorPos.Ch > 0 {
				ind, _ := lex.LineIndent(ed.Buf.Line(ed.CursorPos.Ln), ed.Styles.Text.TabSize)
				if ind > 0 {
					ed.Buf.IndentLine(ed.CursorPos.Ln, ind-1)
					intxt := indent.Bytes(ed.Buf.Opts.IndentChar(), ind-1, ed.Styles.Text.TabSize)
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
		if unicode.IsSpace(kt.KeyRune()) {
			ed.ForceComplete = false
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
	curLn := ed.Buf.Line(pos.Ln)
	lnLen := len(curLn)
	lp, _ := pi.LangSupport.Props(ed.Buf.PiState.Sup)
	if lp != nil && lp.Lang != nil {
		match, newLine = lp.Lang.AutoBracket(&ed.Buf.PiState, kt.KeyRune(), pos, curLn)
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
		if newLine && ed.Buf.Opts.AutoIndent {
			ed.InsertAtCursor([]byte(string(kt.KeyRune()) + "\n"))
			tbe, _, cpos := ed.Buf.AutoIndent(ed.CursorPos.Ln)
			if tbe != nil {
				pos = lex.Pos{Ln: tbe.Reg.End.Ln, Ch: cpos}
				ed.SetCursorShow(pos)
			}
			ed.InsertAtCursor([]byte("\n" + string(ket)))
			ed.Buf.AutoIndent(ed.CursorPos.Ln)
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
		} else if kt.KeyRune() == '}' && ed.Buf.Opts.AutoIndent && ed.CursorPos.Ch == ed.Buf.LineLen(ed.CursorPos.Ln) {
			ed.CancelComplete()
			ed.lastAutoInsert = 0
			ed.InsertAtCursor([]byte(string(kt.KeyRune())))
			tbe, _, cpos := ed.Buf.AutoIndent(ed.CursorPos.Ln)
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
			tp, found := ed.Buf.BraceMatch(kt.KeyRune(), np)
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
		st, _ := rend.SpanPosToRuneIdx(tl.StartSpan, tl.StartIdx)
		end, _ := rend.SpanPosToRuneIdx(tl.EndSpan, tl.EndIdx)
		reg := textbuf.NewRegion(pos.Ln, st, pos.Ln, end)
		_ = reg
		ed.HighlightRegion(reg)
		ed.SetCursorTarget(pos)
		ed.SavePosHistory(ed.CursorPos)
		ed.OpenLink(tl)
	}
	return tl, ok
}

// HandleEditorMouse handles mouse events.Event
func (ed *Editor) HandleEditorMouse() {
	ed.On(events.MouseDown, func(e events.Event) { // note: usual is Click..
		if !ed.StateIs(states.Focused) {
			ed.SetFocusEvent()
		}
		pt := ed.PointToRelPos(e.LocalPos())
		newPos := ed.PixelToCursor(pt)
		switch e.MouseButton() {
		case events.Left:
			ed.SetState(true, states.Focused)
			if _, got := ed.OpenLinkAt(newPos); got {
			} else {
				ed.SetCursorFromMouse(pt, newPos, e.SelectMode())
				ed.SavePosHistory(ed.CursorPos)
			}
		case events.Middle:
			if !ed.IsReadOnly() {
				ed.SetCursorFromMouse(pt, newPos, e.SelectMode())
				ed.SavePosHistory(ed.CursorPos)
				ed.Paste()
			}
		case events.Right:
			ed.SetCursorFromMouse(pt, newPos, e.SelectMode())
			ed.Send(events.ContextMenu, e)
		}
	})
	ed.OnDoubleClick(func(e events.Event) {
		if !ed.StateIs(states.Focused) {
			ed.SetFocusEvent()
			ed.Send(events.Focus, e) // sets focused flag
		}
		updt := ed.UpdateStart()
		e.SetHandled()
		if ed.HasSelection() {
			if ed.SelectReg.Start.Ln == ed.SelectReg.End.Ln {
				sz := ed.Buf.LineLen(ed.SelectReg.Start.Ln)
				if ed.SelectReg.Start.Ch == 0 && ed.SelectReg.End.Ch == sz {
					ed.SelectReset()
				} else { // assume word, go line
					ed.SelectReg.Start.Ch = 0
					ed.SelectReg.End.Ch = sz
				}
			} else {
				ed.SelectReset()
			}
		} else {
			if ed.SelectWord() {
				ed.CursorPos = ed.SelectReg.Start
			}
		}
		ed.UpdateEndRender(updt)
	})
	ed.On(events.SlideMove, func(e events.Event) {
		e.SetHandled()
		if !ed.SelectMode {
			ed.SelectModeToggle()
		}
		pt := ed.PointToRelPos(e.LocalPos())
		newPos := ed.PixelToCursor(pt)
		ed.SetCursorFromMouse(pt, newPos, events.SelectOne)
	})
}

func (ed *Editor) HandleEditorLinkCursor() {
	ed.On(events.MouseMove, func(e events.Event) {
		if !ed.HasLinks {

		}
		pt := ed.PointToRelPos(e.LocalPos())
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
			if e.LocalPos().In(tlb) {
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
	updt := ed.UpdateStart()
	defer ed.UpdateEndRender(updt)

	if !ed.SelectMode && selMode == events.ExtendContinuous {
		if ed.SelectReg == textbuf.RegionNil {
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
			if ln != ed.SelectReg.Start.Ln || ch < ed.SelectReg.Start.Ch || ch > ed.SelectReg.End.Ch {
				ed.SelectReset()
			}
		} else {
			ed.SelectRegUpdate(ed.CursorPos)
		}
		if ed.StateIs(states.Sliding) {
			ed.AutoScroll(pt.Add(ed.Geom.TotalBBox.Min))
		} else {
			ed.ScrollCursorToCenterIfHidden()
		}
	} else if ed.HasSelection() {
		ln := ed.CursorPos.Ln
		ch := ed.CursorPos.Ch
		if ln != ed.SelectReg.Start.Ln || ch < ed.SelectReg.Start.Ch || ch > ed.SelectReg.End.Ch {
			ed.SelectReset()
		}
	}
}
