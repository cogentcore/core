// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import (
	"slices"
	"time"

	"cogentcore.org/core/text/highlighting"
	"cogentcore.org/core/text/parse/lexer"
	"cogentcore.org/core/text/rich"
)

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
// and sets the markup lines.  Must be called under Lock.
func (ls *Lines) markupApplyTags(tags []lexer.Line) {
	tags = ls.markupApplyEdits(tags)
	maxln := min(len(tags), ls.numLines())
	for ln := range maxln {
		ls.hiTags[ln] = tags[ln]
		ls.tags[ln] = ls.adjustedTags(ln)
		mu := highlighting.MarkupLineRich(ls.Highlighter.Style, ls.fontStyle, ls.lines[ln], tags[ln], ls.tags[ln])
		lmu, lay, nbreaks := ls.layoutLine(ls.lines[ln], mu)
		ls.markup[ln] = lmu
		ls.layout[ln] = lay
		ls.nbreaks[ln] = nbreaks
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
		var mu rich.Text
		if err == nil {
			ls.hiTags[ln] = mt
			mu = highlighting.MarkupLineRich(ls.Highlighter.Style, ls.fontStyle, ltxt, mt, ls.adjustedTags(ln))
		} else {
			mu = rich.NewText(ls.fontStyle, ltxt)
			allgood = false
		}
		lmu, lay, nbreaks := ls.layoutLine(ltxt, mu)
		ls.markup[ln] = lmu
		ls.layout[ln] = lay
		ls.nbreaks[ln] = nbreaks
	}
	// Now we trigger a background reparse of everything in a separate parse.FilesState
	// that gets switched into the current.
	return allgood
}
