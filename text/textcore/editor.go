// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"fmt"
	"unicode"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/indent"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/parse"
	"cogentcore.org/core/text/parse/lexer"
	"cogentcore.org/core/text/textpos"
)

// Editor is a widget for editing multiple lines of complicated text (as compared to
// [core.TextField] for a single line of simple text).  The Editor is driven by a
// [lines.Lines] buffer which contains all the text, and manages all the edits,
// sending update events out to the editors.
//
// Use NeedsRender to drive an render update for any change that does
// not change the line-level layout of the text.
//
// Multiple editors can be attached to a given buffer.  All updating in the
// Editor should be within a single goroutine, as it would require
// extensive protections throughout code otherwise.
type Editor struct { //core:embedder
	Base

	// ISearch is the interactive search data.
	ISearch ISearch `set:"-" edit:"-" json:"-" xml:"-"`

	// QReplace is the query replace data.
	QReplace QReplace `set:"-" edit:"-" json:"-" xml:"-"`

	// Complete is the functions and data for text completion.
	Complete *core.Complete `json:"-" xml:"-"`

	// spell is the functions and data for spelling correction.
	spell *spellCheck

	// curFilename is the current filename from Lines. Used to detect changed file.
	curFilename string
}

func (ed *Editor) Init() {
	ed.Base.Init()
	ed.editorSetLines(ed.Lines)
	ed.setSpell()
	ed.AddContextMenu(ed.contextMenu)
	ed.handleKeyChord()
	ed.handleMouse()
	ed.handleLinkCursor()
	ed.handleFocus()
}

// UpdateNewFile checks if there is a new file in the Lines editor and updates
// any relevant editor settings accordingly.
func (ed *Editor) UpdateNewFile() {
	ln := ed.Lines
	if ln == nil {
		ed.curFilename = ""
		ed.viewId = -1
		return
	}
	fnm := ln.Filename()
	if ed.curFilename == fnm {
		return
	}
	ed.curFilename = fnm
	if ln.FileInfo().Known != fileinfo.Unknown {
		_, ps := ln.ParseState()
		ed.setCompleter(ps, completeParse, completeEditParse, lookupParse)
	} else {
		ed.deleteCompleter()
	}
}

// SetLines sets the [lines.Lines] that this is an editor of,
// creating a new view for this editor and connecting to events.
func (ed *Editor) SetLines(ln *lines.Lines) *Editor {
	ed.Base.SetLines(ln)
	ed.editorSetLines(ln)
	return ed
}

// editorSetLines does the editor specific part of SetLines.
func (ed *Editor) editorSetLines(ln *lines.Lines) {
	if ln == nil {
		ed.curFilename = ""
		return
	}
	ln.OnChange(ed.viewId, func(e events.Event) {
		ed.UpdateNewFile()
	})
	ln.FileModPromptFunc = func() {
		FileModPrompt(ed.Scene, ln)
	}
}

// SaveAs saves the current text into given file; does an editDone first to save edits
// and checks for an existing file; if it does exist then prompts to overwrite or not.
func (ed *Editor) SaveAs(filename core.Filename) { //types:add
	ed.editDone()
	SaveAs(ed.Scene, ed.Lines, string(filename), nil)
}

// Save saves the current text into the current filename associated with this buffer.
func (ed *Editor) Save() error { //types:add
	ed.editDone()
	return Save(ed.Scene, ed.Lines)
}

// Close closes the lines viewed by this editor, prompting to save if there are changes.
// If afterFunc is non-nil, then it is called with the status of the user action.
func (ed *Editor) Close(afterFunc func(canceled bool)) bool {
	return Close(ed.Scene, ed.Lines, afterFunc)
}

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
// If the shift key has been released the select region is set to textpos.Region{}
func (ed *Editor) shiftSelect(kt events.Event) {
	hasShift := kt.HasAnyModifier(key.Shift)
	if hasShift {
		if ed.SelectRegion == (textpos.Region{}) {
			ed.selectStart = ed.CursorPos
		}
	} else {
		ed.SelectRegion = textpos.Region{}
	}
}

// shiftSelectExtend updates the select region if the shift key is down and renders the selected lines.
// If the shift key is not down the previously selected text is rerendered to clear the highlight
func (ed *Editor) shiftSelectExtend(kt events.Event) {
	hasShift := kt.HasAnyModifier(key.Shift)
	if hasShift {
		ed.selectRegionUpdate(ed.CursorPos)
	}
}

