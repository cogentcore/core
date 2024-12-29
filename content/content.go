// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package content provides a system for making content-focused
// apps and websites consisting of Markdown, HTML, and Cogent Core.
package content

//go:generate core generate

import (
	"bytes"
	"fmt"
	"io/fs"
	"strconv"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/content/bcontent"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/system"
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
	// used for displaying the table of contents.
	leftFrame *core.Frame

	// tocNodes are all of the tree nodes in the table of contents
	// by lowercase heading name.
	tocNodes map[string]*core.Tree
}

func init() {
	// We want Command+[ and Command+] to work for browser back/forward navigation
	// in content, since we rely on that. They should still be intercepted by
	// Cogent Core for non-content apps for things such as full window dialogs,
	// so we only add these in content.
	system.ReservedWebShortcuts = append(system.ReservedWebShortcuts, "Command+[", "Command+]")
}

func (ct *Content) Init() {
	ct.Splits.Init()
	ct.SetSplits(0.2, 0.8)

	ct.Context = htmlcore.NewContext()
	ct.Context.OpenURL = func(url string) {
		ct.Open(url)
	}
	ct.Context.AddWikilinkHandler(func(text string) (url string, label string) {
		name, label, _ := strings.Cut(text, "|")
		name, heading, _ := strings.Cut(name, "#")
		if name == "" { // A link with a blank page links to the current page
			name = ct.currentPage.Name
		}
		if label == "" {
			if heading != "" {
				label = heading
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
	})
	ct.Context.ElementHandlers["embed-page"] = func(ctx *htmlcore.Context) bool {
		errors.Log(ct.embedPage(ctx))
		return true
	}

	ct.Maker(func(p *tree.Plan) {
		if ct.currentPage == nil {
			return
		}
		tree.Add(p, func(w *core.Frame) {
			ct.leftFrame = w
		})
		tree.Add(p, func(w *core.Frame) {
			w.Maker(func(p *tree.Plan) {
				if ct.currentPage.Name != "" {
					tree.Add(p, func(w *core.Text) {
						w.SetType(core.TextDisplaySmall)
						w.Updater(func() {
							w.SetText(ct.currentPage.Name)
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
	return ct.pagesByName[strings.ToLower(name)]
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
		if path == "" || path == "." {
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

// open opens the page with the given URL and updates the display.
// It optionally adds the page to the history.
func (ct *Content) open(url string, history bool) {
	if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") {
		core.TheApp.OpenURL(url)
		return
	}
	url, heading, _ := strings.Cut(url, "#")
	pg, ok := ct.pagesByURL[url]
	if !ok {
		core.TheApp.OpenURL(url)
		return
	}
	if ct.currentPage == pg {
		ct.openHeading(heading)
		return
	}
	ct.currentPage = pg
	if history {
		ct.historyIndex = len(ct.history)
		ct.history = append(ct.history, pg)
		ct.saveWebURL()
	}
	ct.Scene.Update() // need to update the whole scene to also update the toolbar
	// We can only scroll to the heading after the page layout has been updated, so we defer.
	ct.Defer(func() {
		ct.setStageTitle()
		ct.openHeading(heading)
	})
}

func (ct *Content) openHeading(heading string) {
	if heading == "" {
		return
	}
	tr := ct.tocNodes[strings.ToLower(heading)]
	if tr == nil {
		errors.Log(fmt.Errorf("heading %q not found", heading))
		return
	}
	tr.SelectEvent(events.SelectOne)
}

// loadPage loads the current page content into the given frame if it is not already loaded.
func (ct *Content) loadPage(w *core.Frame) error {
	if ct.renderedPage == ct.currentPage {
		return nil
	}
	w.DeleteChildren()
	b, err := ct.currentPage.ReadContent(ct.pagesByCategory)
	if err != nil {
		return err
	}
	err = htmlcore.ReadMD(ct.Context, w, b)
	if err != nil {
		return err
	}

	w.ScrollDimToContentStart(math32.Y)
	ct.leftFrame.DeleteChildren()
	ct.makeTableOfContents(w)
	ct.makeCategories()
	ct.leftFrame.Update()
	ct.renderedPage = ct.currentPage
	return nil
}

// makeTableOfContents makes the table of contents and adds it to [Content.leftFrame]
// based on the headings in the given frame.
func (ct *Content) makeTableOfContents(w *core.Frame) {
	ct.tocNodes = map[string]*core.Tree{}
	contents := core.NewTree(ct.leftFrame).SetText("<b>Contents</b>")
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
			ct.tocNodes[strings.ToLower(tx.Text)] = tr
			tr.OnSelect(func(e events.Event) {
				tx.ScrollToThis()
			})
		}
		return tree.Continue
	})
}

// makeCategories makes the categories tree for the current page and adds it to [Content.leftFrame].
func (ct *Content) makeCategories() {
	cats := core.NewTree(ct.leftFrame).SetText("<b>Categories</b>")
	for _, cat := range ct.currentPage.Categories {
		catTree := core.NewTree(cats).SetText(cat)
		catTree.OnSelect(func(e events.Event) {
			if catPage := ct.pageByName(cat); catPage != nil {
				ct.Open(catPage.URL)
			}
		})
		for _, pg := range ct.pagesByCategory[cat] {
			if pg == ct.currentPage {
				continue
			}
			pgTree := core.NewTree(catTree).SetText(pg.Name)
			pgTree.OnSelect(func(e events.Event) {
				ct.Open(pg.URL)
			})
		}
	}
}

// embedPage handles an <embed-page> element by embedding the lead section
// (content before the first heading) into the current page, with a
// *Main page: [[Name]]* link added at the start as well. The name of the
// embedded page is the src attribute of the current html element.
func (ct *Content) embedPage(ctx *htmlcore.Context) error {
	src := htmlcore.GetAttr(ctx.Node, "src")
	if src == "" {
		return fmt.Errorf("missing src attribute in <embed-page>")
	}
	pg := ct.pageByName(src)
	if pg == nil {
		return fmt.Errorf("page %q not found", src)
	}
	b, err := pg.ReadContent(ct.pagesByCategory)
	if err != nil {
		return err
	}
	lead, _, _ := bytes.Cut(b, []byte("\n#"))
	res := append([]byte(fmt.Sprintf("*Main page: [[%s]]*", pg.Name)), lead...)
	return htmlcore.ReadMD(ctx, ctx.BlockParent, res)
}

// setStageTitle sets the title of the stage based on the current page URL.
func (ct *Content) setStageTitle() {
	if rw := ct.Scene.RenderWindow(); rw != nil {
		name := ct.currentPage.Name
		if ct.currentPage.URL == "" { // Root page just gets app name
			name = core.TheApp.Name()
		}
		rw.SetStageTitle(name)
	}
}

func (ct *Content) MakeToolbar(p *tree.Plan) {
	tree.Add(p, func(w *core.Button) {
		w.SetIcon(icons.Icon(core.AppIcon))
		w.SetTooltip("Home")
		w.OnClick(func(e events.Event) {
			ct.Open("")
		})
	})
	// Superseded by browser navigation on web.
	if core.TheApp.Platform() != system.Web {
		tree.Add(p, func(w *core.Button) {
			w.SetIcon(icons.ArrowBack).SetKey(keymap.HistPrev)
			w.SetTooltip("Back")
			w.Updater(func() {
				w.SetEnabled(ct.historyIndex > 0)
			})
			w.OnClick(func(e events.Event) {
				ct.historyIndex--
				ct.open(ct.history[ct.historyIndex].URL, false) // do not add to history while navigating history
			})
		})
		tree.Add(p, func(w *core.Button) {
			w.SetIcon(icons.ArrowForward).SetKey(keymap.HistNext)
			w.SetTooltip("Forward")
			w.Updater(func() {
				w.SetEnabled(ct.historyIndex < len(ct.history)-1)
			})
			w.OnClick(func(e events.Event) {
				ct.historyIndex++
				ct.open(ct.history[ct.historyIndex].URL, false) // do not add to history while navigating history
			})
		})
	}
	tree.Add(p, func(w *core.Button) {
		w.SetIcon(icons.Search).SetKey(keymap.Menu)
		w.SetTooltip("Search")
		w.OnClick(func(e events.Event) {
			ct.Scene.MenuSearchDialog("Search", "Search "+core.TheApp.Name())
		})
	})
}

func (ct *Content) MenuSearch(items *[]core.ChooserItem) {
	newItems := make([]core.ChooserItem, len(ct.pages))
	for i, pg := range ct.pages {
		newItems[i] = core.ChooserItem{
			Value: pg,
			Text:  pg.Name,
			Icon:  icons.Article,
			Func: func() {
				ct.Open(pg.URL)
			},
		}
	}
	*items = append(newItems, *items...)
}
