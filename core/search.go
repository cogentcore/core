// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"

	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/runes"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/tree"
)

// SearchResults are the search results for text within [Text] widgets.
type SearchResults struct {
	// Text is the widget
	Text *Text

	// Matches are the matching ranges within the RichText of widget.
	Matches []textpos.Range
}

// TextSearchResults returns all of the [Text] widgets within given widget,
// with matching text for the given find string, with given case sensitivity.
func TextSearchResults(w Widget, find string, ignoreCase bool) []SearchResults {
	fr := []rune(find)
	fsz := len(fr)
	var res []SearchResults
	wb := w.AsWidget()
	wb.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
		tx, ok := cw.(*Text)
		if !ok {
			return tree.Continue
		}
		rn := tx.richText.Join()

		var matches []textpos.Range
		sz := len(rn)
		ci := 0
		for ci < sz {
			var i int
			if ignoreCase {
				i = runes.IndexFold(rn[ci:], fr)
			} else {
				i = runes.Index(rn[ci:], fr)
			}
			if i < 0 {
				break
			}
			i += ci
			ci = i + fsz
			matches = append(matches, textpos.Range{Start: i, End: ci})
		}
		if len(matches) > 0 {
			res = append(res, SearchResults{Text: tx, Matches: matches})
		}
		return tree.Continue
	})
	return res
}

// TextSearchHighlightReset resets the highlights within all [Text] elements
// under given widget.
func TextSearchHighlightReset(w Widget) {
	wb := w.AsWidget()
	wb.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
		tx, ok := cw.(*Text)
		if !ok {
			return tree.Continue
		}
		tx.highlights = nil
		tx.NeedsRender()
		return tree.Continue
	})
}

// TextSearchHighlight adds region highlights for given text search results,
// first calling [TextSearchHighlightReset] to reset any existing highlights.
func TextSearchHighlight(w Widget, res []SearchResults) {
	TextSearchHighlightReset(w)
	for _, r := range res {
		r.Text.highlights = r.Matches
		r.Text.NeedsRender()
	}
}

// TextSearch performs an interactive search for given text,
// on elements within given widget.
func TextSearch(w Widget, find string, ignoreCase bool) {
	var res []SearchResults
	var n int
	idx := 0
	search := func() {
		res = TextSearchResults(w, find, ignoreCase)
		TextSearchHighlight(w, res)
		n = len(res)
	}
	search()

	d := NewBody("Search")
	d.Styler(func(s *styles.Style) {
		s.Direction = styles.Row
	})
	st := NewTextField(d).SetText(find)

	ix := NewText(d).SetText("00/00")
	ix.Updater(func() {
		ix.SetText(fmt.Sprintf("%d/%d", idx, n))
	})
	st.OnChange(func(e events.Event) {
		find = st.Text()
		search()
		ix.Update()
	})
	up := NewButton(d).SetIcon(icons.ArrowUpward)
	up.OnClick(func(e events.Event) {
		if n == 0 {
			return
		}
		idx--
		if idx < 0 {
			idx += n
		}
		ix.Update()
		// todo: select
	})
	dn := NewButton(d).SetIcon(icons.ArrowDownward)
	dn.OnClick(func(e events.Event) {
		if n == 0 {
			return
		}
		idx++
		if idx >= n {
			idx -= n
		}
		ix.Update()
		// todo: select
	})
	x := NewButton(d).SetIcon(icons.Close)
	x.OnClick(func(e events.Event) {
		TextSearchHighlightReset(w)
		d.Close()
	})
	d.RunDialog(w)

}
