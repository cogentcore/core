// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

import (
	"fmt"
	"image"
	"os"
	"time"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/text/highlighting"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/parse/complete"
	"cogentcore.org/core/text/parse/lexer"
	"cogentcore.org/core/text/spell"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/text/token"
)

// Buffer is a buffer of text, which can be viewed by [Editor](s).
// It holds the raw text lines (in original string and rune formats,
// and marked-up from syntax highlighting), and sends signals for making
// edits to the text and coordinating those edits across multiple views.
// Editors always only view a single buffer, so they directly call methods
// on the buffer to drive updates, which are then broadcast.
// It also has methods for loading and saving buffers to files.
// Unlike GUI widgets, its methods generally send events, without an
// explicit Event suffix.
// Internally, the buffer represents new lines using \n = LF, but saving
// and loading can deal with Windows/DOS CRLF format.
type Buffer struct { //types:add
	lines.Lines

	// Filename is the filename of the file that was last loaded or saved.
	// It is used when highlighting code.
	Filename core.Filename `json:"-" xml:"-"`

	// Autosave specifies whether the file should be automatically
	// saved after changes are made.
	Autosave bool

	// Info is the full information about the current file.
	Info fileinfo.FileInfo

	// LineColors are the colors to use for rendering circles
	// next to the line numbers of certain lines.
	LineColors map[int]image.Image

	// editors are the editors that are currently viewing this buffer.
	editors []*Editor

	// posHistory is the history of cursor positions.
	// It can be used to move back through them.
	posHistory []textpos.Pos

	// Complete is the functions and data for text completion.
	Complete *core.Complete `json:"-" xml:"-"`

	// spell is the functions and data for spelling correction.
	spell *spellCheck

	// currentEditor is the current text editor, such as the one that initiated the
	// Complete or Correct process. The cursor position in this view is updated, and
	// it is reset to nil after usage.
	currentEditor *Editor

	// listeners is used for sending standard system events.
	// Change is sent for BufferDone, BufferInsert, and BufferDelete.
	listeners events.Listeners

	// Bool flags:

	// autoSaving is used in atomically safe way to protect autosaving
	autoSaving bool

	// notSaved indicates if the text has been changed (edited) relative to the
	// original, since last Save.  This can be true even when changed flag is
	// false, because changed is cleared on EditDone, e.g., when texteditor
	// is being monitored for OnChange and user does Control+Enter.
	// Use IsNotSaved() method to query state.
	notSaved bool

	// fileModOK have already asked about fact that file has changed since being
	// opened, user is ok
	fileModOK bool
}

// NewBuffer makes a new [Buffer] with default settings
// and initializes it.
func NewBuffer() *Buffer {
	tb := &Buffer{}
	tb.SetHighlighting(highlighting.StyleDefault)
	tb.Options.EditorSettings = core.SystemSettings.Editor
	tb.SetText(nil) // to initialize
	return tb
}

// bufferSignals are signals that [Buffer] can send to [Editor].
type bufferSignals int32 //enums:enum -trim-prefix buffer

const (
	// bufferDone means that editing was completed and applied to Txt field
	// -- data is Txt bytes
	bufferDone bufferSignals = iota

	// bufferNew signals that entirely new text is present.
	// All views should do full layout update.
	bufferNew

	// bufferMods signals that potentially diffuse modifications
	// have been made.  Views should do a Layout and Render.
	bufferMods

	// bufferInsert signals that some text was inserted.
	// data is lines.Edit describing change.
	// The Buf always reflects the current state *after* the edit.
	bufferInsert

	// bufferDelete signals that some text was deleted.
	// data is lines.Edit describing change.
	// The Buf always reflects the current state *after* the edit.
	bufferDelete

	// bufferMarkupUpdated signals that the Markup text has been updated
	// This signal is typically sent from a separate goroutine,
	// so should be used with a mutex
	bufferMarkupUpdated

	// bufferClosed signals that the text was closed.
	bufferClosed
)

// Init initializes the buffer.  Called automatically in SetText.
func (tb *Buffer) Init() {
	if tb.MarkupDoneFunc != nil {
		return
	}
	tb.MarkupDoneFunc = func() {
		tb.signalEditors(bufferMarkupUpdated, nil)
	}
	tb.ChangedFunc = func() {
		tb.notSaved = true
	}
}

// todo: need the init somehow.

