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
	"cogentcore.org/core/colors"
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
	"cogentcore.org/core/text/textcore"
	"cogentcore.org/core/tree"
	"github.com/gomarkdown/markdown/ast"
)

// BindTextEditor is a function set to [cogentcore.org/core/yaegicore.BindTextEditor]
// when importing yaegicore, which provides interactive editing functionality for Go
// code blocks in text editors.
var BindTextEditor func(ed *textcore.Editor, parent *core.Frame, language string)

// handles the id attribute in htmlcore: needed for equation case
func (ct *Content) htmlIDAttributeHandler(ctx *htmlcore.Context, w io.Writer, node ast.Node, entering bool, tag, value string) bool {
	if ct.currentPage == nil {
		return false
	}
	lbl := ct.currentPage.SpecialLabel(value)
	if lbl == "" {
		return false
	}
	ch := node.GetChildren()
	if len(ch) == 2 { // image or table
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

func (ct *Content) htmlPreHandler(ctx *htmlcore.Context) bool {
	hasCode := ctx.Node.FirstChild != nil && ctx.Node.FirstChild.Data == "code"
	if !hasCode {
		return false
	}
	codeEl := ctx.Node.FirstChild
	collapsed := htmlcore.GetAttr(codeEl, "collapsed")
	lang := htmlcore.GetLanguage(htmlcore.GetAttr(codeEl, "class"))
	id := htmlcore.GetAttr(codeEl, "id")
	title := htmlcore.GetAttr(codeEl, "title")
	var ed *textcore.Editor
	parent := ctx.Parent().AsWidget()
	fr := core.NewFrame(parent.This)
	fr.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 0)
		s.Direction = styles.Column
	})
	fr.SetProperty("paginate-block", true) // no split
	if id != "" {
		fr.SetProperty("id", id)
		fr.SetName(id)
		ttx := parent.Children[parent.NumChildren()-2].(core.Widget)
		ttx.AsWidget().SetProperty("id", id) // link target
		tree.MoveToParent(ttx, fr)           // get title text
	}
	if collapsed != "" {
		cl := core.NewCollapser(fr)
		core.NewText(cl.Summary).SetText("Code").SetText(title)
		ed = textcore.NewEditor(cl.Details)
		if collapsed == "false" || collapsed == "-" {
			cl.Open = true
		}
	} else {
		ed = textcore.NewEditor(fr)
	}
	ctx.Node = codeEl
	if lang != "" {
		ed.Lines.SetFileExt(lang)
	}
	ed.Lines.SetString(htmlcore.ExtractText(ctx))
	if BindTextEditor != nil && (lang == "Go" || lang == "Goal") {
		ed.Lines.SpacesToTabs(0, ed.Lines.NumLines()) // Go uses tabs
		parFrame := core.NewFrame(fr)
		parFrame.Styler(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Grow.Set(1, 0)
		})
		// we inherit our Grow.Y from our first child so that
		// elements that want to grow can do so
		parFrame.SetOnChildAdded(func(n tree.Node) {
			if _, ok := n.(*core.Body); ok { // Body should not grow
				return
			}
			wb := core.AsWidget(n)
			if wb.IndexInParent() != 0 {
				return
			}
			wb.FinalStyler(func(s *styles.Style) {
				parFrame.Styles.Grow.Y = s.Grow.Y
			})
		})
		BindTextEditor(ed, parFrame, lang)
	} else {
		ed.SetReadOnly(true)
		ed.Lines.Settings.LineNumbers = false
		ed.Styler(func(s *styles.Style) {
			s.Border.Width.Zero()
			s.MaxBorder.Width.Zero()
			s.StateLayer = 0
			s.Background = colors.Scheme.SurfaceContainer
		})
	}
	return true
}

// widgetHandler is htmlcore widget handler for adding our own actions etc.
func (ct *Content) widgetHandler(w core.Widget) {
	tag := ""
	id := ""
	title := ""
	wb := w.AsWidget()
	if t, ok := wb.Properties["tag"]; ok {
		tag = t.(string)
	}
	if t, ok := wb.Properties["id"]; ok {
		id = t.(string)
	}
	if t, ok := wb.Properties["title"]; ok {
		title = t.(string)
	}
	switch x := w.(type) {
	case *core.Text:
		hdr := len(tag) > 0 && tag[0] == 'h'
		x.Styler(func(s *styles.Style) {
			if tag == "td" {
				s.Margin.SetVertical(units.Em(0)) // use gap
			} else {
				s.Margin.SetVertical(units.Em(core.ConstantSpacing(0.25)))
			}
			s.Font.Size.Value *= core.AppearanceSettings.DocsFontSize / 100
			s.Max.X.In(8) // big enough to not constrain PDF render
			if hdr {
				x.SetProperty("paginate-no-break-after", true)
			}
		})
	case *core.Image:
		ct.widgetHandlerFigure(w, id)
		x.OnDoubleClick(func(e events.Event) {
			d := core.NewBody("Image")
			core.NewImage(d).SetImage(x.Image)
			d.RunWindowDialog(x)
		})
	case *core.SVG:
		ct.widgetHandlerFigure(w, id)
		x.OnDoubleClick(func(e events.Event) {
			d := core.NewBody("SVG")
			sv := core.NewSVG(d)
			sv.SVG = x.SVG
			d.RunWindowDialog(x)
		})
	case *core.Frame:
		switch tag {
		case "table":
			if id != "" {
				lbl := ct.currentPage.SpecialLabel(id)
				cp := "<b>" + lbl + ":</b>"
				if title != "" {
					cp += " " + title
				}
				ct.moveToBlockFrame(w, id, cp, true)
			}
			x.Styler(func(s *styles.Style) {
				s.Align.Self = styles.Center
			})
		}
	}
}

// moveToBlockFrame moves given widget into a block frame with given text
// widget either at the top or bottom of the new frame.
func (ct *Content) moveToBlockFrame(w core.Widget, id, txt string, top bool) {
	wb := w.AsWidget()
	fr := core.NewFrame(wb.Parent)
	fr.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 0)
		s.Direction = styles.Column
	})
	fr.SetProperty("paginate-block", true) // no split
	fr.SetProperty("id", id)
	fr.SetName(id)
	var tx *core.Text
	if top {
		tx = core.NewText(fr).SetText(txt)
		tx.SetProperty("id", id) // good link destination
	}
	tree.MoveToParent(w, fr)
	if !top {
		tx = core.NewText(fr).SetText(txt)
		wb.SetProperty("id", id) // link here
	}
	tx.Styler(func(s *styles.Style) {
		s.Max.X.In(8)
		s.Font.Size.Value *= core.AppearanceSettings.DocsFontSize / 100
	})
}

func (ct *Content) widgetHandlerFigure(w core.Widget, id string) {
	wb := w.AsWidget()
	fig := false
	alt := ""
	if p, ok := wb.Properties["alt"]; ok {
		alt = p.(string)
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
	altf := htmlcore.MDToHTML(ct.Context, []byte(alt))
	lbl := ct.currentPage.SpecialLabel(id)
	lbf := "<b>" + lbl + ":</b> " + string(altf) + " <br><br> "
	ct.moveToBlockFrame(w, id, lbf, false)
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
	if ct.inPDFRender {
		url = "#" + ref
	} else {
		url = "ref://" + ref
	}
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
	if ct.inPDFRender {
		if heading != "" {
			if pg == ct.currentPage {
				return "#" + heading, label
			}
			return ct.getPrintURL() + "/" + pg.URL + "#" + heading, label
		}
		return ct.getPrintURL() + "/" + pg.URL, label
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
