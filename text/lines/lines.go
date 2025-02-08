// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import (
	"bytes"
	"fmt"
	"log"
	"slices"
	"sync"
	"time"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/indent"
	"cogentcore.org/core/base/runes"
	"cogentcore.org/core/base/slicesx"
	"cogentcore.org/core/parse"
	"cogentcore.org/core/parse/lexer"
	"cogentcore.org/core/parse/token"
	"cogentcore.org/core/text/highlighting"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
	"cogentcore.org/core/text/textpos"
)

const (
	// ReplaceMatchCase is used for MatchCase arg in ReplaceText method
	ReplaceMatchCase = true

	// ReplaceNoMatchCase is used for MatchCase arg in ReplaceText method
	ReplaceNoMatchCase = false
)

var (
	// maximum number of lines to look for matching scope syntax (parens, brackets)
	maxScopeLines = 100 // `default:"100" min:"10" step:"10"`

	// maximum number of lines to apply syntax highlighting markup on
	maxMarkupLines = 10000 // `default:"10000" min:"1000" step:"1000"`

	// amount of time to wait before starting a new background markup process, after text changes within a single line (always does after line insertion / deletion)
	markupDelay = 500 * time.Millisecond // `default:"500" min:"100" step:"100"`
)

// Lines manages multi-line monospaced text with a given line width in runes,
// so that all text wrapping, editing, and navigation logic can be managed
// purely in text space, allowing rendering and GUI layout to be relatively fast.
// This is suitable for text editing and terminal applications, among others.
// The text encoded as runes along with a corresponding [rich.Text] markup
// representation with syntax highlighting etc.
// The markup is updated in a separate goroutine for efficiency.
// Everything is protected by an overall sync.Mutex and is safe to concurrent access,
// and thus nothing is exported and all access is through protected accessor functions.
// In general, all unexported methods do NOT lock, and all exported methods do.
type Lines struct {

	// Options are the options for how text editing and viewing works.
	Options Options

	// Highlighter does the syntax highlighting markup, and contains the
	// parameters thereof, such as the language and style.
	Highlighter highlighting.Highlighter

	// ChangedFunc is called whenever the text content is changed.
	// The changed flag is always updated on changes, but this can be
	// used for other flags or events that need to be tracked. The
	// Lock is off when this is called.
	ChangedFunc func()

	// MarkupDoneFunc is called when the offline markup pass is done
	// so that the GUI can be updated accordingly.  The lock is off
	// when this is called.
	MarkupDoneFunc func()

	// width is the current line width in rune characters, used for line wrapping.
	width int

	// FontStyle is the default font styling to use for markup.
	// Is set to use the monospace font.
	fontStyle *rich.Style

	// TextStyle is the default text styling to use for markup.
	textStyle *text.Style

	// todo: probably can unexport this?
	// Undos is the undo manager.
	undos Undo

	// ParseState is the parsing state information for the file.
	parseState parse.FileStates

	// changed indicates whether any changes have been made.
	// Use [IsChanged] method to access.
	changed bool

	// lines are the live lines of text being edited, with the latest modifications.
	// They are encoded as runes per line, which is necessary for one-to-one rune/glyph
	// rendering correspondence. All textpos positions are in rune indexes.
	lines [][]rune

	// breaks are the indexes of the line breaks for each line. The number of display
	// lines per logical line is the number of breaks + 1.
	breaks [][]int

	// markup is the marked-up version of the edited text lines, after being run
	// through the syntax highlighting process. This is what is actually rendered.
	markup []rich.Text

	// tags are the extra custom tagged regions for each line.
	tags []lexer.Line

	// hiTags are the syntax highlighting tags, which are auto-generated.
	hiTags []lexer.Line

	// markupEdits are the edits that were made during the time it takes to generate
	// the new markup tags. this is rare but it does happen.
	markupEdits []*textpos.Edit

	// markupDelayTimer is the markup delay timer.
	markupDelayTimer *time.Timer

	// markupDelayMu is the mutex for updating the markup delay timer.
	markupDelayMu sync.Mutex

	// use Lock(), Unlock() directly for overall mutex on any content updates
	sync.Mutex
}

// numLines returns number of lines
func (ls *Lines) numLines() int {
	return len(ls.lines)
}

// isValidLine returns true if given line number is in range.
func (ls *Lines) isValidLine(ln int) bool {
	if ln < 0 {
		return false
	}
	return ln < ls.numLines()
}

// bytesToLines sets the rune lines from source text
func (ls *Lines) bytesToLines(txt []byte) {
	if txt == nil {
		txt = []byte("")
	}
	ls.setLineBytes(bytes.Split(txt, []byte("\n")))
}

// setLineBytes sets the lines from source [][]byte.
func (ls *Lines) setLineBytes(lns [][]byte) {
	n := len(lns)
	if n > 1 && len(lns[n-1]) == 0 { // lines have lf at end typically
		lns = lns[:n-1]
		n--
	}
	ls.lines = slicesx.SetLength(ls.lines, n)
	ls.breaks = slicesx.SetLength(ls.breaks, n)
	ls.tags = slicesx.SetLength(ls.tags, n)
	ls.hiTags = slicesx.SetLength(ls.hiTags, n)
	ls.markup = slicesx.SetLength(ls.markup, n)
	for ln, txt := range lns {
		ls.lines[ln] = runes.SetFromBytes(ls.lines[ln], txt)
		ls.markup[ln] = rich.NewText(ls.fontStyle, ls.lines[ln]) // start with raw
	}
	ls.initialMarkup()
	ls.startDelayedReMarkup()
}

// bytes returns the current text lines as a slice of bytes.
// with an additional line feed at the end, per POSIX standards.
func (ls *Lines) bytes() []byte {
	nb := ls.width * ls.numLines()
	b := make([]byte, 0, nb)
	for ln := range ls.lines {
		b = append(b, []byte(string(ls.lines[ln]))...)
		b = append(b, []byte("\n")...)
	}
	// https://stackoverflow.com/questions/729692/why-should-text-files-end-with-a-newline
	b = append(b, []byte("\n")...)
	return b
}