// SetText sets the text to the given bytes.
// Pass nil to initialize an empty buffer.
// func (tb *Buffer) SetText(text []byte) *Buffer {
// 	tb.Init()
// 	tb.Lines.SetText(text)
// 	return tb
// }

// FileModCheck checks if the underlying file has been modified since last
// Stat (open, save); if haven't yet prompted, user is prompted to ensure
// that this is OK. It returns true if the file was modified.
func (tb *Buffer) FileModCheck() bool {
	if tb.fileModOK {
		return false
	}
	info, err := os.Stat(string(tb.Filename))
	if err != nil {
		return false
	}
	if info.ModTime() != time.Time(tb.Info.ModTime) {
		if !tb.IsNotSaved() { // we haven't edited: just revert
			tb.Revert()
			return true
		}
		sc := tb.sceneFromEditor()
		d := core.NewBody("File changed on disk: " + fsx.DirAndFile(string(tb.Filename)))
		core.NewText(d).SetType(core.TextSupporting).SetText(fmt.Sprintf("File has changed on disk since being opened or saved by you; what do you want to do?  If you <code>Revert from Disk</code>, you will lose any existing edits in open buffer.  If you <code>Ignore and Proceed</code>, the next save will overwrite the changed file on disk, losing any changes there.  File: %v", tb.Filename))
		d.AddBottomBar(func(bar *core.Frame) {
			core.NewButton(bar).SetText("Save as to different file").OnClick(func(e events.Event) {
				d.Close()
				core.CallFunc(sc, tb.SaveAs)
			})
			core.NewButton(bar).SetText("Revert from disk").OnClick(func(e events.Event) {
				d.Close()
				tb.Revert()
			})
			core.NewButton(bar).SetText("Ignore and proceed").OnClick(func(e events.Event) {
				d.Close()
				tb.fileModOK = true
			})
		})
		d.RunDialog(sc)
		return true
	}
	return false
}

// SaveAsFunc saves the current text into the given file.
// Does an editDone first to save edits and checks for an existing file.
// If it does exist then prompts to overwrite or not.
// If afterFunc is non-nil, then it is called with the status of the user action.
func (tb *Buffer) SaveAsFunc(filename core.Filename, afterFunc func(canceled bool)) {
	tb.editDone()
	if !errors.Log1(fsx.FileExists(string(filename))) {
		tb.saveFile(filename)
		if afterFunc != nil {
			afterFunc(false)
		}
	} else {
		sc := tb.sceneFromEditor()
		d := core.NewBody("File exists")
		core.NewText(d).SetType(core.TextSupporting).SetText(fmt.Sprintf("The file already exists; do you want to overwrite it?  File: %v", filename))
		d.AddBottomBar(func(bar *core.Frame) {
			d.AddCancel(bar).OnClick(func(e events.Event) {
				if afterFunc != nil {
					afterFunc(true)
				}
			})
			d.AddOK(bar).OnClick(func(e events.Event) {
				tb.saveFile(filename)
				if afterFunc != nil {
					afterFunc(false)
				}
			})
		})
		d.RunDialog(sc)
	}
}

// SaveAs saves the current text into given file; does an editDone first to save edits
// and checks for an existing file; if it does exist then prompts to overwrite or not.
func (tb *Buffer) SaveAs(filename core.Filename) { //types:add
	tb.SaveAsFunc(filename, nil)
}

// Save saves the current text into the current filename associated with this buffer.
func (tb *Buffer) Save() error { //types:add
	if tb.Filename == "" {
		return errors.New("core.Buf: filename is empty for Save")
	}
	tb.editDone()
	info, err := os.Stat(string(tb.Filename))
	if err == nil && info.ModTime() != time.Time(tb.Info.ModTime) {
		sc := tb.sceneFromEditor()
		d := core.NewBody("File Changed on Disk")
		core.NewText(d).SetType(core.TextSupporting).SetText(fmt.Sprintf("File has changed on disk since you opened or saved it; what do you want to do?  File: %v", tb.Filename))
		d.AddBottomBar(func(bar *core.Frame) {
			core.NewButton(bar).SetText("Save to different file").OnClick(func(e events.Event) {
				d.Close()
				core.CallFunc(sc, tb.SaveAs)
			})
			core.NewButton(bar).SetText("Open from disk, losing changes").OnClick(func(e events.Event) {
				d.Close()
				tb.Revert()
			})
			core.NewButton(bar).SetText("Save file, overwriting").OnClick(func(e events.Event) {
				d.Close()
				tb.saveFile(tb.Filename)
			})
		})
		d.RunDialog(sc)
	}
	return tb.saveFile(tb.Filename)
}

