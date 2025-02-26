// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import (
	"image"
	"regexp"
	"slices"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/text/highlighting"
	"cogentcore.org/core/text/parse/lexer"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/search"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/text/token"
)

// this file contains the exported API for Lines

// NewLines returns a new empty Lines, with no views.
func NewLines() *Lines {
	ls := &Lines{}
	ls.Defaults()
	ls.setText([]byte(""))
	return ls
}

// NewLinesFromBytes returns a new Lines representation of given bytes of text,
// using given filename to determine the type of content that is represented
// in the bytes, based on the filename extension, and given initial display width.
// A width-specific view is created, with the unique view id returned: this id
// must be used for all subsequent view-specific calls.
// This uses all default styling settings.
func NewLinesFromBytes(filename string, width int, src []byte) (*Lines, int) {
	ls := &Lines{}
	ls.Defaults()
	fi, _ := fileinfo.NewFileInfo(filename)
	ls.setFileInfo(fi)
	_, vid := ls.newView(width)
	ls.setText(src)
	return ls, vid
}

func (ls *Lines) Defaults() {
	ls.Settings.Defaults()
	ls.fontStyle = rich.NewStyle().SetFamily(rich.Monospace)
	ls.links = make(map[int][]rich.Hyperlink)
}

// NewView makes a new view with given initial width,
// with a layout of the existing text at this width.
// The return value is a unique int handle that must be
// used for all subsequent calls that depend on the view.
func (ls *Lines) NewView(width int) int {
	ls.Lock()
	defer ls.Unlock()
	_, vid := ls.newView(width)
	return vid
}

// DeleteView deletes view for given unique view id.
// It is important to delete unused views to maintain efficient updating of
// existing views.
func (ls *Lines) DeleteView(vid int) {
	ls.Lock()
	defer ls.Unlock()
	ls.deleteView(vid)
}

// SetWidth sets the width for line wrapping, for given view id.
// If the width is different than current, the layout is updated,
// and a true is returned, else false.
func (ls *Lines) SetWidth(vid int, wd int) bool {
	ls.Lock()
	defer ls.Unlock()
	vw := ls.view(vid)
	if vw != nil {
		if vw.width == wd {
			return false
		}
		vw.width = wd
		ls.layoutViewLines(vw)
		// fmt.Println("set width:", vw.width, "lines:", vw.viewLines, "mu:", len(vw.markup), len(vw.vlineStarts))
		return true
	}
	return false
}

// Width returns the width for line wrapping for given view id.
func (ls *Lines) Width(vid int) int {
	ls.Lock()
	defer ls.Unlock()
	vw := ls.view(vid)
	if vw != nil {
		return vw.width
	}
	return 0
}

// ViewLines returns the total number of line-wrapped view lines, for given view id.
func (ls *Lines) ViewLines(vid int) int {
	ls.Lock()
	defer ls.Unlock()
	vw := ls.view(vid)
	if vw != nil {
		return vw.viewLines
	}
	return 0
}

// SetFontStyle sets the font style to use in styling and rendering text.
// The Family of the font MUST be set to Monospace.
func (ls *Lines) SetFontStyle(fs *rich.Style) *Lines {
	ls.Lock()
	defer ls.Unlock()
	if fs.Family != rich.Monospace {
		errors.Log(errors.New("lines.Lines font style MUST be Monospace. Setting that but should fix upstream"))
		fs.Family = rich.Monospace
	}
	ls.fontStyle = fs
	return ls
}

// FontStyle returns the font style used for this lines.
func (ls *Lines) FontStyle() *rich.Style {
	ls.Lock()
	defer ls.Unlock()
	return ls.fontStyle
}

// SetText sets the text to the given bytes, and does
// full markup update and sends a Change event.
// Pass nil to initialize an empty lines.
func (ls *Lines) SetText(text []byte) *Lines {
	ls.Lock()
	ls.setText(text)
	ls.Unlock()
	ls.sendChange()
	return ls
}

// SetString sets the text to the given string.
func (ls *Lines) SetString(txt string) *Lines {
	return ls.SetText([]byte(txt))
}

// SetTextLines sets the source lines from given lines of bytes.
func (ls *Lines) SetTextLines(lns [][]byte) {
	ls.Lock()
	ls.setLineBytes(lns)
	ls.Unlock()
	ls.sendChange()
}

// Text returns the current text lines as a slice of bytes,
// with an additional line feed at the end, per POSIX standards.
// It does NOT call EditDone or send a Change event: that should
// happen prior or separately from this call.
func (ls *Lines) Text() []byte {
	ls.Lock()
	defer ls.Unlock()
	return ls.bytes(0)
}

// String returns the current text as a string.
// It does NOT call EditDone or send a Change event: that should
// happen prior or separately from this call.
func (ls *Lines) String() string {
	return string(ls.Text())
}

// SetHighlighting sets the highlighting style.
func (ls *Lines) SetHighlighting(style highlighting.HighlightingName) {
	ls.Lock()
	defer ls.Unlock()
	ls.Highlighter.SetStyle(style)
}

