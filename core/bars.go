// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"slices"

	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
)

// BarFuncs are functions for creating control bars,
// attached to different sides of a [Scene]. Functions
// are called in forward order so first added are called first.
type BarFuncs []func(bar *Frame)

// Add adds the given function for configuring a control bar
func (bf *BarFuncs) Add(fun func(bar *Frame)) *BarFuncs {
	*bf = append(*bf, fun)
	return bf
}

// call calls all the functions for configuring given widget
func (bf *BarFuncs) call(bar *Frame) {
	for _, fun := range *bf {
		fun(bar)
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
			s.Align.Items = styles.Center
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
	st := sc.Stage
	addBack := st.BackButton.Or(st.FullWindow && !st.NewWindow && !(st.Mains != nil && st.Mains.stack.Len() == 0))
	if addBack || st.DisplayTitle {
		sc.Bars.Top = slices.Insert(sc.Bars.Top, 0, func(bar *Frame) {
			if addBack {
				back := NewButton(bar).SetIcon(icons.ArrowBack).SetKey(keymap.HistPrev)
				back.SetType(ButtonAction).SetTooltip("Back")
				back.OnClick(func(e events.Event) {
					sc.Close()
				})
			}
			if st.DisplayTitle {
				title := NewText(bar).SetType(TextHeadlineSmall)
				title.Updater(func() {
					title.SetText(sc.Body.Title)
				})
			}
		})
	}
}

////////  Scene wrappers

// AddTopBar adds the given function for configuring a control bar
// at the top of the window
func (bd *Body) AddTopBar(fun func(bar *Frame)) {
	bd.Scene.Bars.Top.Add(fun)
}

// AddLeftBar adds the given function for configuring a control bar
// on the left of the window
func (bd *Body) AddLeftBar(fun func(bar *Frame)) {
	bd.Scene.Bars.Left.Add(fun)
}

// AddRightBar adds the given function for configuring a control bar
// on the right of the window
func (bd *Body) AddRightBar(fun func(bar *Frame)) {
	bd.Scene.Bars.Right.Add(fun)
}

// AddBottomBar adds the given function for configuring a control bar
// at the bottom of the window
func (bd *Body) AddBottomBar(fun func(bar *Frame)) {
	bd.Scene.Bars.Bottom.Add(fun)
}
