// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"

	"cogentcore.org/core/styles"
)

// Pages is a frame that can easily swap its content between that of
// different possible pages.
type Pages struct {
	Frame

	// Page is the currently open page.
	Page string

	// Pages is a map of page names to functions that configure a page.
	Pages map[string]func(pg *Pages) `set:"-"`

	// page is the currently rendered page.
	page string
}

func (pg *Pages) Init() {
	pg.Frame.Init()
	pg.Pages = map[string]func(pg *Pages){}
	pg.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
	})
	pg.Updater(func() {
		if len(pg.Pages) == 0 {
			return
		}
		if pg.page == pg.Page {
			return
		}
		pg.DeleteChildren()
		fun, ok := pg.Pages[pg.Page]
		if !ok {
			ErrorSnackbar(pg, fmt.Errorf("page %q not found", pg.Page))
			return
		}
		pg.page = pg.Page
		fun(pg)
		pg.DeferShown()
	})
}

// AddPage adds a page with the given name and configuration function.
// If [Pages.Page] is currently unset, it will be set to the given name.
func (pg *Pages) AddPage(name string, f func(pg *Pages)) {
	pg.Pages[name] = f
	if pg.Page == "" {
		pg.Page = name
	}
}

// Open sets the current page to the given name and updates the display.
// In comparison, [Pages.SetPage] does not update the display and should typically
// only be called at the start.
func (pg *Pages) Open(name string) *Pages {
	pg.SetPage(name)
	pg.Update()
	return pg
}
