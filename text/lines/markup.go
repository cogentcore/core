// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import (
	"slices"
	"time"

	"cogentcore.org/core/base/slicesx"
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
		ls.layoutAllLines(vw)
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

// layoutLines performs view-specific layout of current markup.
// the view must already have allocated space for these lines.
// it updates the current number of total lines based on any changes from
// the current number of lines withing given range.
func (ls *Lines) layoutLines(vw *view, st, ed int) {
	inln := 0
	for ln := st; ln <= ed; ln++ {
		inln += 1 + vw.nbreaks[ln]
	}
	nln := 0
	for ln := st; ln <= ed; ln++ {
		ltxt := ls.lines[ln]
		lmu, lay, nbreaks := ls.layoutLine(vw.width, ltxt, ls.markup[ln])
		vw.markup[ln] = lmu
		vw.layout[ln] = lay
		vw.nbreaks[ln] = nbreaks
		nln += 1 + nbreaks
	}
	vw.totalLines += nln - inln
}

// layoutAllLines performs view-specific layout of all lines of current markup.
// ensures that view has capacity to hold all lines, so it can be called on a
// new view.
func (ls *Lines) layoutAllLines(vw *view) {
	n := len(vw.markup)
	vw.markup = slicesx.SetLength(vw.markup, n)
	vw.layout = slicesx.SetLength(vw.layout, n)
	vw.nbreaks = slicesx.SetLength(vw.nbreaks, n)
	nln := 0
	for ln, mu := range ls.markup {
		lmu, lay, nbreaks := ls.layoutLine(vw.width, ls.lines[ln], mu)
		// fmt.Println("\nlayout:\n", lmu)
		vw.markup[ln] = lmu
		vw.layout[ln] = lay
		vw.nbreaks[ln] = nbreaks
		nln += 1 + nbreaks
	}
	vw.totalLines = nln
}