// Close closes the buffer, prompting to save if there are changes, and disconnects
// from editors. If afterFun is non-nil, then it is called with the status of the user
// action.
func (tb *Buffer) Close(afterFun func(canceled bool)) bool {
	if tb.IsNotSaved() {
		tb.StopDelayedReMarkup()
		sc := tb.sceneFromEditor()
		if tb.Filename != "" {
			d := core.NewBody("Close without saving?")
			core.NewText(d).SetType(core.TextSupporting).SetText(fmt.Sprintf("Do you want to save your changes to file: %v?", tb.Filename))
			d.AddBottomBar(func(bar *core.Frame) {
				core.NewButton(bar).SetText("Cancel").OnClick(func(e events.Event) {
					d.Close()
					if afterFun != nil {
						afterFun(true)
					}
				})
				core.NewButton(bar).SetText("Close without saving").OnClick(func(e events.Event) {
					d.Close()
					tb.clearNotSaved()
					tb.AutoSaveDelete()
					tb.Close(afterFun)
				})
				core.NewButton(bar).SetText("Save").OnClick(func(e events.Event) {
					tb.Save()
					tb.Close(afterFun) // 2nd time through won't prompt
				})
			})
			d.RunDialog(sc)
		} else {
			d := core.NewBody("Close without saving?")
			core.NewText(d).SetType(core.TextSupporting).SetText("Do you want to save your changes (no filename for this buffer yet)?  If so, Cancel and then do Save As")
			d.AddBottomBar(func(bar *core.Frame) {
				d.AddCancel(bar).OnClick(func(e events.Event) {
					if afterFun != nil {
						afterFun(true)
					}
				})
				d.AddOK(bar).SetText("Close without saving").OnClick(func(e events.Event) {
					tb.clearNotSaved()
					tb.AutoSaveDelete()
					tb.Close(afterFun)
				})
			})
			d.RunDialog(sc)
		}
		return false // awaiting decisions..
	}
	tb.signalEditors(bufferClosed, nil)
	tb.SetText(nil)
	tb.Filename = ""
	tb.clearNotSaved()
	if afterFun != nil {
		afterFun(false)
	}
	return true
}

// sceneFromEditor returns Scene from text editor, if avail
func (tb *Buffer) sceneFromEditor() *core.Scene {
	if len(tb.editors) > 0 {
		return tb.editors[0].Scene
	}
	return nil
}

// AutoScrollEditors ensures that our editors are always viewing the end of the buffer
func (tb *Buffer) AutoScrollEditors() {
	for _, ed := range tb.editors {
		if ed != nil && ed.This != nil {
			ed.renderLayout()
			ed.SetCursorTarget(tb.EndPos())
		}
	}
}

const (
	// EditSignal is used as an arg for edit methods with a signal arg, indicating
	// that a signal should be emitted.
	EditSignal = true

	// EditNoSignal is used as an arg for edit methods with a signal arg, indicating
	// that a signal should NOT be emitted.
	EditNoSignal = false

	// ReplaceMatchCase is used for MatchCase arg in ReplaceText method
	ReplaceMatchCase = true

	// ReplaceNoMatchCase is used for MatchCase arg in ReplaceText method
	ReplaceNoMatchCase = false
)

// DiffBuffersUnified computes the diff between this buffer and the other buffer,
// returning a unified diff with given amount of context (default of 3 will be
// used if -1)
func (tb *Buffer) DiffBuffersUnified(ob *Buffer, context int) []byte {
	astr := tb.Strings(true) // needs newlines for some reason
	bstr := ob.Strings(true)

	return lines.DiffLinesUnified(astr, bstr, context, string(tb.Filename), tb.Info.ModTime.String(),
		string(ob.Filename), ob.Info.ModTime.String())
}

///////////////////////////////////////////////////////////////////////////////
//    Complete and Spell