// Close should be called when done using the Lines.
// It first sends Close events to all views.
// An Editor widget will likely want to check IsNotSaved()
// and prompt the user to save or cancel first.
func (ls *Lines) Close() {
	ls.sendClose()
	ls.Lock()
	ls.stopDelayedReMarkup()
	ls.views = make(map[int]*view)
	ls.lines = nil
	ls.tags = nil
	ls.hiTags = nil
	ls.markup = nil
	// ls.parseState.Reset() // todo
	ls.undos.Reset()
	ls.markupEdits = nil
	ls.posHistory = nil
	ls.filename = ""
	ls.notSaved = false
	ls.Unlock()
}

// IsChanged reports whether any edits have been applied to text
func (ls *Lines) IsChanged() bool {
	ls.Lock()
	defer ls.Unlock()
	return ls.changed
}

// SetChanged sets the changed flag to given value (e.g., when file saved)
func (ls *Lines) SetChanged(changed bool) {
	ls.Lock()
	defer ls.Unlock()
	ls.changed = changed
}

// SendChange sends an [event.Change] to the views of this lines,
// causing them to update.
func (ls *Lines) SendChange() {
	ls.Lock()
	defer ls.Unlock()
	ls.sendChange()
}

// SendInput sends an [event.Input] to the views of this lines,
// causing them to update.
func (ls *Lines) SendInput() {
	ls.Lock()
	defer ls.Unlock()
	ls.sendInput()
}

// NumLines returns the number of lines.
func (ls *Lines) NumLines() int {
	ls.Lock()
	defer ls.Unlock()
	return ls.numLines()
}

// IsValidLine returns true if given line number is in range.
func (ls *Lines) IsValidLine(ln int) bool {
	if ln < 0 {
		return false
	}
	ls.Lock()
	defer ls.Unlock()
	return ls.isValidLine(ln)
}

// ValidPos returns a position based on given pos that is valid.
func (ls *Lines) ValidPos(pos textpos.Pos) textpos.Pos {
	ls.Lock()
	defer ls.Unlock()

	n := ls.numLines()
	if n == 0 {
		return textpos.Pos{}
	}
	if pos.Line < 0 {
		pos.Line = 0
	}
	if pos.Line >= n {
		pos.Line = n - 1
	}
	llen := len(ls.lines[pos.Line])
	if pos.Char < 0 {
		pos.Char = 0
	}
	if pos.Char > llen {
		pos.Char = llen // end of line is valid
	}
	return pos
}

// Line returns a (copy of) specific line of runes.
func (ls *Lines) Line(ln int) []rune {
	ls.Lock()
	defer ls.Unlock()
	if !ls.isValidLine(ln) {
		return nil
	}
	return slices.Clone(ls.lines[ln])
}

// strings returns the current text as []string array.
// If addNewLine is true, each string line has a \n appended at end.
func (ls *Lines) Strings(addNewLine bool) []string {
	ls.Lock()
	defer ls.Unlock()
	return ls.strings(addNewLine)
}

// LineLen returns the length of the given source line, in runes.
func (ls *Lines) LineLen(ln int) int {
	ls.Lock()
	defer ls.Unlock()
	if !ls.isValidLine(ln) {
		return 0
	}
	return len(ls.lines[ln])
}

// LineChar returns rune at given line and character position.
// returns a 0 if character position is not valid
func (ls *Lines) LineChar(ln, ch int) rune {
	ls.Lock()
	defer ls.Unlock()
	if !ls.isValidLine(ln) {
		return 0
	}
	if len(ls.lines[ln]) <= ch {
		return 0
	}
	return ls.lines[ln][ch]
}

// HiTags returns the highlighting tags for given line, nil if invalid
func (ls *Lines) HiTags(ln int) lexer.Line {
	ls.Lock()
	defer ls.Unlock()
	if !ls.isValidLine(ln) {
		return nil
	}
	return ls.hiTags[ln]
}

// LineLexDepth returns the starting lexical depth in terms of brackets, parens, etc
func (ls *Lines) LineLexDepth(ln int) int {
	ls.Lock()
	defer ls.Unlock()
	n := len(ls.hiTags)
	if ln >= n || len(ls.hiTags[ln]) == 0 {
		return 0
	}
	return ls.hiTags[ln][0].Token.Depth
}

// EndPos returns the ending position at end of lines.
func (ls *Lines) EndPos() textpos.Pos {
	ls.Lock()
	defer ls.Unlock()
	return ls.endPos()
}

// IsValidPos returns an error if the position is not valid.
func (ls *Lines) IsValidPos(pos textpos.Pos) error {
	ls.Lock()
	defer ls.Unlock()
	return ls.isValidPos(pos)
}

// PosToView returns the view position in terms of ViewLines and Char
// offset into that view line for given source line, char position.
func (ls *Lines) PosToView(vid int, pos textpos.Pos) textpos.Pos {
	ls.Lock()
	defer ls.Unlock()
	vw := ls.view(vid)
	return ls.posToView(vw, pos)
}

