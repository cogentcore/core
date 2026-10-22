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
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"golang.org/x/exp/maps"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/base/strcase"
	"cogentcore.org/core/content/bcontent"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/system"
	"cogentcore.org/core/text/csl"
	"cogentcore.org/core/text/paginate"
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

	// history is the history of pages that have been visited, per tab.
	history []*History

	// current is the current location, in current tab.
	current Location

	// rendered is the most recently rendered location, in current tab.
	rendered Location

	// leftFrame is the frame on the left side of the widget,
	// used for displaying the table of contents and the categories.
	leftFrame *core.Frame

	// rightFrame is the frame on the right side of the widget,
	// used for displaying the page content.
	rightFrame *core.Frame

	// tabs are the tabs, only for non-web
	tabs *core.Tabs

	// top toolbar, if present
	toolbar *core.Toolbar

	// tocNodes are all of the tree nodes in the table of contents
	// by kebab-case heading name.
	tocNodes map[string]*core.Tree

	// tocTree is the table of contents tree of the current page in [Content.leftFrame].
	tocTree *core.Tree

	// categoriesTree is the categories tree in [Content.leftFrame]. It is made once
	// and kept between pages; [Content.updateCategories] updates it for each page.
	categoriesTree *core.Tree

	// categoryTrees are the tree nodes of [Content.categories] in [Content.categoriesTree].
	categoryTrees []*core.Tree

	// categoryPages are the pages of the page nodes in [Content.categoriesTree].
	categoryPages map[*core.Tree]*bcontent.Page

	// inPDFRender indicates that it is rendering a PDF now, turning off
	// elements that are not appropriate for that.
	inPDFRender bool

	// The previous and next page, if applicable. They must be stored on this struct
	// to avoid stale local closure variables.
	prevPage, nextPage *bcontent.Page

	// if non-nil, a message is sent when content page is rendered
	renderChan chan struct{}

	// loads counts the pages loaded by [Content.loadPage], so that the rest of
	// a staged page (see [Content.loadRest]) is only added to its own page.
	loads int

	// leftScroll is the vertical scroll offset of [Content.leftFrame] when
	// a staged page was loaded, which [Content.loadRest] restores.
	leftScroll float32
}

// stagePages is whether long pages are shown in two stages on the web: the
// first part of the page is shown first, and the rest is added right after
// (see [Content.loadRest]). Building, styling and laying out the widgets of a
// long page takes much longer on the web than natively, and most of them are
// not visible at first.
var stagePages = true

func init() {
	// We want Command+[ and Command+] to work for browser back/forward navigation
	// in content, since we rely on that. They should still be intercepted by
	// Cogent Core for non-content apps for things such as full window dialogs,
	// so we only add these in content.
	system.ReservedWebShortcuts = append(system.ReservedWebShortcuts, "Command+[", "Command+]")
}

// keepCategories is whether the categories tree is kept between pages,
// instead of being made again for each page (tests compare the two).
var keepCategories = true

// NewPageInitFunc is called when a new page is just being rendered.
// This can do any necessary new-page initialization, e.g.,
// [yaegicore.ResetGoalInterpreter]
var NewPageInitFunc func()

// ImageCacheBytes is the maximum total size in bytes of the decoded
// images that a [Content] keeps in the [htmlcore.ImageCache] of its
// [Content.Context], so that revisiting a page does not decode its
// images again.
var ImageCacheBytes = 64 << 20