// strings returns the current text as []string array.
// If addNewLine is true, each string line has a \n appended at end.
func (ls *Lines) strings(addNewLine bool) []string {
	str := make([]string, ls.numLines())
	for i, l := range ls.lines {
		str[i] = string(l)
		if addNewLine {
			str[i] += "\n"
		}
	}
	return str
}

////////   Appending Lines

// endPos returns the ending position at end of lines
func (ls *Lines) endPos() textpos.Pos {
	n := ls.numLines()
	if n == 0 {
		return textpos.Pos{}
	}
	return textpos.Pos{n - 1, len(ls.lines[n-1])}
}

// appendTextMarkup appends new lines of text to end of lines,
// using insert, returns edit, and uses supplied markup to render it.
func (ls *Lines) appendTextMarkup(text []rune, markup []rich.Text) *textpos.Edit {
	if len(text) == 0 {
		return &textpos.Edit{}
	}
	ed := ls.endPos()
	tbe := ls.insertText(ed, text)

	st := tbe.Region.Start.Line
	el := tbe.Region.End.Line
	// n := (el - st) + 1
	for ln := st; ln <= el; ln++ {
		ls.markup[ln] = markup[ln-st]
	}
	return tbe
}

// appendTextLineMarkup appends one line of new text to end of lines, using
// insert, and appending a LF at the end of the line if it doesn't already
// have one. User-supplied markup is used. Returns the edit region.
func (ls *Lines) appendTextLineMarkup(text []rune, markup rich.Text) *textpos.Edit {
	ed := ls.endPos()
	sz := len(text)
	addLF := true
	if sz > 0 {
		if text[sz-1] == '\n' {
			addLF = false
		}
	}
	efft := text
	if addLF {
		efft = make([]rune, sz+1)
		copy(efft, text)
		efft[sz] = '\n'
	}
	tbe := ls.insertText(ed, efft)
	ls.markup[tbe.Region.Start.Line] = markup
	return tbe
}

////////   Edits

// isValidPos returns an error if position is invalid.
func (ls *Lines) isValidPos(pos textpos.Pos) error {
	n := ls.numLines()
	if n == 0 {
		if pos.Line != 0 || pos.Char != 0 {
			return fmt.Errorf("invalid position for empty text: %s", pos)
		}
	}
	if pos.Line < 0 || pos.Line >= n {
		return fmt.Errorf("invalid line number for n lines %d: %s", n, pos)
	}
	llen := len(ls.lines[pos.Line])
	if pos.Char < 0 || pos.Char > llen {
		return fmt.Errorf("invalid character position for pos %d: %s", llen, pos)
	}
	return nil
}

// region returns a Edit representation of text between start and end positions
// returns nil and logs an error if not a valid region.
// sets the timestamp on the Edit to now
func (ls *Lines) region(st, ed textpos.Pos) *textpos.Edit {
	if errors.Log(ls.isValidPos(st)) != nil {
		return nil
	}
	if errors.Log(ls.isValidPos(ed)) != nil {
		return nil
	}
	if st == ed {
		return nil
	}
	if !st.IsLess(ed) {
		log.Printf("lines.region: starting position must be less than ending!: st: %v, ed: %v\n", st, ed)
		return nil
	}
	tbe := &textpos.Edit{Region: textpos.NewRegionPosTime(st, ed)}
	if ed.Line == st.Line {
		sz := ed.Char - st.Char
		tbe.Text = make([][]rune, 1)
		tbe.Text[0] = make([]rune, sz)
		copy(tbe.Text[0][:sz], ls.lines[st.Line][st.Char:ed.Char])
	} else {
		nln := tbe.Region.NumLines()
		tbe.Text = make([][]rune, nln)
		stln := st.Line
		if st.Char > 0 {
			ec := len(ls.lines[st.Line])
			sz := ec - st.Char
			if sz > 0 {
				tbe.Text[0] = make([]rune, sz)
				copy(tbe.Text[0], ls.lines[st.Line][st.Char:])
			}
			stln++
		}
		edln := ed.Line
		if ed.Char < len(ls.lines[ed.Line]) {
			tbe.Text[ed.Line-st.Line] = make([]rune, ed.Char)
			copy(tbe.Text[ed.Line-st.Line], ls.lines[ed.Line][:ed.Char])
			edln--
		}
		for ln := stln; ln <= edln; ln++ {
			ti := ln - st.Line
			sz := len(ls.lines[ln])
			tbe.Text[ti] = make([]rune, sz)
			copy(tbe.Text[ti], ls.lines[ln])
		}
	}
	return tbe
}

// regionRect returns a Edit representation of text between start and end
// positions as a rectangle.
// returns nil and logs an error if not a valid region.
// sets the timestamp on the Edit to now
func (ls *Lines) regionRect(st, ed textpos.Pos) *textpos.Edit {
	if errors.Log(ls.isValidPos(st)) != nil {
		return nil
	}
	if errors.Log(ls.isValidPos(ed)) != nil {
		return nil
	}
	if st == ed {
		return nil
	}
	if !st.IsLess(ed) || st.Char >= ed.Char {
		log.Printf("core.Buf.RegionRect: starting position must be less than ending!: st: %v, ed: %v\n", st, ed)
		return nil
	}
	tbe := &textpos.Edit{Region: textpos.NewRegionPosTime(st, ed)}
	tbe.Rect = true
	nln := tbe.Region.NumLines()
	nch := (ed.Char - st.Char)
	tbe.Text = make([][]rune, nln)
	for i := range nln {
		ln := st.Line + i
		lr := ls.lines[ln]
		ll := len(lr)
		var txt []rune
		if ll > st.Char {
			sz := min(ll-st.Char, nch)
			txt = make([]rune, sz, nch)
			edl := min(ed.Char, ll)
			copy(txt, lr[st.Char:edl])
		}
		if len(txt) < nch { // rect
			txt = append(txt, runes.Repeat([]rune(" "), nch-len(txt))...)
		}
		tbe.Text[i] = txt
	}
	return tbe
}

