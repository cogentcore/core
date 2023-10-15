// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

import (
	"fmt"
	"image"
	"unicode"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/texteditor/textbuf"
	"goki.dev/girl/paint"
	"goki.dev/girl/states"
	"goki.dev/glop/indent"
	"goki.dev/goosi/events"
	"goki.dev/goosi/events/key"
	"goki.dev/pi/v2/lex"
	"goki.dev/pi/v2/pi"
)

// ViewEvents sets connections between mouse and key events and actions
func (tv *Editor) HandleTextViewEvents() {
	tv.HandleLayoutEvents()
	tv.HandleTextViewKeyChord()
	tv.HandleTextViewMouse()
}

///////////////////////////////////////////////////////////////////////////////
//    KeyInput handling

func (tv *Editor) HandleTextViewKeyChord() {
	tv.OnKeyChord(func(e events.Event) {
		tv.KeyInput(e)
	})
}

// ShiftSelect sets the selection start if the shift key is down but wasn't on the last key move.
// If the shift key has been released the select region is set to textbuf.RegionNil
func (tv *Editor) ShiftSelect(kt events.Event) {
	hasShift := kt.HasAnyModifier(key.Shift)
	if hasShift {
		if tv.SelectReg == textbuf.RegionNil {
			tv.SelectStart = tv.CursorPos
		}
	} else {
		tv.SelectReg = textbuf.RegionNil
	}
}

// ShiftSelectExtend updates the select region if the shift key is down and renders the selected text.
// If the shift key is not down the previously selected text is rerendered to clear the highlight
func (tv *Editor) ShiftSelectExtend(kt events.Event) {
	hasShift := kt.HasAnyModifier(key.Shift)
	if hasShift {
		tv.SelectRegUpdate(tv.CursorPos)
	}
}

