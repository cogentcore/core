// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package content provides a system for making content-focused
// apps and websites consisting of Markdown, HTML, and Cogent Core.
package content

//go:generate core generate

import (
	"io/fs"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
)

// Content manages and displays the content of a set of pages.
type Content struct {
	core.Frame

	// Source is the source filesystem for the content.
	// It should be set using [Content.SetSource] or [Content.SetContent].
	Source fs.FS `set:"-"`

	// pages are the pages that constitute the content.
	pages []*Page

	// pagesByURL has the [Page] for each URL.
	pagesByURL map[string]*Page

	// currentPage is the currently open page.
	currentPage *Page

	// renderedPage is the most recently rendered page.
	renderedPage *Page
}

func (ct *Content) Init() {
	ct.Frame.Init()
	ct.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})
	ct.Maker(func(p *tree.Plan) {
		if ct.currentPage == nil {
			return
		}
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
				if ct.renderedPage == ct.currentPage {
					return
				}
				w.DeleteChildren()
				b, err := ct.currentPage.ReadContent()
				if errors.Log(err) != nil {
					return
				}
				err = htmlcore.ReadMD(htmlcore.NewContext(), w, b)
				if errors.Log(err) != nil {
					return
				}
				ct.renderedPage = ct.currentPage
			})
		})
	})
}

// SetSource sets the source filesystem for the content.
func (ct *Content) SetSource(source fs.FS) *Content {
	ct.Source = source
	ct.pages = []*Page{}
	ct.pagesByURL = map[string]*Page{}
	errors.Log(fs.WalkDir(ct.Source, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "" || path == "." {
			return nil
		}
		pg, err := NewPage(ct.Source, path)
		if err != nil {
			return err
		}
		ct.pages = append(ct.pages, pg)
		ct.pagesByURL[pg.URL] = pg
		return nil
	}))
	if root, ok := ct.pagesByURL[""]; ok {
		ct.currentPage = root
	} else {
		ct.currentPage = ct.pages[0]
	}
	return ct
}

// SetContent is a helper function that calls [Content.SetSource]
// with the "content" subdirectory of the given filesystem.
func (ct *Content) SetContent(content fs.FS) *Content {
	return ct.SetSource(fsx.Sub(content, "content"))
}

// Open opens the page with the given URL and updates the display.
func (ct *Content) Open(url string) *Content {
	ct.currentPage = ct.pagesByURL[url]
	ct.Update()
	return ct
}