// keyInput handles keyboard input into the text field and from the completion menu
func (ed *Editor) keyInput(e events.Event) {
	ed.isScrolling = false
	if core.DebugSettings.KeyEventTrace {
		fmt.Printf("View KeyInput: %v\n", ed.Path())
	}
	kf := keymap.Of(e.KeyChord())

	if e.IsHandled() {
		return
	}
	if ed.Lines == nil || ed.Lines.NumLines() == 0 {
		return
	}

	// cancelAll cancels search, completer, and..
	cancelAll := func() {
		ed.CancelComplete()
		ed.cancelCorrect()
		ed.iSearchCancel()
		ed.qReplaceCancel()
		ed.lastAutoInsert = 0
	}

	if kf != keymap.Recenter { // always start at centering
		ed.lastRecenter = 0
	}

	if kf != keymap.Undo && ed.lastWasUndo {
		ed.Lines.EmacsUndoSave()
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
		ed.iSpellKeyInput(e)
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
		ed.iSpellKeyInput(e)
	case keymap.MoveDown:
		cancelAll()
		e.SetHandled()
		ed.shiftSelect(e)
		ed.cursorDown(1)
		ed.shiftSelectExtend(e)
		ed.iSpellKeyInput(e)
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
		ed.cursorLineStart()
		ed.shiftSelectExtend(e)
	case keymap.End:
		cancelAll()
		e.SetHandled()
		ed.shiftSelect(e)
		ed.cursorLineEnd()
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
		if ed.Lines != nil && ed.posHistoryIndex == ed.Lines.PosHistoryLen()-1 {
			ed.savePosHistory(ed.CursorPos) // save current if end
			ed.posHistoryIndex--
		}
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
			ed.OpenLinkAt(ed.CursorPos)
		case kf == keymap.FocusPrev: // tab
			e.SetHandled()
			ed.CursorPrevLink(true)
			ed.OpenLinkAt(ed.CursorPos)
		case kf == keymap.None && ed.ISearch.On:
			if unicode.IsPrint(e.KeyRune()) && !e.HasAnyModifier(key.Control, key.Meta) {
				ed.iSearchKeyInput(e)
			}
		case e.KeyRune() == ' ' || kf == keymap.Accept || kf == keymap.Enter:
			e.SetHandled()
			ed.CursorPos = ed.Lines.MoveBackward(ed.CursorPos, 1)
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
			ed.iSpellKeyInput(e)
			ed.offerComplete()
		}
	case keymap.Kill:
		cancelAll()
		e.SetHandled()
		ed.cursorKill()
	case keymap.Delete:
		cancelAll()
		e.SetHandled()
		ed.cursorDelete(1)
		ed.iSpellKeyInput(e)
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
		if ed.isSpellEnabled(ed.CursorPos) {
			ed.offerCorrect()
		} else {
			ed.offerComplete()
		}
	case keymap.Enter:
		cancelAll()
		if !e.HasAnyModifier(key.Control, key.Meta) {
			e.SetHandled()
			if ed.Lines.Settings.AutoIndent {
				lp, _ := ed.Lines.ParseState()
				if lp != nil && lp.Lang != nil && lp.HasFlag(parse.ReAutoIndent) {
					// only re-indent current line for supported types
					tbe, _, _ := ed.Lines.AutoIndent(ed.CursorPos.Line) // reindent current line
					if tbe != nil {
						// go back to end of line!
						npos := textpos.Pos{Line: ed.CursorPos.Line, Char: ed.Lines.LineLen(ed.CursorPos.Line)}
						ed.setCursor(npos)
					}
				}
				ed.InsertAtCursor([]byte("\n"))
				tbe, _, cpos := ed.Lines.AutoIndent(ed.CursorPos.Line)
				if tbe != nil {
					ed.SetCursorShow(textpos.Pos{Line: tbe.Region.End.Line, Char: cpos})
				}
			} else {
				ed.InsertAtCursor([]byte("\n"))
			}
			ed.iSpellKeyInput(e)
		}
		// todo: KeFunFocusPrev -- unindent
	case keymap.FocusNext: // tab
		cancelAll()
		if !e.HasAnyModifier(key.Control, key.Meta) {
			e.SetHandled()
			lasttab := ed.lastWasTabAI
			if !lasttab && ed.CursorPos.Char == 0 && ed.Lines.Settings.AutoIndent {
				_, _, cpos := ed.Lines.AutoIndent(ed.CursorPos.Line)
				ed.CursorPos.Char = cpos
				ed.renderCursor(true)
				gotTabAI = true
			} else {
				ed.InsertAtCursor(indent.Bytes(ed.Lines.Settings.IndentChar(), 1, ed.Styles.Text.TabSize))
			}
			ed.NeedsRender()
			ed.iSpellKeyInput(e)
		}
	case keymap.FocusPrev: // shift-tab
		cancelAll()
		if !e.HasAnyModifier(key.Control, key.Meta) {
			e.SetHandled()
			if ed.CursorPos.Char > 0 {
				ind, _ := lexer.LineIndent(ed.Lines.Line(ed.CursorPos.Line), ed.Styles.Text.TabSize)
				if ind > 0 {
					ed.Lines.IndentLine(ed.CursorPos.Line, ind-1)
					intxt := indent.Bytes(ed.Lines.Settings.IndentChar(), ind-1, ed.Styles.Text.TabSize)
					npos := textpos.Pos{Line: ed.CursorPos.Line, Char: len(intxt)}
					ed.SetCursorShow(npos)
				}
			}
			ed.iSpellKeyInput(e)
		}
	case keymap.None:
		if unicode.IsPrint(e.KeyRune()) {
			if !e.HasAnyModifier(key.Control, key.Meta) {
				ed.keyInputInsertRune(e)
			}
		}
		ed.iSpellKeyInput(e)
	}
	ed.lastWasTabAI = gotTabAI
}

