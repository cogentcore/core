// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package content provides a system for making content-focused
// apps and websites consisting of Markdown, HTML, and Cogent Core.
package content

//go:generate core generate

import (
	"bytes"
	"cmp"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/gomarkdown/markdown/ast"
	"golang.org/x/exp/maps"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/base/strcase"
	"cogentcore.org/core/content/bcontent"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/system"
	"cogentcore.org/core/text/csl"
	"cogentcore.org/core/tree"
)

// Content manages and displays the content of a set of pages.
type Content struct {
	core.Splits

	// Source is the source filesystem for the content.
	// It should be set using [Content.SetSource] or [Content.SetContent].
	Source fs.FS `set:"-"`

	// Context is the [htmlcore.Context] used to render the content,
	// which can be modified for things such as adding wikilink handlers.
	Context *htmlcore.Context `set:"-"`

	// References is a list of references used for generating citation text
	// for literature reference wikilinks in the format [[@CiteKey]].
	References *csl.KeyList

	// pages are the pages that constitute the content.
	pages []*bcontent.Page

	// pagesByName has the [bcontent.Page] for each [bcontent.Page.Name]
	// transformed into lowercase. See [Content.pageByName] for a helper
	// function that automatically transforms into lowercase.
	pagesByName map[string]*bcontent.Page

	// pagesByURL has the [bcontent.Page] for each [bcontent.Page.URL].
	pagesByURL map[string]*bcontent.Page

	// pagesByCategory has the [bcontent.Page]s for each of all [bcontent.Page.Categories].
	pagesByCategory map[string][]*bcontent.Page

	// categories has all unique [bcontent.Page.Categories], sorted such that the categories
	// with the most pages are listed first. "Other" is always last, and is used for pages that
	// do not have a category, unless they are a category themselves.
	categories []string

	// history is the history of pages that have been visited.
	// The oldest page is first.
	history []*bcontent.Page

	// historyIndex is the current position in [Content.history].
	historyIndex int

	// currentPage is the currently open page.
	currentPage *bcontent.Page

	// renderedPage is the most recently rendered page.
	renderedPage *bcontent.Page

	// leftFrame is the frame on the left side of the widget,
	// used for displaying the table of contents and the categories.
	leftFrame *core.Frame

	// rightFrame is the frame on the right side of the widget,
	// used for displaying the page content.
	rightFrame *core.Frame

	// tocNodes are all of the tree nodes in the table of contents
	// by kebab-case heading name.
	tocNodes map[string]*core.Tree

	// currentHeading is the currently selected heading in the table of contents,
	// if any (in kebab-case).
	currentHeading string

	// The previous and next page, if applicable. They must be stored on this struct
	// to avoid stale local closure variables.
	prevPage, nextPage *bcontent.Page
}

func init() {
	// We want Command+[ and Command+] to work for browser back/forward navigation
	// in content, since we rely on that. They should still be intercepted by
	// Cogent Core for non-content apps for things such as full window dialogs,
	// so we only add these in content.
	system.ReservedWebShortcuts = append(system.ReservedWebShortcuts, "Command+[", "Command+]")
}

// NewPageInitFunc is called when a new page is just being rendered.
// This can do any necessary new-page initialization, e.g.,
// [yaegicore.ResetGoalInterpreter]
var NewPageInitFunc func()

