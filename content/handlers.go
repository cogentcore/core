// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package content

import (
	"fmt"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/labels"
	"cogentcore.org/core/base/strcase"
	"cogentcore.org/core/content/bcontent"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/text/csl"
	"cogentcore.org/core/tree"
)

// citeWikilink processes citation links, which start with @
func (ct *Content) citeWikilink(text string) (url string, label string) {
	if len(text) == 0 || text[0] != '@' { // @CiteKey reference citations
		return "", ""
	}
	ref := text[1:]
	cs := csl.Parenthetical
	if len(ref) > 1 && ref[0] == '^' {
		cs = csl.Narrative
		ref = ref[1:]
	}
	url = "ref://" + ref
	if ct.References == nil {
		return url, ref
	}
	it, has := ct.References.AtTry(ref)
	if has {
		return url, csl.CiteDefault(cs, it)
	}
	return url, ref
}

// mainWikilink processes all other wikilinks.
// page -> Page, page
// page|label -> Page, label
// page#heading -> Page#heading, heading
// #heading -> ThisPage#heading, heading
// Page is the resolved page name.
// heading can be a special id, or id:element to find elements within a special,
// e.g., #sim_neuron:Run Cycles
func (ct *Content) mainWikilink(text string) (url string, label string) {
	name, label, _ := strings.Cut(text, "|")
	name, heading, _ := strings.Cut(name, "#")
	noName := false
	if name == "" { // A link with a blank page links to the current page
		name = ct.currentPage.Name
		noName = true
	}
	if label == "" {
		if heading != "" {
			label = heading
			if noName {
				sl := ct.currentPage.SpecialLabel(heading)
				if sl != "" {
					label = sl
				} else {
					colon := strings.Index(heading, ":")
					if colon > 0 {
						sl = ct.currentPage.SpecialLabel(heading[:colon])
						if sl != "" {
							label = sl + ":" + heading[colon+1:]
						}
					}
				}
			}
		} else {
			label = name
		}
	}
	if pg := ct.pageByName(name); pg != nil {
		if heading != "" {
			return pg.URL + "#" + heading, label
		}
		return pg.URL, label
	}
	return "", ""
}

// open opens the page with the given URL and updates the display.
// It optionally adds the page to the history.
func (ct *Content) open(url string, history bool) {
	if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") {
		core.TheApp.OpenURL(url)
		return
	}
	if strings.HasPrefix(url, "ref://") {
		ct.openRef(url)
		return
	}
	url = strings.ReplaceAll(url, "/#", "#")
	url, heading, _ := strings.Cut(url, "#")
	pg := ct.pagesByURL[url]
	if pg == nil {
		// We want only the URL after the last slash for automatic redirects
		// (old URLs could have nesting).
		last := url
		if li := strings.LastIndex(url, "/"); li >= 0 {
			last = url[li+1:]
		}
		pg = ct.similarPage(last)
		if pg == nil {
			core.ErrorSnackbar(ct, errors.New("no pages available"))
		} else {
			core.MessageSnackbar(ct, fmt.Sprintf("Redirected from %s", url))
		}
	}
	heading = bcontent.SpecialToKebab(heading)
	ct.currentHeading = heading
	if ct.currentPage == pg {
		ct.openHeading(heading)
		return
	}
	ct.currentPage = pg
	if history {
		ct.addHistory(pg)
	}
	ct.Scene.Update() // need to update the whole scene to also update the toolbar
	// We can only scroll to the heading after the page layout has been updated, so we defer.
	ct.Defer(func() {
		ct.setStageTitle()
		ct.openHeading(heading)
	})
}

// openRef opens a ref:// reference url.
func (ct *Content) openRef(url string) {
	pg := ct.pagesByURL["references"]
	if pg == nil {
		core.MessageSnackbar(ct, "references page not generated, use mdcite in csl package")
		return
	}
	ref := strings.TrimPrefix(url, "ref://")
	ct.currentPage = pg
	ct.addHistory(pg)
	ct.Scene.Update()
	ct.Defer(func() {
		ct.setStageTitle()
		ct.openID(ref, "")
	})
}

func (ct *Content) openHeading(heading string) {
	if heading == "" {
		ct.rightFrame.ScrollDimToContentStart(math32.Y)
		return
	}
	idname := "" // in case of #id:element
	element := ""
	colon := strings.Index(heading, ":")
	if colon > 0 {
		idname = heading[:colon]
		element = heading[colon+1:]
	}
	tr := ct.tocNodes[strcase.ToKebab(heading)]
	if tr == nil {
		found := false
		if idname != "" && element != "" {
			found = ct.openID(idname, element)
			if !found {
				found = ct.openID(heading, "")
			}
		} else {
			found = ct.openID(heading, "")
		}
		if !found {
			errors.Log(fmt.Errorf("heading %q not found", heading))
		}
		return
	}
	tr.SelectEvent(events.SelectOne)
}

func (ct *Content) openID(id, element string) bool {
	if id == "" {
		ct.rightFrame.ScrollDimToContentStart(math32.Y)
		return true
	}
	var found *core.WidgetBase
	ct.rightFrame.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
		// if found != nil {
		// 	return tree.Break
		// }
		if cwb.Name != id {
			return tree.Continue
		}
		found = cwb
		return tree.Break
	})
	if found == nil {
		return false
	}
	if element != "" {
		// fmt.Println("looking for el:", element, "in:", found)
		el := ct.elementInSpecial(found, element)
		if el != nil {
			found = el
		}
	}
	found.SetFocus()
	found.SetState(true, states.Active)
	found.Style()
	found.NeedsRender()
	return true
}

// elementInSpecial looks for given element within a special item.
func (ct *Content) elementInSpecial(sp *core.WidgetBase, element string) *core.WidgetBase {
	pathPrefix := ""
	hasPath := false
	if strings.Contains(element, "/") {
		pathPrefix, element, hasPath = strings.Cut(element, "/")
	}
	if _, ok := sp.Parent.(*core.Collapser); ok { // for code
		sp = sp.Parent.AsTree().Parent.(core.Widget).AsWidget()
	}

	var found *core.WidgetBase
	sp.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
		if found != nil {
			return tree.Break
		}
		if !cwb.IsDisplayable() {
			return tree.Continue
		}
		if hasPath && !strings.Contains(cw.AsTree().Path(), pathPrefix) {
			return tree.Continue
		}
		label := labels.ToLabel(cw)
		if !strings.EqualFold(label, element) {
			return tree.Continue
		}
		if cwb.AbilityIs(abilities.Focusable) {
			found = cwb
			return tree.Break
		}
		next := core.AsWidget(tree.Next(cwb))
		if next.AbilityIs(abilities.Focusable) {
			found = next
			return tree.Break
		}
		return tree.Continue
	})
	return found
}