// keyInputInsertBracket handle input of opening bracket-like entity
// (paren, brace, bracket)
func (ed *Editor) keyInputInsertBracket(kt events.Event) {
	pos := ed.CursorPos
	match := true
	newLine := false
	curLn := ed.Lines.Line(pos.Line)
	lnLen := len(curLn)
	lp, ps := ed.Lines.ParseState()
	if lp != nil && lp.Lang != nil {
		match, newLine = lp.Lang.AutoBracket(ps, kt.KeyRune(), pos, curLn)
	} else {
		if kt.KeyRune() == '{' {
			if pos.Char == lnLen {
				if lnLen == 0 || unicode.IsSpace(curLn[pos.Char-1]) {
					newLine = true
				}
				match = true
			} else {
				match = unicode.IsSpace(curLn[pos.Char])
			}
		} else {
			match = pos.Char == lnLen || unicode.IsSpace(curLn[pos.Char]) // at end or if space after
		}
	}
	if match {
		ket, _ := lexer.BracePair(kt.KeyRune())
		if newLine && ed.Lines.Settings.AutoIndent {
			ed.InsertAtCursor([]byte(string(kt.KeyRune()) + "\n"))
			tbe, _, cpos := ed.Lines.AutoIndent(ed.CursorPos.Line)
			if tbe != nil {
				pos = textpos.Pos{Line: tbe.Region.End.Line, Char: cpos}
				ed.SetCursorShow(pos)
			}
			ed.InsertAtCursor([]byte("\n" + string(ket)))
			ed.Lines.AutoIndent(ed.CursorPos.Line)
		} else {
			ed.InsertAtCursor([]byte(string(kt.KeyRune()) + string(ket)))
			pos.Char++
		}
		ed.lastAutoInsert = ket
	} else {
		ed.InsertAtCursor([]byte(string(kt.KeyRune())))
		pos.Char++
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
		} else if kt.KeyRune() == '}' && ed.Lines.Settings.AutoIndent && ed.CursorPos.Char == ed.Lines.LineLen(ed.CursorPos.Line) {
			ed.CancelComplete()
			ed.lastAutoInsert = 0
			ed.InsertAtCursor([]byte(string(kt.KeyRune())))
			tbe, _, cpos := ed.Lines.AutoIndent(ed.CursorPos.Line)
			if tbe != nil {
				ed.SetCursorShow(textpos.Pos{Line: tbe.Region.End.Line, Char: cpos})
			}
		} else if ed.lastAutoInsert == kt.KeyRune() { // if we type what we just inserted, just move past
			ed.CursorPos.Char++
			ed.SetCursorShow(ed.CursorPos)
			ed.lastAutoInsert = 0
		} else {
			ed.lastAutoInsert = 0
			ed.InsertAtCursor([]byte(string(kt.KeyRune())))
			if kt.KeyRune() == ' ' {
				ed.CancelComplete()
			} else {
				ed.offerComplete()
			}
		}
		if kt.KeyRune() == '}' || kt.KeyRune() == ')' || kt.KeyRune() == ']' {
			cp := ed.CursorPos
			np := cp
			np.Char--
			tp, found := ed.Lines.BraceMatchRune(kt.KeyRune(), np)
			if found {
				ed.addScopelights(np, tp)
			}
		}
	}
}