// PosFromView returns the original source position from given
// view position in terms of ViewLines and Char offset into that view line.
// If the Char position is beyond the end of the line, it returns the
// end of the given line.
func (ls *Lines) PosFromView(vid int, pos textpos.Pos) textpos.Pos {
	ls.Lock()
	defer ls.Unlock()
	vw := ls.view(vid)
	return ls.posFromView(vw, pos)
}

// ViewLineLen returns the length in chars (runes) of the given view line.
func (ls *Lines) ViewLineLen(vid int, vln int) int {
	ls.Lock()
	defer ls.Unlock()
	vw := ls.view(vid)
	return ls.viewLineLen(vw, vln)
}

// ViewLineRegion returns the region in view coordinates of the given view line.
func (ls *Lines) ViewLineRegion(vid int, vln int) textpos.Region {
	ls.Lock()
	defer ls.Unlock()
	vw := ls.view(vid)
	return ls.viewLineRegion(vw, vln)
}

// ViewLineRegionLocked returns the region in view coordinates of the given view line,
// for case where Lines is already locked.
func (ls *Lines) ViewLineRegionLocked(vid int, vln int) textpos.Region {
	vw := ls.view(vid)
	return ls.viewLineRegion(vw, vln)
}

// RegionToView converts the given region in source coordinates into view coordinates.
func (ls *Lines) RegionToView(vid int, reg textpos.Region) textpos.Region {
	ls.Lock()
	defer ls.Unlock()
	vw := ls.view(vid)
	return ls.regionToView(vw, reg)
}

// RegionFromView converts the given region in view coordinates into source coordinates.
func (ls *Lines) RegionFromView(vid int, reg textpos.Region) textpos.Region {
	ls.Lock()
	defer ls.Unlock()
	vw := ls.view(vid)
	return ls.regionFromView(vw, reg)
}

// Region returns a Edit representation of text between start and end positions.
// returns nil if not a valid region.  sets the timestamp on the Edit to now.
func (ls *Lines) Region(st, ed textpos.Pos) *textpos.Edit {
	ls.Lock()
	defer ls.Unlock()
	return ls.region(st, ed)
}

// RegionRect returns a Edit representation of text between
// start and end positions as a rectangle,
// returns nil if not a valid region.  sets the timestamp on the Edit to now.
func (ls *Lines) RegionRect(st, ed textpos.Pos) *textpos.Edit {
	ls.Lock()
	defer ls.Unlock()
	return ls.regionRect(st, ed)
}

// AdjustRegion adjusts given text region for any edits that
// have taken place since time stamp on region (using the Undo stack).
// If region was wholly within a deleted region, then RegionNil will be
// returned, otherwise it is clipped appropriately as function of deletes.
func (ls *Lines) AdjustRegion(reg textpos.Region) textpos.Region {
	ls.Lock()
	defer ls.Unlock()
	return ls.undos.AdjustRegion(reg)
}

////////   Edits

// DeleteText is the primary method for deleting text from the lines.
// It deletes region of text between start and end positions.
// Sets the timestamp on resulting Edit to now.
// An Undo record is automatically saved depending on Undo.Off setting.
// Calls sendInput to send an Input event to views, so they update.
func (ls *Lines) DeleteText(st, ed textpos.Pos) *textpos.Edit {
	ls.Lock()
	ls.fileModCheck()
	tbe := ls.deleteText(st, ed)
	if tbe != nil && ls.Autosave {
		go ls.autoSave()
	}
	ls.Unlock()
	ls.sendInput()
	return tbe
}

// DeleteTextRect deletes rectangular region of text between start, end
// defining the upper-left and lower-right corners of a rectangle.
// Fails if st.Char >= ed.Char. Sets the timestamp on resulting Edit to now.
// An Undo record is automatically saved depending on Undo.Off setting.
// Calls sendInput to send an Input event to views, so they update.
func (ls *Lines) DeleteTextRect(st, ed textpos.Pos) *textpos.Edit {
	ls.Lock()
	ls.fileModCheck()
	tbe := ls.deleteTextRect(st, ed)
	if tbe != nil && ls.Autosave {
		go ls.autoSave()
	}
	ls.Unlock()
	ls.sendInput()
	return tbe
}

// InsertTextBytes is the primary method for inserting text,
// at given starting position. Sets the timestamp on resulting Edit to now.
// An Undo record is automatically saved depending on Undo.Off setting.
// Calls sendInput to send an Input event to views, so they update.
func (ls *Lines) InsertTextBytes(st textpos.Pos, text []byte) *textpos.Edit {
	ls.Lock()
	ls.fileModCheck()
	tbe := ls.insertText(st, []rune(string(text)))
	if tbe != nil && ls.Autosave {
		go ls.autoSave()
	}
	ls.Unlock()
	ls.sendInput()
	return tbe
}

