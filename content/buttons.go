// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package content

import (
	"slices"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/system"
	"cogentcore.org/core/tree"
)

// MakeToolbar adds the standard toolbar buttons for the content.
// See [Content.MakeToolbarPDF] for the optional PDF button.
func (ct *Content) MakeToolbar(p *tree.Plan) {
	if false && ct.SizeClass() == core.SizeCompact { // TODO: implement hamburger menu for compact
		tree.Add(p, func(w *core.Button) {
			w.SetIcon(icons.Menu)
			w.SetTooltip("Navigate pages and headings")
			w.OnClick(func(e events.Event) {
				d := core.NewBody("Navigate")
				// tree.MoveToParent(ct.leftFrame, d)
				d.AddBottomBar(func(bar *core.Frame) {
					d.AddCancel(bar)
				})
				d.RunDialog(w)
			})
		})
	}
	tree.Add(p, func(w *core.Button) {
		w.SetIcon(icons.Icon(core.AppIcon))
		w.SetTooltip("Home")
		w.OnClick(func(e events.Event) {
			ct.OpenEvent("", e)
		})
	})
	// Superseded by browser navigation on web.
	if core.TheApp.Platform() != system.Web {
		tree.Add(p, func(w *core.Button) {
			w.SetIcon(icons.ArrowBack).SetKey(keymap.HistPrev)
			w.SetTooltip("Back")
			w.Updater(func() {
				w.SetEnabled(ct.historyHasBack())
			})
			w.OnClick(func(e events.Event) {
				ct.historyBack()
			})
		})
		tree.Add(p, func(w *core.Button) {
			w.SetIcon(icons.ArrowForward).SetKey(keymap.HistNext)
			w.SetTooltip("Forward")
			w.Updater(func() {
				w.SetEnabled(ct.historyHasForward())
			})
			w.OnClick(func(e events.Event) {
				ct.historyForward()
			})
		})
	}
	tree.Add(p, func(w *core.Button) {
		w.SetText("Search").SetIcon(icons.Search).SetKey(keymap.Menu)
		w.Styler(func(s *styles.Style) {
			s.Background = colors.Scheme.SurfaceVariant
			s.Padding.Right.Em(5)
		})
		w.OnClick(func(e events.Event) {
			ct.Scene.MenuSearchDialog("Search", "Search "+core.TheApp.Name())
		})
	})
}

// MakeToolbarPDF adds the PDF button to the toolbar. This is optional.
func (ct *Content) MakeToolbarPDF(p *tree.Plan) {
	tree.Add(p, func(w *core.Button) {
		w.SetText("PDF").SetIcon(icons.PictureAsPdf).SetTooltip("PDF generates and opens / downloads the current page as a printable PDF file. See the Settings/Printer panel (Command+,) for settings.")
		w.OnClick(func(e events.Event) {
			ct.PagePDF("pdfs")
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

// makeBottomButtons makes the previous and next buttons if relevant.
func (ct *Content) makeBottomButtons(p *tree.Plan) {
	if len(ct.currentPage.Categories) == 0 {
		return
	}
	cat := ct.currentPage.Categories[0]
	pages := ct.pagesByCategory[cat]
	idx := slices.Index(pages, ct.currentPage)

	ct.prevPage, ct.nextPage = nil, nil

	if idx > 0 {
		ct.prevPage = pages[idx-1]
	}
	if idx < len(pages)-1 {
		ct.nextPage = pages[idx+1]
	}

	if ct.prevPage == nil && ct.nextPage == nil {
		return
	}

	tree.Add(p, func(w *core.Frame) {
		w.Styler(func(s *styles.Style) {
			s.Align.Items = styles.Center
			s.Grow.Set(1, 0)
		})
		w.Maker(func(p *tree.Plan) {
			if ct.prevPage != nil {
				tree.Add(p, func(w *core.Button) {
					w.SetText("Previous").SetIcon(icons.ArrowBack).SetType(core.ButtonTonal)
					ct.Context.LinkButtonUpdating(w, func() string { // needed to prevent stale URL variable
						return ct.prevPage.URL
					})
				})
			}
			if ct.nextPage != nil {
				tree.Add(p, func(w *core.Stretch) {})
				tree.Add(p, func(w *core.Button) {
					w.SetText("Next").SetIcon(icons.ArrowForward).SetType(core.ButtonTonal)
					ct.Context.LinkButtonUpdating(w, func() string { // needed to prevent stale URL variable
						return ct.nextPage.URL
					})
				})
			}
		})
	})
}