func (ct *Content) Init() {
	ct.Splits.Init()
	ct.SetSplits(0.2, 0.8)

	ct.Context = htmlcore.NewContext()
	ct.Context.DelayedImageLoad = false // not useful for content
	// The source does not change until SetSource, which resets the cache.
	ct.Context.ImageCache = htmlcore.NewImageCache(ImageCacheBytes)
	ct.Context.OpenURL = func(url string, e events.Event) {
		ct.OpenEvent(url, e)
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
	ct.Context.ElementHandlers["pre"] = ct.htmlPreHandler
	ct.Context.AttributeHandlers["id"] = ct.htmlIDAttributeHandler
	ct.Context.AddWidgetHandler(ct.widgetHandler)

	ct.Maker(func(p *tree.Plan) {
		if ct.current.Page == nil {
			return
		}
		tree.Add(p, func(w *core.Frame) {
			ct.leftFrame = w
		})
		tree.Add(p, func(w *core.Frame) {
			ct.rightFrame = w
			w.Maker(func(p *tree.Plan) {
				if core.TheApp.Platform() == system.Web || system.GenerateHTMLArg() {
					ct.pageMaker(p, 0)
				} else {
					tree.Add(p, func(w *core.Tabs) {
						ct.tabs = w
						w.SetType(core.FunctionalTabs).SetNewTabButton(true)
						w.NewTabFunc = func(index int) {
							ct.newTab(ct.tabs.TabAtIndex(index))
							ct.open("", true)
						}
						w.CloseTabFunc = func(index int) {
							ct.history = slices.Delete(ct.history, index, index+1)
							_, ci := ct.tabs.CurrentTab()
							h := ct.history[ci]
							lc := h.Records[h.Index]
							ct.open(lc.URL, false)
							if ct.toolbar != nil {
								ct.toolbar.Update()
							}
						}
						fr, _ := w.NewTab("Content")
						h := &History{}
						h.Save(ct.current.Clone())
						ct.history = []*History{h}
						fr.Maker(func(p *tree.Plan) { ct.pageMaker(p, 0) })
					})
				}
			})
		})
	})

	// Must be done after the default title is set elsewhere in normal OnShow
	ct.OnFinal(events.Show, func(e events.Event) {
		ct.setStageTitle()
	})
	ct.handleWebPopState()
}

// pageFrame returns the current frame for rendering the page.
func (ct *Content) pageFrame() *core.Frame {
	if ct.tabs == nil {
		return ct.rightFrame
	}
	w, _ := ct.tabs.CurrentTab()
	return w.(*core.Frame)
}

// pageMaker is the maker function for a page
func (ct *Content) pageMaker(p *tree.Plan, tabIdx int) {
	tree.Add(p, func(w *core.Frame) {
		if system.GenerateHTMLArg() {
			w.SetProperty("class", "cc-page-frame") // responsive padding in the web content.css
		}
		w.Styler(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Grow.Set(1, 1)
			switch w.SizeClass() {
			case core.SizeCompact, core.SizeMedium:
				s.Padding.SetHorizontal(units.Em(0.5))
			case core.SizeExpanded:
				s.Padding.SetHorizontal(units.Em(3))
			}
		})
		w.Maker(func(p *tree.Plan) {
			if ct.tabs != nil {
				_, ci := ct.tabs.CurrentTab()
				if ci != tabIdx {
					return
				}
				h := ct.history[tabIdx]
				lc := h.Records[h.Index]
				ct.current = *lc
			}
			if !ct.inPDFRender && ct.current.Page.Title != "" {
				tree.Add(p, func(w *core.Text) {
					w.Updater(func() {
						w.SetText(ct.current.Page.Title)
					})
					w.SetType(core.TextDisplaySmall)
					if system.GenerateHTMLArg() {
						w.SetProperty("tag", "h1")
						w.SetProperty("page-title", true) // not in table of contents
					}
				})
			}
			if !ct.inPDFRender && len(ct.current.Page.Authors) > 0 {
				tree.Add(p, func(w *core.Text) {
					w.SetType(core.TextTitleLarge)
					w.Updater(func() {
						w.SetText("By " + ct.current.Page.Authors)
					})
				})
			}
			if !ct.inPDFRender && len(ct.current.Page.Affiliations) > 0 {
				tree.Add(p, func(w *core.Text) {
					w.SetType(core.TextBodyLarge)
					w.Updater(func() {
						w.SetText(ct.current.Page.Affiliations)
					})
				})
			}
			if !ct.inPDFRender && !ct.current.Page.Date.IsZero() {
				tree.Add(p, func(w *core.Text) {
					w.SetType(core.TextTitleMedium)
					w.Updater(func() {
						if ct.current.Page.Version != "" {
							w.SetText(ct.current.Page.Date.Format("January 2, 2006") + ", Version: " + ct.current.Page.Version)
						} else {
							w.SetText(ct.current.Page.Date.Format("January 2, 2006"))
						}
					})
				})
			}
			if !ct.inPDFRender && ct.current.Page.Date.IsZero() && ct.current.Page.Version != "" {
				tree.Add(p, func(w *core.Text) {
					w.SetType(core.TextTitleMedium)
					w.Updater(func() {
						w.SetText(" Version: " + ct.current.Page.Version)
					})
				})
			}
			if !ct.inPDFRender && len(ct.current.Page.Reference) > 0 {
				tree.Add(p, func(w *core.Text) {
					w.SetType(core.TextBodyLarge)
					w.Updater(func() {
						w.SetText(ct.current.Page.Reference)
					})
				})
			}
			if !ct.inPDFRender && len(ct.current.Page.Abstract) > 0 {
				tree.Add(p, func(w *core.Text) {
					w.SetType(core.TextBodyLarge)
					w.Updater(func() {
						w.SetText("<b>Abstract:</b> " + ct.current.Page.Abstract)
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
					if ct.toolbar != nil {
						ct.toolbar.Update()
					}
				})
			})
			if !ct.inPDFRender {
				ct.makeBottomButtons(p)
			}
		})
	})
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
	if ct.Context.ImageCache != nil {
		ct.Context.ImageCache.Reset()
	}
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
	// The categories tree of the old source is made again.
	if ct.categoriesTree != nil {
		ct.categoriesTree.Delete()
		ct.categoriesTree = nil
	}

	if url := ct.getWebURL(); url != "" {
		return ct.Open(url)
	}
	if root, ok := ct.pagesByURL[""]; ok {
		return ct.Open(root.URL)
	}
	return ct.Open(ct.pages[0].URL)
}

// SetContent is a helper function that calls [Content.SetSource]
// with the "content" subdirectory of the given filesystem.
func (ct *Content) SetContent(content fs.FS) *Content {
	return ct.SetSource(fsx.Sub(content, "content"))
}

// OpenEvent opens the page with the given URL and updates the display.
// If no pages correspond to the URL, it is opened in the default browser.
// This version is for widget event cases, where the keyboard modifiers
// are used to control the way the page is opened: Ctrl/Meta = new tab.
func (ct *Content) OpenEvent(url string, e events.Event) *Content {
	if e == nil || !e.HasAnyModifier(key.Control, key.Meta) {
		return ct.Open(url)
	}
	if ct.tabs == nil {
		if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") {
			core.TheApp.OpenURL(url)
			return ct
		}
		if strings.HasPrefix(url, "ref://") {
			ct.openRef(url)
			return ct
		}
		_, pg, heading := ct.parseURL(url)
		_, nw, err := ct.pageURL(pg, heading)
		if err != nil {
			return ct
		}
		core.TheApp.OpenURL(nw.String())
		return ct
	}
	nt := ct.tabs.NumTabs()
	nm := fmt.Sprintf("Tab %d", nt+1)
	ct.newTab(ct.tabs.NewTab(nm))
	return ct.Open(url)
}