// callChangedFunc calls the ChangedFunc if it is set,
// starting from a Lock state, losing and then regaining the lock.
func (ls *Lines) callChangedFunc() {
	if ls.ChangedFunc == nil {
		return
	}
	ls.Unlock()
	ls.ChangedFunc()
	ls.Lock()
}

// deleteText is the primary method for deleting text,
// between start and end positions.
// An Undo record is automatically saved depending on Undo.Off setting.
func (ls *Lines) deleteText(st, ed textpos.Pos) *textpos.Edit {
	tbe := ls.deleteTextImpl(st, ed)
	ls.saveUndo(tbe)
	return tbe
}

func (ls *Lines) deleteTextImpl(st, ed textpos.Pos) *textpos.Edit {
	tbe := ls.region(st, ed)
	if tbe == nil {
		return nil
	}
	tbe.Delete = true
	nl := ls.numLines()
	if ed.Line == st.Line {
		if st.Line < nl {
			ec := min(ed.Char, len(ls.lines[st.Line])) // somehow region can still not be valid.
			ls.lines[st.Line] = append(ls.lines[st.Line][:st.Char], ls.lines[st.Line][ec:]...)
			ls.linesEdited(tbe)
		}
	} else {
		// first get chars on start and end
		stln := st.Line + 1
		cpln := st.Line
		ls.lines[st.Line] = ls.lines[st.Line][:st.Char]
		eoedl := 0
		if ed.Line >= nl {
			// todo: somehow this is happening in patch diffs -- can't figure out why
			// fmt.Println("err in range:", ed.Line, nl, ed.Char)
			ed.Line = nl - 1
		}
		if ed.Char < len(ls.lines[ed.Line]) {
			eoedl = len(ls.lines[ed.Line][ed.Char:])
		}
		var eoed []rune
		if eoedl > 0 { // save it
			eoed = make([]rune, eoedl)
			copy(eoed, ls.lines[ed.Line][ed.Char:])
		}
		ls.lines = append(ls.lines[:stln], ls.lines[ed.Line+1:]...)
		if eoed != nil {
			ls.lines[cpln] = append(ls.lines[cpln], eoed...)
		}
		ls.linesDeleted(tbe)
	}
	ls.changed = true
	ls.callChangedFunc()
	return tbe
}

// deleteTextRect deletes rectangular region of text between start, end
// defining the upper-left and lower-right corners of a rectangle.
// Fails if st.Char >= ed.Char. Sets the timestamp on resulting Edit to now.
// An Undo record is automatically saved depending on Undo.Off setting.
func (ls *Lines) deleteTextRect(st, ed textpos.Pos) *textpos.Edit {
	tbe := ls.deleteTextRectImpl(st, ed)
	ls.saveUndo(tbe)
	return tbe
}

func (ls *Lines) deleteTextRectImpl(st, ed textpos.Pos) *textpos.Edit {
	tbe := ls.regionRect(st, ed)
	if tbe == nil {
		return nil
	}
	tbe.Delete = true
	for ln := st.Line; ln <= ed.Line; ln++ {
		l := ls.lines[ln]
		if len(l) > st.Char {
			if ed.Char < len(l)-1 {
				ls.lines[ln] = append(l[:st.Char], l[ed.Char:]...)
			} else {
				ls.lines[ln] = l[:st.Char]
			}
		}
	}
	ls.linesEdited(tbe)
	ls.changed = true
	ls.callChangedFunc()
	return tbe
}

// insertText is the primary method for inserting text,
// at given starting position.  Sets the timestamp on resulting Edit to now.
// An Undo record is automatically saved depending on Undo.Off setting.
func (ls *Lines) insertText(st textpos.Pos, text []rune) *textpos.Edit {
	tbe := ls.insertTextImpl(st, textpos.NewEditFromRunes(text))
	ls.saveUndo(tbe)
	return tbe
}

func (ls *Lines) insertTextImpl(st textpos.Pos, ins *textpos.Edit) *textpos.Edit {
	if errors.Log(ls.isValidPos(st)) != nil {
		return nil
	}
	lns := runes.Split(text, []rune("\n"))
	sz := len(lns)
	ed := st
	var tbe *textpos.Edit
	st.Char = min(len(ls.lines[st.Line]), st.Char)
	if sz == 1 {
		ls.lines[st.Line] = slices.Insert(ls.lines[st.Line], st.Char, lns[0]...)
		ed.Char += len(lns[0])
		tbe = ls.region(st, ed)
		ls.linesEdited(tbe)
	} else {
		if ls.lines[st.Line] == nil {
			ls.lines[st.Line] = []rune{}
		}
		eostl := len(ls.lines[st.Line][st.Char:]) // end of starting line
		var eost []rune
		if eostl > 0 { // save it
			eost = make([]rune, eostl)
			copy(eost, ls.lines[st.Line][st.Char:])
		}
		ls.lines[st.Line] = append(ls.lines[st.Line][:st.Char], lns[0]...)
		nsz := sz - 1
		stln := st.Line + 1
		ls.lines = slices.Insert(ls.lines, stln, lns[1:]...)
		ed.Line += nsz
		ed.Char = len(ls.lines[ed.Line])
		if eost != nil {
			ls.lines[ed.Line] = append(ls.lines[ed.Line], eost...)
		}
		tbe = ls.region(st, ed)
		ls.linesInserted(tbe)
	}
	ls.changed = true
	ls.callChangedFunc()
	return tbe
}

// insertTextRect inserts a rectangle of text defined in given Edit record,
// (e.g., from RegionRect or DeleteRect).
// Returns a copy of the Edit record with an updated timestamp.
// An Undo record is automatically saved depending on Undo.Off setting.
func (ls *Lines) insertTextRect(tbe *textpos.Edit) *textpos.Edit {
	re := ls.insertTextRectImpl(tbe)
	ls.saveUndo(re)
	return tbe
}