func (ct *Content) Init() {
	ct.Splits.Init()
	ct.SetSplits(0.2, 0.8)

	ct.Context = htmlcore.NewContext()
	ct.Context.OpenURL = func(url string) {
		ct.Open(url)
	}
	ct.Context.GetURL = func(url string) (*http.Response, error) {
		return htmlcore.GetURLFromFS(ct.Source, url)
	}
	ct.Context.AddWikilinkHandler(ct.citeWikilink)
	ct.Context.AddWikilinkHandler(ct.mainWikilink)
	ct.Context.ElementHandlers["embed-page"] = func(ctx *htmlcore.Context) bool {
		errors.Log(ct.embedPage(ctx))
		return true
	}
	ct.Context.AttributeHandlers["id"] = func(ctx *htmlcore.Context, w io.Writer, node ast.Node, entering bool, tag, value string) bool {
		if ct.currentPage == nil {
			return false
		}
		lbl := ct.currentPage.SpecialLabel(value)
		ch := node.GetChildren()
		if len(ch) == 2 { // image or table
			if entering {
				sty := htmlcore.MDGetAttr(node, "style")
				if sty != "" {
					if img, ok := ch[1].(*ast.Image); ok {
						htmlcore.MDSetAttr(img, "style", sty)
						delete(node.AsContainer().Attribute.Attrs, "style")
					}
				}
				return false
			}
			cp := "\n<p><b>" + lbl + ":</b>"
			if img, ok := ch[1].(*ast.Image); ok {
				// fmt.Printf("Image: %s\n", string(img.Destination))
				// fmt.Printf("Image: %#v\n", img)
				nc := len(img.Children)
				if nc > 0 {
					if txt, ok := img.Children[0].(*ast.Text); ok {
						// fmt.Printf("text: %s\n", string(txt.Literal)) // not formatted!
						cp += " " + string(txt.Literal) // todo: not formatted!
					}
				}
			} else {
				title := htmlcore.MDGetAttr(node, "title")
				if title != "" {
					cp += " " + title
				}
			}
			cp += "</p>\n"
			w.Write([]byte(cp))
		} else if entering {
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
	ct.Context.AddWidgetHandler(func(w core.Widget) {
		switch x := w.(type) {
		case *core.Text:
			x.Styler(func(s *styles.Style) {
				s.Max.X.Ch(120)
			})
		case *core.Image:
			x.Styler(func(s *styles.Style) {
				s.SetAbilities(true, abilities.Clickable, abilities.DoubleClickable)
				s.Overflow.Set(styles.OverflowAuto)
			})
			x.OnDoubleClick(func(e events.Event) {
				d := core.NewBody("Image")
				core.NewImage(d).SetImage(x.Image)
				d.RunWindowDialog(x)
			})
		}
	})

	ct.Maker(func(p *tree.Plan) {
		if ct.currentPage == nil {
			return
		}
		tree.Add(p, func(w *core.Frame) {
			ct.leftFrame = w
		})
		tree.Add(p, func(w *core.Frame) {
			ct.rightFrame = w
			w.Styler(func(s *styles.Style) {
				switch w.SizeClass() {
				case core.SizeCompact, core.SizeMedium:
					s.Padding.SetHorizontal(units.Em(0.5))
				case core.SizeExpanded:
					s.Padding.SetHorizontal(units.Em(3))
				}
			})
			w.Maker(func(p *tree.Plan) {
				if ct.currentPage.Title != "" {
					tree.Add(p, func(w *core.Text) {
						w.SetType(core.TextDisplaySmall)
						w.Updater(func() {
							w.SetText(ct.currentPage.Title)
						})
					})
				}
				if len(ct.currentPage.Authors) > 0 {
					tree.Add(p, func(w *core.Text) {
						w.SetType(core.TextTitleLarge)
						w.Updater(func() {
							w.SetText("By " + strcase.FormatList(ct.currentPage.Authors...))
						})
					})
				}
				if !ct.currentPage.Date.IsZero() {
					tree.Add(p, func(w *core.Text) {
						w.SetType(core.TextTitleMedium)
						w.Updater(func() {
							w.SetText(ct.currentPage.Date.Format("January 2, 2006"))
						})
					})
				}
				tree.Add(p, func(w *core.Frame) {
					w.Styler(func(s *styles.Style) {
						s.Direction = styles.Column
						s.Grow.Set(1, 1)
					})
					w.Updater(func() {
						errors.Log(ct.loadPage(w))
					})
				})
				ct.makeBottomButtons(p)
			})
		})
	})

	// Must be done after the default title is set elsewhere in normal OnShow
	ct.OnFinal(events.Show, func(e events.Event) {
		ct.setStageTitle()
	})
	ct.handleWebPopState()
}

// pageByName returns [Content.pagesByName] of the lowercase version of the given name.
func (ct *Content) pageByName(name string) *bcontent.Page {
	ln := strings.ToLower(name)
	if pg, ok := ct.pagesByName[ln]; ok {
		return pg
	}
	nd := strings.ReplaceAll(ln, "-", " ")
	if pg, ok := ct.pagesByName[nd]; ok {
		return pg
	}
	return nil
}

// SetSource sets the source filesystem for the content.
func (ct *Content) SetSource(source fs.FS) *Content {
	ct.Source = source
	ct.pages = []*bcontent.Page{}
	ct.pagesByName = map[string]*bcontent.Page{}
	ct.pagesByURL = map[string]*bcontent.Page{}
	ct.pagesByCategory = map[string][]*bcontent.Page{}
	errors.Log(fs.WalkDir(ct.Source, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if path == "" || path == "." {
			return nil
		}
		ext := filepath.Ext(path)
		if !(ext == ".md" || ext == ".html") {
			return nil
		}
		pg, err := bcontent.NewPage(ct.Source, path)
		if err != nil {
			return err
		}
		ct.pages = append(ct.pages, pg)
		ct.pagesByName[strings.ToLower(pg.Name)] = pg
		ct.pagesByURL[pg.URL] = pg
		for _, cat := range pg.Categories {
			ct.pagesByCategory[cat] = append(ct.pagesByCategory[cat], pg)
		}
		return nil
	}))
	ct.categories = maps.Keys(ct.pagesByCategory)
	slices.SortFunc(ct.categories, func(a, b string) int {
		if a == "Other" {
			return 1
		}
		if b == "Other" {
			return -1
		}
		v := cmp.Compare(len(ct.pagesByCategory[b]), len(ct.pagesByCategory[a]))
		if v != 0 {
			return v
		}
		return cmp.Compare(a, b)
	})
	// Pages that are a category are already represented in the categories tree,
	// so they do not belong in the "Other" category.
	ct.pagesByCategory["Other"] = slices.DeleteFunc(ct.pagesByCategory["Other"], func(pg *bcontent.Page) bool {
		_, isCategory := ct.pagesByCategory[pg.Name]
		if isCategory {
			pg.Categories = slices.DeleteFunc(pg.Categories, func(c string) bool {
				return c == "Other"
			})
		}
		return isCategory
	})

	if url := ct.getWebURL(); url != "" {
		ct.Open(url)
		return ct
	}
	if root, ok := ct.pagesByURL[""]; ok {
		ct.Open(root.URL)
		return ct
	}
	ct.Open(ct.pages[0].URL)
	return ct
}

// SetContent is a helper function that calls [Content.SetSource]
// with the "content" subdirectory of the given filesystem.
func (ct *Content) SetContent(content fs.FS) *Content {
	return ct.SetSource(fsx.Sub(content, "content"))
}

// Open opens the page with the given URL and updates the display.
// If no pages correspond to the URL, it is opened in the default browser.
func (ct *Content) Open(url string) *Content {
	ct.open(url, true)
	return ct
}

func (ct *Content) addHistory(pg *bcontent.Page) {
	ct.historyIndex = len(ct.history)
	ct.history = append(ct.history, pg)
	ct.saveWebURL()
}

// loadPage loads the current page content into the given frame if it is not already loaded.
func (ct *Content) loadPage(w *core.Frame) error {
	if ct.renderedPage == ct.currentPage {
		return nil
	}
	if NewPageInitFunc != nil {
		NewPageInitFunc()
	}
	w.DeleteChildren()
	b, err := ct.currentPage.ReadContent(ct.pagesByCategory)
	if err != nil {
		return err
	}
	ct.currentPage.ParseSpecials(b)
	err = htmlcore.ReadMD(ct.Context, w, b)
	if err != nil {
		return err
	}

	ct.leftFrame.DeleteChildren()
	ct.makeTableOfContents(w, ct.currentPage)
	ct.makeCategories()
	ct.leftFrame.Update()
	ct.renderedPage = ct.currentPage
	return nil
}

// makeTableOfContents makes the table of contents and adds it to [Content.leftFrame]
// based on the headings in the given frame.
func (ct *Content) makeTableOfContents(w *core.Frame, pg *bcontent.Page) {
	ct.tocNodes = map[string]*core.Tree{}
	contents := core.NewTree(ct.leftFrame).SetText("<b>Contents</b>")
	contents.SetReadOnly(true)
	contents.OnSelect(func(e events.Event) {
		if contents.IsRootSelected() {
			ct.rightFrame.ScrollDimToContentStart(math32.Y)
			ct.currentHeading = ""
			ct.saveWebURL()
		}
	})
	// last is the most recent tree node for each heading level, used for nesting.
	last := map[int]*core.Tree{}
	w.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
		tx, ok := cw.(*core.Text)
		if !ok {
			return tree.Continue
		}
		tag := tx.Property("tag")
		switch tag {
		case "h1", "h2", "h3", "h4", "h5", "h6":
			num := errors.Log1(strconv.Atoi(tag.(string)[1:]))
			parent := contents
			// Our parent is the last heading with a lower level (closer to h1).
			for i := num - 1; i >= 1; i-- {
				if last[i] != nil {
					parent = last[i]
					break
				}
			}
			tr := core.NewTree(parent).SetText(tx.Text)
			last[num] = tr
			kebab := strcase.ToKebab(tr.Text)
			ct.tocNodes[kebab] = tr
			tr.OnSelect(func(e events.Event) {
				tx.ScrollThisToTop()
				ct.currentHeading = kebab
				ct.saveWebURL()
			})
		}
		return tree.Continue
	})

	if contents.NumChildren() == 0 {
		contents.Delete()
	}
}