func (ct *Content) newTab(fr *core.Frame, tb *core.Tab) {
	ct.rendered.Reset()
	nt := ct.tabs.NumTabs()
	nm := fmt.Sprintf("Tab %d", nt)
	tb.SetText(nm)
	h := &History{}
	h.Save(ct.current.Clone())
	ct.history = append(ct.history, h)
	fr.Maker(func(p *tree.Plan) {
		ct.pageMaker(p, nt-1)
	})
	ct.tabs.SelectTabIndex(nt - 1)
}

// Open opens the page with the given URL and updates the display.
// If no pages correspond to the URL, it is opened in the default browser.
// This version is for programmatic use -- see also OpenEvent.
func (ct *Content) Open(url string) *Content {
	ct.open(url, true)
	return ct
}

// reloadPage reloads the current page
func (ct *Content) reloadPage() {
	ct.rendered.Reset()
	ct.Update()
}

// loadPage loads the current page content into the given frame if it is not already loaded.
func (ct *Content) loadPage(w *core.Frame) error {
	if ct.rendered == ct.current { // this prevents tabs from rendering
		// fmt.Println("repeat")
		// return nil
	}
	if NewPageInitFunc != nil {
		NewPageInitFunc()
	}
	resetGoal(ct.current.Page.Filename)
	w.DeleteChildren()
	b, err := ct.current.Page.ReadContent(ct.pagesByCategory)
	if err != nil {
		return err
	}
	ct.current.Page.ParseSpecials(b)

	// A long page opened at its start is shown in two stages on the web.
	ct.loads++
	staged := stagePages && core.TheApp.Platform() == system.Web && ct.current.Heading == "" && !ct.inPDFRender
	if staged {
		ct.Context.StageSize = ct.stageSize()
		ct.leftScroll = ct.leftFrame.Geom.Scroll.Y
	}
	err = htmlcore.ReadMD(ct.Context, w, b)
	ct.Context.StageSize = 0
	if err != nil {
		return err
	}
	if ct.Context.HasRest() {
		loads := ct.loads
		w.Defer(func() {
			go ct.loadRest(w, loads)
		})
	}

	ct.makeTableOfContents(w, ct.current.Page)
	// A new categories tree is styled with the table of contents; a kept one only
	// restyles the nodes whose selection changes, in updateCategories.
	if ct.makeCategories() {
		ct.leftFrame.Update()
	} else {
		ct.updateCategories(true)
		if ct.tocTree != nil {
			ct.tocTree.Update()
		}
		ct.leftFrame.NeedsLayout()
	}
	ct.rendered = ct.current
	return nil
}

