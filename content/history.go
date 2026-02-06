// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package content

import (
	"cogentcore.org/core/content/bcontent"
)

func (ct *Content) historyAdd(pg *bcontent.Page, heading, url string) {
	if ct.tabs == nil {
		ct.saveWebURL()
		return
	}
	_, ci := ct.tabs.CurrentTab()
	ct.history[ci].Save(&Location{Page: pg, Heading: heading, URL: url})
}

func (ct *Content) historyHasBack() bool {
	if ct.tabs == nil { // shouldn't happen
		return false
	}
	_, ci := ct.tabs.CurrentTab()
	h := ct.history[ci]
	return h.Index > 0
}

func (ct *Content) historyBack() {
	if ct.tabs == nil { // shouldn't happen
		return
	}
	_, ci := ct.tabs.CurrentTab()
	lc, _ := ct.history[ci].Back()
	// fmt.Println("back:", lc.URL)
	ct.open(lc.URL, false) // no add more history
}

func (ct *Content) historyHasForward() bool {
	if ct.tabs == nil { // shouldn't happen
		return false
	}
	_, ci := ct.tabs.CurrentTab()
	h := ct.history[ci]
	return h.Index < len(h.Records)-1
}

func (ct *Content) historyForward() {
	if ct.tabs == nil { // shouldn't happen
		return
	}
	_, ci := ct.tabs.CurrentTab()
	lc, _ := ct.history[ci].Forward()
	ct.open(lc.URL, false) // no add more history
}

///////// History

// Location holds one location of browsing history.
type Location struct {
	Page    *bcontent.Page
	Heading string
	URL     string
}

func (lc *Location) Reset() {
	lc.Page = nil
	lc.Heading = ""
	lc.URL = ""
}

// History records the history of browsing locations, for back arrow
// navigation.
type History struct {
	// Index is the current index in the Records.
	// This is the location to use when the back arrow happens.
	Index int

	// Records is the list of saved locations.
	Records []*Location
}

func (hs *History) Save(lc *Location) {
	if hs.Records == nil {
		hs.Records = make([]*Location, 1)
		hs.Index = 0
		hs.Records[0] = lc
		return
	}
	hs.Index++
	if len(hs.Records) > hs.Index {
		hs.Records = hs.Records[:hs.Index+1]
		hs.Records[hs.Index] = lc
	} else {
		hs.Index = len(hs.Records) // note: going to end first
		hs.Records = append(hs.Records, lc)
	}
}

// Back returns current back location and decrements Index.
// If already at the start, returns false.
func (hs *History) Back() (*Location, bool) {
	if hs.Index <= 0 {
		return hs.Records[0], false
	}
	hs.Index--
	lc := hs.Records[hs.Index]
	return lc, true
}

// Forward returns next location and increments Index.
// returns false if already at end (returns end location).
func (hs *History) Forward() (*Location, bool) {
	if hs.Index == len(hs.Records)-1 {
		return hs.Records[hs.Index], false
	}
	hs.Index++
	return hs.Records[hs.Index], true
}
