// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package webcore is a framework designed for easily building content-focused sites
package webcore

//go:generate core generate

import (
	"bytes"
	"fmt"
	"io/fs"
	"log/slog"
	"net/url"
	"path"
	"strings"

	"cogentcore.org/core/coredom"
	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/goosi"
	"cogentcore.org/core/grows/tomls"
	"cogentcore.org/core/grr"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/strcase"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/webcore/wpath"
)

// Page represents one site page
type Page struct {
	gi.Frame

	// Source is the filesystem in which the content is located.
	Source fs.FS

	// Context is the page's [coredom.Context].
	Context *coredom.Context `set:"-"`

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

var _ ki.Ki = (*Page)(nil)

func (pg *Page) OnInit() {
	pg.Frame.OnInit()
	pg.Context = coredom.NewContext()
	pg.Context.OpenURL = func(url string) {
		pg.OpenURL(url, true)
	}
	pg.Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})
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
		gi.ErrorSnackbar(pg, err, "Invalid URL")
		return
	}
	if u.Scheme != "" {
		goosi.TheApp.OpenURL(u.String())
		return
	}

	if pg.Source == nil {
		gi.MessageSnackbar(pg, "Programmer error: page source must not be nil")
		return
	}

	// if we are not rooted, we go relative to our current fs path
	if !strings.HasPrefix(rawURL, "/") {
		rawURL = path.Join(path.Dir(pg.PagePath), rawURL)
	}

	// the paths in the fs are never rooted, so we trim a rooted one
	rawURL = strings.TrimPrefix(rawURL, "/")

	pg.Context.PageURL = rawURL
	if addToHistory {
		pg.HistoryIndex = len(pg.History)
		pg.History = append(pg.History, pg.Context.PageURL)
	}

	pg.PagePath = pg.URLToPagePath[pg.Context.PageURL]

	b, err := fs.ReadFile(pg.Source, pg.PagePath)
	if err != nil {
		gi.ErrorSnackbar(pg, err, "Error opening page")
		return
	}

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
			grr.Log(tomls.ReadBytes(&fm, fmb))
			fmt.Println("front matter", fm)
		}
	}

	// need to reset
	NumExamples[pg.Context.PageURL] = 0

	fr := pg.FindPath("splits/body").(*gi.Frame)
	fr.DeleteChildren()
	err = coredom.ReadMD(pg.Context, fr, b)
	if err != nil {
		gi.ErrorSnackbar(pg, err, "Error loading page")
		return
	}
	fr.Update()
}

func (pg *Page) Config() {
	if pg.HasChildren() {
		return
	}
	sp := gi.NewSplits(pg, "splits")

	nfr := gi.NewFrame(sp, "nav-frame")
	nav := giv.NewTreeView(nfr, "nav").SetText(gi.TheApp.Name())
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

	grr.Log(fs.WalkDir(pg.Source, ".", func(fpath string, d fs.DirEntry, err error) error {
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

		par := nav
		if pdir != "" && pdir != "." {
			par = nav.FindPath(pdir).(*giv.TreeView)
		}

		nm := strings.TrimSuffix(base, ext)
		txt := strcase.ToSentence(nm)
		tv := giv.NewTreeView(par, nm).SetText(txt)

		// need index.md for page path
		if d.IsDir() {
			fpath += "/index.md"
		}
		pg.URLToPagePath[tv.PathFrom(nav)] = fpath
		return nil
	}))

	gi.NewFrame(sp, "body").Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})

	sp.SetSplits(0.2, 0.8)

	if pg.PagePath == "" {
		pg.OpenURL("/", true)
	}
}

// AppBar is the default app bar for a [Page]
func (pg *Page) AppBar(tb *gi.Toolbar) {
	ch := tb.AppChooser()

	back := tb.ChildByName("back").(*gi.Button)
	back.OnClick(func(e events.Event) {
		if pg.HistoryIndex > 0 {
			pg.HistoryIndex--
			// we reverse the order
			// ch.SelectItem(len(pg.History) - pg.HistoryIndex - 1)
			// we need a slash so that it doesn't think it's a relative URL
			pg.OpenURL("/"+pg.History[pg.HistoryIndex], false)
		}
	})

	ch.AllowNew = true
	// TODO(kai/abc)
	// ch.AddItemsFunc(func() {
	// 	ch.Items = make([]any, len(pg.History))
	// 	for i, u := range pg.History {
	// 		// we reverse the order
	// 		ch.Items[len(pg.History)-i-1] = u
	// 	}
	// })
	// ch.OnChange(func(e events.Event) {
	// 	// we need a slash so that it doesn't think it's a relative URL
	// 	pg.OpenURL("/"+ch.CurrentLabel, true)
	// 	e.SetHandled()
	// })
}
