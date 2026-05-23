// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package content

import (
	"fmt"

	"cogentcore.org/core/base/strcase"
	"cogentcore.org/core/content/bcontent"
	"cogentcore.org/core/core"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/tree"
)

func (ct *Content) nearestHeading() string {
	fr := ct.pageFrame()
	if fr == nil {
		return ""
	}
	d := math32.Y
	if !fr.HasScroll[d] || fr.Scrolls[d] == nil {
		return ""
	}
	// go through the lines and find one at pos
	var lastHead string
	fr.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
		wpos := cwb.Geom.Pos.Total.Y - fr.Geom.Pos.Content.Y
		if wpos > 100 {
			return tree.Break
		}
		tx, ok := cw.(*core.Text)
		if !ok {
			return tree.Continue
		}
		tag, ok := tx.Property("tag").(string)
		if ok && len(tag) == 2 && tag[0] == 'h' {
			lastHead = tx.Text
		}
		return tree.Continue
	})
	return lastHead
}

// save an updated history record based on current position on current page.
// args are for the _next_ page that will be opened: don't add updated history
// if the same as these args.
func (ct *Content) saveUpdatedPos(curPg Location, pg *bcontent.Page, heading string) {
	nhead := ct.nearestHeading()
	if nhead == "" {
		return
	}
	url := curPg.URL
	_, nw, _ := ct.pageURL(curPg.Page, strcase.ToKebab(nhead))
	if nw != nil {
		url = nw.String()
	} else {
		url = curPg.Page.URL + "#" + strcase.ToKebab(nhead)
	}
	ct.historyAdd(curPg.Page, nhead, url)
}

func (ct *Content) historyAdd(pg *bcontent.Page, heading, url string) {
	if ct.tabs == nil {
		ct.saveWebURL(&Location{Page: pg, Heading: heading, URL: url})
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

func (lc *Location) String() string {
	if lc.Page != nil {
		return fmt.Sprintf("%q (page: %q head: %q)", lc.URL, lc.Page.URL, lc.Heading)
	}
	return fmt.Sprintf("%q", lc.URL)
}

func (lc *Location) Clone() *Location {
	nc := &Location{}
	*nc = *lc
	return nc
}

// History records the history of browsing locations, for back arrow
// navigation.
type History struct {
	// Index is the current index in the Records, which is typically the
	// current page as saved. Back decrements then returns that.
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
		if hs.Index > 0 && *(hs.Records[hs.Index-1]) == *lc { // no repeats
			return
		}
		hs.Records = hs.Records[:hs.Index+1]
		hs.Records[hs.Index] = lc
	} else {
		hs.Index = len(hs.Records)                            // note: going to end first
		if hs.Index > 0 && *(hs.Records[hs.Index-1]) == *lc { // no repeats
			return
		}
		hs.Records = append(hs.Records, lc)
	}
}

// Back decrements Index and returns that location.
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
