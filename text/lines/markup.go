// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import (
	"slices"
	"time"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/text/highlighting"
	"cogentcore.org/core/text/parse/lexer"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/text/token"
)

// setFileInfo sets the syntax highlighting and other parameters
// based on the type of file specified by given [fileinfo.FileInfo].
func (ls *Lines) setFileInfo(info *fileinfo.FileInfo) {
	ls.parseState.SetSrc(string(info.Path), "", info.Known)
	ls.Highlighter.Init(info, &ls.parseState)
	ls.Settings.ConfigKnown(info.Known)
	if ls.numLines() > 0 {
		ls.initialMarkup()
		ls.startDelayedReMarkup()
	}
}

// initialMarkup does the first-pass markup on the file
func (ls *Lines) initialMarkup() {
	if !ls.Highlighter.Has || ls.numLines() == 0 {
		return
	}
	txt := ls.bytes(100)
	if ls.Highlighter.UsingParse() {
		fs := ls.parseState.Done() // initialize
		fs.Src.SetBytes(txt)
	}
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

// stopDelayedReMarkup stops timer for doing markup after an interval
func (ls *Lines) stopDelayedReMarkup() {
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
	ls.stopDelayedReMarkup()
	go ls.asyncMarkup()
}

// asyncMarkup does the markupTags from a separate goroutine.
// Does not start or end with lock, but acquires at end to apply.
func (ls *Lines) asyncMarkup() {
	ls.Lock()
	txt := ls.bytes(0)
	ls.markupEdits = nil // only accumulate after this point; very rare
	ls.Unlock()

	tags, err := ls.markupTags(txt)
	if err != nil {
		return
	}
	ls.Lock()
	ls.markupApplyTags(tags)
	ls.Unlock()
	ls.sendChange()
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
	// fmt.Println("edits:", edits)
	if len(ls.markupEdits) == 0 {
		return tags // todo: somehow needs to actually do process below even if no edits
		// but I can't remember right now what the issues are.
	}
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
		for ln := range tags { // todo: something weird about this -- not working in test
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
// and sets the markup lines. Must be called under Lock.
func (ls *Lines) markupApplyTags(tags []lexer.Line) {
	tags = ls.markupApplyEdits(tags)
	maxln := min(len(tags), ls.numLines())
	for ln := range maxln {
		ls.hiTags[ln] = tags[ln]
		ls.tags[ln] = ls.adjustedTags(ln)
		// fmt.Println("#####\n", ln, "tags:\n", tags[ln])
		ls.markup[ln] = highlighting.MarkupLineRich(ls.Highlighter.Style, ls.fontStyle, ls.lines[ln], tags[ln], ls.tags[ln])
	}
	for _, vw := range ls.views {
		ls.layoutAll(vw)
	}
}

// markupLines generates markup of given range of lines.
// end is *inclusive* line. Called after edits, under Lock().
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
		var mu rich.Text
		if err == nil {
			ls.hiTags[ln] = mt
			mu = highlighting.MarkupLineRich(ls.Highlighter.Style, ls.fontStyle, ltxt, mt, ls.adjustedTags(ln))
		} else {
			mu = rich.NewText(ls.fontStyle, ltxt)
			allgood = false
		}
		ls.markup[ln] = mu
	}
	for _, vw := range ls.views {
		ls.layoutLines(vw, st, ed)
	}
	// Now we trigger a background reparse of everything in a separate parse.FilesState
	// that gets switched into the current.
	return allgood
}

////////  Lines and tags

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

	ls.markupEdits = append(ls.markupEdits, tbe)
	ls.markup = slices.Insert(ls.markup, stln, make([]rich.Text, nsz)...)
	ls.tags = slices.Insert(ls.tags, stln, make([]lexer.Line, nsz)...)
	ls.hiTags = slices.Insert(ls.hiTags, stln, make([]lexer.Line, nsz)...)

	// todo:
	// for _, vw := range ls.views {
	// 	vw.markup = slices.Insert(vw.markup, stln, make([]rich.Text, nsz)...)
	// 	vw.nbreaks = slices.Insert(vw.nbreaks, stln, make([]int, nsz)...)
	// 	vw.layout = slices.Insert(vw.layout, stln, make([][]textpos.Pos16, nsz)...)
	// }

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

	// todo:
	// for _, vw := range ls.views {
	// 	vw.markup = append(vw.markup[:stln], vw.markup[edln:]...)
	// 	vw.nbreaks = append(vw.nbreaks[:stln], vw.nbreaks[edln:]...)
	// 	vw.layout = append(vw.layout[:stln], vw.layout[edln:]...)
	// }

	if ls.Highlighter.UsingParse() {
		pfs := ls.parseState.Done()
		pfs.Src.LinesDeleted(stln, edln)
	}
	st := tbe.Region.Start.Line
	ls.markupLines(st, st)
	ls.startDelayedReMarkup()
}

// AdjustRegion adjusts given text region for any edits that
// have taken place since time stamp on region (using the Undo stack).
// If region was wholly within a deleted region, then RegionNil will be
// returned -- otherwise it is clipped appropriately as function of deletes.
func (ls *Lines) AdjustRegion(reg textpos.Region) textpos.Region {
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
		reg := textpos.Region{Start: textpos.Pos{Line: ln, Char: tg.Start}, End: textpos.Pos{Line: ln, Char: tg.End}}
		reg.Time = tg.Time
		reg = ls.undos.AdjustRegion(reg)
		if !reg.IsNil() {
			ntr := ntags.AddLex(tg.Token, reg.Start.Char, reg.End.Char)
			ntr.Time.Now()
		}
	}
	return ntags
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
	if lx.End > lln {
		return ""
	}
	stlx := lexer.ObjPathAt(ls.hiTags[ln], lx)
	if stlx.Start >= lx.End {
		return ""
	}
	return string(ls.lines[ln][stlx.Start:lx.End])
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