// stageSize returns the size of the first stage of a staged page (see
// [htmlcore.Context.StageSize]): about three times the text that fits in
// the window.
func (ct *Content) stageSize() int {
	sz := ct.Scene.SceneGeom.Size
	charsPerLine := max(sz.X/9, 40)
	lines := max(sz.Y/24, 20)
	return 3 * charsPerLine * lines
}

// loadRest adds the rest of a staged page to the page frame w after the
// first stage has been shown, unless another page has been loaded since
// (the given loads is [Content.loads] when the page was loaded).
func (ct *Content) loadRest(w *core.Frame, loads int) {
	// The browser shows the first stage before the rest is added.
	time.Sleep(time.Millisecond)
	ct.AsyncLock()
	defer ct.AsyncUnlock()
	if ct.loads != loads || w.This == nil || !ct.Context.HasRest() {
		return
	}

	// The widgets of the rest are updated, styled and laid out, and the
	// table of contents is made again with their headings.
	parent, start, err := ct.Context.ReadRest()
	errors.Log(err)
	pw := parent.AsWidget()
	for _, c := range pw.Children[start:] {
		c.(core.Widget).AsWidget().Update()
	}
	ct.makeTableOfContents(w, ct.current.Page)
	if ct.tocTree != nil {
		ct.tocTree.Update()
	}
	ct.leftFrame.NeedsLayout()
	w.NeedsLayout()

	// The sidebar is scrolled back to where it was before the page was
	// loaded, which the shorter table of contents of the first stage may
	// have limited, once the layout has updated its range.
	if scroll := ct.leftScroll; scroll != 0 {
		ct.leftFrame.Defer(func() {
			if sb := ct.leftFrame.Scrolls[math32.Y]; sb != nil && ct.loads == loads {
				sb.SetValue(-scroll)
				ct.leftFrame.ScrollChanged(math32.Y, sb)
			}
		})
	}
}

func (ct *Content) RenderWidget() {
	ct.Splits.RenderWidget()
	if ct.renderChan != nil {
		ct.renderChan <- struct{}{}
	}
}