func (ls *Lines) insertTextRectImpl(tbe *textpos.Edit) *textpos.Edit {
	st := tbe.Region.Start
	ed := tbe.Region.End
	nlns := (ed.Line - st.Line) + 1
	if nlns <= 0 {
		return nil
	}
	ls.changed = true
	// make sure there are enough lines -- add as needed
	cln := ls.numLines()
	if cln <= ed.Line {
		nln := (1 + ed.Line) - cln
		tmp := make([][]rune, nln)
		ls.lines = append(ls.lines, tmp...)
		ie := &textpos.Edit{}
		ie.Region.Start.Line = cln - 1
		ie.Region.End.Line = ed.Line
		ls.linesInserted(ie)
	}
	// nch := (ed.Char - st.Char)
	for i := 0; i < nlns; i++ {
		ln := st.Line + i
		lr := ls.lines[ln]
		ir := tbe.Text[i]
		if len(lr) < st.Char {
			lr = append(lr, runes.Repeat([]rune{' '}, st.Char-len(lr))...)
		}
		nt := slices.Insert(nt, st.Char, ir...)
		ls.lines[ln] = nt
	}
	re := tbe.Clone()
	re.Delete = false
	re.Region.TimeNow()
	ls.linesEdited(re)
	return re
}

// ReplaceText does DeleteText for given region, and then InsertText at given position
// (typically same as delSt but not necessarily).
// if matchCase is true, then the lexer.MatchCase function is called to match the
// case (upper / lower) of the new inserted text to that of the text being replaced.
// returns the Edit for the inserted text.
// An Undo record is automatically saved depending on Undo.Off setting.
func (ls *Lines) replaceText(delSt, delEd, insPos textpos.Pos, insTxt string, matchCase bool) *textpos.Edit {
	if matchCase {
		red := ls.region(delSt, delEd)
		cur := string(red.ToBytes())
		insTxt = lexer.MatchCase(cur, insTxt)
	}
	if len(insTxt) > 0 {
		ls.deleteText(delSt, delEd)
		return ls.insertText(insPos, []rune(insTxt))
	}
	return ls.deleteText(delSt, delEd)
}

/////////////////////////////////////////////////////////////////////////////
//   Undo

// saveUndo saves given edit to undo stack
func (ls *Lines) saveUndo(tbe *textpos.Edit) {
	if tbe == nil {
		return
	}
	ls.undos.Save(tbe)
}

// undo undoes next group of items on the undo stack
func (ls *Lines) undo() []*textpos.Edit {
	tbe := ls.undos.UndoPop()
	if tbe == nil {
		// note: could clear the changed flag on tbe == nil in parent
		return nil
	}
	stgp := tbe.Group
	var eds []*textpos.Edit
	for {
		if tbe.Rect {
			if tbe.Delete {
				utbe := ls.insertTextRectImpl(tbe)
				utbe.Group = stgp + tbe.Group
				if ls.Options.EmacsUndo {
					ls.undos.SaveUndo(utbe)
				}
				eds = append(eds, utbe)
			} else {
				utbe := ls.deleteTextRectImpl(tbe.Region.Start, tbe.Region.End)
				utbe.Group = stgp + tbe.Group
				if ls.Options.EmacsUndo {
					ls.undos.SaveUndo(utbe)
				}
				eds = append(eds, utbe)
			}
		} else {
			if tbe.Delete {
				utbe := ls.insertTextImpl(tbe.Region.Start, tbe)
				utbe.Group = stgp + tbe.Group
				if ls.Options.EmacsUndo {
					ls.undos.SaveUndo(utbe)
				}
				eds = append(eds, utbe)
			} else {
				utbe := ls.deleteTextImpl(tbe.Region.Start, tbe.Region.End)
				utbe.Group = stgp + tbe.Group
				if ls.Options.EmacsUndo {
					ls.undos.SaveUndo(utbe)
				}
				eds = append(eds, utbe)
			}
		}
		tbe = ls.undos.UndoPopIfGroup(stgp)
		if tbe == nil {
			break
		}
	}
	return eds
}

// EmacsUndoSave is called by View at end of latest set of undo commands.
// If EmacsUndo mode is active, saves the current UndoStack to the regular Undo stack
// at the end, and moves undo to the very end -- undo is a constant stream.
func (ls *Lines) EmacsUndoSave() {
	if !ls.Options.EmacsUndo {
		return
	}
	ls.undos.UndoStackSave()
}

// redo redoes next group of items on the undo stack,
// and returns the last record, nil if no more
func (ls *Lines) redo() []*textpos.Edit {
	tbe := ls.undos.RedoNext()
	if tbe == nil {
		return nil
	}
	var eds []*textpos.Edit
	stgp := tbe.Group
	for {
		if tbe.Rect {
			if tbe.Delete {
				ls.deleteTextRectImpl(tbe.Region.Start, tbe.Region.End)
			} else {
				ls.insertTextRectImpl(tbe)
			}
		} else {
			if tbe.Delete {
				ls.deleteTextImpl(tbe.Region.Start, tbe.Region.End)
			} else {
				ls.insertTextImpl(tbe.Region.Start, tbe)
			}
		}
		eds = append(eds, tbe)
		tbe = ls.undos.RedoNextIfGroup(stgp)
		if tbe == nil {
			break
		}
	}
	return eds
}

// DiffBuffers computes the diff between this buffer and the other buffer,
// reporting a sequence of operations that would convert this buffer (a) into
// the other buffer (b).  Each operation is either an 'r' (replace), 'd'
// (delete), 'i' (insert) or 'e' (equal).  Everything is line-based (0, offset).
func (ls *Lines) DiffBuffers(ob *Lines) Diffs {
	ls.Lock()
	defer ls.Unlock()
	return ls.diffBuffers(ob)
}