// InsertText is the primary method for inserting text,
// at given starting position. Sets the timestamp on resulting Edit to now.
// An Undo record is automatically saved depending on Undo.Off setting.
// Calls sendInput to send an Input event to views, so they update.
func (ls *Lines) InsertText(st textpos.Pos, text []rune) *textpos.Edit {
	ls.Lock()
	ls.fileModCheck()
	tbe := ls.insertText(st, text)
	if tbe != nil && ls.Autosave {
		go ls.autoSave()
	}
	ls.Unlock()
	ls.sendInput()
	return tbe
}

// InsertTextLines is the primary method for inserting text,
// at given starting position. Sets the timestamp on resulting Edit to now.
// An Undo record is automatically saved depending on Undo.Off setting.
// Calls sendInput to send an Input event to views, so they update.
func (ls *Lines) InsertTextLines(st textpos.Pos, text [][]rune) *textpos.Edit {
	ls.Lock()
	ls.fileModCheck()
	tbe := ls.insertTextImpl(st, text)
	if tbe != nil && ls.Autosave {
		go ls.autoSave()
	}
	ls.Unlock()
	ls.sendInput()
	return tbe
}

// InsertTextRect inserts a rectangle of text defined in given Edit record,
// (e.g., from RegionRect or DeleteRect).
// Returns a copy of the Edit record with an updated timestamp.
// An Undo record is automatically saved depending on Undo.Off setting.
// Calls sendInput to send an Input event to views, so they update.
func (ls *Lines) InsertTextRect(tbe *textpos.Edit) *textpos.Edit {
	ls.Lock()
	ls.fileModCheck()
	tbe = ls.insertTextRect(tbe)
	if tbe != nil && ls.Autosave {
		go ls.autoSave()
	}
	ls.Unlock()
	ls.sendInput()
	return tbe
}

// ReplaceText does DeleteText for given region, and then InsertText at given position
// (typically same as delSt but not necessarily).
// if matchCase is true, then the lexer.MatchCase function is called to match the
// case (upper / lower) of the new inserted text to that of the text being replaced.
// returns the Edit for the inserted text.
// An Undo record is automatically saved depending on Undo.Off setting.
// Calls sendInput to send an Input event to views, so they update.
func (ls *Lines) ReplaceText(delSt, delEd, insPos textpos.Pos, insTxt string, matchCase bool) *textpos.Edit {
	ls.Lock()
	ls.fileModCheck()
	tbe := ls.replaceText(delSt, delEd, insPos, insTxt, matchCase)
	if tbe != nil && ls.Autosave {
		go ls.autoSave()
	}
	ls.Unlock()
	ls.sendInput()
	return tbe
}

// AppendTextMarkup appends new text to end of lines, using insert, returns
// edit, and uses supplied markup to render it, for preformatted output.
// Calls sendInput to send an Input event to views, so they update.
func (ls *Lines) AppendTextMarkup(text [][]rune, markup []rich.Text) *textpos.Edit {
	ls.Lock()
	ls.fileModCheck()
	tbe := ls.appendTextMarkup(text, markup)
	if tbe != nil && ls.Autosave {
		go ls.autoSave()
	}
	ls.collectLinks()
	ls.layoutViews()
	ls.Unlock()
	ls.sendInput()
	return tbe
}

// ReMarkup starts a background task of redoing the markup
func (ls *Lines) ReMarkup() {
	ls.Lock()
	defer ls.Unlock()
	ls.reMarkup()
}

// SetUndoOn turns on or off the recording of undo records for every edit.
func (ls *Lines) SetUndoOn(on bool) {
	ls.Lock()
	defer ls.Unlock()
	ls.undos.Off = !on
}

// NewUndoGroup increments the undo group counter for batchiung
// the subsequent actions.
func (ls *Lines) NewUndoGroup() {
	ls.Lock()
	defer ls.Unlock()
	ls.undos.NewGroup()
}

// UndoReset resets all current undo records.
func (ls *Lines) UndoReset() {
	ls.Lock()
	defer ls.Unlock()
	ls.undos.Reset()
}

// Undo undoes next group of items on the undo stack,
// and returns all the edits performed.
// Calls sendInput to send an Input event to views, so they update.
func (ls *Lines) Undo() []*textpos.Edit {
	ls.Lock()
	autoSave := ls.batchUpdateStart()
	tbe := ls.undo()
	if tbe == nil || ls.undos.Pos == 0 { // no more undo = fully undone
		ls.changed = false
		ls.notSaved = false
		ls.autosaveDelete()
	}
	ls.batchUpdateEnd(autoSave)
	ls.Unlock()
	ls.sendInput()
	return tbe
}

// Redo redoes next group of items on the undo stack,
// and returns all the edits performed.
// Calls sendInput to send an Input event to views, so they update.
func (ls *Lines) Redo() []*textpos.Edit {
	ls.Lock()
	autoSave := ls.batchUpdateStart()
	tbe := ls.redo()
	ls.batchUpdateEnd(autoSave)
	ls.Unlock()
	ls.sendInput()
	return tbe
}

