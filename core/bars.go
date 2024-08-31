// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"slices"
	"strings"

	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
)

// BarFuncs are functions for creating control bars,
// attached to different sides of a [Scene]. Functions
// are called in forward order so first added are called first.
type BarFuncs []func(parent Widget)

// Add adds the given function for configuring a control bar
func (bf *BarFuncs) Add(fun func(parent Widget)) *BarFuncs {
	*bf = append(*bf, fun)
	return bf
}

// call calls all the functions for configuring given widget
func (bf *BarFuncs) call(parent Widget) {
	for _, fun := range *bf {
		fun(parent)
	}
}

// isEmpty returns true if there are no functions added
func (bf *BarFuncs) isEmpty() bool {
	return len(*bf) == 0
}

// makeSceneBars configures the side control bars, for main scenes.
func (sc *Scene) makeSceneBars() {
	sc.addDefaultBars()
	if !sc.Bars.Top.isEmpty() {
		head := NewFrame(sc)
		head.SetName("top-bar")
		head.Styler(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Grow.Set(1, 0)
		})
		sc.Bars.Top.call(head)
	}
	if !sc.Bars.Left.isEmpty() || !sc.Bars.Right.isEmpty() {
		mid := NewFrame(sc)
		mid.SetName("body-area")
		if !sc.Bars.Left.isEmpty() {
			left := NewFrame(mid)
			left.SetName("left-bar")
			left.Styler(func(s *styles.Style) {
				s.Direction = styles.Column
				s.Align.Items = styles.Center
				s.Grow.Set(0, 1)
			})
			sc.Bars.Left.call(left)
		}
		if sc.Body != nil {
			mid.AddChild(sc.Body)
		}
		if !sc.Bars.Right.isEmpty() {
			right := NewFrame(mid)
			right.SetName("right-bar")
			right.Styler(func(s *styles.Style) {
				s.Direction = styles.Column
				s.Align.Items = styles.Center
				s.Grow.Set(0, 1)
			})
			sc.Bars.Right.call(right)
		}
	} else {
		if sc.Body != nil {
			sc.AddChild(sc.Body)
		}
	}
	if !sc.Bars.Bottom.isEmpty() {
		foot := NewFrame(sc)
		foot.SetName("bottom-bar")
		foot.Styler(func(s *styles.Style) {
			s.Justify.Content = styles.End
			s.Align.Items = styles.Center
			s.Grow.Set(1, 0)
		})
		sc.Bars.Bottom.call(foot)
	}
}

func (sc *Scene) addDefaultBars() {
	// TODO(appbar): remove default default app bar adding
	if sc.Stage.Type.isMain() && (sc.Stage.NewWindow || sc.Stage.FullWindow) {
		if len(sc.AppBars) > 0 {
			sc.Bars.Top.Add(makeAppBar)
		}
	}

	st := sc.Stage
	needBackButton := st.FullWindow && !st.NewWindow && !(st.Mains != nil && st.Mains.stack.Len() == 0)
	if st.DisplayTitle || needBackButton {
		sc.Bars.Top = slices.Insert(sc.Bars.Top, 0, func(parent Widget) {
			titleRow := NewFrame(parent)
			titleRow.SetName("title-row")
			titleRow.Styler(func(s *styles.Style) {
				s.Grow.Set(1, 0)
				s.Align.Items = styles.Center
			})
			if needBackButton {
				back := NewButton(titleRow).SetIcon(icons.ArrowBack).SetKey(keymap.HistPrev)
				back.SetType(ButtonAction).SetTooltip("Back")
				back.OnClick(func(e events.Event) {
					sc.Close()
				})
			}
			if st.DisplayTitle {
				title := NewText(titleRow).SetType(TextHeadlineSmall)
				title.Updater(func() {
					title.SetText(sc.Body.Title)
				})
			}
		})
	}
}

// TODO(appbar): remove GetBar and GetTopAppBar

// GetBar returns Bar layout widget at given side, nil if not there.
func (sc *Scene) GetBar(side styles.SideIndexes) *Frame {
	nm := strings.ToLower(side.String()) + "-bar"
	bar := sc.ChildByName(nm)
	if bar != nil {
		return bar.(*Frame)
	}
	return nil
}

// GetTopAppBar returns the TopAppBar Toolbar if it exists, nil otherwise.
func (sc *Scene) GetTopAppBar() *Toolbar {
	tb := sc.GetBar(styles.Top)
	if tb == nil {
		return nil
	}
	return tree.ChildByType[*Toolbar](tb)
}

//////////////////////////////////////////////////////////////
// 	Scene wrappers

// AddTopBar adds the given function for configuring a control bar
// at the top of the window
func (bd *Body) AddTopBar(fun func(parent Widget)) {
	bd.Scene.Bars.Top.Add(fun)
}

// AddLeftBar adds the given function for configuring a control bar
// on the left of the window
func (bd *Body) AddLeftBar(fun func(parent Widget)) {
	bd.Scene.Bars.Left.Add(fun)
}

// AddRightBar adds the given function for configuring a control bar
// on the right of the window
func (bd *Body) AddRightBar(fun func(parent Widget)) {
	bd.Scene.Bars.Right.Add(fun)
}

// AddBottomBar adds the given function for configuring a control bar
// at the bottom of the window
func (bd *Body) AddBottomBar(fun func(parent Widget)) {
	bd.Scene.Bars.Bottom.Add(fun)
}

// AddAppBar adds plan maker function(s) for the top app bar, which can be used to add items to it.
func (bd *Body) AddAppBar(m ...func(p *tree.Plan)) {
	bd.Scene.AppBars = append(bd.Scene.AppBars, m...)
}