// PatchFromBuffer patches (edits) using content from other,
// according to diff operations (e.g., as generated from DiffBufs).
func (ls *Lines) PatchFromBuffer(ob *Lines, diffs Diffs) bool {
	ls.Lock()
	defer ls.Unlock()
	return ls.patchFromBuffer(ob, diffs)
}

/////////////////////////////////////////////////////////////////////////////
//   Syntax Highlighting Markup

// linesEdited re-marks-up lines in edit (typically only 1).
func (ls *Lines) linesEdited(tbe *textpos.Edit) {
	st, ed := tbe.Region.Start.Line, tbe.Region.End.Line
	for ln := st; ln <= ed; ln++ {
		ls.markup[ln] = rich.NewText(ls.fontStyle, ls.lines[ln])
	}
	ls.markupLines(st, ed)
	ls.startDelayedReMarkup()
}

// linesInserted inserts new lines for all other line-based slices
// corresponding to lines inserted in the lines slice.
func (ls *Lines) linesInserted(tbe *textpos.Edit) {
	stln := tbe.Region.Start.Line + 1
	nsz := (tbe.Region.End.Line - tbe.Region.Start.Line)

	// todo: breaks!
	ls.markupEdits = append(ls.markupEdits, tbe)
	ls.markup = slices.Insert(ls.markup, stln, make([]rich.Text, nsz)...)
	ls.tags = slices.Insert(ls.tags, stln, make([]lexer.Line, nsz)...)
	ls.hiTags = slices.Insert(ls.hiTags, stln, make([]lexer.Line, nsz)...)

	if ls.Highlighter.UsingParse() {
		pfs := ls.parseState.Done()
		pfs.Src.LinesInserted(stln, nsz)
	}
	ls.linesEdited(tbe)
}

// linesDeleted deletes lines in Markup corresponding to lines
// deleted in Lines text.
func (ls *Lines) linesDeleted(tbe *textpos.Edit) {
	ls.markupEdits = append(ls.markupEdits, tbe)
	stln := tbe.Region.Start.Line
	edln := tbe.Region.End.Line
	ls.markup = append(ls.markup[:stln], ls.markup[edln:]...)
	ls.tags = append(ls.tags[:stln], ls.tags[edln:]...)
	ls.hiTags = append(ls.hiTags[:stln], ls.hiTags[edln:]...)

	if ls.Highlighter.UsingParse() {
		pfs := ls.parseState.Done()
		pfs.Src.LinesDeleted(stln, edln)
	}
	st := tbe.Region.Start.Line
	// todo:
	// ls.markup[st] = highlighting.HtmlEscapeRunes(ls.lines[st])
	ls.markupLines(st, st)
	ls.startDelayedReMarkup()
}

///////////////////////////////////////////////////////////////////////////////////////
//  Markup

// initialMarkup does the first-pass markup on the file
func (ls *Lines) initialMarkup() {
	if !ls.Highlighter.Has || ls.numLines() == 0 {
		return
	}
	if ls.Highlighter.UsingParse() {
		fs := ls.parseState.Done() // initialize
		fs.Src.SetBytes(ls.bytes())
	}
	mxhi := min(100, ls.numLines())
	txt := ls.bytes()
	tags, err := ls.markupTags(txt)
	if err == nil {
		ls.markupApplyTags(tags)
	}
}

// startDelayedReMarkup starts a timer for doing markup after an interval.
func (ls *Lines) startDelayedReMarkup() {
	ls.markupDelayMu.Lock()
	defer ls.markupDelayMu.Unlock()

	if !ls.Highlighter.Has || ls.numLines() == 0 || ls.numLines() > maxMarkupLines {
		return
	}
	if ls.markupDelayTimer != nil {
		ls.markupDelayTimer.Stop()
		ls.markupDelayTimer = nil
	}
	ls.markupDelayTimer = time.AfterFunc(markupDelay, func() {
		ls.markupDelayTimer = nil
		ls.asyncMarkup() // already in a goroutine
	})
}

// StopDelayedReMarkup stops timer for doing markup after an interval
func (ls *Lines) StopDelayedReMarkup() {
	ls.markupDelayMu.Lock()
	defer ls.markupDelayMu.Unlock()

	if ls.markupDelayTimer != nil {
		ls.markupDelayTimer.Stop()
		ls.markupDelayTimer = nil
	}
}

// reMarkup runs re-markup on text in background
func (ls *Lines) reMarkup() {
	if !ls.Highlighter.Has || ls.numLines() == 0 || ls.numLines() > maxMarkupLines {
		return
	}
	ls.StopDelayedReMarkup()
	go ls.asyncMarkup()
}

// AdjustRegion adjusts given text region for any edits that
// have taken place since time stamp on region (using the Undo stack).
// If region was wholly within a deleted region, then RegionNil will be
// returned -- otherwise it is clipped appropriately as function of deletes.
func (ls *Lines) AdjustRegion(reg textpos.RegionTime) textpos.RegionTime {
	return ls.undos.AdjustRegion(reg)
}

// adjustedTags updates tag positions for edits, for given list of tags
func (ls *Lines) adjustedTags(ln int) lexer.Line {
	if !ls.isValidLine(ln) {
		return nil
	}
	return ls.adjustedTagsLine(ls.tags[ln], ln)
}

// adjustedTagsLine updates tag positions for edits, for given list of tags
func (ls *Lines) adjustedTagsLine(tags lexer.Line, ln int) lexer.Line {
	sz := len(tags)
	if sz == 0 {
		return nil
	}
	ntags := make(lexer.Line, 0, sz)
	for _, tg := range tags {
		reg := RegionTime{Start: textpos.Pos{Ln: ln, Ch: tg.St}, End: textpos.Pos{Ln: ln, Ch: tg.Ed}}
		reg.Time = tg.Time
		reg = ls.undos.AdjustRegion(reg)
		if !reg.IsNil() {
			ntr := ntags.AddLex(tg.Token, reg.Start.Char, reg.End.Char)
			ntr.Time.Now()
		}
	}
	return ntags
}