// KeyInput handles keyboard input into the text field and from the completion menu
func (tv *Editor) KeyInput(kt events.Event) {
	if gi.KeyEventTrace {
		fmt.Printf("View KeyInput: %v\n", tv.Path())
	}
	kf := gi.KeyFun(kt.KeyChord())

	// todo:
	// cpop := win.CurPopup()
	// if gi.PopupIsCompleter(cpop) {
	// 	setprocessed := tv.Buf.Complete.KeyInput(kf)
	// 	if setprocessed {
	// 		kt.SetHandled()
	// 	}
	// }
	//
	// if gi.PopupIsCorrector(cpop) {
	// 	setprocessed := tv.Buf.Spell.KeyInput(kf)
	// 	if setprocessed {
	// 		kt.SetHandled()
	// 	}
	// }

	if kt.IsHandled() {
		return
	}
	if tv.Buf == nil || tv.Buf.NumLines() == 0 {
		return
	}

	// cancelAll cancels search, completer, and..
	cancelAll := func() {
		tv.CancelComplete()
		tv.CancelCorrect()
		tv.ISearchCancel()
		tv.QReplaceCancel()
		tv.lastAutoInsert = 0
	}

	if kf != gi.KeyFunRecenter { // always start at centering
		tv.lastRecenter = 0
	}

	if kf != gi.KeyFunUndo && tv.Is(ViewLastWasUndo) {
		tv.Buf.EmacsUndoSave()
		tv.SetFlag(false, ViewLastWasUndo)
	}

	gotTabAI := false // got auto-indent tab this time

	// first all the keys that work for both inactive and active
	switch kf {
	case gi.KeyFunMoveRight:
		cancelAll()
		kt.SetHandled()
		tv.ShiftSelect(kt)
		tv.CursorForward(1)
		tv.ShiftSelectExtend(kt)
		tv.ISpellKeyInput(kt)
	case gi.KeyFunWordRight:
		cancelAll()
		kt.SetHandled()
		tv.ShiftSelect(kt)
		tv.CursorForwardWord(1)
		tv.ShiftSelectExtend(kt)
	case gi.KeyFunMoveLeft:
		cancelAll()
		kt.SetHandled()
		tv.ShiftSelect(kt)
		tv.CursorBackward(1)
		tv.ShiftSelectExtend(kt)
	case gi.KeyFunWordLeft:
		cancelAll()
		kt.SetHandled()
		tv.ShiftSelect(kt)
		tv.CursorBackwardWord(1)
		tv.ShiftSelectExtend(kt)
	case gi.KeyFunMoveUp:
		cancelAll()
		kt.SetHandled()
		tv.ShiftSelect(kt)
		tv.CursorUp(1)
		tv.ShiftSelectExtend(kt)
		tv.ISpellKeyInput(kt)
	case gi.KeyFunMoveDown:
		cancelAll()
		kt.SetHandled()
		tv.ShiftSelect(kt)
		tv.CursorDown(1)
		tv.ShiftSelectExtend(kt)
		tv.ISpellKeyInput(kt)
	case gi.KeyFunPageUp:
		cancelAll()
		kt.SetHandled()
		tv.ShiftSelect(kt)
		tv.CursorPageUp(1)
		tv.ShiftSelectExtend(kt)
	case gi.KeyFunPageDown:
		cancelAll()
		kt.SetHandled()
		tv.ShiftSelect(kt)
		tv.CursorPageDown(1)
		tv.ShiftSelectExtend(kt)
	case gi.KeyFunHome:
		cancelAll()
		kt.SetHandled()
		tv.ShiftSelect(kt)
		tv.CursorStartLine()
		tv.ShiftSelectExtend(kt)
	case gi.KeyFunEnd:
		cancelAll()
		kt.SetHandled()
		tv.ShiftSelect(kt)
		tv.CursorEndLine()
		tv.ShiftSelectExtend(kt)
	case gi.KeyFunDocHome:
		cancelAll()
		kt.SetHandled()
		tv.ShiftSelect(kt)
		tv.CursorStartDoc()
		tv.ShiftSelectExtend(kt)
	case gi.KeyFunDocEnd:
		cancelAll()
		kt.SetHandled()
		tv.ShiftSelect(kt)
		tv.CursorEndDoc()
		tv.ShiftSelectExtend(kt)
	case gi.KeyFunRecenter:
		cancelAll()
		kt.SetHandled()
		tv.ReMarkup()
		tv.CursorRecenter()
	case gi.KeyFunSelectMode:
		cancelAll()
		kt.SetHandled()
		tv.SelectModeToggle()
	case gi.KeyFunCancelSelect:
		tv.CancelComplete()
		kt.SetHandled()
		tv.EscPressed() // generic cancel
	case gi.KeyFunSelectAll:
		cancelAll()
		kt.SetHandled()
		tv.SelectAll()
	case gi.KeyFunCopy:
		cancelAll()
		kt.SetHandled()
		tv.Copy(true) // reset
	case gi.KeyFunSearch:
		kt.SetHandled()
		tv.QReplaceCancel()
		tv.CancelComplete()
		tv.ISearchStart()
	case gi.KeyFunAbort:
		cancelAll()
		kt.SetHandled()
		tv.EscPressed()
	case gi.KeyFunJump:
		cancelAll()
		kt.SetHandled()
		tv.JumpToLinePrompt()
	case gi.KeyFunHistPrev:
		cancelAll()
		kt.SetHandled()
		tv.CursorToHistPrev()
	case gi.KeyFunHistNext:
		cancelAll()
		kt.SetHandled()
		tv.CursorToHistNext()
	case gi.KeyFunLookup:
		cancelAll()
		kt.SetHandled()
		tv.Lookup()
	}
	if tv.IsDisabled() {
		switch {
		case kf == gi.KeyFunFocusNext: // tab
			kt.SetHandled()
			tv.CursorNextLink(true)
		case kf == gi.KeyFunFocusPrev: // tab
			kt.SetHandled()
			tv.CursorPrevLink(true)
		case kf == gi.KeyFunNil && tv.ISearch.On:
			if unicode.IsPrint(kt.KeyRune()) && !kt.HasAnyModifier(key.Control, key.Meta) {
				tv.ISearchKeyInput(kt)
			}
		case kt.KeyRune() == ' ' || kf == gi.KeyFunAccept || kf == gi.KeyFunEnter:
			kt.SetHandled()
			tv.CursorPos.Ch--
			tv.CursorNextLink(true) // todo: cursorcurlink
			tv.OpenLinkAt(tv.CursorPos)
		}
		return
	}
	if kt.IsHandled() {
		tv.SetFlag(gotTabAI, ViewLastWasTabAI)
		return
	}
	switch kf {
	case gi.KeyFunReplace:
		kt.SetHandled()
		tv.CancelComplete()
		tv.ISearchCancel()
		tv.QReplacePrompt()
	// case gi.KeyFunAccept: // ctrl+enter
	// 	tv.ISearchCancel()
	// 	tv.QReplaceCancel()
	// 	kt.SetHandled()
	// 	tv.FocusNext()
	case gi.KeyFunBackspace:
		// todo: previous item in qreplace
		if tv.ISearch.On {
			tv.ISearchBackspace()
		} else {
			kt.SetHandled()
			tv.CursorBackspace(1)
			tv.ISpellKeyInput(kt)
			tv.OfferComplete()
		}
	case gi.KeyFunKill:
		cancelAll()
		kt.SetHandled()
		tv.CursorKill()
	case gi.KeyFunDelete:
		cancelAll()
		kt.SetHandled()
		tv.CursorDelete(1)
		tv.ISpellKeyInput(kt)
	case gi.KeyFunBackspaceWord:
		cancelAll()
		kt.SetHandled()
		tv.CursorBackspaceWord(1)
	case gi.KeyFunDeleteWord:
		cancelAll()
		kt.SetHandled()
		tv.CursorDeleteWord(1)
	case gi.KeyFunCut:
		cancelAll()
		kt.SetHandled()
		tv.Cut()
	case gi.KeyFunPaste:
		cancelAll()
		kt.SetHandled()
		tv.Paste()
	case gi.KeyFunTranspose:
		cancelAll()
		kt.SetHandled()
		tv.CursorTranspose()
	case gi.KeyFunTransposeWord:
		cancelAll()
		kt.SetHandled()
		tv.CursorTransposeWord()
	case gi.KeyFunPasteHist:
		cancelAll()
		kt.SetHandled()
		tv.PasteHist()
	case gi.KeyFunUndo:
		cancelAll()
		kt.SetHandled()
		tv.Undo()
		tv.SetFlag(true, ViewLastWasUndo)
	case gi.KeyFunRedo:
		cancelAll()
		kt.SetHandled()
		tv.Redo()
	case gi.KeyFunComplete:
		tv.ISearchCancel()
		kt.SetHandled()
		if tv.Buf.IsSpellEnabled(tv.CursorPos) {
			tv.OfferCorrect()
		} else {
			tv.ForceComplete = true
			tv.OfferComplete()
			tv.ForceComplete = false
		}
	case gi.KeyFunEnter:
		cancelAll()
		if !kt.HasAnyModifier(key.Control, key.Meta) {
			kt.SetHandled()
			if tv.Buf.Opts.AutoIndent {
				lp, _ := pi.LangSupport.Props(tv.Buf.PiState.Sup)
				if lp != nil && lp.Lang != nil && lp.HasFlag(pi.ReAutoIndent) {
					// only re-indent current line for supported types
					tbe, _, _ := tv.Buf.AutoIndent(tv.CursorPos.Ln) // reindent current line
					if tbe != nil {
						// go back to end of line!
						npos := lex.Pos{Ln: tv.CursorPos.Ln, Ch: tv.Buf.LineLen(tv.CursorPos.Ln)}
						tv.SetCursor(npos)
					}
				}
				tv.InsertAtCursor([]byte("\n"))
				tbe, _, cpos := tv.Buf.AutoIndent(tv.CursorPos.Ln)
				if tbe != nil {
					tv.SetCursorShow(lex.Pos{Ln: tbe.Reg.End.Ln, Ch: cpos})
				}
			} else {
				tv.InsertAtCursor([]byte("\n"))
			}
			tv.ISpellKeyInput(kt)
		}
		// todo: KeFunFocusPrev -- unindent
	case gi.KeyFunFocusNext: // tab
		cancelAll()
		if !kt.HasAnyModifier(key.Control, key.Meta) {
			kt.SetHandled()
			updt := tv.UpdateStart()
			lasttab := tv.Is(ViewLastWasTabAI)
			if !lasttab && tv.CursorPos.Ch == 0 && tv.Buf.Opts.AutoIndent {
				_, _, cpos := tv.Buf.AutoIndent(tv.CursorPos.Ln)
				tv.CursorPos.Ch = cpos
				tv.RenderCursor(true)
				gotTabAI = true
			} else {
				tv.InsertAtCursor(indent.Bytes(tv.Buf.Opts.IndentChar(), 1, tv.Styles.Text.TabSize))
			}
			tv.UpdateEndRender(updt)
			tv.ISpellKeyInput(kt)
		}
	case gi.KeyFunFocusPrev: // shift-tab
		cancelAll()
		if !kt.HasAnyModifier(key.Control, key.Meta) {
			kt.SetHandled()
			if tv.CursorPos.Ch > 0 {
				ind, _ := lex.LineIndent(tv.Buf.Line(tv.CursorPos.Ln), tv.Styles.Text.TabSize)
				if ind > 0 {
					tv.Buf.IndentLine(tv.CursorPos.Ln, ind-1)
					intxt := indent.Bytes(tv.Buf.Opts.IndentChar(), ind-1, tv.Styles.Text.TabSize)
					npos := lex.Pos{Ln: tv.CursorPos.Ln, Ch: len(intxt)}
					tv.SetCursorShow(npos)
				}
			}
			tv.ISpellKeyInput(kt)
		}
	case gi.KeyFunNil:
		if unicode.IsPrint(kt.KeyRune()) {
			if !kt.HasAnyModifier(key.Control, key.Meta) {
				tv.KeyInputInsertRune(kt)
			}
		}
		if unicode.IsSpace(kt.KeyRune()) {
			tv.ForceComplete = false
		}
		tv.ISpellKeyInput(kt)
	}
	tv.SetFlag(gotTabAI, ViewLastWasTabAI)
}

