// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package content

import (
	"fmt"
	"io"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/labels"
	"cogentcore.org/core/base/strcase"
	"cogentcore.org/core/content/bcontent"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/csl"
	"cogentcore.org/core/tree"
	"github.com/gomarkdown/markdown/ast"
)

// handles the id attribute in htmlcore -- deals with all non-image id cases
func (ct *Content) htmlIDAttributeHandler(ctx *htmlcore.Context, w io.Writer, node ast.Node, entering bool, tag, value string) bool {
	if ct.currentPage == nil {
		return false
	}
	lbl := ct.currentPage.SpecialLabel(value)
	ch := node.GetChildren()
	if len(ch) == 2 { // image or table
		if entering {
			return false
		}
		if _, ok := ch[1].(*ast.Image); ok {
			return false
		}
		cp := "\n<p><b>" + lbl + ":</b>"
		title := htmlcore.MDGetAttr(node, "title")
		if title != "" {
			cp += " " + title
		}
		cp += "</p>\n"
		w.Write([]byte(cp))
		return false
	}
	if entering {
		cp := "\n<span id=\"" + value + "\"><b>" + lbl + ":</b>"
		title := htmlcore.MDGetAttr(node, "title")
		if title != "" {
			cp += " " + title
		}
		cp += "</span>\n"
		w.Write([]byte(cp))
		// fmt.Println("id:", value, lbl)
		// fmt.Printf("%#v\n", node)
	}
	return false
}

// widgetHandler is htmlcore widget handler for adding our own actions etc.
func (ct *Content) widgetHandler(w core.Widget) {
	switch x := w.(type) {
	case *core.Text:
		hdr := false
		if t, ok := x.Properties["tag"]; ok {
			if len(t.(string)) > 0 && (t.(string))[0] == 'h' {
				hdr = true
			}
		}
		x.Styler(func(s *styles.Style) {
			s.Margin.SetVertical(units.Em(core.ConstantSpacing(0.25)))
			s.Font.Size.Value *= core.AppearanceSettings.DocsFontSize / 100
			s.Max.X.In(8) // big enough to not constrain PDF render
			if hdr {
				x.SetProperty("paginate-no-break-after", true)
			}
		})
	case *core.Image:
		ct.widgetHandlerFigure(w)
		x.OnDoubleClick(func(e events.Event) {
			d := core.NewBody("Image")
			core.NewImage(d).SetImage(x.Image)
			d.RunWindowDialog(x)
		})
	case *core.SVG:
		ct.widgetHandlerFigure(w)
		x.OnDoubleClick(func(e events.Event) {
			d := core.NewBody("SVG")
			sv := core.NewSVG(d)
			sv.SVG = x.SVG
			d.RunWindowDialog(x)
		})
	case *core.Frame:
		table := false
		if t, ok := x.Properties["tag"]; ok {
			table = (t.(string) == "table")
		}
		if table {
			x.Styler(func(s *styles.Style) {
				s.Align.Self = styles.Center
			})
		}
	}
}

func (ct *Content) widgetHandlerFigure(w core.Widget) {
	wb := w.AsWidget()
	fig := false
	alt := ""
	id := ""
	if p, ok := wb.Properties["alt"]; ok {
		alt = p.(string)
	}
	if p, ok := wb.Properties["id"]; ok {
		id = p.(string)
	}
	if alt != "" && id != "" {
		fig = true
	}
	wb.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Clickable, abilities.DoubleClickable)
		s.Overflow.Set(styles.OverflowAuto)
		if fig {
			s.Align.Self = styles.Center
		}
	})
	if !fig {
		return
	}
	fr := core.NewFrame(wb.Parent)
	fr.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 0)
		s.Direction = styles.Column
	})
	fr.SetProperty("paginate-block", true) // no split
	fr.SetProperty("id", id)
	tree.MoveToParent(w, fr)
	fr.SetName(id)
	lbl := ct.currentPage.SpecialLabel(id)
	lbf := "<b>" + lbl + ":</b> "
	tx := core.NewText(fr).SetText(lbf + alt + "<br> <br> ")
	tx.Styler(func(s *styles.Style) {
		s.Max.X.In(8) // big enough to not constrain PDF render
		s.Font.Size.Value *= core.AppearanceSettings.DocsFontSize / 100
	})
}

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
	if name == "" { // A link with a blank page links to the current page
		name = ct.currentPage.Name
	} else if heading == "" {
		if pg := ct.pageByName(name); pg == ct.currentPage {
			// if just a link to current page, don't render link
			// this can happen for embedded pages that refer to embedder
			return "", ""
		}
	}
	pg := ct.pageByName(name)
	if pg == nil {
		return "", ""
	}
	if label == "" {
		if heading != "" {
			label = ct.wikilinkLabel(pg, heading)
		} else {
			label = name
		}
	}
	if heading != "" {
		return pg.URL + "#" + heading, label
	}
	return pg.URL, label
}

// wikilinkLabel returns a label for given heading, for given page.
func (ct *Content) wikilinkLabel(pg *bcontent.Page, heading string) string {
	label := heading
	sl := pg.SpecialLabel(heading)
	if sl != "" {
		label = sl
	} else {
		colon := strings.Index(heading, ":")
		if colon > 0 {
			sl = pg.SpecialLabel(heading[:colon])
			if sl != "" {
				label = sl + ":" + heading[colon+1:]
			}
		}
	}
	return label
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
	if cl, ok := sp.Parent.(*core.Collapser); ok { // for code
		nxt := tree.NextSibling(cl)
		if nxt != nil {
			sp = nxt.(core.Widget).AsWidget()
		} else {
			sp = cl.Parent.(core.Widget).AsWidget() // todo: not sure when this is good
		}
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
