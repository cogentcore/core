// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

//go:generate core generate -add-types

import (
	"bytes"
	"fmt"
	"image"
	"log"
	"slices"
	"sync"
	"time"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/metadata"
	"cogentcore.org/core/base/slicesx"
	"cogentcore.org/core/text/highlighting"
	"cogentcore.org/core/text/parse"
	"cogentcore.org/core/text/parse/lexer"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/runes"
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

	// amount of time to wait before starting a new background markup process,
	// after text changes within a single line (always does after line insertion / deletion)
	markupDelay = 500 * time.Millisecond // `default:"500" min:"100" step:"100"`

	// text buffer max lines to use diff-based revert to more quickly update
	// e.g., after file has been reformatted
	diffRevertLines = 10000 // `default:"10000" min:"0" step:"1000"`

	// text buffer max diffs to use diff-based revert to more quickly update
	// e.g., after file has been reformatted -- if too many differences, just revert.
	diffRevertDiffs = 20 // `default:"20" min:"0" step:"1"`
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

	// Settings are the options for how text editing and viewing works.
	Settings Settings

	// Highlighter does the syntax highlighting markup, and contains the
	// parameters thereof, such as the language and style.
	Highlighter highlighting.Highlighter

	// Autosave specifies whether an autosave copy of the file should
	// be automatically saved after changes are made.
	Autosave bool

	// FileModPromptFunc is called when a file has been modified in the filesystem
	// and it is about to be modified through an edit, in the fileModCheck function.
	// The prompt should determine whether the user wants to revert, overwrite, or
	// save current version as a different file. It must block until the user responds.
	FileModPromptFunc func()

	// Meta can be used to maintain misc metadata associated with the Lines text,
	// which allows the Lines object to be the primary data type for applications
	// dealing with text data, if there are just a few additional data elements needed.
	// Use standard Go camel-case key names, standards in [metadata].
	Meta metadata.Data

	// fontStyle is the default font styling to use for markup.
	// Is set to use the monospace font.
	fontStyle *rich.Style

	// undos is the undo manager.
	undos Undo

	// filename is the filename of the file that was last loaded or saved.
	// If this is empty then no file-related functionality is engaged.
	filename string

	// readOnly marks the contents as not editable. This is for the outer GUI
	// elements to consult, and is not enforced within Lines itself.
	readOnly bool

	// fileInfo is the full information about the current file, if one is set.
	fileInfo fileinfo.FileInfo

	// parseState is the parsing state information for the file.
	parseState parse.FileStates

	// changed indicates whether any changes have been made.
	// Use [IsChanged] method to access.
	changed bool

	// lines are the live lines of text being edited, with the latest modifications.
	// They are encoded as runes per line, which is necessary for one-to-one rune/glyph
	// rendering correspondence. All textpos positions are in rune indexes.
	lines [][]rune

	// tags are the extra custom tagged regions for each line.
	tags []lexer.Line

	// hiTags are the syntax highlighting tags, which are auto-generated.
	hiTags []lexer.Line

	// markup is the [rich.Text] encoded marked-up version of the text lines,
	// with the results of syntax highlighting. It just has the raw markup without
	// additional layout for a specific line width, which goes in a [view].
	markup []rich.Text

	// views are the distinct views of the lines, accessed via a unique view handle,
	// which is the key in the map. Each view can have its own width, and thus its own
	// markup and layout.
	views map[int]*view

	// lineColors associate a color with a given line number (key of map),
	// e.g., for a breakpoint or other such function.
	lineColors map[int]image.Image

	// markupEdits are the edits that were made during the time it takes to generate
	// the new markup tags. this is rare but it does happen.
	markupEdits []*textpos.Edit

	// markupDelayTimer is the markup delay timer.
	markupDelayTimer *time.Timer

	// markupDelayMu is the mutex for updating the markup delay timer.
	markupDelayMu sync.Mutex

	// posHistory is the history of cursor positions.
	// It can be used to move back through them.
	posHistory []textpos.Pos

	// links is the collection of all hyperlinks within the markup source,
	// indexed by the markup source line.
	// only updated at the full markup sweep.
	links map[int][]rich.Hyperlink

	// batchUpdating indicates that a batch update is under way,
	// so Input signals are not sent until the end.
	batchUpdating bool

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

	// use Lock(), Unlock() directly for overall mutex on any content updates
	sync.Mutex
}

func (ls *Lines) Metadata() *metadata.Data { return &ls.Meta }

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

// setText sets the rune lines from source text,
// and triggers initial markup and delayed full markup.
func (ls *Lines) setText(txt []byte) {
	ls.bytesToLines(txt)
	ls.initialMarkup()
	ls.startDelayedReMarkup()
}

// bytesToLines sets the rune lines from source text.
// it does not trigger any markup but does allocate everything.
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
	if ls.fontStyle == nil {
		ls.Defaults()
	}
	ls.lines = slicesx.SetLength(ls.lines, n)
	ls.tags = slicesx.SetLength(ls.tags, n)
	ls.hiTags = slicesx.SetLength(ls.hiTags, n)
	ls.markup = slicesx.SetLength(ls.markup, n)
	for ln, txt := range lns {
		ls.lines[ln] = runes.SetFromBytes(ls.lines[ln], txt)
		ls.markup[ln] = rich.NewText(ls.fontStyle, ls.lines[ln]) // start with raw
	}
}

