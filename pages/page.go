// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package pages provides an easy way to make content-focused
// sites consisting of Markdown, HTML, and Cogent Core.
package pages

//go:generate core generate

import (
	"bytes"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strings"
	"time"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/base/iox/tomlx"
	"cogentcore.org/core/base/slicesx"
	"cogentcore.org/core/base/strcase"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/pages/ppath"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
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
	history []string

	// historyIndex is the current place we are at in the History
	historyIndex int

	// pagePath is the fs path of the current page in [Page.Source]
	pagePath string

	// urlToPagePath is a map between user-facing page URLs and underlying
	// FS page paths.
	urlToPagePath map[string]string

	// nav is the navigation tree.
	nav *core.Tree

	// body is the page body frame.
	body *core.Frame
}

var _ tree.Node = (*Page)(nil)

// getWebURL, if non-nil, returns the current relative web URL that should
// be passed to [Page.OpenURL] on startup.
var getWebURL func(p *Page) string

// saveWebURL, if non-nil, saves the given web URL to the user's browser address bar and history.
var saveWebURL func(p *Page, u string)

// needsPath indicates that a URL in [Page.URLToPagePath] needs its path
// to be set to the first valid child path, since its index.md file does
// not exist.
const needsPath = "$__NEEDS_PATH__$"

func (pg *Page) Init() {
	pg.Frame.Init()
	pg.Context = htmlcore.NewContext()
	pg.Context.OpenURL = func(url string) {
		pg.OpenURL(url, true)
	}
	pg.Context.GetURL = func(rawURL string) (*http.Response, error) {
		u, err := url.Parse(rawURL)
		if err != nil {
			return nil, err
		}
		if u.Scheme != "" {
			return http.Get(rawURL)
		}
		rawURL = strings.TrimPrefix(rawURL, "/")
		filename := ""
		dirPath, ok := pg.urlToPagePath[path.Dir(rawURL)]
		if ok {
			filename = path.Join(path.Dir(dirPath), path.Base(rawURL))
		} else {
			filename = rawURL
		}
		f, err := pg.Source.Open(filename)
		if err != nil {
			return nil, err
		}
		return &http.Response{
			Status:        "200 OK",
			StatusCode:    200,
			Body:          f,
			ContentLength: -1,
		}, nil
	}
	pg.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})

	// must be done after the default title is set elsewhere in normal OnShow
	pg.OnFinal(events.Show, func(e events.Event) {
		pg.setStageTitle()
	})

	tree.AddChild(pg, func(w *core.Splits) {
		w.SetSplits(0.2, 0.8)
		tree.AddChild(w, func(w *core.Frame) {
			w.SetProperty("tag", "tree") // ignore in generatehtml
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

				pg.urlToPagePath = map[string]string{"": "index.md"}
				errors.Log(fs.WalkDir(pg.Source, ".", func(fpath string, d fs.DirEntry, err error) error {
					if err != nil {
						return err
					}

					// already handled
					if fpath == "" || fpath == "." {
						return nil
					}

					if system.TheApp.Platform() == system.Web && ppath.Draft(fpath) {
						return nil
					}
					p := ppath.Format(fpath)
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
					tr := core.NewTree(parent).SetText(txt)
					tr.SetName(nm)

					// need index.md for page path
					if d.IsDir() {
						fpath += "/index.md"
					}
					exists, err := fsx.FileExistsFS(pg.Source, fpath)
					if err != nil {
						return err
					}
					if !exists {
						fpath = needsPath
						tr.SetProperty("no-index", true)
					}
					pg.urlToPagePath[tr.PathFrom(w)] = fpath
					// everyone who needs a path gets our path
					for u, p := range pg.urlToPagePath {
						if p == needsPath {
							pg.urlToPagePath[u] = fpath
						}
					}
					return nil
				}))
				// If we still need a path, we shouldn't exist.
				for u, p := range pg.urlToPagePath {
					if p == needsPath {
						delete(pg.urlToPagePath, u)
						if n := w.FindPath(u); n != nil {
							n.AsTree().Delete()
						}
					}
				}
				// open the default page if there is no currently open page
				if pg.pagePath == "" {
					if getWebURL != nil {
						pg.OpenURL(getWebURL(pg), true)
					} else {
						pg.OpenURL("/", true)
					}
				}
			})
		})
		tree.AddChild(w, func(w *core.Frame) {
			pg.body = w
			w.Styler(func(s *styles.Style) {
				s.Direction = styles.Column
				s.Padding.Set(units.Dp(8))
			})
		})
	})
}