// EmacsUndoSave is called by an editor at end of latest set of undo commands.
// If EmacsUndo mode is active, saves the current UndoStack to the regular Undo stack
// at the end, and moves undo to the very end; undo is a constant stream.
func (ls *Lines) EmacsUndoSave() {
	ls.Lock()
	defer ls.Unlock()
	if !ls.Settings.EmacsUndo {
		return
	}
	ls.undos.UndoStackSave()
}

/////////   Moving

// MoveForward moves given source position forward given number of rune steps.
func (ls *Lines) MoveForward(pos textpos.Pos, steps int) textpos.Pos {
	ls.Lock()
	defer ls.Unlock()
	return ls.moveForward(pos, steps)
}

// MoveBackward moves given source position backward given number of rune steps.
func (ls *Lines) MoveBackward(pos textpos.Pos, steps int) textpos.Pos {
	ls.Lock()
	defer ls.Unlock()
	return ls.moveBackward(pos, steps)
}

// MoveForwardWord moves given source position forward given number of word steps.
func (ls *Lines) MoveForwardWord(pos textpos.Pos, steps int) textpos.Pos {
	ls.Lock()
	defer ls.Unlock()
	return ls.moveForwardWord(pos, steps)
}

// MoveBackwardWord moves given source position backward given number of word steps.
func (ls *Lines) MoveBackwardWord(pos textpos.Pos, steps int) textpos.Pos {
	ls.Lock()
	defer ls.Unlock()
	return ls.moveBackwardWord(pos, steps)
}

// MoveDown moves given source position down given number of display line steps,
// always attempting to use the given column position if the line is long enough.
func (ls *Lines) MoveDown(vid int, pos textpos.Pos, steps, col int) textpos.Pos {
	ls.Lock()
	defer ls.Unlock()
	vw := ls.view(vid)
	return ls.moveDown(vw, pos, steps, col)
}

// MoveUp moves given source position up given number of display line steps,
// always attempting to use the given column position if the line is long enough.
func (ls *Lines) MoveUp(vid int, pos textpos.Pos, steps, col int) textpos.Pos {
	ls.Lock()
	defer ls.Unlock()
	vw := ls.view(vid)
	return ls.moveUp(vw, pos, steps, col)
}

// MoveLineStart moves given source position to start of view line.
func (ls *Lines) MoveLineStart(vid int, pos textpos.Pos) textpos.Pos {
	ls.Lock()
	defer ls.Unlock()
	vw := ls.view(vid)
	return ls.moveLineStart(vw, pos)
}

// MoveLineEnd moves given source position to end of view line.
func (ls *Lines) MoveLineEnd(vid int, pos textpos.Pos) textpos.Pos {
	ls.Lock()
	defer ls.Unlock()
	vw := ls.view(vid)
	return ls.moveLineEnd(vw, pos)
}

////////  Words

// IsWordEnd returns true if the cursor is just past the last letter of a word.
func (ls *Lines) IsWordEnd(pos textpos.Pos) bool {
	ls.Lock()
	defer ls.Unlock()
	if errors.Log(ls.isValidPos(pos)) != nil {
		return false
	}
	txt := ls.lines[pos.Line]
	sz := len(txt)
	if sz == 0 {
		return false
	}
	if pos.Char >= len(txt) { // end of line
		r := txt[len(txt)-1]
		return textpos.IsWordBreak(r, -1)
	}
	if pos.Char == 0 { // start of line
		r := txt[0]
		return !textpos.IsWordBreak(r, -1)
	}
	r1 := txt[pos.Char-1]
	r2 := txt[pos.Char]
	return !textpos.IsWordBreak(r1, rune(-1)) && textpos.IsWordBreak(r2, rune(-1))
}

// IsWordMiddle returns true if the cursor is anywhere inside a word,
// i.e. the character before the cursor and the one after the cursor
// are not classified as word break characters
func (ls *Lines) IsWordMiddle(pos textpos.Pos) bool {
	ls.Lock()
	defer ls.Unlock()
	if errors.Log(ls.isValidPos(pos)) != nil {
		return false
	}
	txt := ls.lines[pos.Line]
	sz := len(txt)
	if sz < 2 {
		return false
	}
	if pos.Char >= len(txt) { // end of line
		return false
	}
	if pos.Char == 0 { // start of line
		return false
	}
	r1 := txt[pos.Char-1]
	r2 := txt[pos.Char]
	return !textpos.IsWordBreak(r1, rune(-1)) && !textpos.IsWordBreak(r2, rune(-1))
}

// WordAt returns a Region for a word starting at given position.
// If the current position is a word break then go to next
// break after the first non-break.
func (ls *Lines) WordAt(pos textpos.Pos) textpos.Region {
	ls.Lock()
	defer ls.Unlock()
	if errors.Log(ls.isValidPos(pos)) != nil {
		return textpos.Region{}
	}
	txt := ls.lines[pos.Line]
	rng := textpos.WordAt(txt, pos.Char)
	st := pos
	st.Char = rng.Start
	ed := pos
	ed.Char = rng.End
	return textpos.NewRegionPos(st, ed)
}