// bytes returns the current text lines as a slice of bytes, up to
// given number of lines if maxLines > 0.
// Adds an additional line feed at the end, per POSIX standards.
func (ls *Lines) bytes(maxLines int) []byte {
	nl := ls.numLines()
	if maxLines > 0 {
		nl = min(nl, maxLines)
	}
	nb := 80 * nl
	b := make([]byte, 0, nb)
	for ln := range nl {
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
	return textpos.Pos{Line: n - 1, Char: len(ls.lines[n-1])}
}

// appendTextMarkup appends new lines of text to end of lines,
// using insert, returns edit, and uses supplied markup to render it.
func (ls *Lines) appendTextMarkup(text [][]rune, markup []rich.Text) *textpos.Edit {
	if len(text) == 0 {
		return &textpos.Edit{}
	}
	text = append(text, []rune{})
	ed := ls.endPos()
	tbe := ls.insertTextImpl(ed, text)
	if tbe == nil {
		fmt.Println("nil insert", ed, text)
		return nil
	}
	st := tbe.Region.Start.Line
	el := tbe.Region.End.Line
	for ln := st; ln < el; ln++ {
		ls.markup[ln] = markup[ln-st]
	}
	return tbe
}

////////   Edits

// isValidPos returns an error if position is invalid. Note that the end
// of the line (at length) is valid.
func (ls *Lines) isValidPos(pos textpos.Pos) error {
	n := ls.numLines()
	if n == 0 {
		if pos.Line != 0 || pos.Char != 0 {
			// return fmt.Errorf("invalid position for empty text: %s", pos)
			panic(fmt.Errorf("invalid position for empty text: %s", pos).Error())
		}
	}
	if pos.Line < 0 || pos.Line >= n {
		// return fmt.Errorf("invalid line number for n lines %d: %s", n, pos)
		panic(fmt.Errorf("invalid line number for n lines %d: %s", n, pos).Error())
	}
	llen := len(ls.lines[pos.Line])
	if pos.Char < 0 || pos.Char > llen {
		// return fmt.Errorf("invalid character position for pos %d: %s", llen, pos)
		panic(fmt.Errorf("invalid character position for pos %d: %s", llen, pos).Error())
	}
	return nil
}

// region returns a Edit representation of text between start and end positions
// returns nil and logs an error if not a valid region.
// sets the timestamp on the Edit to now
func (ls *Lines) region(st, ed textpos.Pos) *textpos.Edit {
	n := ls.numLines()
	if errors.Log(ls.isValidPos(st)) != nil {
		return nil
	}
	if ed.Line == n && ed.Char == 0 { // end line: goes to endpos
		ed.Line = n - 1
		ed.Char = len(ls.lines[ed.Line])
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
	tbe := &textpos.Edit{Region: textpos.NewRegionPos(st, ed)}
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
	tbe := &textpos.Edit{Region: textpos.NewRegionPos(st, ed)}
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
	// fmt.Println("del:", tbe.Region)
	tbe.Delete = true
	for ln := st.Line; ln <= ed.Line; ln++ {
		l := ls.lines[ln]
		// fmt.Println(ln, string(l))
		if len(l) > st.Char {
			if ed.Char <= len(l)-1 {
				ls.lines[ln] = slices.Delete(l, st.Char, ed.Char)
				// fmt.Println(ln, "del:", st.Char, ed.Char, string(ls.lines[ln]))
			} else {
				ls.lines[ln] = l[:st.Char]
				// fmt.Println(ln, "trunc", st.Char, ed.Char, string(ls.lines[ln]))
			}
		} else {
			panic(fmt.Sprintf("deleteTextRectImpl: line does not have text: %d < st.Char: %d", len(l), st.Char))
		}
	}
	ls.linesEdited(tbe)
	ls.changed = true
	return tbe
}

// insertText is the primary method for inserting text,
// at given starting position.  Sets the timestamp on resulting Edit to now.
// An Undo record is automatically saved depending on Undo.Off setting.
func (ls *Lines) insertText(st textpos.Pos, txt []rune) *textpos.Edit {
	tbe := ls.insertTextImpl(st, runes.Split(txt, []rune("\n")))
	ls.saveUndo(tbe)
	return tbe
}

// insertTextImpl inserts the Text at given starting position.
func (ls *Lines) insertTextImpl(st textpos.Pos, txt [][]rune) *textpos.Edit {
	if errors.Log(ls.isValidPos(st)) != nil {
		return nil
	}
	nl := len(txt)
	var tbe *textpos.Edit
	ed := st
	if nl == 1 {
		ls.lines[st.Line] = slices.Insert(ls.lines[st.Line], st.Char, txt[0]...)
		ed.Char += len(txt[0])
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
		ls.lines[st.Line] = append(ls.lines[st.Line][:st.Char], txt[0]...)
		nsz := nl - 1
		stln := st.Line + 1
		ls.lines = slices.Insert(ls.lines, stln, txt[1:]...)
		ed.Line += nsz
		ed.Char = len(ls.lines[ed.Line])
		if eost != nil {
			ls.lines[ed.Line] = append(ls.lines[ed.Line], eost...)
		}
		tbe = ls.region(st, ed)
		ls.linesInserted(tbe)
	}
	ls.changed = true
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
	nch := (ed.Char - st.Char)
	for i := 0; i < nlns; i++ {
		ln := st.Line + i
		lr := ls.lines[ln]
		ir := tbe.Text[i]
		if len(ir) != nch {
			panic(fmt.Sprintf("insertTextRectImpl: length of rect line: %d, %d != expected from region: %d", i, len(ir), nch))
		}
		if len(lr) < st.Char {
			lr = append(lr, runes.Repeat([]rune{' '}, st.Char-len(lr))...)
		}
		nt := slices.Insert(lr, st.Char, ir...)
		ls.lines[ln] = nt
	}
	re := tbe.Clone()
	re.Rect = true
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
