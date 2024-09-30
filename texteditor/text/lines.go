// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package text

import (
	"bytes"
	"log"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/indent"
	"cogentcore.org/core/base/runes"
	"cogentcore.org/core/base/slicesx"
	"cogentcore.org/core/base/stringsx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/parse"
	"cogentcore.org/core/parse/lexer"
	"cogentcore.org/core/parse/token"
	"cogentcore.org/core/texteditor/highlighting"
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

// Lines manages multi-line text, with original source text encoded as bytes
// and runes, and a corresponding markup representation with syntax highlighting
// and other HTML-encoded text markup on top of the raw text.
// The markup is updated in a separate goroutine for efficiency.
// Everything is protected by an overall sync.Mutex and safe to concurrent access,
// and thus nothing is exported and all access is through protected accessor functions.
// In general, all unexported methods do NOT lock, and all exported methods do.
type Lines struct {
	// Options are the options for how text editing and viewing works.
	Options Options

	// Highlighter does the syntax highlighting markup, and contains the
	// parameters thereof, such as the language and style.
	Highlighter highlighting.Highlighter

	// Undos is the undo manager.
	Undos Undo

	// Markup is the marked-up version of the edited text lines, after being run
	// through the syntax highlighting process. This is what is actually rendered.
	// You MUST access it only under a Lock()!
	Markup [][]byte

	// ParseState is the parsing state information for the file.
	ParseState parse.FileStates

	// ChangedFunc is called whenever the text content is changed.
	// The changed flag is always updated on changes, but this can be
	// used for other flags or events that need to be tracked. The
	// Lock is off when this is called.
	ChangedFunc func()

	// MarkupDoneFunc is called when the offline markup pass is done
	// so that the GUI can be updated accordingly.  The lock is off
	// when this is called.
	MarkupDoneFunc func()

	// changed indicates whether any changes have been made.
	// Use [IsChanged] method to access.
	changed bool

	// lineBytes are the live lines of text being edited,
	// with the latest modifications, continuously updated
	// back-and-forth with the lines runes.
	lineBytes [][]byte

	// Lines are the live lines of text being edited, with the latest modifications.
	// They are encoded as runes per line, which is necessary for one-to-one rune/glyph
	// rendering correspondence. All TextPos positions are in rune indexes, not byte
	// indexes.
	lines [][]rune

	// tags are the extra custom tagged regions for each line.
	tags []lexer.Line

	// hiTags are the syntax highlighting tags, which are auto-generated.
	hiTags []lexer.Line

	// markupEdits are the edits that were made during the time it takes to generate
	// the new markup tags -- rare but it does happen.
	markupEdits []*Edit

	// markupDelayTimer is the markup delay timer.
	markupDelayTimer *time.Timer

	// markupDelayMu is the mutex for updating the markup delay timer.
	markupDelayMu sync.Mutex

	// use Lock(), Unlock() directly for overall mutex on any content updates
	sync.Mutex
}

// SetText sets the text to the given bytes (makes a copy).
// Pass nil to initialize an empty buffer.
func (ls *Lines) SetText(text []byte) {
	ls.Lock()
	defer ls.Unlock()

	ls.bytesToLines(text)
	ls.initFromLineBytes()
}

// SetTextLines sets linesBytes from given lines of bytes, making a copy
// and removing any trailing \r carriage returns, to standardize.
func (ls *Lines) SetTextLines(lns [][]byte) {
	ls.Lock()
	defer ls.Unlock()

	ls.setLineBytes(lns)
	ls.initFromLineBytes()
}

// Bytes returns the current text lines as a slice of bytes,
// with an additional line feed at the end, per POSIX standards.
func (ls *Lines) Bytes() []byte {
	ls.Lock()
	defer ls.Unlock()
	return ls.bytes()
}

// SetFileInfo sets the syntax highlighting and other parameters
// based on the type of file specified by given fileinfo.FileInfo.
func (ls *Lines) SetFileInfo(info *fileinfo.FileInfo) {
	ls.Lock()
	defer ls.Unlock()

	ls.ParseState.SetSrc(string(info.Path), "", info.Known)
	ls.Highlighter.Init(info, &ls.ParseState)
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
	return ls.isValidLine(ln)
}

// Line returns a (copy of) specific line of runes.
func (ls *Lines) Line(ln int) []rune {
	if !ls.IsValidLine(ln) {
		return nil
	}
	ls.Lock()
	defer ls.Unlock()
	return slices.Clone(ls.lines[ln])
}

// LineBytes returns a (copy of) specific line of bytes.
func (ls *Lines) LineBytes(ln int) []byte {
	if !ls.IsValidLine(ln) {
		return nil
	}
	ls.Lock()
	defer ls.Unlock()
	return slices.Clone(ls.lineBytes[ln])
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
	if !ls.IsValidLine(ln) {
		return 0
	}
	ls.Lock()
	defer ls.Unlock()
	return len(ls.lines[ln])
}

// LineChar returns rune at given line and character position.
// returns a 0 if character position is not valid
func (ls *Lines) LineChar(ln, ch int) rune {
	if !ls.IsValidLine(ln) {
		return 0
	}
	ls.Lock()
	defer ls.Unlock()
	if len(ls.lines[ln]) <= ch {
		return 0
	}
	return ls.lines[ln][ch]
}

// HiTags returns the highlighting tags for given line, nil if invalid
func (ls *Lines) HiTags(ln int) lexer.Line {
	if !ls.IsValidLine(ln) {
		return nil
	}
	ls.Lock()
	defer ls.Unlock()
	return ls.hiTags[ln]
}

// EndPos returns the ending position at end of lines.
func (ls *Lines) EndPos() lexer.Pos {
	ls.Lock()
	defer ls.Unlock()
	return ls.endPos()
}

// ValidPos returns a position that is in a valid range.
func (ls *Lines) ValidPos(pos lexer.Pos) lexer.Pos {
	ls.Lock()
	defer ls.Unlock()
	return ls.validPos(pos)
}

// Region returns a Edit representation of text between start and end positions.
// returns nil if not a valid region.  sets the timestamp on the Edit to now.
func (ls *Lines) Region(st, ed lexer.Pos) *Edit {
	ls.Lock()
	defer ls.Unlock()
	return ls.region(st, ed)
}

// RegionRect returns a Edit representation of text between
// start and end positions as a rectangle,
// returns nil if not a valid region.  sets the timestamp on the Edit to now.
func (ls *Lines) RegionRect(st, ed lexer.Pos) *Edit {
	ls.Lock()
	defer ls.Unlock()
	return ls.regionRect(st, ed)
}

/////////////////////////////////////////////////////////////////////////////
//   Edits

// DeleteText is the primary method for deleting text from the lines.
// It deletes region of text between start and end positions.
// Sets the timestamp on resulting Edit to now.
// An Undo record is automatically saved depending on Undo.Off setting.
func (ls *Lines) DeleteText(st, ed lexer.Pos) *Edit {
	ls.Lock()
	defer ls.Unlock()
	return ls.deleteText(st, ed)
}

// DeleteTextRect deletes rectangular region of text between start, end
// defining the upper-left and lower-right corners of a rectangle.
// Fails if st.Ch >= ed.Ch. Sets the timestamp on resulting Edit to now.
// An Undo record is automatically saved depending on Undo.Off setting.
func (ls *Lines) DeleteTextRect(st, ed lexer.Pos) *Edit {
	ls.Lock()
	defer ls.Unlock()
	return ls.deleteTextRect(st, ed)
}

// InsertText is the primary method for inserting text,
// at given starting position.  Sets the timestamp on resulting Edit to now.
// An Undo record is automatically saved depending on Undo.Off setting.
func (ls *Lines) InsertText(st lexer.Pos, text []byte) *Edit {
	ls.Lock()
	defer ls.Unlock()
	return ls.insertText(st, text)
}

// InsertTextRect inserts a rectangle of text defined in given Edit record,
// (e.g., from RegionRect or DeleteRect).
// Returns a copy of the Edit record with an updated timestamp.
// An Undo record is automatically saved depending on Undo.Off setting.
func (ls *Lines) InsertTextRect(tbe *Edit) *Edit {
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
func (ls *Lines) ReplaceText(delSt, delEd, insPos lexer.Pos, insTxt string, matchCase bool) *Edit {
	ls.Lock()
	defer ls.Unlock()
	return ls.replaceText(delSt, delEd, insPos, insTxt, matchCase)
}

// AppendTextMarkup appends new text to end of lines, using insert, returns
// edit, and uses supplied markup to render it, for preformatted output.
func (ls *Lines) AppendTextMarkup(text []byte, markup []byte) *Edit {
	ls.Lock()
	defer ls.Unlock()
	return ls.appendTextMarkup(text, markup)
}

// AppendTextLineMarkup appends one line of new text to end of lines, using
// insert, and appending a LF at the end of the line if it doesn't already
// have one. User-supplied markup is used. Returns the edit region.
func (ls *Lines) AppendTextLineMarkup(text []byte, markup []byte) *Edit {
	ls.Lock()
	defer ls.Unlock()
	return ls.appendTextLineMarkup(text, markup)
}

// ReMarkup starts a background task of redoing the markup
func (ls *Lines) ReMarkup() {
	ls.Lock()
	defer ls.Unlock()
	ls.reMarkup()
}

// Undo undoes next group of items on the undo stack,
// and returns all the edits performed.
func (ls *Lines) Undo() []*Edit {
	ls.Lock()
	defer ls.Unlock()
	return ls.undo()
}

// Redo redoes next group of items on the undo stack,
// and returns all the edits performed.
func (ls *Lines) Redo() []*Edit {
	ls.Lock()
	defer ls.Unlock()
	return ls.redo()
}

/////////////////////////////////////////////////////////////////////////////
//   Edit helpers

// InComment returns true if the given text position is within
// a commented region.
func (ls *Lines) InComment(pos lexer.Pos) bool {
	ls.Lock()
	defer ls.Unlock()
	return ls.inComment(pos)
}

// HiTagAtPos returns the highlighting (markup) lexical tag at given position
// using current Markup tags, and index, -- could be nil if none or out of range.
func (ls *Lines) HiTagAtPos(pos lexer.Pos) (*lexer.Lex, int) {
	ls.Lock()
	defer ls.Unlock()
	return ls.hiTagAtPos(pos)
}

// InTokenSubCat returns true if the given text position is marked with lexical
// type in given SubCat sub-category.
func (ls *Lines) InTokenSubCat(pos lexer.Pos, subCat token.Tokens) bool {
	ls.Lock()
	defer ls.Unlock()
	return ls.inTokenSubCat(pos, subCat)
}

// InLitString returns true if position is in a string literal.
func (ls *Lines) InLitString(pos lexer.Pos) bool {
	ls.Lock()
	defer ls.Unlock()
	return ls.inLitString(pos)
}

// InTokenCode returns true if position is in a Keyword,
// Name, Operator, or Punctuation.
// This is useful for turning off spell checking in docs
func (ls *Lines) InTokenCode(pos lexer.Pos) bool {
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
func (ls *Lines) IndentLine(ln, ind int) *Edit {
	ls.Lock()
	defer ls.Unlock()
	return ls.indentLine(ln, ind)
}

// autoIndent indents given line to the level of the prior line, adjusted
// appropriately if the current line starts with one of the given un-indent
// strings, or the prior line ends with one of the given indent strings.
// Returns any edit that took place (could be nil), along with the auto-indented
// level and character position for the indent of the current line.
func (ls *Lines) AutoIndent(ln int) (tbe *Edit, indLev, chPos int) {
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

func (ls *Lines) CountWordsLinesRegion(reg Region) (words, lines int) {
	ls.Lock()
	defer ls.Unlock()
	words, lines = CountWordsLinesRegion(ls.lines, reg)
	return
}

/////////////////////////////////////////////////////////////////////////////
//   Search etc

// Search looks for a string (no regexp) within buffer,
// with given case-sensitivity, returning number of occurrences
// and specific match position list. Column positions are in runes.
func (ls *Lines) Search(find []byte, ignoreCase, lexItems bool) (int, []Match) {
	ls.Lock()
	defer ls.Unlock()
	if lexItems {
		return SearchLexItems(ls.lines, ls.hiTags, find, ignoreCase)
	} else {
		return SearchRuneLines(ls.lines, find, ignoreCase)
	}
}

// SearchRegexp looks for a string (regexp) within buffer,
// returning number of occurrences and specific match position list.
// Column positions are in runes.
func (ls *Lines) SearchRegexp(re *regexp.Regexp) (int, []Match) {
	ls.Lock()
	defer ls.Unlock()
	return SearchByteLinesRegexp(ls.lineBytes, re)
}

// BraceMatch finds the brace, bracket, or parens that is the partner
// of the one passed to function.
func (ls *Lines) BraceMatch(r rune, st lexer.Pos) (en lexer.Pos, found bool) {
	ls.Lock()
	defer ls.Unlock()
	return lexer.BraceMatch(ls.lines, ls.hiTags, r, st, maxScopeLines)
}

//////////////////////////////////////////////////////////////////////
//   Impl below

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

// bytesToLines sets the lineBytes from source .text,
// making a copy of the bytes so they don't refer back to text,
// and removing any trailing \r carriage returns, to standardize.
func (ls *Lines) bytesToLines(txt []byte) {
	if txt == nil {
		txt = []byte("")
	}
	ls.setLineBytes(bytes.Split(txt, []byte("\n")))
}

// setLineBytes sets the lineBytes from source [][]byte, making copies,
// and removing any trailing \r carriage returns, to standardize.
// also removes any trailing blank line if line ended with \n
func (ls *Lines) setLineBytes(lns [][]byte) {
	n := len(lns)
	ls.lineBytes = slicesx.SetLength(ls.lineBytes, n)
	for i, l := range lns {
		ls.lineBytes[i] = slicesx.CopyFrom(ls.lineBytes[i], stringsx.ByteTrimCR(l))
	}
	if n > 1 && len(ls.lineBytes[n-1]) == 0 { // lines have lf at end typically
		ls.lineBytes = ls.lineBytes[:n-1]
	}
}

// initFromLineBytes initializes everything from lineBytes
func (ls *Lines) initFromLineBytes() {
	n := len(ls.lineBytes)
	ls.lines = slicesx.SetLength(ls.lines, n)
	ls.tags = slicesx.SetLength(ls.tags, n)
	ls.hiTags = slicesx.SetLength(ls.hiTags, n)
	ls.Markup = slicesx.SetLength(ls.Markup, n)
	for ln, txt := range ls.lineBytes {
		ls.lines[ln] = runes.SetFromBytes(ls.lines[ln], txt)
		ls.Markup[ln] = highlighting.HtmlEscapeRunes(ls.lines[ln])
	}
	ls.initialMarkup()
	ls.startDelayedReMarkup()
}

// bytes returns the current text lines as a slice of bytes.
// with an additional line feed at the end, per POSIX standards.
func (ls *Lines) bytes() []byte {
	txt := bytes.Join(ls.lineBytes, []byte("\n"))
	// https://stackoverflow.com/questions/729692/why-should-text-files-end-with-a-newline
	txt = append(txt, []byte("\n")...)
	return txt
}

// lineOffsets returns the index offsets for the start of each line
// within an overall slice of bytes (e.g., from bytes).
func (ls *Lines) lineOffsets() []int {
	n := len(ls.lineBytes)
	of := make([]int, n)
	bo := 0
	for ln, txt := range ls.lineBytes {
		of[ln] = bo
		bo += len(txt) + 1 // lf
	}
	return of
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

/////////////////////////////////////////////////////////////////////////////
//   Appending Lines

// endPos returns the ending position at end of lines
func (ls *Lines) endPos() lexer.Pos {
	n := ls.numLines()
	if n == 0 {
		return lexer.PosZero
	}
	return lexer.Pos{n - 1, len(ls.lines[n-1])}
}

// appendTextMarkup appends new text to end of lines, using insert, returns
// edit, and uses supplied markup to render it.
func (ls *Lines) appendTextMarkup(text []byte, markup []byte) *Edit {
	if len(text) == 0 {
		return &Edit{}
	}
	ed := ls.endPos()
	tbe := ls.insertText(ed, text)

	st := tbe.Reg.Start.Ln
	el := tbe.Reg.End.Ln
	sz := (el - st) + 1
	msplt := bytes.Split(markup, []byte("\n"))
	if len(msplt) < sz {
		log.Printf("Buf AppendTextMarkup: markup text less than appended text: is: %v, should be: %v\n", len(msplt), sz)
		el = min(st+len(msplt)-1, el)
	}
	for ln := st; ln <= el; ln++ {
		ls.Markup[ln] = msplt[ln-st]
	}
	return tbe
}

// appendTextLineMarkup appends one line of new text to end of lines, using
// insert, and appending a LF at the end of the line if it doesn't already
// have one. User-supplied markup is used. Returns the edit region.
func (ls *Lines) appendTextLineMarkup(text []byte, markup []byte) *Edit {
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
		efft = make([]byte, sz+1)
		copy(efft, text)
		efft[sz] = '\n'
	}
	tbe := ls.insertText(ed, efft)
	ls.Markup[tbe.Reg.Start.Ln] = markup
	return tbe
}

/////////////////////////////////////////////////////////////////////////////
//   Edits

// validPos returns a position that is in a valid range
func (ls *Lines) validPos(pos lexer.Pos) lexer.Pos {
	n := ls.numLines()
	if n == 0 {
		return lexer.PosZero
	}
	if pos.Ln < 0 {
		pos.Ln = 0
	}
	if pos.Ln >= n {
		pos.Ln = n - 1
		pos.Ch = len(ls.lines[pos.Ln])
		return pos
	}
	pos.Ln = min(pos.Ln, n-1)
	llen := len(ls.lines[pos.Ln])
	pos.Ch = min(pos.Ch, llen)
	if pos.Ch < 0 {
		pos.Ch = 0
	}
	return pos
}

// region returns a Edit representation of text between start and end positions
// returns nil if not a valid region.  sets the timestamp on the Edit to now
func (ls *Lines) region(st, ed lexer.Pos) *Edit {
	st = ls.validPos(st)
	ed = ls.validPos(ed)
	n := ls.numLines()
	// not here:
	// if ed.Ln >= n {
	// 	fmt.Println("region err in range:", ed.Ln, len(ls.lines), ed.Ch)
	// }
	if st == ed {
		return nil
	}
	if !st.IsLess(ed) {
		log.Printf("text.region: starting position must be less than ending!: st: %v, ed: %v\n", st, ed)
		return nil
	}
	tbe := &Edit{Reg: NewRegionPos(st, ed)}
	if ed.Ln == st.Ln {
		sz := ed.Ch - st.Ch
		if sz <= 0 {
			return nil
		}
		tbe.Text = make([][]rune, 1)
		tbe.Text[0] = make([]rune, sz)
		copy(tbe.Text[0][:sz], ls.lines[st.Ln][st.Ch:ed.Ch])
	} else {
		// first get chars on start and end
		if ed.Ln >= n {
			ed.Ln = n - 1
			ed.Ch = len(ls.lines[ed.Ln])
		}
		nlns := (ed.Ln - st.Ln) + 1
		tbe.Text = make([][]rune, nlns)
		stln := st.Ln
		if st.Ch > 0 {
			ec := len(ls.lines[st.Ln])
			sz := ec - st.Ch
			if sz > 0 {
				tbe.Text[0] = make([]rune, sz)
				copy(tbe.Text[0][0:sz], ls.lines[st.Ln][st.Ch:])
			}
			stln++
		}
		edln := ed.Ln
		if ed.Ch < len(ls.lines[ed.Ln]) {
			tbe.Text[ed.Ln-st.Ln] = make([]rune, ed.Ch)
			copy(tbe.Text[ed.Ln-st.Ln], ls.lines[ed.Ln][:ed.Ch])
			edln--
		}
		for ln := stln; ln <= edln; ln++ {
			ti := ln - st.Ln
			sz := len(ls.lines[ln])
			tbe.Text[ti] = make([]rune, sz)
			copy(tbe.Text[ti], ls.lines[ln])
		}
	}
	return tbe
}

// regionRect returns a Edit representation of text between
// start and end positions as a rectangle,
// returns nil if not a valid region.  sets the timestamp on the Edit to now
func (ls *Lines) regionRect(st, ed lexer.Pos) *Edit {
	st = ls.validPos(st)
	ed = ls.validPos(ed)
	if st == ed {
		return nil
	}
	if !st.IsLess(ed) || st.Ch >= ed.Ch {
		log.Printf("core.Buf.RegionRect: starting position must be less than ending!: st: %v, ed: %v\n", st, ed)
		return nil
	}
	tbe := &Edit{Reg: NewRegionPos(st, ed)}
	tbe.Rect = true
	// first get chars on start and end
	nlns := (ed.Ln - st.Ln) + 1
	nch := (ed.Ch - st.Ch)
	tbe.Text = make([][]rune, nlns)
	for i := 0; i < nlns; i++ {
		ln := st.Ln + i
		lr := ls.lines[ln]
		ll := len(lr)
		var txt []rune
		if ll > st.Ch {
			sz := min(ll-st.Ch, nch)
			txt = make([]rune, sz, nch)
			edl := min(ed.Ch, ll)
			copy(txt, lr[st.Ch:edl])
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
func (ls *Lines) deleteText(st, ed lexer.Pos) *Edit {
	tbe := ls.deleteTextImpl(st, ed)
	ls.saveUndo(tbe)
	return tbe
}

func (ls *Lines) deleteTextImpl(st, ed lexer.Pos) *Edit {
	tbe := ls.region(st, ed)
	if tbe == nil {
		return nil
	}
	tbe.Delete = true
	nl := ls.numLines()
	if ed.Ln == st.Ln {
		if st.Ln < nl {
			ec := min(ed.Ch, len(ls.lines[st.Ln])) // somehow region can still not be valid.
			ls.lines[st.Ln] = append(ls.lines[st.Ln][:st.Ch], ls.lines[st.Ln][ec:]...)
			ls.linesEdited(tbe)
		}
	} else {
		// first get chars on start and end
		stln := st.Ln + 1
		cpln := st.Ln
		ls.lines[st.Ln] = ls.lines[st.Ln][:st.Ch]
		eoedl := 0
		if ed.Ln >= nl {
			// todo: somehow this is happening in patch diffs -- can't figure out why
			// fmt.Println("err in range:", ed.Ln, nl, ed.Ch)
			ed.Ln = nl - 1
		}
		if ed.Ch < len(ls.lines[ed.Ln]) {
			eoedl = len(ls.lines[ed.Ln][ed.Ch:])
		}
		var eoed []rune
		if eoedl > 0 { // save it
			eoed = make([]rune, eoedl)
			copy(eoed, ls.lines[ed.Ln][ed.Ch:])
		}
		ls.lines = append(ls.lines[:stln], ls.lines[ed.Ln+1:]...)
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
// Fails if st.Ch >= ed.Ch. Sets the timestamp on resulting Edit to now.
// An Undo record is automatically saved depending on Undo.Off setting.
func (ls *Lines) deleteTextRect(st, ed lexer.Pos) *Edit {
	tbe := ls.deleteTextRectImpl(st, ed)
	ls.saveUndo(tbe)
	return tbe
}

func (ls *Lines) deleteTextRectImpl(st, ed lexer.Pos) *Edit {
	tbe := ls.regionRect(st, ed)
	if tbe == nil {
		return nil
	}
	tbe.Delete = true
	for ln := st.Ln; ln <= ed.Ln; ln++ {
		l := ls.lines[ln]
		if len(l) > st.Ch {
			if ed.Ch < len(l)-1 {
				ls.lines[ln] = append(l[:st.Ch], l[ed.Ch:]...)
			} else {
				ls.lines[ln] = l[:st.Ch]
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
func (ls *Lines) insertText(st lexer.Pos, text []byte) *Edit {
	tbe := ls.insertTextImpl(st, text)
	ls.saveUndo(tbe)
	return tbe
}

func (ls *Lines) insertTextImpl(st lexer.Pos, text []byte) *Edit {
	if len(text) == 0 {
		return nil
	}
	st = ls.validPos(st)
	lns := bytes.Split(text, []byte("\n"))
	sz := len(lns)
	rs := bytes.Runes(lns[0])
	rsz := len(rs)
	ed := st
	var tbe *Edit
	st.Ch = min(len(ls.lines[st.Ln]), st.Ch)
	if sz == 1 {
		ls.lines[st.Ln] = slices.Insert(ls.lines[st.Ln], st.Ch, rs...)
		ed.Ch += rsz
		tbe = ls.region(st, ed)
		ls.linesEdited(tbe)
	} else {
		if ls.lines[st.Ln] == nil {
			ls.lines[st.Ln] = []rune("")
		}
		eostl := len(ls.lines[st.Ln][st.Ch:]) // end of starting line
		var eost []rune
		if eostl > 0 { // save it
			eost = make([]rune, eostl)
			copy(eost, ls.lines[st.Ln][st.Ch:])
		}
		ls.lines[st.Ln] = append(ls.lines[st.Ln][:st.Ch], rs...)
		nsz := sz - 1
		tmp := make([][]rune, nsz)
		for i := 1; i < sz; i++ {
			tmp[i-1] = bytes.Runes(lns[i])
		}
		stln := st.Ln + 1
		ls.lines = slices.Insert(ls.lines, stln, tmp...)
		ed.Ln += nsz
		ed.Ch = len(ls.lines[ed.Ln])
		if eost != nil {
			ls.lines[ed.Ln] = append(ls.lines[ed.Ln], eost...)
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
func (ls *Lines) insertTextRect(tbe *Edit) *Edit {
	re := ls.insertTextRectImpl(tbe)
	ls.saveUndo(re)
	return tbe
}

func (ls *Lines) insertTextRectImpl(tbe *Edit) *Edit {
	st := tbe.Reg.Start
	ed := tbe.Reg.End
	nlns := (ed.Ln - st.Ln) + 1
	if nlns <= 0 {
		return nil
	}
	ls.changed = true
	// make sure there are enough lines -- add as needed
	cln := ls.numLines()
	if cln <= ed.Ln {
		nln := (1 + ed.Ln) - cln
		tmp := make([][]rune, nln)
		ls.lines = append(ls.lines, tmp...)
		ie := &Edit{}
		ie.Reg.Start.Ln = cln - 1
		ie.Reg.End.Ln = ed.Ln
		ls.linesInserted(ie)
	}
	nch := (ed.Ch - st.Ch)
	for i := 0; i < nlns; i++ {
		ln := st.Ln + i
		lr := ls.lines[ln]
		ir := tbe.Text[i]
		if len(lr) < st.Ch {
			lr = append(lr, runes.Repeat([]rune(" "), st.Ch-len(lr))...)
		}
		nt := append(lr, ir...)          // first append to end to extend capacity
		copy(nt[st.Ch+nch:], nt[st.Ch:]) // move stuff to end
		copy(nt[st.Ch:], ir)             // copy into position
		ls.lines[ln] = nt
	}
	re := tbe.Clone()
	re.Delete = false
	re.Reg.TimeNow()
	ls.linesEdited(re)
	return re
}

// ReplaceText does DeleteText for given region, and then InsertText at given position
// (typically same as delSt but not necessarily).
// if matchCase is true, then the lexer.MatchCase function is called to match the
// case (upper / lower) of the new inserted text to that of the text being replaced.
// returns the Edit for the inserted text.
// An Undo record is automatically saved depending on Undo.Off setting.
func (ls *Lines) replaceText(delSt, delEd, insPos lexer.Pos, insTxt string, matchCase bool) *Edit {
	if matchCase {
		red := ls.region(delSt, delEd)
		cur := string(red.ToBytes())
		insTxt = lexer.MatchCase(cur, insTxt)
	}
	if len(insTxt) > 0 {
		ls.deleteText(delSt, delEd)
		return ls.insertText(insPos, []byte(insTxt))
	}
	return ls.deleteText(delSt, delEd)
}

/////////////////////////////////////////////////////////////////////////////
//   Undo

// saveUndo saves given edit to undo stack
func (ls *Lines) saveUndo(tbe *Edit) {
	if tbe == nil {
		return
	}
	ls.Undos.Save(tbe)
}

// undo undoes next group of items on the undo stack
func (ls *Lines) undo() []*Edit {
	tbe := ls.Undos.UndoPop()
	if tbe == nil {
		// note: could clear the changed flag on tbe == nil in parent
		return nil
	}
	stgp := tbe.Group
	var eds []*Edit
	for {
		if tbe.Rect {
			if tbe.Delete {
				utbe := ls.insertTextRectImpl(tbe)
				utbe.Group = stgp + tbe.Group
				if ls.Options.EmacsUndo {
					ls.Undos.SaveUndo(utbe)
				}
				eds = append(eds, utbe)
			} else {
				utbe := ls.deleteTextRectImpl(tbe.Reg.Start, tbe.Reg.End)
				utbe.Group = stgp + tbe.Group
				if ls.Options.EmacsUndo {
					ls.Undos.SaveUndo(utbe)
				}
				eds = append(eds, utbe)
			}
		} else {
			if tbe.Delete {
				utbe := ls.insertTextImpl(tbe.Reg.Start, tbe.ToBytes())
				utbe.Group = stgp + tbe.Group
				if ls.Options.EmacsUndo {
					ls.Undos.SaveUndo(utbe)
				}
				eds = append(eds, utbe)
			} else {
				utbe := ls.deleteTextImpl(tbe.Reg.Start, tbe.Reg.End)
				utbe.Group = stgp + tbe.Group
				if ls.Options.EmacsUndo {
					ls.Undos.SaveUndo(utbe)
				}
				eds = append(eds, utbe)
			}
		}
		tbe = ls.Undos.UndoPopIfGroup(stgp)
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
	ls.Undos.UndoStackSave()
}

// redo redoes next group of items on the undo stack,
// and returns the last record, nil if no more
func (ls *Lines) redo() []*Edit {
	tbe := ls.Undos.RedoNext()
	if tbe == nil {
		return nil
	}
	var eds []*Edit
	stgp := tbe.Group
	for {
		if tbe.Rect {
			if tbe.Delete {
				ls.deleteTextRectImpl(tbe.Reg.Start, tbe.Reg.End)
			} else {
				ls.insertTextRectImpl(tbe)
			}
		} else {
			if tbe.Delete {
				ls.deleteTextImpl(tbe.Reg.Start, tbe.Reg.End)
			} else {
				ls.insertTextImpl(tbe.Reg.Start, tbe.ToBytes())
			}
		}
		eds = append(eds, tbe)
		tbe = ls.Undos.RedoNextIfGroup(stgp)
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
func (ls *Lines) linesEdited(tbe *Edit) {
	st, ed := tbe.Reg.Start.Ln, tbe.Reg.End.Ln
	for ln := st; ln <= ed; ln++ {
		ls.lineBytes[ln] = []byte(string(ls.lines[ln]))
		ls.Markup[ln] = highlighting.HtmlEscapeRunes(ls.lines[ln])
	}
	ls.markupLines(st, ed)
	ls.startDelayedReMarkup()
}

// linesInserted inserts new lines for all other line-based slices
// corresponding to lines inserted in the lines slice.
func (ls *Lines) linesInserted(tbe *Edit) {
	stln := tbe.Reg.Start.Ln + 1
	nsz := (tbe.Reg.End.Ln - tbe.Reg.Start.Ln)

	ls.markupEdits = append(ls.markupEdits, tbe)
	ls.lineBytes = slices.Insert(ls.lineBytes, stln, make([][]byte, nsz)...)
	ls.Markup = slices.Insert(ls.Markup, stln, make([][]byte, nsz)...)
	ls.tags = slices.Insert(ls.tags, stln, make([]lexer.Line, nsz)...)
	ls.hiTags = slices.Insert(ls.hiTags, stln, make([]lexer.Line, nsz)...)

	if ls.Highlighter.UsingParse() {
		pfs := ls.ParseState.Done()
		pfs.Src.LinesInserted(stln, nsz)
	}
	ls.linesEdited(tbe)
}

// linesDeleted deletes lines in Markup corresponding to lines
// deleted in Lines text.
func (ls *Lines) linesDeleted(tbe *Edit) {
	ls.markupEdits = append(ls.markupEdits, tbe)
	stln := tbe.Reg.Start.Ln
	edln := tbe.Reg.End.Ln
	ls.lineBytes = append(ls.lineBytes[:stln], ls.lineBytes[edln:]...)
	ls.Markup = append(ls.Markup[:stln], ls.Markup[edln:]...)
	ls.tags = append(ls.tags[:stln], ls.tags[edln:]...)
	ls.hiTags = append(ls.hiTags[:stln], ls.hiTags[edln:]...)

	if ls.Highlighter.UsingParse() {
		pfs := ls.ParseState.Done()
		pfs.Src.LinesDeleted(stln, edln)
	}
	st := tbe.Reg.Start.Ln
	ls.lineBytes[st] = []byte(string(ls.lines[st]))
	ls.Markup[st] = highlighting.HtmlEscapeRunes(ls.lines[st])
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
		fs := ls.ParseState.Done() // initialize
		fs.Src.SetBytes(ls.bytes())
	}
	mxhi := min(100, ls.numLines())
	txt := bytes.Join(ls.lineBytes[:mxhi], []byte("\n"))
	txt = append(txt, []byte("\n")...)
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
func (ls *Lines) AdjustRegion(reg Region) Region {
	return ls.Undos.AdjustRegion(reg)
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
		reg := Region{Start: lexer.Pos{Ln: ln, Ch: tg.St}, End: lexer.Pos{Ln: ln, Ch: tg.Ed}}
		reg.Time = tg.Time
		reg = ls.Undos.AdjustRegion(reg)
		if !reg.IsNil() {
			ntr := ntags.AddLex(tg.Token, reg.Start.Ch, reg.End.Ch)
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
		pfs := ls.ParseState.Done()
		for _, tbe := range edits {
			if tbe.Delete {
				stln := tbe.Reg.Start.Ln
				edln := tbe.Reg.End.Ln
				pfs.Src.LinesDeleted(stln, edln)
			} else {
				stln := tbe.Reg.Start.Ln + 1
				nlns := (tbe.Reg.End.Ln - tbe.Reg.Start.Ln)
				pfs.Src.LinesInserted(stln, nlns)
			}
		}
		for ln := range tags {
			tags[ln] = pfs.LexLine(ln) // does clone, combines comments too
		}
	} else {
		for _, tbe := range edits {
			if tbe.Delete {
				stln := tbe.Reg.Start.Ln
				edln := tbe.Reg.End.Ln
				tags = append(tags[:stln], tags[edln:]...)
			} else {
				stln := tbe.Reg.Start.Ln + 1
				nlns := (tbe.Reg.End.Ln - tbe.Reg.Start.Ln)
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
		ls.Markup[ln] = highlighting.MarkupLine(ls.lines[ln], tags[ln], ls.tags[ln], highlighting.EscapeHTML)
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
			ls.Markup[ln] = highlighting.MarkupLine(ltxt, mt, ls.adjustedTags(ln), highlighting.EscapeHTML)
		} else {
			ls.Markup[ln] = highlighting.HtmlEscapeRunes(ltxt)
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
func (ls *Lines) AddTagEdit(tbe *Edit, tag token.Tokens) {
	ls.AddTag(tbe.Reg.Start.Ln, tbe.Reg.Start.Ch, tbe.Reg.End.Ch, tag)
}

// RemoveTag removes tag (optionally only given tag if non-zero)
// at given position if it exists. returns tag.
func (ls *Lines) RemoveTag(pos lexer.Pos, tag token.Tokens) (reg lexer.Lex, ok bool) {
	if !ls.IsValidLine(pos.Ln) {
		return
	}
	ls.Lock()
	defer ls.Unlock()

	ls.tags[pos.Ln] = ls.adjustedTags(pos.Ln) // re-adjust for current info
	for i, t := range ls.tags[pos.Ln] {
		if t.ContainsPos(pos.Ch) {
			if tag > 0 && t.Token.Token != tag {
				continue
			}
			ls.tags[pos.Ln].DeleteIndex(i)
			reg = t
			ok = true
			break
		}
	}
	if ok {
		ls.markupLines(pos.Ln, pos.Ln)
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
func (ls *Lines) hiTagAtPos(pos lexer.Pos) (*lexer.Lex, int) {
	if !ls.isValidLine(pos.Ln) {
		return nil, -1
	}
	return ls.hiTags[pos.Ln].AtPos(pos.Ch)
}

// inTokenSubCat returns true if the given text position is marked with lexical
// type in given SubCat sub-category.
func (ls *Lines) inTokenSubCat(pos lexer.Pos, subCat token.Tokens) bool {
	lx, _ := ls.hiTagAtPos(pos)
	return lx != nil && lx.Token.Token.InSubCat(subCat)
}

// inLitString returns true if position is in a string literal
func (ls *Lines) inLitString(pos lexer.Pos) bool {
	return ls.inTokenSubCat(pos, token.LitStr)
}

// inTokenCode returns true if position is in a Keyword,
// Name, Operator, or Punctuation.
// This is useful for turning off spell checking in docs
func (ls *Lines) inTokenCode(pos lexer.Pos) bool {
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
func (ls *Lines) indentLine(ln, ind int) *Edit {
	tabSz := ls.Options.TabSize
	ichr := indent.Tab
	if ls.Options.SpaceIndent {
		ichr = indent.Space
	}
	curind, _ := lexer.LineIndent(ls.lines[ln], tabSz)
	if ind > curind {
		return ls.insertText(lexer.Pos{Ln: ln}, indent.Bytes(ichr, ind-curind, tabSz))
	} else if ind < curind {
		spos := indent.Len(ichr, ind, tabSz)
		cpos := indent.Len(ichr, curind, tabSz)
		return ls.deleteText(lexer.Pos{Ln: ln, Ch: spos}, lexer.Pos{Ln: ln, Ch: cpos})
	}
	return nil
}

// autoIndent indents given line to the level of the prior line, adjusted
// appropriately if the current line starts with one of the given un-indent
// strings, or the prior line ends with one of the given indent strings.
// Returns any edit that took place (could be nil), along with the auto-indented
// level and character position for the indent of the current line.
func (ls *Lines) autoIndent(ln int) (tbe *Edit, indLev, chPos int) {
	tabSz := ls.Options.TabSize
	lp, _ := parse.LanguageSupport.Properties(ls.ParseState.Known)
	var pInd, delInd int
	if lp != nil && lp.Lang != nil {
		pInd, delInd, _, _ = lp.Lang.IndentLine(&ls.ParseState, ls.lines, ls.hiTags, ln, tabSz)
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
func (ls *Lines) inComment(pos lexer.Pos) bool {
	if ls.inTokenSubCat(pos, token.Comment) {
		return true
	}
	cs := ls.commentStart(pos.Ln)
	if cs < 0 {
		return false
	}
	return pos.Ch > cs
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
			ls.insertText(lexer.Pos{Ln: ln, Ch: ch}, []byte(comst))
			if comed != "" {
				lln := len(ls.lines[ln])
				ls.insertText(lexer.Pos{Ln: ln, Ch: lln}, []byte(comed))
			}
		} else {
			idx := ls.commentStart(ln)
			if idx >= 0 {
				ls.deleteText(lexer.Pos{Ln: ln, Ch: idx}, lexer.Pos{Ln: ln, Ch: idx + len(comst)})
			}
			if comed != "" {
				idx := runes.IndexFold(ls.lines[ln], []rune(comed))
				if idx >= 0 {
					ls.deleteText(lexer.Pos{Ln: ln, Ch: idx}, lexer.Pos{Ln: ln, Ch: idx + len(comed)})
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
				stp := lexer.Pos{Ln: ln + 1}
				if ln == startLine {
					stp.Ln--
				}
				ep := lexer.Pos{Ln: curEd - 1}
				if curEd == endLine {
					ep.Ln = curEd
				}
				eln := ls.lines[ep.Ln]
				ep.Ch = len(eln)
				tlb := bytes.Join(ls.lineBytes[stp.Ln:ep.Ln+1], []byte(" "))
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
	st := lexer.Pos{Ln: ln}
	ed := lexer.Pos{Ln: ln}
	i := 0
	for {
		if i >= len(lr) {
			break
		}
		r := lr[i]
		if r == '\t' {
			po := i % tabSz
			nspc := tabSz - po
			st.Ch = i
			ed.Ch = i + 1
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
	st := lexer.Pos{Ln: ln}
	ed := lexer.Pos{Ln: ln}
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
				st.Ch = i - (tabSz - 1)
				ed.Ch = i + 1
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

///////////////////////////////////////////////////////////////////
//  Diff

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
			ls.deleteText(lexer.Pos{Ln: df.I1}, lexer.Pos{Ln: df.I2})
			// fmt.Printf("patch rep del: %v %v\n", tbe.Reg, string(tbe.ToBytes()))
			ot := ob.Region(lexer.Pos{Ln: df.J1}, lexer.Pos{Ln: df.J2})
			ls.insertText(lexer.Pos{Ln: df.I1}, ot.ToBytes())
			// fmt.Printf("patch rep ins: %v %v\n", tbe.Reg, string(tbe.ToBytes()))
			mods = true
		case 'd':
			ls.deleteText(lexer.Pos{Ln: df.I1}, lexer.Pos{Ln: df.I2})
			// fmt.Printf("patch del: %v %v\n", tbe.Reg, string(tbe.ToBytes()))
			mods = true
		case 'i':
			ot := ob.Region(lexer.Pos{Ln: df.J1}, lexer.Pos{Ln: df.J2})
			ls.insertText(lexer.Pos{Ln: df.I1}, ot.ToBytes())
			// fmt.Printf("patch ins: %v %v\n", tbe.Reg, string(tbe.ToBytes()))
			mods = true
		}
	}
	return mods
}