// handleMouse handles mouse events
func (ed *Editor) handleMouse() {
	ed.OnClick(func(e events.Event) {
		ed.SetFocus()
		pt := ed.PointToRelPos(e.Pos())
		newPos := ed.PixelToCursor(pt)
		if newPos == textpos.PosErr {
			return
		}
		switch e.MouseButton() {
		case events.Left:
			lk, _ := ed.OpenLinkAt(newPos)
			if lk == nil {
				if !e.HasAnyModifier(key.Shift, key.Meta, key.Alt) {
					ed.SelectReset()
				}
				ed.setCursorFromMouse(pt, newPos, e.SelectMode())
				ed.savePosHistory(ed.CursorPos)
			}
		case events.Middle:
			if !ed.IsReadOnly() {
				ed.Paste()
			}
		}
	})
	ed.OnDoubleClick(func(e events.Event) {
		if !ed.StateIs(states.Focused) {
			ed.SetFocus()
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
			ed.SetFocus()
			ed.Send(events.Focus, e) // sets focused flag
		}
		e.SetHandled()
		sz := ed.Lines.LineLen(ed.CursorPos.Line)
		if sz > 0 {
			ed.SelectRegion.Start.Line = ed.CursorPos.Line
			ed.SelectRegion.Start.Char = 0
			ed.SelectRegion.End.Line = ed.CursorPos.Line
			ed.SelectRegion.End.Char = sz
		}
		ed.NeedsRender()
	})
	ed.On(events.Scroll, func(e events.Event) {
		ed.isScrolling = true
	})
	ed.On(events.SlideStart, func(e events.Event) {
		e.SetHandled()
		ed.SetState(true, states.Sliding)
		ed.isScrolling = true
		pt := ed.PointToRelPos(e.Pos())
		newPos := ed.PixelToCursor(pt)
		if ed.selectMode || e.SelectMode() != events.SelectOne { // extend existing select
			ed.setCursorFromMouse(pt, newPos, e.SelectMode())
		} else {
			ed.CursorPos = newPos
			if !ed.selectMode {
				ed.selectModeToggle()
			}
		}
		ed.savePosHistory(ed.CursorPos)
	})
	ed.On(events.SlideMove, func(e events.Event) {
		e.SetHandled()
		ed.selectMode = true
		pt := ed.PointToRelPos(e.Pos())
		newPos := ed.PixelToCursor(pt)
		ed.setCursorFromMouse(pt, newPos, events.SelectOne)
	})
	ed.On(events.SlideStop, func(e events.Event) {
		e.SetHandled()
		ed.isScrolling = false
	})
}

func (ed *Editor) handleLinkCursor() {
	ed.On(events.MouseMove, func(e events.Event) {
		pt := ed.PointToRelPos(e.Pos())
		newPos := ed.PixelToCursor(pt)
		if newPos == textpos.PosErr {
			return
		}
		lk, _ := ed.linkAt(newPos)
		if lk != nil {
			ed.Styles.Cursor = cursors.Pointer
		} else {
			ed.Styles.Cursor = cursors.Text
		}
	})
}

////////  Context Menu

// ShowContextMenu displays the context menu with options dependent on situation
func (ed *Editor) ShowContextMenu(e events.Event) {
	if ed.spell != nil && !ed.HasSelection() && ed.isSpellEnabled(ed.CursorPos) {
		if ed.offerCorrect() {
			return
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
		core.NewFuncButton(m).SetFunc(ed.Save).SetIcon(icons.Save)
		core.NewFuncButton(m).SetFunc(ed.SaveAs).SetIcon(icons.SaveAs)
		core.NewFuncButton(m).SetFunc(ed.Lines.Open).SetIcon(icons.Open)
		core.NewFuncButton(m).SetFunc(ed.Lines.Revert).SetIcon(icons.Reset)
	} else {
		core.NewButton(m).SetText("Clear").SetIcon(icons.ClearAll).
			OnClick(func(e events.Event) {
				ed.Clear()
			})
		if ed.Lines != nil && ed.Lines.FileInfo().Generated {
			core.NewButton(m).SetText("Set editable").SetIcon(icons.Edit).
				OnClick(func(e events.Event) {
					ed.SetReadOnly(false)
					ed.Lines.FileInfo().Generated = false
					ed.Update()
				})
		}
	}
}

// JumpToLinePrompt jumps to given line number (minus 1) from prompt
func (ed *Editor) JumpToLinePrompt() {
	val := ""
	d := core.NewBody("Jump to line")
	core.NewText(d).SetType(core.TextSupporting).SetText("Line number to jump to")
	tf := core.NewTextField(d).SetPlaceholder("Line number")
	tf.OnChange(func(e events.Event) {
		val = tf.Text()
	})
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Jump").OnClick(func(e events.Event) {
			val = tf.Text()
			ln, err := reflectx.ToInt(val)
			if err == nil {
				ed.jumpToLine(int(ln))
			}
		})
	})
	d.RunDialog(ed)
}

// jumpToLine jumps to given line number (minus 1)
func (ed *Editor) jumpToLine(ln int) {
	ed.SetCursorShow(textpos.Pos{Line: ln - 1})
	ed.savePosHistory(ed.CursorPos)
}
