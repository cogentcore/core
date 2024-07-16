// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package pages provides an easy way to make content-focused
// sites consisting of Markdown, HTML, and Cogent Core.
package pages

//go:generate core generate

import (
	"bytes"
	"fmt"
	"io/fs"
	"log/slog"
	"net/url"
	"path"
	"slices"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/base/iox/tomlx"
	"cogentcore.org/core/base/strcase"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/pages/wpath"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/system"
	"cogentcore.org/core/tree"
)

// Page represents a content page with support for navigating
// to other pages within the same source content.
type Page struct {
	core.Frame

	// Source is the filesystem in which the content is located.
	Source fs.FS

	// Context is the page's [htmlcore.Context].
	Context *htmlcore.Context `set:"-"`

	// The history of URLs that have been visited. The oldest page is first.
	History []string `set:"-"`

	// HistoryIndex is the current place we are at in the History
	HistoryIndex int `set:"-"`

	// PagePath is the fs path of the current page in [Page.Source]
	PagePath string `set:"-"`

	// URLToPagePath is a map between user-facing page URLs and underlying
	// FS page paths.
	URLToPagePath map[string]string `set:"-"`

	// nav is the navigation tree.
	nav *core.Tree

	// body is the page body frame.
	body *core.Frame
}

var _ tree.Node = (*Page)(nil)

// getWebURL, if non-nil, returns the current relative web URL that should
// be passed to [Page.OpenURL] on startup.
var getWebURL func() string

// saveWebURL, if non-nil, saves the given web URL to the user's browser address bar and history.
var saveWebURL func(u string)

func (pg *Page) Init() {
	pg.Frame.Init()
	pg.Context = htmlcore.NewContext()
	pg.Context.OpenURL = func(url string) {
		pg.OpenURL(url, true)
	}
	pg.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})

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
	pg.OnFinal(events.Show, func(e events.Event) {
		pg.setStageTitle()
	})

	tree.AddChild(pg, func(w *core.Splits) {
		w.SetSplits(0.2, 0.8)
		tree.AddChild(w, func(w *core.Frame) {
			w.Styler(func(s *styles.Style) {
				s.Background = colors.Scheme.SurfaceContainerLow
			})
			tree.AddChild(w, func(w *core.Tree) {
				pg.nav = w
				w.SetText(core.TheApp.Name())
				w.SetReadOnly(true)
				w.OnSelect(func(e events.Event) {
					if len(w.SelectedNodes) == 0 {
						return
					}
					sn := w.SelectedNodes[0]
					url := "/"
					if sn != w {
						// we need a slash so that it doesn't think it's a relative URL
						url = "/" + sn.AsTree().PathFrom(w)
					}
					pg.OpenURL(url, true)
				})

				pg.URLToPagePath = map[string]string{"": "index.md"}
				errors.Log(fs.WalkDir(pg.Source, ".", func(fpath string, d fs.DirEntry, err error) error {
					if err != nil {
						return err
					}

					// already handled
					if fpath == "" || fpath == "." {
						return nil
					}

					if system.TheApp.Platform() == system.Web && wpath.Draft(fpath) {
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

					parent := w
					if pdir != "" && pdir != "." {
						parent = w.FindPath(pdir).(*core.Tree)
					}

					nm := strings.TrimSuffix(base, ext)
					txt := strcase.ToSentence(nm)
					tv := core.NewTree(parent).SetText(txt)
					tv.SetName(nm)

					// need index.md for page path
					if d.IsDir() {
						fpath += "/index.md"
					}
					pg.URLToPagePath[tv.PathFrom(w)] = fpath
					return nil
				}))
			})
		})
		tree.AddChild(w, func(w *core.Frame) {
			pg.body = w
			w.Styler(func(s *styles.Style) {
				s.Direction = styles.Column
			})
		})
	})
}

// setStageTitle sets the title of the stage based on the current page URL.
func (pg *Page) setStageTitle() {
	if rw := pg.Scene.RenderWindow(); rw != nil {
		rw.SetStageTitle(wpath.Label(pg.Context.PageURL, core.TheApp.Name()))
	}
}

// SetContent is a helper function that calls [Page.SetSource]
// with the "content" subdirectory of the given filesystem.
func (pg *Page) SetContent(content fs.FS) *Page {
	return pg.SetSource(fsx.Sub(content, "content"))
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

	// if we are not rooted, we go relative to our current URL
	if !strings.HasPrefix(rawURL, "/") {
		current := pg.Context.PageURL
		if !strings.HasSuffix(pg.PagePath, "index.md") && !strings.HasSuffix(pg.PagePath, "index.html") {
			current = path.Dir(current) // we must go up one if we are not the index page (which is already up one)
		}
		rawURL = path.Join(current, rawURL)
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
				if system.TheApp.Platform() == system.Web && wpath.Draft(path) {
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

	pg.nav.UnselectAll()
	utv := pg.nav.FindPath(rawURL).(*core.Tree)
	utv.Select()
	utv.ScrollToThis()

	pg.body.DeleteChildren()
	if wpath.Draft(pg.PagePath) {
		draft := core.NewText(pg.body).SetType(core.TextDisplayLarge).SetText("DRAFT")
		draft.Styler(func(s *styles.Style) {
			s.Color = colors.Scheme.Error.Base
			s.Font.Weight = styles.WeightBold
		})
	}
	err = htmlcore.ReadMD(pg.Context, pg.body, b)
	if err != nil {
		core.ErrorSnackbar(pg, err, "Error loading page")
		return
	}
	pg.body.Update()
}

func (pg *Page) MakeToolbar(p *tree.Plan) {
	tree.AddInit(p, "back", func(w *core.Button) {
		w.OnClick(func(e events.Event) {
			if pg.HistoryIndex > 0 {
				pg.HistoryIndex--
				// we need a slash so that it doesn't think it's a relative URL
				pg.OpenURL("/"+pg.History[pg.HistoryIndex], false)
				e.SetHandled()
			}
		})
	})
	tree.AddInit(p, "app-chooser", func(w *core.Chooser) {
		w.AddItemsFunc(func() {
			urls := []string{}
			for u := range pg.URLToPagePath {
				urls = append(urls, u)
			}
			slices.Sort(urls)
			for _, u := range urls {
				w.Items = append(w.Items, core.ChooserItem{
					Value: u,
					Text:  wpath.Label(u, core.TheApp.Name()),
					Func: func() {
						pg.OpenURL("/"+u, true)
					},
				})
			}
		})
	})
}