// setCompleter sets completion functions so that completions will
// automatically be offered as the user types
func (tb *Buffer) setCompleter(data any, matchFun complete.MatchFunc, editFun complete.EditFunc,
	lookupFun complete.LookupFunc) {
	if tb.Complete != nil {
		if tb.Complete.Context == data {
			tb.Complete.MatchFunc = matchFun
			tb.Complete.EditFunc = editFun
			tb.Complete.LookupFunc = lookupFun
			return
		}
		tb.deleteCompleter()
	}
	tb.Complete = core.NewComplete().SetContext(data).SetMatchFunc(matchFun).
		SetEditFunc(editFun).SetLookupFunc(lookupFun)
	tb.Complete.OnSelect(func(e events.Event) {
		tb.completeText(tb.Complete.Completion)
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

func (tb *Buffer) deleteCompleter() {
	if tb.Complete == nil {
		return
	}
	tb.Complete = nil
}

// completeText edits the text using the string chosen from the completion menu
func (tb *Buffer) completeText(s string) {
	if s == "" {
		return
	}
	// give the completer a chance to edit the completion before insert,
	// also it return a number of runes past the cursor to delete
	st := textpos.Pos{tb.Complete.SrcLn, 0}
	en := textpos.Pos{tb.Complete.SrcLn, tb.LineLen(tb.Complete.SrcLn)}
	var tbes string
	tbe := tb.Region(st, en)
	if tbe != nil {
		tbes = string(tbe.ToBytes())
	}
	c := tb.Complete.GetCompletion(s)
	pos := textpos.Pos{tb.Complete.SrcLn, tb.Complete.SrcCh}
	ed := tb.Complete.EditFunc(tb.Complete.Context, tbes, tb.Complete.SrcCh, c, tb.Complete.Seed)
	if ed.ForwardDelete > 0 {
		delEn := textpos.Pos{tb.Complete.SrcLn, tb.Complete.SrcCh + ed.ForwardDelete}
		tb.DeleteText(pos, delEn, EditNoSignal)
	}
	// now the normal completion insertion
	st = pos
	st.Char -= len(tb.Complete.Seed)
	tb.ReplaceText(st, pos, st, ed.NewText, EditSignal, ReplaceNoMatchCase)
	if tb.currentEditor != nil {
		ep := st
		ep.Char += len(ed.NewText) + ed.CursorAdjust
		tb.currentEditor.SetCursorShow(ep)
		tb.currentEditor = nil
	}
}

// isSpellEnabled returns true if spelling correction is enabled,
// taking into account given position in text if it is relevant for cases
// where it is only conditionally enabled
func (tb *Buffer) isSpellEnabled(pos textpos.Pos) bool {
	if tb.spell == nil || !tb.Options.SpellCorrect {
		return false
	}
	switch tb.Info.Cat {
	case fileinfo.Doc: // not in code!
		return !tb.InTokenCode(pos)
	case fileinfo.Code:
		return tb.InComment(pos) || tb.InLitString(pos)
	default:
		return false
	}
}

// setSpell sets spell correct functions so that spell correct will
// automatically be offered as the user types
func (tb *Buffer) setSpell() {
	if tb.spell != nil {
		return
	}
	initSpell()
	tb.spell = newSpell()
	tb.spell.onSelect(func(e events.Event) {
		tb.correctText(tb.spell.correction)
	})
}

// correctText edits the text using the string chosen from the correction menu
func (tb *Buffer) correctText(s string) {
	st := textpos.Pos{tb.spell.srcLn, tb.spell.srcCh} // start of word
	tb.RemoveTag(st, token.TextSpellErr)
	oend := st
	oend.Char += len(tb.spell.word)
	tb.ReplaceText(st, oend, st, s, EditSignal, ReplaceNoMatchCase)
	if tb.currentEditor != nil {
		ep := st
		ep.Char += len(s)
		tb.currentEditor.SetCursorShow(ep)
		tb.currentEditor = nil
	}
}

// SpellCheckLineErrors runs spell check on given line, and returns Lex tags
// with token.TextSpellErr for any misspelled words
func (tb *Buffer) SpellCheckLineErrors(ln int) lexer.Line {
	if !tb.IsValidLine(ln) {
		return nil
	}
	return spell.CheckLexLine(tb.Line(ln), tb.HiTags(ln))
}

// spellCheckLineTag runs spell check on given line, and sets Tags for any
// misspelled words and updates markup for that line.
func (tb *Buffer) spellCheckLineTag(ln int) {
	if !tb.IsValidLine(ln) {
		return
	}
	ser := tb.SpellCheckLineErrors(ln)
	ntgs := tb.AdjustedTags(ln)
	ntgs.DeleteToken(token.TextSpellErr)
	for _, t := range ser {
		ntgs.AddSort(t)
	}
	tb.SetTags(ln, ntgs)
	tb.MarkupLines(ln, ln)
	tb.StartDelayedReMarkup()
}