// KeyInputInsertBra handle input of opening bracket-like entity (paren, brace, bracket)
func (tv *Editor) KeyInputInsertBra(kt events.Event) {
	pos := tv.CursorPos
	match := true
	newLine := false
	curLn := tv.Buf.Line(pos.Ln)
	lnLen := len(curLn)
	lp, _ := pi.LangSupport.Props(tv.Buf.PiState.Sup)
	if lp != nil && lp.Lang != nil {
		match, newLine = lp.Lang.AutoBracket(&tv.Buf.PiState, kt.KeyRune(), pos, curLn)
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
		if newLine && tv.Buf.Opts.AutoIndent {
			tv.InsertAtCursor([]byte(string(kt.KeyRune()) + "\n"))
			tbe, _, cpos := tv.Buf.AutoIndent(tv.CursorPos.Ln)
			if tbe != nil {
				pos = lex.Pos{Ln: tbe.Reg.End.Ln, Ch: cpos}
				tv.SetCursorShow(pos)
			}
			tv.InsertAtCursor([]byte("\n" + string(ket)))
			tv.Buf.AutoIndent(tv.CursorPos.Ln)
		} else {
			tv.InsertAtCursor([]byte(string(kt.KeyRune()) + string(ket)))
			pos.Ch++
		}
		tv.lastAutoInsert = ket
	} else {
		tv.InsertAtCursor([]byte(string(kt.KeyRune())))
		pos.Ch++
	}
	tv.SetCursorShow(pos)
	tv.SetCursorCol(tv.CursorPos)
}