// WordBefore returns the word before the given source position.
// uses IsWordBreak to determine the bounds of the word
func (ls *Lines) WordBefore(pos textpos.Pos) *textpos.Edit {
	ls.Lock()
	defer ls.Unlock()
	if errors.Log(ls.isValidPos(pos)) != nil {
		return &textpos.Edit{}
	}
	txt := ls.lines[pos.Line]
	ch := pos.Char
	ch = min(ch, len(txt))
	st := ch
	for i := ch - 1; i >= 0; i-- {
		if i == 0 { // start of line
			st = 0
			break
		}
		r1 := txt[i]
		r2 := txt[i-1]
		if textpos.IsWordBreak(r1, r2) {
			st = i + 1
			break
		}
	}
	if st != ch {
		return ls.region(textpos.Pos{Line: pos.Line, Char: st}, pos)
	}
	return nil
}

////////  PosHistory

// PosHistorySave saves the cursor position in history stack of cursor positions.
// Tracks across views. Returns false if position was on same line as last one saved.
func (ls *Lines) PosHistorySave(pos textpos.Pos) bool {
	ls.Lock()
	defer ls.Unlock()
	if ls.posHistory == nil {
		ls.posHistory = make([]textpos.Pos, 0, 1000)
	}
	sz := len(ls.posHistory)
	if sz > 0 {
		if ls.posHistory[sz-1].Line == pos.Line {
			return false
		}
	}
	ls.posHistory = append(ls.posHistory, pos)
	// fmt.Printf("saved pos hist: %v\n", pos)
	return true
}

// PosHistoryLen returns the length of the position history stack.
func (ls *Lines) PosHistoryLen() int {
	ls.Lock()
	defer ls.Unlock()
	return len(ls.posHistory)
}

// PosHistoryAt returns the position history at given index.
// returns false if not a valid index.
func (ls *Lines) PosHistoryAt(idx int) (textpos.Pos, bool) {
	ls.Lock()
	defer ls.Unlock()
	if idx < 0 || idx >= len(ls.posHistory) {
		return textpos.Pos{}, false
	}
	return ls.posHistory[idx], true
}

/////////   Edit helpers

// InComment returns true if the given text position is within
// a commented region.
func (ls *Lines) InComment(pos textpos.Pos) bool {
	ls.Lock()
	defer ls.Unlock()
	return ls.inComment(pos)
}

// HiTagAtPos returns the highlighting (markup) lexical tag at given position
// using current Markup tags, and index, -- could be nil if none or out of range.
func (ls *Lines) HiTagAtPos(pos textpos.Pos) (*lexer.Lex, int) {
	ls.Lock()
	defer ls.Unlock()
	return ls.hiTagAtPos(pos)
}

// InTokenSubCat returns true if the given text position is marked with lexical
// type in given SubCat sub-category.
func (ls *Lines) InTokenSubCat(pos textpos.Pos, subCat token.Tokens) bool {
	ls.Lock()
	defer ls.Unlock()
	return ls.inTokenSubCat(pos, subCat)
}

// InLitString returns true if position is in a string literal.
func (ls *Lines) InLitString(pos textpos.Pos) bool {
	ls.Lock()
	defer ls.Unlock()
	return ls.inLitString(pos)
}

// InTokenCode returns true if position is in a Keyword,
// Name, Operator, or Punctuation.
// This is useful for turning off spell checking in docs
func (ls *Lines) InTokenCode(pos textpos.Pos) bool {
	ls.Lock()
	defer ls.Unlock()
	return ls.inTokenCode(pos)
}

// LexObjPathString returns the string at given lex, and including prior
// lex-tagged regions that include sequences of PunctSepPeriod and NameTag
// which are used for object paths -- used for e.g., debugger to pull out
// variable expressions that can be evaluated.
func (ls *Lines) LexObjPathString(ln int, lx *lexer.Lex) string {
	ls.Lock()
	defer ls.Unlock()
	return ls.lexObjPathString(ln, lx)
}

////////   Tags

// AddTag adds a new custom tag for given line, at given position.
func (ls *Lines) AddTag(ln, st, ed int, tag token.Tokens) {
	ls.Lock()
	defer ls.Unlock()
	if !ls.isValidLine(ln) {
		return
	}

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
	ls.Lock()
	defer ls.Unlock()
	if !ls.isValidLine(pos.Line) {
		return
	}

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
	ls.Lock()
	defer ls.Unlock()
	if !ls.isValidLine(ln) {
		return
	}
	ls.tags[ln] = tags
}

// AdjustedTags updates tag positions for edits, for given line
// and returns the new tags
func (ls *Lines) AdjustedTags(ln int) lexer.Line {
	ls.Lock()
	defer ls.Unlock()
	return ls.adjustedTags(ln)
}

