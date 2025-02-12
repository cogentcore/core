// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import (
	"regexp"
	"slices"
	"strings"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/core"
	"cogentcore.org/core/text/parse/lexer"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/text/token"
)

// this file contains the exported API for lines

// SetText sets the text to the given bytes (makes a copy).
// Pass nil to initialize an empty buffer.
func (ls *Lines) SetText(text []byte) {
	ls.Lock()
	defer ls.Unlock()

	ls.bytesToLines(text)
}

// SetTextLines sets the source lines from given lines of bytes.
func (ls *Lines) SetTextLines(lns [][]byte) {
	ls.Lock()
	defer ls.Unlock()

	ls.setLineBytes(lns)
}

// Bytes returns the current text lines as a slice of bytes,
// with an additional line feed at the end, per POSIX standards.
func (ls *Lines) Bytes() []byte {
	ls.Lock()
	defer ls.Unlock()
	return ls.bytes(0)
}

// SetFileInfo sets the syntax highlighting and other parameters
// based on the type of file specified by given fileinfo.FileInfo.
func (ls *Lines) SetFileInfo(info *fileinfo.FileInfo) {
	ls.Lock()
	defer ls.Unlock()

	ls.parseState.SetSrc(string(info.Path), "", info.Known)
	ls.Highlighter.Init(info, &ls.parseState)
	ls.Options.ConfigKnown(info.Known)
	if ls.numLines() > 0 {
		ls.initialMarkup()
		ls.startDelayedReMarkup()
	}
}

// SetFileType sets the syntax highlighting and other parameters
// based on the given fileinfo.Known file type
func (ls *Lines) SetLanguage(ftyp fileinfo.Known) {
	ls.SetFileInfo(fileinfo.NewFileInfoType(ftyp))
}

// SetFileExt sets syntax highlighting and other parameters
// based on the given file extension (without the . prefix),
// for cases where an actual file with [fileinfo.FileInfo] is not
// available.
func (ls *Lines) SetFileExt(ext string) {
	if len(ext) == 0 {
		return
	}
	if ext[0] == '.' {
		ext = ext[1:]
	}
	fn := "_fake." + strings.ToLower(ext)
	fi, _ := fileinfo.NewFileInfo(fn)
	ls.SetFileInfo(fi)
}