// asyncMarkup does the markupTags from a separate goroutine.
// Does not start or end with lock, but acquires at end to apply.
func (ls *Lines) asyncMarkup() {
	ls.Lock()
	txt := ls.bytes()
	ls.markupEdits = nil // only accumulate after this point; very rare
	ls.Unlock()

	tags, err := ls.markupTags(txt)
	if err != nil {
		return
	}
	ls.Lock()
	ls.markupApplyTags(tags)
	ls.Unlock()
	if ls.MarkupDoneFunc != nil {
		ls.MarkupDoneFunc()
	}
}

// markupTags generates the new markup tags from the highligher.
// this is a time consuming step, done via asyncMarkup typically.
// does not require any locking.
func (ls *Lines) markupTags(txt []byte) ([]lexer.Line, error) {
	return ls.Highlighter.MarkupTagsAll(txt)
}

// markupApplyEdits applies any edits in markupEdits to the
// tags prior to applying the tags.  returns the updated tags.
func (ls *Lines) markupApplyEdits(tags []lexer.Line) []lexer.Line {
	edits := ls.markupEdits
	ls.markupEdits = nil
	if ls.Highlighter.UsingParse() {
		pfs := ls.parseState.Done()
		for _, tbe := range edits {
			if tbe.Delete {
				stln := tbe.Region.Start.Line
				edln := tbe.Region.End.Line
				pfs.Src.LinesDeleted(stln, edln)
			} else {
				stln := tbe.Region.Start.Line + 1
				nlns := (tbe.Region.End.Line - tbe.Region.Start.Line)
				pfs.Src.LinesInserted(stln, nlns)
			}
		}
		for ln := range tags {
			tags[ln] = pfs.LexLine(ln) // does clone, combines comments too
		}
	} else {
		for _, tbe := range edits {
			if tbe.Delete {
				stln := tbe.Region.Start.Line
				edln := tbe.Region.End.Line
				tags = append(tags[:stln], tags[edln:]...)
			} else {
				stln := tbe.Region.Start.Line + 1
				nlns := (tbe.Region.End.Line - tbe.Region.Start.Line)
				stln = min(stln, len(tags))
				tags = slices.Insert(tags, stln, make([]lexer.Line, nlns)...)
			}
		}
	}
	return tags
}

// markupApplyTags applies given tags to current text
// and sets the markup lines.  Must be called under Lock.
func (ls *Lines) markupApplyTags(tags []lexer.Line) {
	tags = ls.markupApplyEdits(tags)
	maxln := min(len(tags), ls.numLines())
	for ln := range maxln {
		ls.hiTags[ln] = tags[ln]
		ls.tags[ln] = ls.adjustedTags(ln)
		ls.markup[ln] = highlighting.MarkupLine(ls.lines[ln], tags[ln], ls.tags[ln], highlighting.EscapeHTML)
	}
}

// markupLines generates markup of given range of lines.
// end is *inclusive* line.  Called after edits, under Lock().
// returns true if all lines were marked up successfully.
func (ls *Lines) markupLines(st, ed int) bool {
	n := ls.numLines()
	if !ls.Highlighter.Has || n == 0 {
		return false
	}
	if ed >= n {
		ed = n - 1
	}

	allgood := true
	for ln := st; ln <= ed; ln++ {
		ltxt := ls.lines[ln]
		mt, err := ls.Highlighter.MarkupTagsLine(ln, ltxt)
		if err == nil {
			ls.hiTags[ln] = mt
			ls.markup[ln] = highlighting.MarkupLine(ltxt, mt, ls.adjustedTags(ln), highlighting.EscapeHTML)
		} else {
			ls.markup[ln] = highlighting.HtmlEscapeRunes(ltxt)
			allgood = false
		}
	}
	// Now we trigger a background reparse of everything in a separate parse.FilesState
	// that gets switched into the current.
	return allgood
}

/////////////////////////////////////////////////////////////////////////////
//   Tags

// AddTag adds a new custom tag for given line, at given position.
func (ls *Lines) AddTag(ln, st, ed int, tag token.Tokens) {
	if !ls.IsValidLine(ln) {
		return
	}
	ls.Lock()
	defer ls.Unlock()

	tr := lexer.NewLex(token.KeyToken{Token: tag}, st, ed)
	tr.Time.Now()
	if len(ls.tags[ln]) == 0 {
		ls.tags[ln] = append(ls.tags[ln], tr)
	} else {
		ls.tags[ln] = ls.adjustedTags(ln) // must re-adjust before adding new ones!
		ls.tags[ln].AddSort(tr)
	}
	ls.markupLines(ln, ln)
}

// AddTagEdit adds a new custom tag for given line, using Edit for location.
func (ls *Lines) AddTagEdit(tbe *textpos.Edit, tag token.Tokens) {
	ls.AddTag(tbe.Region.Start.Line, tbe.Region.Start.Char, tbe.Region.End.Char, tag)
}

// RemoveTag removes tag (optionally only given tag if non-zero)
// at given position if it exists. returns tag.
func (ls *Lines) RemoveTag(pos textpos.Pos, tag token.Tokens) (reg lexer.Lex, ok bool) {
	if !ls.IsValidLine(pos.Line) {
		return
	}
	ls.Lock()
	defer ls.Unlock()

	ls.tags[pos.Line] = ls.adjustedTags(pos.Line) // re-adjust for current info
	for i, t := range ls.tags[pos.Line] {
		if t.ContainsPos(pos.Char) {
			if tag > 0 && t.Token.Token != tag {
				continue
			}
			ls.tags[pos.Line].DeleteIndex(i)
			reg = t
			ok = true
			break
		}
	}
	if ok {
		ls.markupLines(pos.Line, pos.Line)
	}
	return
}

// SetTags tags for given line.
func (ls *Lines) SetTags(ln int, tags lexer.Line) {
	if !ls.IsValidLine(ln) {
		return
	}
	ls.Lock()
	defer ls.Unlock()
	ls.tags[ln] = tags
}