// makeCategories makes the categories tree for the current page and adds it to [Content.leftFrame].
func (ct *Content) makeCategories() {
	if len(ct.categories) == 0 {
		return
	}

	cats := core.NewTree(ct.leftFrame).SetText("<b>Categories</b>")
	cats.SetReadOnly(true)
	cats.OnSelect(func(e events.Event) {
		if cats.IsRootSelected() {
			ct.Open("")
		}
	})
	for _, cat := range ct.categories {
		catTree := core.NewTree(cats).SetText(cat).SetClosed(true)
		if ct.currentPage.Name == cat {
			catTree.SetSelected(true)
		}
		catTree.OnSelect(func(e events.Event) {
			if catPage := ct.pageByName(cat); catPage != nil {
				ct.Open(catPage.URL)
			} else {
				catTree.Open() // no page to open so open the tree
			}
		})
		for _, pg := range ct.pagesByCategory[cat] {
			if strings.EqualFold(cat, pg.Name) {
				continue
			}
			pgTree := core.NewTree(catTree).SetText(pg.Name)
			if pg == ct.currentPage {
				pgTree.SetSelected(true)
				catTree.SetClosed(false)
			}
			pgTree.OnSelect(func(e events.Event) {
				ct.Open(pg.URL)
			})
		}
	}
}