// setStageTitle sets the title of the stage based on the current page URL.
func (pg *Page) setStageTitle() {
	if rw := pg.Scene.RenderWindow(); rw != nil {
		rw.SetStageTitle(ppath.Label(pg.Context.PageURL, core.TheApp.Name()))
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
		if !strings.HasSuffix(pg.pagePath, "index.md") && !strings.HasSuffix(pg.pagePath, "index.html") {
			current = path.Dir(current) // we must go up one if we are not the index page (which is already up one)
		}
		rawURL = path.Join(current, rawURL)
	}
	if rawURL == ".." {
		rawURL = ""
	}

	// the paths in the fs are never rooted, so we trim a rooted one
	rawURL = strings.TrimPrefix(rawURL, "/")
	rawURL = strings.TrimSuffix(rawURL, "/")

	pg.pagePath = pg.urlToPagePath[rawURL]

	b, err := fs.ReadFile(pg.Source, pg.pagePath)
	if err != nil {
		core.ErrorSnackbar(pg, err, "Error opening page "+rawURL)
		return
	}

	pg.Context.PageURL = rawURL
	if addToHistory {
		pg.historyIndex = len(pg.history)
		pg.history = append(pg.history, pg.Context.PageURL)
	}
	if saveWebURL != nil {
		saveWebURL(pg, pg.Context.PageURL)
	}

	var frontMatter map[string]any
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
			errors.Log(tomlx.ReadBytes(&frontMatter, fmb))
		}
	}

	// need to reset
	NumExamples[pg.Context.PageURL] = 0

	pg.nav.UnselectAll()
	curNav := pg.nav.FindPath(rawURL).(*core.Tree)
	// we must select the first tree that does not have "no-index"
	if curNav.Property("no-index") != nil {
		got := false
		curNav.WalkDown(func(n tree.Node) bool {
			if got {
				return tree.Break
			}
			tr, ok := n.(*core.Tree)
			if !ok || n.AsTree().Property("no-index") != nil {
				return tree.Continue
			}
			curNav = tr
			got = true
			return tree.Break
		})
	}
	curNav.Select()
	curNav.ScrollToThis()
	pg.Context.PageURL = curNav.PathFrom(pg.nav)
	pg.setStageTitle()

	pg.body.DeleteChildren()
	if ppath.Draft(pg.pagePath) {
		draft := core.NewText(pg.body).SetType(core.TextDisplayMedium).SetText("DRAFT")
		draft.Styler(func(s *styles.Style) {
			s.Color = colors.Scheme.Error.Base
			s.Font.Weight = styles.WeightBold
		})
	}
	if curNav != pg.nav {
		bc := core.NewText(pg.body).SetText(ppath.Breadcrumbs(pg.Context.PageURL, core.TheApp.Name()))
		bc.HandleTextClick(func(tl *paint.TextLink) {
			pg.Context.OpenURL(tl.URL)
		})
		core.NewText(pg.body).SetType(core.TextDisplaySmall).SetText(curNav.Text)
	}
	if author := frontMatter["author"]; author != nil {
		author := slicesx.As[any, string](author.([]any))
		core.NewText(pg.body).SetType(core.TextTitleLarge).SetText("By " + strcase.FormatList(author...))
	}
	base := strings.TrimPrefix(path.Base(pg.pagePath), "-")
	if len(base) >= 10 {
		date := base[:10]
		if t, err := time.Parse("2006-01-02", date); err == nil {
			core.NewText(pg.body).SetType(core.TextTitleMedium).SetText(t.Format("1/2/2006"))
		}
	}
	err = htmlcore.ReadMD(pg.Context, pg.body, b)
	if err != nil {
		core.ErrorSnackbar(pg, err, "Error loading page")
		return
	}

	if curNav != pg.nav {
		buttons := core.NewFrame(pg.body)
		buttons.Styler(func(s *styles.Style) {
			s.Align.Items = styles.Center
			s.Grow.Set(1, 0)
		})
		if previous, ok := tree.Previous(curNav).(*core.Tree); ok {
			// we must skip over trees with "no-index" to get to a real new page
			for previous != nil && previous.Property("no-index") != nil {
				previous, _ = tree.Previous(previous).(*core.Tree)
			}
			if previous != nil {
				bt := core.NewButton(buttons).SetText("Previous").SetIcon(icons.ArrowBack).SetType(core.ButtonTonal)
				bt.OnClick(func(e events.Event) {
					curNav.Unselect()
					previous.SelectEvent(events.SelectOne)
				})
			}
		}
		if next, ok := tree.Next(curNav).(*core.Tree); ok {
			core.NewStretch(buttons)
			bt := core.NewButton(buttons).SetText("Next").SetIcon(icons.ArrowForward).SetType(core.ButtonTonal)
			bt.OnClick(func(e events.Event) {
				curNav.Unselect()
				next.SelectEvent(events.SelectOne)
			})
		}
	}

	pg.body.Update()
	pg.body.ScrollDimToContentStart(math32.Y)
}

func (pg *Page) MakeToolbar(p *tree.Plan) {
	tree.AddInit(p, "back", func(w *core.Button) {
		w.OnClick(func(e events.Event) {
			if pg.historyIndex > 0 {
				pg.historyIndex--
				// we need a slash so that it doesn't think it's a relative URL
				pg.OpenURL("/"+pg.history[pg.historyIndex], false)
				e.SetHandled()
			}
		})
	})
	tree.AddInit(p, "app-chooser", func(w *core.Chooser) {
		w.AddItemsFunc(func() {
			urls := []string{}
			for u := range pg.urlToPagePath {
				urls = append(urls, u)
			}
			slices.Sort(urls)
			for _, u := range urls {
				w.Items = append(w.Items, core.ChooserItem{
					Value: u,
					Text:  ppath.Label(u, core.TheApp.Name()),
					Func: func() {
						pg.OpenURL("/"+u, true)
					},
				})
			}
		})
	})
}