// lexObjPathString returns the string at given lex, and including prior
// lex-tagged regions that include sequences of PunctSepPeriod and NameTag
// which are used for object paths -- used for e.g., debugger to pull out
// variable expressions that can be evaluated.
func (ls *Lines) lexObjPathString(ln int, lx *lexer.Lex) string {
	if !ls.isValidLine(ln) {
		return ""
	}
	lln := len(ls.lines[ln])
	if lx.Ed > lln {
		return ""
	}
	stlx := lexer.ObjPathAt(ls.hiTags[ln], lx)
	if stlx.St >= lx.Ed {
		return ""
	}
	return string(ls.lines[ln][stlx.St:lx.Ed])
}

// hiTagAtPos returns the highlighting (markup) lexical tag at given position
// using current Markup tags, and index, -- could be nil if none or out of range
func (ls *Lines) hiTagAtPos(pos textpos.Pos) (*lexer.Lex, int) {
	if !ls.isValidLine(pos.Line) {
		return nil, -1
	}
	return ls.hiTags[pos.Line].AtPos(pos.Char)
}

// inTokenSubCat returns true if the given text position is marked with lexical
// type in given SubCat sub-category.
func (ls *Lines) inTokenSubCat(pos textpos.Pos, subCat token.Tokens) bool {
	lx, _ := ls.hiTagAtPos(pos)
	return lx != nil && lx.Token.Token.InSubCat(subCat)
}

// inLitString returns true if position is in a string literal
func (ls *Lines) inLitString(pos textpos.Pos) bool {
	return ls.inTokenSubCat(pos, token.LitStr)
}

// inTokenCode returns true if position is in a Keyword,
// Name, Operator, or Punctuation.
// This is useful for turning off spell checking in docs
func (ls *Lines) inTokenCode(pos textpos.Pos) bool {
	lx, _ := ls.hiTagAtPos(pos)
	if lx == nil {
		return false
	}
	return lx.Token.Token.IsCode()
}

/////////////////////////////////////////////////////////////////////////////
//   Indenting

// see parse/lexer/indent.go for support functions

// indentLine indents line by given number of tab stops, using tabs or spaces,
// for given tab size (if using spaces) -- either inserts or deletes to reach target.
// Returns edit record for any change.
func (ls *Lines) indentLine(ln, ind int) *textpos.Edit {
	tabSz := ls.Options.TabSize
	ichr := indent.Tab
	if ls.Options.SpaceIndent {
		ichr = indent.Space
	}
	curind, _ := lexer.LineIndent(ls.lines[ln], tabSz)
	if ind > curind {
		return ls.insertText(textpos.Pos{Ln: ln}, indent.Bytes(ichr, ind-curind, tabSz))
	} else if ind < curind {
		spos := indent.Len(ichr, ind, tabSz)
		cpos := indent.Len(ichr, curind, tabSz)
		return ls.deleteText(textpos.Pos{Ln: ln, Ch: spos}, textpos.Pos{Ln: ln, Ch: cpos})
	}
	return nil
}

// autoIndent indents given line to the level of the prior line, adjusted
// appropriately if the current line starts with one of the given un-indent
// strings, or the prior line ends with one of the given indent strings.
// Returns any edit that took place (could be nil), along with the auto-indented
// level and character position for the indent of the current line.
func (ls *Lines) autoIndent(ln int) (tbe *textpos.Edit, indLev, chPos int) {
	tabSz := ls.Options.TabSize
	lp, _ := parse.LanguageSupport.Properties(ls.parseState.Known)
	var pInd, delInd int
	if lp != nil && lp.Lang != nil {
		pInd, delInd, _, _ = lp.Lang.IndentLine(&ls.parseState, ls.lines, ls.hiTags, ln, tabSz)
	} else {
		pInd, delInd, _, _ = lexer.BracketIndentLine(ls.lines, ls.hiTags, ln, tabSz)
	}
	ichr := ls.Options.IndentChar()
	indLev = pInd + delInd
	chPos = indent.Len(ichr, indLev, tabSz)
	tbe = ls.indentLine(ln, indLev)
	return
}

// autoIndentRegion does auto-indent over given region; end is *exclusive*
func (ls *Lines) autoIndentRegion(start, end int) {
	end = min(ls.numLines(), end)
	for ln := start; ln < end; ln++ {
		ls.autoIndent(ln)
	}
}

// commentStart returns the char index where the comment
// starts on given line, -1 if no comment.
func (ls *Lines) commentStart(ln int) int {
	if !ls.isValidLine(ln) {
		return -1
	}
	comst, _ := ls.Options.CommentStrings()
	if comst == "" {
		return -1
	}
	return runes.Index(ls.lines[ln], []rune(comst))
}

// inComment returns true if the given text position is within
// a commented region.
func (ls *Lines) inComment(pos textpos.Pos) bool {
	if ls.inTokenSubCat(pos, token.Comment) {
		return true
	}
	cs := ls.commentStart(pos.Line)
	if cs < 0 {
		return false
	}
	return pos.Char > cs
}

// lineCommented returns true if the given line is a full-comment
// line (i.e., starts with a comment).
func (ls *Lines) lineCommented(ln int) bool {
	if !ls.isValidLine(ln) {
		return false
	}
	tags := ls.hiTags[ln]
	if len(tags) == 0 {
		return false
	}
	return tags[0].Token.Token.InCat(token.Comment)
}