// embedPage handles an <embed-page> element by embedding the lead section
// (content before the first heading) into the current page, with a heading
// and a *Main page: [[Name]]* link added at the start as well. The name of
// the embedded page is the case-insensitive src attribute of the current
// html element. A title attribute may also be specified to override the
// heading text.
func (ct *Content) embedPage(ctx *htmlcore.Context) error {
	src := htmlcore.GetAttr(ctx.Node, "src")
	if src == "" {
		return fmt.Errorf("missing src attribute in <embed-page>")
	}
	pg := ct.pageByName(src)
	if pg == nil {
		return fmt.Errorf("page %q not found in <embed-page>", src)
	}
	title := htmlcore.GetAttr(ctx.Node, "title")
	if title == "" {
		title = pg.Name
	}
	b, err := pg.ReadContent(ct.pagesByCategory)
	if err != nil {
		return err
	}
	lead, _, _ := bytes.Cut(b, []byte("\n#"))
	heading := fmt.Sprintf("## %s\n\n*Main page: [[%s]]*\n\n", title, pg.Name)
	res := append([]byte(heading), lead...)
	return htmlcore.ReadMD(ctx, ctx.BlockParent, res)
}

// setStageTitle sets the title of the stage based on the current page URL.
func (ct *Content) setStageTitle() {
	if rw := ct.Scene.RenderWindow(); rw != nil && ct.currentPage != nil {
		name := ct.currentPage.Name
		if ct.currentPage.URL == "" { // Root page just gets app name
			name = core.TheApp.Name()
		}
		rw.SetStageTitle(name)
	}
}
