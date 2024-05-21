// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package pages provides an easy way to make content-focused
// sites consisting of Markdown, HTML, and Cogent Core pages.
package pages

//go:generate core generate

import (
	"bytes"
	"fmt"
	"io/fs"
	"log/slog"
	"net/url"
	"path"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/iox/tomlx"
	"cogentcore.org/core/base/strcase"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/htmlview"
	"cogentcore.org/core/pages/wpath"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/system"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/views"
)

// Page represents a content page with support for navigating
// to other pages within the same source content.
type Page struct {
	core.Frame

	// Source is the filesystem in which the content is located.
	Source fs.FS

	// Context is the page's [htmlview.Context].
	Context *htmlview.Context `set:"-"`

	// The history of URLs that have been visited. The oldest page is first.
	History []string `set:"-"`

	// HistoryIndex is the current place we are at in the History
	HistoryIndex int `set:"-"`

	// PagePath is the fs path of the current page in [Page.Source]
	PagePath string `set:"-"`

	// URLToPagePath is a map between user-facing page URLs and underlying
	// FS page paths.
	URLToPagePath map[string]string `set:"-"`
}

var _ tree.Node = (*Page)(nil)

// getWebURL, if non-nil, returns the current relative web URL that should
// be passed to [Page.OpenURL] on startup.
var getWebURL func() string

// saveWebURL, if non-nil, saves the given web URL to the user's browser address bar and history.
var saveWebURL func(u string)

func (pg *Page) OnInit() {
	pg.Frame.OnInit()
	pg.Context = htmlview.NewContext()
	pg.Context.OpenURL = func(url string) {
		pg.OpenURL(url, true)
	}
	pg.Style(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})
}

func (pg *Page) OnAdd() {
	pg.WidgetBase.OnAdd()
	pg.OnShow(func(e events.Event) {
		if pg.PagePath == "" {
			if getWebURL != nil {
				pg.OpenURL(getWebURL(), true)
			} else {
				pg.OpenURL("/", true)
			}
		}
	})
	// must be done after the default title is set elsewhere in normal OnShow
	pg.Scene.OnFinal(events.Show, func(e events.Event) {
		pg.setStageTitle()
	})
}

// setStageTitle sets the title of the stage based on the current page URL.
func (pg *Page) setStageTitle() {
	if rw := pg.Scene.RenderWindow(); rw != nil {
		rw.SetStageTitle(wpath.Label(pg.Context.PageURL, core.TheApp.Name()))
	}
}

// OpenURL sets the content of the page from the given url. If the given URL
// has no scheme (eg: "/about"), then it sets the content of the page to the
// file specified by the URL. This is either the "index.md" file in the
// corresponding directory (eg: "/about/index.md") or the corresponding
// md file (eg: "/about.md"). If it has a scheme, (eg: "https://example.com"),
// then it opens it in the user's default browser.
func (pg *Page) OpenURL(rawURL string, addToHistory bool) {
	u, err := url.Parse(rawURL)
	if err != nil {
		core.ErrorSnackbar(pg, err, "Invalid URL")
		return
	}
	if u.Scheme != "" {
		system.TheApp.OpenURL(u.String())
		return
	}

	if pg.Source == nil {
		core.MessageSnackbar(pg, "Programmer error: page source must not be nil")
		return
	}

	// if we are not rooted, we go relative to our current fs path
	if !strings.HasPrefix(rawURL, "/") {
		rawURL = path.Join(path.Dir(pg.PagePath), rawURL)
	}

	// the paths in the fs are never rooted, so we trim a rooted one
	rawURL = strings.TrimPrefix(rawURL, "/")
	rawURL = strings.TrimSuffix(rawURL, "/")

	pg.PagePath = pg.URLToPagePath[rawURL]

	b, err := fs.ReadFile(pg.Source, pg.PagePath)
	if err != nil {
		// we go to the first page in the directory if there is no index page
		if errors.Is(err, fs.ErrNotExist) && (strings.HasSuffix(pg.PagePath, "index.md") || strings.HasSuffix(pg.PagePath, "index.html")) {
			err = fs.WalkDir(pg.Source, path.Dir(pg.PagePath), func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if path == pg.PagePath || d.IsDir() {
					return nil
				}
				pg.PagePath = path
				return fs.SkipAll
			})
			// need to update rawURL with new page path
			for u, p := range pg.URLToPagePath {
				if p == pg.PagePath {
					rawURL = u
					break
				}
			}
			if err == nil {
				b, err = fs.ReadFile(pg.Source, pg.PagePath)
			}
		}
		if errors.Log(err) != nil {
			core.ErrorSnackbar(pg, err, "Error opening page")
			return
		}
	}

	pg.Context.PageURL = rawURL
	if addToHistory {
		pg.HistoryIndex = len(pg.History)
		pg.History = append(pg.History, pg.Context.PageURL)
	}
	if saveWebURL != nil {
		saveWebURL(pg.Context.PageURL)
	}
	pg.setStageTitle()

	btp := []byte("+++")
	if bytes.HasPrefix(b, btp) {
		b = bytes.TrimPrefix(b, btp)
		fmb, content, ok := bytes.Cut(b, btp)
		if !ok {
			slog.Error("got unclosed front matter")
			b = fmb
			fmb = nil
		} else {
			b = content
		}
		if len(fmb) > 0 {
			var fm map[string]string
			errors.Log(tomlx.ReadBytes(&fm, fmb))
			fmt.Println("front matter", fm)
		}
	}

	// need to reset
	NumExamples[pg.Context.PageURL] = 0

	nav := pg.FindPath("splits/nav-frame/nav").(*views.TreeView)
	nav.UnselectAll()
	utv := nav.FindPath(rawURL).(*views.TreeView)
	utv.Select()
	utv.ScrollToMe()

	fr := pg.FindPath("splits/body").(*core.Frame)
	fr.DeleteChildren()
	err = htmlview.ReadMD(pg.Context, fr, b)
	if err != nil {
		core.ErrorSnackbar(pg, err, "Error loading page")
		return
	}
	fr.Update()
}