// commentRegion inserts comment marker on given lines; end is *exclusive*.
func (ls *Lines) commentRegion(start, end int) {
	tabSz := ls.Options.TabSize
	ch := 0
	ind, _ := lexer.LineIndent(ls.lines[start], tabSz)
	if ind > 0 {
		if ls.Options.SpaceIndent {
			ch = ls.Options.TabSize * ind
		} else {
			ch = ind
		}
	}

	comst, comed := ls.Options.CommentStrings()
	if comst == "" {
		log.Printf("text.Lines: attempt to comment region without any comment syntax defined")
		return
	}

	eln := min(ls.numLines(), end)
	ncom := 0
	nln := eln - start
	for ln := start; ln < eln; ln++ {
		if ls.lineCommented(ln) {
			ncom++
		}
	}
	trgln := max(nln-2, 1)
	doCom := true
	if ncom >= trgln {
		doCom = false
	}

	for ln := start; ln < eln; ln++ {
		if doCom {
			ls.insertText(textpos.Pos{Ln: ln, Ch: ch}, []byte(comst))
			if comed != "" {
				lln := len(ls.lines[ln])
				ls.insertText(textpos.Pos{Ln: ln, Ch: lln}, []byte(comed))
			}
		} else {
			idx := ls.commentStart(ln)
			if idx >= 0 {
				ls.deleteText(textpos.Pos{Ln: ln, Ch: idx}, textpos.Pos{Ln: ln, Ch: idx + len(comst)})
			}
			if comed != "" {
				idx := runes.IndexFold(ls.lines[ln], []rune(comed))
				if idx >= 0 {
					ls.deleteText(textpos.Pos{Ln: ln, Ch: idx}, textpos.Pos{Ln: ln, Ch: idx + len(comed)})
				}
			}
		}
	}
}

// joinParaLines merges sequences of lines with hard returns forming paragraphs,
// separated by blank lines, into a single line per paragraph,
// within the given line regions; endLine is *inclusive*.
func (ls *Lines) joinParaLines(startLine, endLine int) {
	// current end of region being joined == last blank line
	curEd := endLine
	for ln := endLine; ln >= startLine; ln-- { // reverse order
		lb := ls.lineBytes[ln]
		lbt := bytes.TrimSpace(lb)
		if len(lbt) == 0 || ln == startLine {
			if ln < curEd-1 {
				stp := textpos.Pos{Ln: ln + 1}
				if ln == startLine {
					stp.Line--
				}
				ep := textpos.Pos{Ln: curEd - 1}
				if curEd == endLine {
					ep.Line = curEd
				}
				eln := ls.lines[ep.Line]
				ep.Char = len(eln)
				tlb := bytes.Join(ls.lineBytes[stp.Line:ep.Line+1], []byte(" "))
				ls.replaceText(stp, ep, stp, string(tlb), ReplaceNoMatchCase)
			}
			curEd = ln
		}
	}
}

// tabsToSpacesLine replaces tabs with spaces in the given line.
func (ls *Lines) tabsToSpacesLine(ln int) {
	tabSz := ls.Options.TabSize

	lr := ls.lines[ln]
	st := textpos.Pos{Ln: ln}
	ed := textpos.Pos{Ln: ln}
	i := 0
	for {
		if i >= len(lr) {
			break
		}
		r := lr[i]
		if r == '\t' {
			po := i % tabSz
			nspc := tabSz - po
			st.Char = i
			ed.Char = i + 1
			ls.replaceText(st, ed, st, indent.Spaces(1, nspc), ReplaceNoMatchCase)
			i += nspc
			lr = ls.lines[ln]
		} else {
			i++
		}
	}
}

// tabsToSpaces replaces tabs with spaces over given region; end is *exclusive*.
func (ls *Lines) tabsToSpaces(start, end int) {
	end = min(ls.numLines(), end)
	for ln := start; ln < end; ln++ {
		ls.tabsToSpacesLine(ln)
	}
}

// spacesToTabsLine replaces spaces with tabs in the given line.
func (ls *Lines) spacesToTabsLine(ln int) {
	tabSz := ls.Options.TabSize

	lr := ls.lines[ln]
	st := textpos.Pos{Ln: ln}
	ed := textpos.Pos{Ln: ln}
	i := 0
	nspc := 0
	for {
		if i >= len(lr) {
			break
		}
		r := lr[i]
		if r == ' ' {
			nspc++
			if nspc == tabSz {
				st.Char = i - (tabSz - 1)
				ed.Char = i + 1
				ls.replaceText(st, ed, st, "\t", ReplaceNoMatchCase)
				i -= tabSz - 1
				lr = ls.lines[ln]
				nspc = 0
			} else {
				i++
			}
		} else {
			nspc = 0
			i++
		}
	}
}

// spacesToTabs replaces tabs with spaces over given region; end is *exclusive*
func (ls *Lines) spacesToTabs(start, end int) {
	end = min(ls.numLines(), end)
	for ln := start; ln < end; ln++ {
		ls.spacesToTabsLine(ln)
	}
}

////////  Diff

// diffBuffers computes the diff between this buffer and the other buffer,
// reporting a sequence of operations that would convert this buffer (a) into
// the other buffer (b).  Each operation is either an 'r' (replace), 'd'
// (delete), 'i' (insert) or 'e' (equal).  Everything is line-based (0, offset).
func (ls *Lines) diffBuffers(ob *Lines) Diffs {
	astr := ls.strings(false)
	bstr := ob.strings(false)
	return DiffLines(astr, bstr)
}

// patchFromBuffer patches (edits) using content from other,
// according to diff operations (e.g., as generated from DiffBufs).
func (ls *Lines) patchFromBuffer(ob *Lines, diffs Diffs) bool {
	sz := len(diffs)
	mods := false
	for i := sz - 1; i >= 0; i-- { // go in reverse so changes are valid!
		df := diffs[i]
		switch df.Tag {
		case 'r':
			ls.deleteText(textpos.Pos{Ln: df.I1}, textpos.Pos{Ln: df.I2})
			ot := ob.Region(textpos.Pos{Ln: df.J1}, textpos.Pos{Ln: df.J2})
			ls.insertText(textpos.Pos{Ln: df.I1}, ot.ToBytes())
			mods = true
		case 'd':
			ls.deleteText(textpos.Pos{Ln: df.I1}, textpos.Pos{Ln: df.I2})
			mods = true
		case 'i':
			ot := ob.Region(textpos.Pos{Ln: df.J1}, textpos.Pos{Ln: df.J2})
			ls.insertText(textpos.Pos{Ln: df.I1}, ot.ToBytes())
			mods = true
		}
	}
	return mods
}