// makeTableOfContents makes the table of contents and adds it to [Content.leftFrame]
// based on the headings in the given frame.
func (ct *Content) makeTableOfContents(w *core.Frame, pg *bcontent.Page) {
	ct.deleteTableOfContents()
	ct.tocNodes = map[string]*core.Tree{}
	// The table of contents goes before the categories tree, which is kept between pages.
	contents := core.NewTree()
	ct.leftFrame.InsertChild(contents, 0)
	contents.SetText("<b>Contents</b>")
	ct.tocTree = contents
	contents.SetReadOnly(true)
	contents.OnSelect(func(e events.Event) {
		if contents.IsRootSelected() {
			ct.pageFrame().ScrollDimToContentStart(math32.Y)
			ct.current.Heading = ""
			ct.saveWebURL(&ct.current)
		}
	})
	// last is the most recent tree node for each heading level, used for nesting.
	last := map[int]*core.Tree{}
	w.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
		tx, ok := cw.(*core.Text)
		if !ok {
			return tree.Continue
		}
		tag, ok := tx.Property("tag").(string)
		if ok && len(tag) == 2 && tag[0] == 'h' {
			num := errors.Log1(strconv.Atoi(tag[1:]))
			parent := contents
			// Our parent is the last heading with a lower level (closer to h1).
			for i := num - 1; i >= 1; i-- {
				if last[i] != nil {
					parent = last[i]
					break
				}
			}
			tr := core.NewTree(parent).SetText(tx.Text)
			tr.SetProperty("page-text", tx)
			last[num] = tr
			kebab := strcase.ToKebab(tr.Text)
			ct.tocNodes[kebab] = tr
			tr.OnSelect(func(e events.Event) {
				ct.saveUpdatedPos(ct.current, ct.current.Page, ct.current.Heading)
				tx.ScrollThisToTop()
				ct.OpenEvent(ct.current.Page.URL+"#"+kebab, e)
			})
		}
		return tree.Continue
	})

	if contents.NumChildren() == 0 {
		ct.deleteTableOfContents()
	}
}

// deleteTableOfContents deletes the table of contents tree, if there is one.
func (ct *Content) deleteTableOfContents() {
	if ct.tocTree != nil {
		ct.tocTree.Delete()
		ct.tocTree = nil
	}
}

// hasCategoriesTree returns whether [Content.categoriesTree] is the
// current categories tree in [Content.leftFrame].
func (ct *Content) hasCategoriesTree() bool {
	cats := ct.categoriesTree
	return cats != nil && cats.This != nil && ct.leftFrame != nil && cats.Parent == ct.leftFrame.This
}

// makeCategories makes the categories tree for the current page and adds it to
// [Content.leftFrame] if it does not have it yet, returning whether it made it.
func (ct *Content) makeCategories() bool {
	if len(ct.categories) == 0 || ct.hasCategoriesTree() {
		return false
	}

	cats := core.NewTree(ct.leftFrame).SetText("<b>Categories</b>")
	cats.SetReadOnly(true)
	cats.OnSelect(func(e events.Event) {
		if cats.IsRootSelected() {
			ct.OpenEvent("", e)
		}
	})
	ct.categoriesTree = cats
	ct.categoryTrees = make([]*core.Tree, len(ct.categories))
	ct.categoryPages = map[*core.Tree]*bcontent.Page{}
	for i, cat := range ct.categories {
		catTree := core.NewTree(cats).SetText(cat).SetClosed(true)
		ct.categoryTrees[i] = catTree
		catTree.OnSelect(func(e events.Event) {
			if catPage := ct.pageByName(cat); catPage != nil {
				ct.OpenEvent(catPage.URL, e)
			} else {
				catTree.Open() // no page to open so open the tree
			}
		})
		for _, pg := range ct.pagesByCategory[cat] {
			if strings.EqualFold(cat, pg.Name) {
				continue
			}
			pgTree := core.NewTree(catTree).SetText(pg.Name)
			ct.categoryPages[pgTree] = pg
			pgTree.OnSelect(func(e events.Event) {
				ct.OpenEvent(pg.URL, e)
			})
		}
	}
	ct.updateCategories(false)
	return true
}

// updateCategories selects the nodes of the categories tree for the current page
// and opens the categories that contain it, closing the others, as a new tree has them.
// If restyle is true, the nodes whose selection changes are styled again.
func (ct *Content) updateCategories(restyle bool) {
	var selected []core.Treer
	// setSelected sets whether the given node is selected.
	setSelected := func(tr *core.Tree, sel bool) {
		if sel {
			selected = append(selected, tr)
		}
		if tr.StateIs(states.Selected) == sel {
			return
		}
		tr.SetState(sel, states.Selected)
		if restyle {
			styleTreeNode(tr)
		}
	}

	// The root is selected by a click on it until the home page opens.
	setSelected(ct.categoriesTree, false)
	for i, catTree := range ct.categoryTrees {
		setSelected(catTree, ct.current.Page.Name == ct.categories[i])
		hasPage := false
		for _, k := range catTree.Children {
			pgTree := k.(*core.Tree)
			isPage := ct.categoryPages[pgTree] == ct.current.Page
			hasPage = hasPage || isPage
			setSelected(pgTree, isPage)
		}
		if hasPage {
			catTree.Open()
		} else {
			catTree.Close()
		}
	}
	ct.categoriesTree.SelectedNodes = selected
}