// KeyInputInsertRune handles the insertion of a typed character
func (tv *Editor) KeyInputInsertRune(kt events.Event) {
	kt.SetHandled()
	if tv.ISearch.On {
		tv.CancelComplete()
		tv.ISearchKeyInput(kt)
	} else if tv.QReplace.On {
		tv.CancelComplete()
		tv.QReplaceKeyInput(kt)
	} else {
		if kt.KeyRune() == '{' || kt.KeyRune() == '(' || kt.KeyRune() == '[' {
			tv.KeyInputInsertBra(kt)
		} else if kt.KeyRune() == '}' && tv.Buf.Opts.AutoIndent && tv.CursorPos.Ch == tv.Buf.LineLen(tv.CursorPos.Ln) {
			tv.CancelComplete()
			tv.lastAutoInsert = 0
			tv.InsertAtCursor([]byte(string(kt.KeyRune())))
			tbe, _, cpos := tv.Buf.AutoIndent(tv.CursorPos.Ln)
			if tbe != nil {
				tv.SetCursorShow(lex.Pos{Ln: tbe.Reg.End.Ln, Ch: cpos})
			}
		} else if tv.lastAutoInsert == kt.KeyRune() { // if we type what we just inserted, just move past
			tv.CursorPos.Ch++
			tv.SetCursorShow(tv.CursorPos)
			tv.lastAutoInsert = 0
		} else {
			tv.lastAutoInsert = 0
			tv.InsertAtCursor([]byte(string(kt.KeyRune())))
			if kt.KeyRune() == ' ' {
				tv.CancelComplete()
			} else {
				tv.OfferComplete()
			}
		}
		if kt.KeyRune() == '}' || kt.KeyRune() == ')' || kt.KeyRune() == ']' {
			cp := tv.CursorPos
			np := cp
			np.Ch--
			tp, found := tv.Buf.BraceMatch(kt.KeyRune(), np)
			if found {
				tv.Scopelights = append(tv.Scopelights, textbuf.NewRegionPos(tp, lex.Pos{tp.Ln, tp.Ch + 1}))
				tv.Scopelights = append(tv.Scopelights, textbuf.NewRegionPos(np, lex.Pos{cp.Ln, cp.Ch}))
			}
		}
	}
}

