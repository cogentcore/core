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

	// inPDFRender indicates that it is rendering a PDF now, turning off
	// elements that are not appropriate for that.
	inPDFRender bool

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
	ct.Context.DelayedImageLoad = false // not useful for content
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
			if !ct.current.Page.Date.IsZero() {
				tree.Add(p, func(w *core.Text) {
					w.SetType(core.TextTitleMedium)
					w.Updater(func() {
						w.SetText(ct.current.Page.Date.Format("January 2, 2006"))
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
	w.DeleteChildren()
	b, err := ct.current.Page.ReadContent(ct.pagesByCategory)
	if err != nil {
		return err
	}
	ct.current.Page.ParseSpecials(b)
	err = htmlcore.ReadMD(ct.Context, w, b)
	if err != nil {
		return err
	}

	ct.leftFrame.DeleteChildren()
	ct.makeTableOfContents(w, ct.current.Page)
	ct.makeCategories()
	ct.leftFrame.Update()
	ct.rendered = ct.current
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
			tr.SetProperty("page-text", tx)
			last[num] = tr
			kebab := strcase.ToKebab(tr.Text)
			ct.tocNodes[kebab] = tr
			tr.OnSelect(func(e events.Event) {
				tx.ScrollThisToTop()
				ct.OpenEvent(ct.current.Page.URL+"#"+kebab, e)
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
			ct.OpenEvent("", e)
		}
	})
	for _, cat := range ct.categories {
		catTree := core.NewTree(cats).SetText(cat).SetClosed(true)
		if ct.current.Page.Name == cat {
			catTree.SetSelected(true)
		}
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
			if pg == ct.current.Page {
				pgTree.SetSelected(true)
				catTree.SetClosed(false)
			}
			pgTree.OnSelect(func(e events.Event) {
				ct.OpenEvent(pg.URL, e)
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
