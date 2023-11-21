// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"goki.dev/girl/styles"
	"goki.dev/ki/v2"
)

// BarFuncs are functions for creating control bars,
// attached to different sides of a Scene (e.g., TopAppBar at Top,
// NavBar at Bottom, etc).  Functions are called in forward order
// so first added are called first.
type BarFuncs []func(pw Widget)

// Add adds given function for configuring a control bar
func (bf *BarFuncs) Add(fun func(pw Widget)) *BarFuncs {
	*bf = append(*bf, fun)
	return bf
}

// Call calls all the functions for configuring given widget
func (bf *BarFuncs) Call(pw Widget) {
	for _, fun := range *bf {
		fun(pw)
	}
}

// IsEmpty returns true if there are no functions added
func (bf *BarFuncs) IsEmpty() bool {
	return len(*bf) == 0
}

// ConfigScene is called by the Stage to configure the Scene during Run process
func (sc *Scene) ConfigScene() {
	if !sc.Bars.Top.IsEmpty() {
		head := NewLayout(sc, "top-bar").Style(func(s *styles.Style) {
			s.Grow.Set(1, 0)
		})
		sc.Bars.Top.Call(head)
	}
	if !sc.Bars.Left.IsEmpty() || !sc.Bars.Right.IsEmpty() {
		mid := NewLayout(sc, "body-area")
		if !sc.Bars.Left.IsEmpty() {
			left := NewLayout(mid, "left-bar")
			sc.Bars.Left.Call(left)
		}
		if sc.Body != nil {
			mid.AddChild(sc.Body)
		}
		if !sc.Bars.Right.IsEmpty() {
			right := NewLayout(mid, "right-bar")
			sc.Bars.Right.Call(right)
		}
	} else {
		if sc.Body != nil {
			sc.AddChild(sc.Body)
		}
	}
	if !sc.Bars.Bottom.IsEmpty() {
		foot := NewLayout(sc, "bottom-bar").Style(func(s *styles.Style) {
			s.Justify.Content = styles.End
			s.Grow.Set(1, 0)
		})
		sc.Bars.Bottom.Call(foot)
	}
}

// TopAppBar constructs or returns the TopAppBar in given parent Widget
func (sc *Scene) TopAppBar(pw Widget) *TopAppBar {
	tb := pw.ChildByType(TopAppBarType, ki.Embeds)
	if tb != nil {
		return tb.(*TopAppBar)
	}
	return NewTopAppBar(pw)
}

// DefaultTopAppBar constructs or returns the TopAppBar in given parent Widget.
// if DefaultTopAppBar function is != nil, and we are making a new
// TopAppBar widget, then we call DefaultTopAppBar first.
func (sc *Scene) DefaultTopAppBar(pw Widget) *TopAppBar {
	tbi := pw.ChildByType(TopAppBarType, ki.Embeds)
	if tbi != nil {
		return tbi.(*TopAppBar)
	}
	tb := NewTopAppBar(pw)
	if DefaultTopAppBar != nil {
		DefaultTopAppBar(tb)
	}
	return tb
}

// AddDefaultTopAppBar adds code to Bars.Top to make a
// DefaultTopAppBar on this Scene
func (sc *Scene) AddDefaultTopAppBar() {
	sc.Bars.Top.Add(func(pw Widget) {
		if DefaultTopAppBar != nil {
			sc.DefaultTopAppBar(pw)
		}
	})
}

// InheritBars inherits Bars functions from given other scene
// for each side that the other scene marks as inherited.
func (sc *Scene) InheritBars(osc *Scene) {
	if osc == nil {
		return
	}
	if osc.BarsInherit.Top || sc.BarsInherit.Top {
		sc.Bars.Top = osc.Bars.Top
		sc.BarsInherit.Top = true
	}
	if osc.BarsInherit.Bottom || sc.BarsInherit.Bottom {
		sc.Bars.Bottom = osc.Bars.Bottom
		sc.BarsInherit.Bottom = true
	}
	if osc.BarsInherit.Left || sc.BarsInherit.Left {
		sc.Bars.Left = osc.Bars.Left
		sc.BarsInherit.Left = true
	}
	if osc.BarsInherit.Right || sc.BarsInherit.Right {
		sc.Bars.Right = osc.Bars.Right
		sc.BarsInherit.Right = true
	}
}

//////////////////////////////////////////////////////////////
// 	Body wrappers

// Add adds given function for configuring a control bar
// at the top of the window
func (bd *Body) AddTopBar(fun func(pw Widget)) {
	bd.Sc.Bars.Top.Add(fun)
}

// Add adds given function for configuring a control bar
// at the top of the window
func (bd *Body) AddLeftBar(fun func(pw Widget)) {
	bd.Sc.Bars.Left.Add(fun)
}

// Add adds given function for configuring a control bar
// at the top of the window
func (bd *Body) AddRightBar(fun func(pw Widget)) {
	bd.Sc.Bars.Right.Add(fun)
}

// Add adds given function for configuring a control bar
// at the top of the window
func (bd *Body) AddBottomBar(fun func(pw Widget)) {
	bd.Sc.Bars.Bottom.Add(fun)
}

// TopAppBar constructs or returns the TopAppBar in given parent Widget
func (bd *Body) TopAppBar(pw Widget) *TopAppBar {
	return bd.Sc.TopAppBar(pw)
}

// DefaultTopAppBar constructs or returns the TopAppBar in given parent Widget
func (bd *Body) DefaultTopAppBar(pw Widget) *TopAppBar {
	return bd.Sc.DefaultTopAppBar(pw)
}

// AddDefaultTopAppBar adds code to Bars.Top to make a
// DefaultTopAppBar on this Scene
func (bd *Body) AddDefaultTopAppBar() {
	bd.Sc.AddDefaultTopAppBar()
}