// OpenLink opens given link, either by sending LinkSig signal if there are
// receivers, or by calling the TextLinkHandler if non-nil, or URLHandler if
// non-nil (which by default opens user's default browser via
// oswin/App.OpenURL())
func (tv *Editor) OpenLink(tl *paint.TextLink) {
	// tl.Widget = tv.This().(gi.Widget)
	// fmt.Printf("opening link: %v\n", tl.URL)
	// if len(tv.LinkSig.Cons) == 0 {
	// 	if paint.TextLinkHandler != nil {
	// 		if paint.TextLinkHandler(*tl) {
	// 			return
	// 		}
	// 		if paint.URLHandler != nil {
	// 			paint.URLHandler(tl.URL)
	// 		}
	// 	}
	// 	return
	// }
	// tv.LinkSig.Emit(tv.This(), 0, tl.URL) // todo: could potentially signal different target=_blank kinds of options here with the sig
}

// LinkAt returns link at given cursor position, if one exists there --
// returns true and the link if there is a link, and false otherwise
func (tv *Editor) LinkAt(pos lex.Pos) (*paint.TextLink, bool) {
	if !(pos.Ln < len(tv.Renders) && len(tv.Renders[pos.Ln].Links) > 0) {
		return nil, false
	}
	cpos := tv.CharStartPos(pos).ToPointCeil()
	cpos.Y += 2
	cpos.X += 2
	lpos := tv.CharStartPos(lex.Pos{Ln: pos.Ln})
	rend := &tv.Renders[pos.Ln]
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
func (tv *Editor) OpenLinkAt(pos lex.Pos) (*paint.TextLink, bool) {
	tl, ok := tv.LinkAt(pos)
	if ok {
		rend := &tv.Renders[pos.Ln]
		st, _ := rend.SpanPosToRuneIdx(tl.StartSpan, tl.StartIdx)
		ed, _ := rend.SpanPosToRuneIdx(tl.EndSpan, tl.EndIdx)
		reg := textbuf.NewRegion(pos.Ln, st, pos.Ln, ed)
		tv.HighlightRegion(reg)
		tv.SetCursorShow(pos)
		tv.SavePosHistory(tv.CursorPos)
		tv.OpenLink(tl)
	}
	return tl, ok
}

// HandleTextViewMouse handles mouse events.Event
func (tv *Editor) HandleTextViewMouse() {
	tv.On(events.MouseDown, func(e events.Event) { // note: usual is Click..
		if tv.StateIs(states.Disabled) {
			return
		}
		if !tv.StateIs(states.Focused) {
			tv.GrabFocus()
		}
		pt := tv.PointToRelPos(e.LocalPos())
		newPos := tv.PixelToCursor(pt)
		switch e.MouseButton() {
		case events.Left:
			tv.SetState(true, states.Focused)
			if _, got := tv.OpenLinkAt(newPos); got {
			} else {
				tv.SetCursorFromMouse(pt, newPos, e.SelectMode())
				tv.SavePosHistory(tv.CursorPos)
			}
		case events.Middle:
			if !tv.IsDisabled() {
				tv.SetCursorFromMouse(pt, newPos, e.SelectMode())
				tv.SavePosHistory(tv.CursorPos)
				tv.Paste()
			}
		case events.Right:
			tv.SetCursorFromMouse(pt, newPos, e.SelectMode())
			// tv.EmitContextMenuSignal()
			tv.This().(gi.Widget).ContextMenu()

		}
	})
	tv.OnDoubleClick(func(e events.Event) {
		if tv.StateIs(states.Disabled) {
			return
		}
		if !tv.StateIs(states.Focused) {
			tv.GrabFocus()
			tv.Send(events.Focus, e) // sets focused flag
		}
		e.SetHandled()
		if tv.HasSelection() {
			if tv.SelectReg.Start.Ln == tv.SelectReg.End.Ln {
				sz := tv.Buf.LineLen(tv.SelectReg.Start.Ln)
				if tv.SelectReg.Start.Ch == 0 && tv.SelectReg.End.Ch == sz {
					tv.SelectReset()
				} else { // assume word, go line
					tv.SelectReg.Start.Ch = 0
					tv.SelectReg.End.Ch = sz
				}
			} else {
				tv.SelectReset()
			}
		} else {
			if tv.SelectWord() {
				tv.CursorPos = tv.SelectReg.Start
			}
		}
	})
	tv.On(events.SlideMove, func(e events.Event) {
		if tv.StateIs(states.Disabled) {
			return
		}
		e.SetHandled()
		if !tv.SelectMode {
			tv.SelectModeToggle()
		}
		pt := tv.PointToRelPos(e.LocalPos())
		newPos := tv.PixelToCursor(pt)
		tv.SetCursorFromMouse(pt, newPos, events.SelectOne)
	})
}

// todo: needs this in event filtering update!
// if !tv.HasLinks {
// 	return
// }

/*
// MouseMoveEvent
func (tv *View) MouseMoveEvent() {
	we.AddFunc(events.MouseMove, gi.RegPri, func(recv, send ki.Ki, sig int64, d any) {
		me := d.(events.Event)
		me.SetHandled()
		tvv := recv.Embed(TypeView).(*View)
		pt := tv.PointToRelPos(me.LocalPos())
		mpos := tvv.PixelToCursor(pt)
		if mpos.Ln >= tvv.NLines {
			return
		}
		pos := tv.RenderStartPos()
		pos.Y += tv.Offs[mpos.Ln]
		pos.X += tv.LineNoOff
		rend := &tvv.Renders[mpos.Ln]
		inLink := false
		for _, tl := range rend.Links {
			tlb := tl.Bounds(rend, pos)
			if me.Pos().In(tlb) {
				inLink = true
				break
			}
		}
		// TODO: figure out how to handle links with new cursor setup
		// if inLink {
		// 	goosi.TheApp.Cursor(tv.ParentRenderWin().RenderWin).PushIfNot(cursors.Pointer)
		// } else {
		// 	goosi.TheApp.Cursor(tv.ParentRenderWin().RenderWin).PopIf(cursors.Pointer)
		// }

	})
}

*/

// SetCursorFromMouse sets cursor position from mouse mouse action -- handles
// the selection updating etc.
func (tv *Editor) SetCursorFromMouse(pt image.Point, newPos lex.Pos, selMode events.SelectModes) {
	oldPos := tv.CursorPos
	if newPos == oldPos {
		return
	}
	//	fmt.Printf("set cursor fm mouse: %v\n", newPos)
	updt := tv.UpdateStart()
	defer tv.UpdateEndRender(updt)

	if !tv.SelectMode && selMode == events.ExtendContinuous {
		if tv.SelectReg == textbuf.RegionNil {
			tv.SelectStart = tv.CursorPos
		}
		tv.SetCursor(newPos)
		tv.SelectRegUpdate(tv.CursorPos)
		tv.RenderCursor(true)
		return
	}

	tv.SetCursor(newPos)
	if tv.SelectMode || selMode != events.SelectOne {
		if !tv.SelectMode && selMode != events.SelectOne {
			tv.SelectMode = true
			tv.SelectStart = newPos
			tv.SelectRegUpdate(tv.CursorPos)
		}
		if !tv.StateIs(states.Sliding) && selMode == events.SelectOne {
			ln := tv.CursorPos.Ln
			ch := tv.CursorPos.Ch
			if ln != tv.SelectReg.Start.Ln || ch < tv.SelectReg.Start.Ch || ch > tv.SelectReg.End.Ch {
				tv.SelectReset()
			}
		} else {
			tv.SelectRegUpdate(tv.CursorPos)
		}
		if tv.StateIs(states.Sliding) {
			tv.AutoScroll(pt.Add(tv.ScBBox.Min))
		} else {
			tv.ScrollCursorToCenterIfHidden()
		}
	} else if tv.HasSelection() {
		ln := tv.CursorPos.Ln
		ch := tv.CursorPos.Ch
		if ln != tv.SelectReg.Start.Ln || ch < tv.SelectReg.Start.Ch || ch > tv.SelectReg.End.Ch {
			tv.SelectReset()
		}
	}
}