// AdjustedTagsLine updates tag positions for edits, for given list of tags,
// associated with given line of text.
func (ls *Lines) AdjustedTagsLine(tags lexer.Line, ln int) lexer.Line {
	ls.Lock()
	defer ls.Unlock()
	return ls.adjustedTagsLine(tags, ln)
}

// MarkupLines generates markup of given range of lines.
// end is *inclusive* line.  Called after edits, under Lock().
// returns true if all lines were marked up successfully.
func (ls *Lines) MarkupLines(st, ed int) bool {
	ls.Lock()
	defer ls.Unlock()
	return ls.markupLines(st, ed)
}

// StartDelayedReMarkup starts a timer for doing markup after an interval.
func (ls *Lines) StartDelayedReMarkup() {
	ls.Lock()
	defer ls.Unlock()
	ls.startDelayedReMarkup()
}

// StopDelayedReMarkup stops the timer for doing markup after an interval.
func (ls *Lines) StopDelayedReMarkup() {
	ls.Lock()
	defer ls.Unlock()
	ls.stopDelayedReMarkup()
}

//////// Misc edit functions

// IndentLine indents line by given number of tab stops, using tabs or spaces,
// for given tab size (if using spaces). Either inserts or deletes to reach target.
// Returns edit record for any change.
// Calls sendInput to send an Input event to views, so they update.
func (ls *Lines) IndentLine(ln, ind int) *textpos.Edit {
	ls.Lock()
	autoSave := ls.batchUpdateStart()
	tbe := ls.indentLine(ln, ind)
	ls.batchUpdateEnd(autoSave)
	ls.Unlock()
	ls.sendInput()
	return tbe
}

// AutoIndent indents given line to the level of the prior line, adjusted
// appropriately if the current line starts with one of the given un-indent
// strings, or the prior line ends with one of the given indent strings.
// Returns any edit that took place (could be nil), along with the auto-indented
// level and character position for the indent of the current line.
// Calls sendInput to send an Input event to views, so they update.
func (ls *Lines) AutoIndent(ln int) (tbe *textpos.Edit, indLev, chPos int) {
	ls.Lock()
	autoSave := ls.batchUpdateStart()
	tbe, indLev, chPos = ls.autoIndent(ln)
	ls.batchUpdateEnd(autoSave)
	ls.Unlock()
	ls.sendInput()
	return
}

// AutoIndentRegion does auto-indent over given region; end is *exclusive*.
// Calls sendInput to send an Input event to views, so they update.
func (ls *Lines) AutoIndentRegion(start, end int) {
	ls.Lock()
	autoSave := ls.batchUpdateStart()
	ls.autoIndentRegion(start, end)
	ls.batchUpdateEnd(autoSave)
	ls.Unlock()
	ls.sendInput()
}

// CommentRegion inserts comment marker on given lines; end is *exclusive*.
// Calls sendInput to send an Input event to views, so they update.
func (ls *Lines) CommentRegion(start, end int) {
	ls.Lock()
	autoSave := ls.batchUpdateStart()
	ls.commentRegion(start, end)
	ls.batchUpdateEnd(autoSave)
	ls.Unlock()
	ls.sendInput()
}

// JoinParaLines merges sequences of lines with hard returns forming paragraphs,
// separated by blank lines, into a single line per paragraph,
// within the given line regions; endLine is *inclusive*.
// Calls sendInput to send an Input event to views, so they update.
func (ls *Lines) JoinParaLines(startLine, endLine int) {
	ls.Lock()
	autoSave := ls.batchUpdateStart()
	ls.joinParaLines(startLine, endLine)
	ls.batchUpdateEnd(autoSave)
	ls.Unlock()
	ls.sendInput()
}

// TabsToSpaces replaces tabs with spaces over given region; end is *exclusive*.
// Calls sendInput to send an Input event to views, so they update.
func (ls *Lines) TabsToSpaces(start, end int) {
	ls.Lock()
	autoSave := ls.batchUpdateStart()
	ls.tabsToSpaces(start, end)
	ls.batchUpdateEnd(autoSave)
	ls.Unlock()
	ls.sendInput()
}

// SpacesToTabs replaces tabs with spaces over given region; end is *exclusive*
// Calls sendInput to send an Input event to views, so they update.
func (ls *Lines) SpacesToTabs(start, end int) {
	ls.Lock()
	autoSave := ls.batchUpdateStart()
	ls.spacesToTabs(start, end)
	ls.batchUpdateEnd(autoSave)
	ls.Unlock()
	ls.sendInput()
}

// CountWordsLinesRegion returns the count of words and lines in given region.
func (ls *Lines) CountWordsLinesRegion(reg textpos.Region) (words, lines int) {
	ls.Lock()
	defer ls.Unlock()
	words, lines = CountWordsLinesRegion(ls.lines, reg)
	return
}