// styleTreeNode styles the given tree node and its parts, which inherit
// its selection colors, but not its child nodes.
func styleTreeNode(tr *core.Tree) {
	tr.Style()
	if tr.Parts != nil {
		tr.Parts.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
			cw.Style()
			return tree.Continue
		})
	}
	tr.NeedsRender()
}

// updateScene updates the scene for a new page, as [core.WidgetBase.Update]
// does, except for the categories tree, which [Content.loadPage] updates.
func (ct *Content) updateScene() {
	cats := ct.categoriesTree
	if !ct.hasCategoriesTree() {
		cats = nil
	}
	ct.Scene.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
		if cats != nil && cwb == cats.AsWidget() {
			return tree.Break
		}
		cwb.UpdateWidget()
		cw.Style()
		return tree.Continue
	})
	ct.Scene.NeedsLayout()
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
	if rw := ct.Scene.RenderWindow(); rw != nil && ct.current.Page != nil {
		name := ct.current.Page.Name
		if ct.current.Page.URL == "" { // Root page just gets app name
			name = core.TheApp.Name()
		}
		rw.SetStageTitle(name)
	}
}

// PagePDF generates a PDF of the current page, to given file path
// (directory). the page name is the file name.
func (ct *Content) PagePDF(path string) error {
	if ct.current.Page == nil {
		return errors.Log(errors.New("Page empty"))
	}
	core.MessageSnackbar(ct, "Generating PDF...")

	Settings.PDF.FontScale = (100.0 / core.AppearanceSettings.DocsFontSize)

	ct.inPDFRender = true
	ct.reloadPage()
	ct.inPDFRender = false

	refs := ct.PageRefs(ct.current.Page)

	fname := ct.current.Page.Name + ".pdf"
	if path != "" {
		errors.Log(os.MkdirAll(path, 0777))
		fname = filepath.Join(path, fname)
	}
	f, err := os.Create(fname)
	if errors.Log(err) != nil {
		return err
	}
	opts := Settings.PageSettings(ct, ct.current.Page)
	if refs != nil {
		paginate.PDF(f, opts.PDF, ct.pageFrame(), refs)
	} else {
		paginate.PDF(f, opts.PDF, ct.pageFrame())
	}
	err = f.Close()

	ct.reloadPage()

	core.MessageSnackbar(ct, "PDF saved to: "+fname)
	af := errors.Log1(filepath.Abs(fname))
	core.TheApp.OpenURL("file://" + af)
	return err
}

// PageRefs returns a core.Frame with the contents of the references cited
// on the given page. if References is nil, or error, result will be nil.
func (ct *Content) PageRefs(page *bcontent.Page) *core.Frame {
	if ct.References == nil {
		return nil
	}
	sty := csl.APA // todo: settings
	var b bytes.Buffer
	_, err := csl.GenerateMarkdown(&b, ct.Source, "## References", ct.References, sty, page.Filename)
	if errors.Log(err) != nil {
		return nil
	}
	// os.WriteFile("tmp-refs.md", b.Bytes(), 0666)

	fr := core.NewFrame()
	fr.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	err = htmlcore.ReadMD(ct.Context, fr, b.Bytes())
	if errors.Log(err) != nil {
		return nil
	}
	fr.StyleTree()
	fr.SetScene(ct.Scene)
	return fr
}

// LoadEachPage is a helper function to iterate through loading each page,
// which is used for updating the math cache.
func (ct *Content) LoadEachPage(after func()) error {
	ct.renderChan = make(chan struct{})
	go func() {
		for _, pg := range ct.pages {
			fmt.Println("\n#### page:", pg.URL)
			ct.OpenEvent(pg.URL, nil)
			<-ct.renderChan
			fmt.Println("rendered:", pg.URL)
		}
		if after != nil {
			after()
		}
	}()
	return nil
}