// SetHighlighting sets the highlighting style.
func (ls *Lines) SetHighlighting(style core.HighlightingName) {
	ls.Lock()
	defer ls.Unlock()
	ls.Highlighter.SetStyle(style)
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

// LineLen returns the length of the given line, in runes.
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

////////   Edits

// DeleteText is the primary method for deleting text from the lines.
// It deletes region of text between start and end positions.
// Sets the timestamp on resulting Edit to now.
// An Undo record is automatically saved depending on Undo.Off setting.
func (ls *Lines) DeleteText(st, ed textpos.Pos) *textpos.Edit {
	ls.Lock()
	defer ls.Unlock()
	return ls.deleteText(st, ed)
}

// DeleteTextRect deletes rectangular region of text between start, end
// defining the upper-left and lower-right corners of a rectangle.
// Fails if st.Char >= ed.Char. Sets the timestamp on resulting Edit to now.
// An Undo record is automatically saved depending on Undo.Off setting.
func (ls *Lines) DeleteTextRect(st, ed textpos.Pos) *textpos.Edit {
	ls.Lock()
	defer ls.Unlock()
	return ls.deleteTextRect(st, ed)
}

// InsertTextBytes is the primary method for inserting text,
// at given starting position. Sets the timestamp on resulting Edit to now.
// An Undo record is automatically saved depending on Undo.Off setting.
func (ls *Lines) InsertTextBytes(st textpos.Pos, text []byte) *textpos.Edit {
	ls.Lock()
	defer ls.Unlock()
	return ls.insertText(st, []rune(string(text)))
}

// InsertText is the primary method for inserting text,
// at given starting position. Sets the timestamp on resulting Edit to now.
// An Undo record is automatically saved depending on Undo.Off setting.
func (ls *Lines) InsertText(st textpos.Pos, text []rune) *textpos.Edit {
	ls.Lock()
	defer ls.Unlock()
	return ls.insertText(st, text)
}

// InsertTextRect inserts a rectangle of text defined in given Edit record,
// (e.g., from RegionRect or DeleteRect).
// Returns a copy of the Edit record with an updated timestamp.
// An Undo record is automatically saved depending on Undo.Off setting.
func (ls *Lines) InsertTextRect(tbe *textpos.Edit) *textpos.Edit {
	ls.Lock()
	defer ls.Unlock()
	return ls.insertTextRect(tbe)
}

// ReplaceText does DeleteText for given region, and then InsertText at given position
// (typically same as delSt but not necessarily).
// if matchCase is true, then the lexer.MatchCase function is called to match the
// case (upper / lower) of the new inserted text to that of the text being replaced.
// returns the Edit for the inserted text.
// An Undo record is automatically saved depending on Undo.Off setting.
func (ls *Lines) ReplaceText(delSt, delEd, insPos textpos.Pos, insTxt string, matchCase bool) *textpos.Edit {
	ls.Lock()
	defer ls.Unlock()
	return ls.replaceText(delSt, delEd, insPos, insTxt, matchCase)
}

// AppendTextMarkup appends new text to end of lines, using insert, returns
// edit, and uses supplied markup to render it, for preformatted output.
func (ls *Lines) AppendTextMarkup(text []rune, markup []rich.Text) *textpos.Edit {
	ls.Lock()
	defer ls.Unlock()
	return ls.appendTextMarkup(text, markup)
}

// ReMarkup starts a background task of redoing the markup
func (ls *Lines) ReMarkup() {
	ls.Lock()
	defer ls.Unlock()
	ls.reMarkup()
}

// Undo undoes next group of items on the undo stack,
// and returns all the edits performed.
func (ls *Lines) Undo() []*textpos.Edit {
	ls.Lock()
	defer ls.Unlock()
	return ls.undo()
}

// Redo redoes next group of items on the undo stack,
// and returns all the edits performed.
func (ls *Lines) Redo() []*textpos.Edit {
	ls.Lock()
	defer ls.Unlock()
	return ls.redo()
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

// IndentLine indents line by given number of tab stops, using tabs or spaces,
// for given tab size (if using spaces) -- either inserts or deletes to reach target.
// Returns edit record for any change.
func (ls *Lines) IndentLine(ln, ind int) *textpos.Edit {
	ls.Lock()
	defer ls.Unlock()
	return ls.indentLine(ln, ind)
}

// autoIndent indents given line to the level of the prior line, adjusted
// appropriately if the current line starts with one of the given un-indent
// strings, or the prior line ends with one of the given indent strings.
// Returns any edit that took place (could be nil), along with the auto-indented
// level and character position for the indent of the current line.
func (ls *Lines) AutoIndent(ln int) (tbe *textpos.Edit, indLev, chPos int) {
	ls.Lock()
	defer ls.Unlock()
	return ls.autoIndent(ln)
}

// AutoIndentRegion does auto-indent over given region; end is *exclusive*.
func (ls *Lines) AutoIndentRegion(start, end int) {
	ls.Lock()
	defer ls.Unlock()
	ls.autoIndentRegion(start, end)
}

// CommentRegion inserts comment marker on given lines; end is *exclusive*.
func (ls *Lines) CommentRegion(start, end int) {
	ls.Lock()
	defer ls.Unlock()
	ls.commentRegion(start, end)
}

// JoinParaLines merges sequences of lines with hard returns forming paragraphs,
// separated by blank lines, into a single line per paragraph,
// within the given line regions; endLine is *inclusive*.
func (ls *Lines) JoinParaLines(startLine, endLine int) {
	ls.Lock()
	defer ls.Unlock()
	ls.joinParaLines(startLine, endLine)
}

// TabsToSpaces replaces tabs with spaces over given region; end is *exclusive*.
func (ls *Lines) TabsToSpaces(start, end int) {
	ls.Lock()
	defer ls.Unlock()
	ls.tabsToSpaces(start, end)
}

// SpacesToTabs replaces tabs with spaces over given region; end is *exclusive*
func (ls *Lines) SpacesToTabs(start, end int) {
	ls.Lock()
	defer ls.Unlock()
	ls.spacesToTabs(start, end)
}

func (ls *Lines) CountWordsLinesRegion(reg textpos.Region) (words, lines int) {
	ls.Lock()
	defer ls.Unlock()
	words, lines = CountWordsLinesRegion(ls.lines, reg)
	return
}

////////   Search etc

// Search looks for a string (no regexp) within buffer,
// with given case-sensitivity, returning number of occurrences
// and specific match position list. Column positions are in runes.
func (ls *Lines) Search(find []byte, ignoreCase, lexItems bool) (int, []textpos.Match) {
	ls.Lock()
	defer ls.Unlock()
	if lexItems {
		return SearchLexItems(ls.lines, ls.hiTags, find, ignoreCase)
	}
	return SearchRuneLines(ls.lines, find, ignoreCase)
}

// SearchRegexp looks for a string (regexp) within buffer,
// returning number of occurrences and specific match position list.
// Column positions are in runes.
func (ls *Lines) SearchRegexp(re *regexp.Regexp) (int, []textpos.Match) {
	ls.Lock()
	defer ls.Unlock()
	return SearchRuneLinesRegexp(ls.lines, re)
}

// BraceMatch finds the brace, bracket, or parens that is the partner
// of the one passed to function.
func (ls *Lines) BraceMatch(r rune, st textpos.Pos) (en textpos.Pos, found bool) {
	ls.Lock()
	defer ls.Unlock()
	return lexer.BraceMatch(ls.lines, ls.hiTags, r, st, maxScopeLines)
}