// Diffs computes the diff between this lines and the other lines,
// reporting a sequence of operations that would convert this lines (a) into
// the other lines (b).  Each operation is either an 'r' (replace), 'd'
// (delete), 'i' (insert) or 'e' (equal).  Everything is line-based (0, offset).
func (ls *Lines) Diffs(ob *Lines) Diffs {
	ls.Lock()
	defer ls.Unlock()
	return ls.diffs(ob)
}

// PatchFrom patches (edits) using content from other,
// according to diff operations (e.g., as generated from Diffs).
func (ls *Lines) PatchFrom(ob *Lines, diffs Diffs) bool {
	ls.Lock()
	defer ls.Unlock()
	return ls.patchFrom(ob, diffs)
}

// DiffsUnified computes the diff between this lines and the other lines,
// returning a unified diff with given amount of context (default of 3 will be
// used if -1)
func (ls *Lines) DiffsUnified(ob *Lines, context int) []byte {
	astr := ls.Strings(true) // needs newlines for some reason
	bstr := ob.Strings(true)

	return DiffLinesUnified(astr, bstr, context, ls.Filename(), ls.FileInfo().ModTime.String(),
		ob.Filename(), ob.FileInfo().ModTime.String())
}

////////   Search etc

// Search looks for a string (no regexp) within buffer,
// with given case-sensitivity, returning number of occurrences
// and specific match position list. Column positions are in runes.
func (ls *Lines) Search(find []byte, ignoreCase, lexItems bool) (int, []textpos.Match) {
	ls.Lock()
	defer ls.Unlock()
	if lexItems {
		return search.LexItems(ls.lines, ls.hiTags, find, ignoreCase)
	}
	return search.RuneLines(ls.lines, find, ignoreCase)
}

// SearchRegexp looks for a string (regexp) within buffer,
// returning number of occurrences and specific match position list.
// Column positions are in runes.
func (ls *Lines) SearchRegexp(re *regexp.Regexp) (int, []textpos.Match) {
	ls.Lock()
	defer ls.Unlock()
	return search.RuneLinesRegexp(ls.lines, re)
}

// BraceMatch finds the brace, bracket, or parens that is the partner
// of the one at the given position, if there is one of those at this position.
func (ls *Lines) BraceMatch(pos textpos.Pos) (textpos.Pos, bool) {
	ls.Lock()
	defer ls.Unlock()
	return ls.braceMatch(pos)
}

// BraceMatchRune finds the brace, bracket, or parens that is the partner
// of the given rune, starting at given position.
func (ls *Lines) BraceMatchRune(r rune, pos textpos.Pos) (textpos.Pos, bool) {
	ls.Lock()
	defer ls.Unlock()
	return lexer.BraceMatch(ls.lines, ls.hiTags, r, pos, maxScopeLines)
}

// LinkAt returns a hyperlink at given source position, if one exists,
// nil otherwise. this is fast so no problem to call frequently.
func (ls *Lines) LinkAt(pos textpos.Pos) *rich.Hyperlink {
	ls.Lock()
	defer ls.Unlock()
	return ls.linkAt(pos)
}

// NextLink returns the next hyperlink after given source position,
// if one exists, and the line it is on. nil, -1 otherwise.
func (ls *Lines) NextLink(pos textpos.Pos) (*rich.Hyperlink, int) {
	ls.Lock()
	defer ls.Unlock()
	return ls.nextLink(pos)
}

// PrevLink returns the previous hyperlink before given source position,
// if one exists, and the line it is on. nil, -1 otherwise.
func (ls *Lines) PrevLink(pos textpos.Pos) (*rich.Hyperlink, int) {
	ls.Lock()
	defer ls.Unlock()
	return ls.prevLink(pos)
}

// Links returns the full list of hyperlinks
func (ls *Lines) Links() map[int][]rich.Hyperlink {
	ls.Lock()
	defer ls.Unlock()
	return ls.links
}

////////   LineColors

// SetLineColor sets the color to use for rendering a circle next to the line
// number at the given line.
func (ls *Lines) SetLineColor(ln int, color image.Image) {
	ls.Lock()
	defer ls.Unlock()
	if ls.lineColors == nil {
		ls.lineColors = make(map[int]image.Image)
	}
	ls.lineColors[ln] = color
}

// LineColor returns the line color for given line, and bool indicating if set.
func (ls *Lines) LineColor(ln int) (image.Image, bool) {
	ls.Lock()
	defer ls.Unlock()
	if ln < 0 {
		return nil, false
	}
	if ls.lineColors == nil {
		return nil, false
	}
	clr, has := ls.lineColors[ln]
	return clr, has
}

// DeleteLineColor deletes the line color at the given line.
// Passing a -1 clears all current line colors.
func (ls *Lines) DeleteLineColor(ln int) {
	ls.Lock()
	defer ls.Unlock()

	if ln < 0 {
		ls.lineColors = nil
		return
	}
	if ls.lineColors == nil {
		return
	}
	delete(ls.lineColors, ln)
}