func (pg *Page) Make(p *core.Plan) {
	if pg.HasChildren() {
		return
	}
	sp := core.NewSplits(pg).SetSplits(0.2, 0.8)
	sp.SetName("splits")

	nav := views.NewTreeViewFrame(sp).SetText(core.TheApp.Name())
	nav.Parent().SetName("nav-frame")
	nav.SetName("nav")
	nav.SetReadOnly(true)
	nav.ParentWidget().Style(func(s *styles.Style) {
		s.Background = colors.C(colors.Scheme.SurfaceContainerLow)
	})
	nav.OnSelect(func(e events.Event) {
		if len(nav.SelectedNodes) == 0 {
			return
		}
		sn := nav.SelectedNodes[0]
		url := "/"
		if sn != nav {
			// we need a slash so that it doesn't think it's a relative URL
			url = "/" + sn.PathFrom(nav)
		}
		pg.OpenURL(url, true)
	})

	pg.URLToPagePath = map[string]string{"": "index.md"}

	errors.Log(fs.WalkDir(pg.Source, ".", func(fpath string, d fs.DirEntry, err error) error {
		// already handled
		if fpath == "" || fpath == "." {
			return nil
		}

		p := wpath.Format(fpath)

		pdir := path.Dir(p)
		base := path.Base(p)

		// already handled
		if base == "index.md" {
			return nil
		}

		ext := path.Ext(base)
		if ext != "" && ext != ".md" {
			return nil
		}

		parent := nav
		if pdir != "" && pdir != "." {
			parent = nav.FindPath(pdir).(*views.TreeView)
		}

		nm := strings.TrimSuffix(base, ext)
		txt := strcase.ToSentence(nm)
		tv := views.NewTreeView(parent).SetText(txt)
		tv.SetName(nm)

		// need index.md for page path
		if d.IsDir() {
			fpath += "/index.md"
		}
		pg.URLToPagePath[tv.PathFrom(nav)] = fpath
		return nil
	}))

	core.NewFrame(sp).Style(func(s *styles.Style) {
		s.Direction = styles.Column
	}).SetName("body")
}

// AppBar is the default app bar for a [Page]
func (pg *Page) AppBar(c *core.Plan) {
	// todo: needs a different config
	/*
		ch := tb.AppChooser()

		back := tb.ChildByName("back").(*core.Button)
		back.OnClick(func(e events.Event) {
			if pg.HistoryIndex > 0 {
				pg.HistoryIndex--
				// we reverse the order
				// ch.SelectItem(len(pg.History) - pg.HistoryIndex - 1)
				// we need a slash so that it doesn't think it's a relative URL
				pg.OpenURL("/"+pg.History[pg.HistoryIndex], false)
			}
		})

		ch.AddItemsFunc(func() {
			urls := []string{}
			for u := range pg.URLToPagePath {
				urls = append(urls, u)
			}
			slices.Sort(urls)
			for _, u := range urls {
				ch.Items = append(ch.Items, core.ChooserItem{
					Value: u,
					Text:  wpath.Label(u, core.TheApp.Name()),
					Func: func() {
						pg.OpenURL("/"+u, true)
					},
				})
			}
		})
	*/
}
