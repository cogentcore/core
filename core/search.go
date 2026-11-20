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

var (
	// LastSearch remembers the last search string.
	LastSearch string

	// LastUseCase remembers the last search useCase value.
	LastUseCase CaseSensitive
)

// CaseSensitive specifies whether a search is case-sensitive: [UseCase] or
// [IgnoreCase]. It is [textpos.CaseSensitive], which the text search
// functions use.
type CaseSensitive = textpos.CaseSensitive

// SelectionReset specifies whether a select function resets the selection
// ([SelectReset]) or sets it ([SelectSet]).
type SelectionReset bool

// SelectionScroll specifies whether a select function scrolls to the selection:
// [SelectScroll] or [SelectNoScroll].
type SelectionScroll bool

const (
	// UseCase can be used as arg for [Search] functions,
	// to use case-sensitive search.
	UseCase = textpos.UseCase

	// IgnoreCase can be used as arg for [Search] functions
	// to use case-insensitive search.
	IgnoreCase = textpos.IgnoreCase

	// SelectReset can be used as reset arg for Select functions,
	// to reset the selection.
	SelectReset SelectionReset = true

	// SelectSet can be used as reset arg for Select functions,
	// to actually set the selection.
	SelectSet SelectionReset = false

	// SelectScroll can be used as scroll arg for Select functions,
	// to scroll to given location.
	SelectScroll SelectionScroll = true

	// SelectNoScroll can be used as scroll arg for Select functions,
	// to NOT scroll to given location.
	SelectNoScroll SelectionScroll = false
)

// SearchResult is one set of search results for text in a widget.
type SearchResult struct {
	// Widget with the text. Can be a collection too, to manage scrolling.
	Widget Widget

	// Matches are the matching ranges within the RichText of widget.
	Matches []textpos.Match
}

// SearchResults are the collection of search results.
type SearchResults []SearchResult

// Len returns the total number of search results.
func (sr SearchResults) Len() int {
	n := 0
	for _, r := range sr {
		n += len(r.Matches)
	}
	return n
}

// AtIndex returns the result at given index into the full set
// of results [0..Len()), returning the local index into Matches
// within the given result.
func (sr SearchResults) AtIndex(idx int) (*SearchResult, int) {
	n := 0
	for i := range sr {
		r := &sr[i]
		nr := len(r.Matches)
		if idx >= n && idx < n+nr {
			return r, idx - n
		}
		n += nr
	}
	return nil, -1
}

// SearchRunes is a helper function that returns [textpos.Matches]
// as Region Char indexes into the runes.
func SearchRunes(rn []rune, find string, useCase CaseSensitive) []textpos.Match {
	var matches []textpos.Match
	fr := []rune(find)
	fsz := len(fr)
	sz := len(rn)
	ci := 0
	for ci < sz {
		var i int
		if useCase {
			i = runes.Index(rn[ci:], fr)
		} else {
			i = runes.IndexFold(rn[ci:], fr)
		}
		if i < 0 {
			break
		}
		i += ci
		ci = i + fsz
		matches = append(matches, textpos.NewMatch(rn, i, ci, 0))
	}
	return matches
}

// SearchWidgets performs text in widgets under given widget,
// for the given find string, with given case sensitivity,
// using the Search interface method.
func SearchWidgets(w Widget, find string, useCase CaseSensitive) SearchResults {
	var res SearchResults
	wb := w.AsWidget()
	wb.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
		if !cwb.IsDisplayable() {
			return tree.Break
		}
		matches, handled := cw.Search(find, useCase)
		if len(matches) > 0 {
			res = append(res, SearchResult{Widget: cw, Matches: matches})
		}
		if handled {
			return tree.Break
		}
		return tree.Continue
	})
	return res
}

// SearchHighlight highlights all the [SearchResults] under given
// widget, resetting any existing first. Pass a nil to just reset.
func SearchHighlight(w Widget, results SearchResults) {
	wb := w.AsWidget()
	wb.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
		handled := cw.HighlightMatches(nil)
		if handled {
			return tree.Break
		}
		return tree.Continue
	})
	if results == nil {
		return
	}
	for _, r := range results {
		r.Widget.HighlightMatches(r.Matches)
	}
}

// SearchSelect selects the [SearchResult] at given results index
// from given [SearchResults], or resets the selection if reset = true.
func SearchSelect(results SearchResults, index int, reset SelectionReset) {
	res, i := results.AtIndex(index)
	if res == nil {
		return
	}
	res.Widget.SelectMatch(res.Matches, i, SelectionScroll(!reset), reset)
}

// Search performs an interactive search for given text,
// on elements within given widget.
func Search(w Widget, find string, useCase CaseSensitive) {
	LastSearch = find
	var res SearchResults
	var n int
	idx := 0
	search := func() {
		res = SearchWidgets(w, find, useCase)
		SearchHighlight(w, res)
		n = res.Len()
	}

	sel := func() {
		SearchSelect(res, idx, SelectSet) // not reset
	}
	unSel := func() {
		SearchSelect(res, idx, SelectReset) // reset
	}
	search()
	sel()

	d := NewBody("Search")
	d.Styler(func(s *styles.Style) {
		s.Direction = styles.Row
	})
	st := NewTextField(d).SetText(find)

	ix := NewText(d).SetText("00/00")
	ix.Updater(func() {
		ix.SetText(fmt.Sprintf("%d/%d", idx+1, n))
	})
	st.OnChange(func(e events.Event) {
		find = st.Text()
		LastSearch = find
		unSel()
		search()
		sel()
		ix.Update()
	})
	up := NewButton(d).SetIcon(icons.ArrowUpward)
	up.OnClick(func(e events.Event) {
		if n == 0 {
			return
		}
		unSel()
		idx--
		if idx < 0 {
			idx += n
		}
		sel()
		ix.Update()
		// todo: select
	})
	dn := NewButton(d).SetIcon(icons.ArrowDownward)
	dn.OnClick(func(e events.Event) {
		if n == 0 {
			return
		}
		unSel()
		idx++
		if idx >= n {
			idx -= n
		}
		sel()
		ix.Update()
		// todo: select
	})
	cs := NewSwitch(d).SetText("Use case")
	cs.SetTooltip("Use case sensitive search.")
	Bind(&useCase, cs)
	cs.OnChange(func(e events.Event) {
		unSel()
		search()
		sel()
	})
	x := NewButton(d).SetIcon(icons.Close)
	x.SetTooltip("Close search dialog")
	x.OnClick(func(e events.Event) {
		unSel()
		SearchHighlight(w, nil)
		d.Close()
	})
	d.NewDialog(w).SetModal(false).Run()
}
