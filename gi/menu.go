// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"

	"goki.dev/colors"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/ki/v2"
)

// MenuSceneConfigStyles configures the default styles
// for the given pop-up menu frame with the given parent.
// It should be called on menu frames when they are created.
func MenuSceneConfigStyles(msc *Scene) {
	msc.Style(func(s *styles.Style) {
		s.Grow.Set(0, 0)
		s.Padding.Set(units.Dp(2))
		s.Border.Radius = styles.BorderRadiusExtraSmall
		s.BackgroundColor.SetSolid(colors.Scheme.SurfaceContainer)
		s.BoxShadow = styles.BoxShadow2()
		s.Gap.Zero()
	})
	msc.OnWidgetAdded(func(w Widget) {
		if bt := AsButton(w); bt != nil {
			bt.Type = ButtonMenu
			return
		}
		if sp, ok := w.(*Separator); ok {
			sp.Horiz = true
		}
	})
}

// MenuMaxHeight is the maximum height of any menu popup panel in units of font height
// scroll bars are enforced beyond that size.
var MenuMaxHeight = 30

// NewMenuScene constructs a [Scene] for displaying a menu, using the
// given menu constructor function. If no name is provided, it defaults
// to "menu".  If no menu items added, returns nil.
func NewMenuScene(menu func(m *Scene), name ...string) *Scene {
	nm := "menu"
	if len(name) > 0 {
		nm = name[0] + "-menu"
	}
	msc := NewScene(nm)
	MenuSceneConfigStyles(msc)
	menu(msc)
	if !msc.HasChildren() {
		return nil
	}

	hasSelected := false
	msc.WidgetWalkPre(func(wi Widget, wb *WidgetBase) bool {
		if wi.This() == msc.This() {
			return ki.Continue
		}
		if bt := AsButton(wi); bt != nil {
			if bt.Menu == nil {
				bt.HandleClickDismissMenu()
			}
		}
		if !hasSelected && wb.StateIs(states.Selected) {
			// fmt.Println("start focus sel:", wb)
			msc.EventMgr.SetStartFocus(wb)
			hasSelected = true
		}
		return ki.Continue
	})
	if !hasSelected && msc.HasChildren() {
		// fmt.Println("start focus first:", msc.Child(0).(Widget))
		msc.EventMgr.SetStartFocus(msc.Child(0).(Widget))
	}
	return msc
}

// NewMenuStage returns a new Menu stage with given scene contents,
// in connection with given widget, which provides key context
// for constructing the menu, at given RenderWin position
// (e.g., use ContextMenuPos or WinPos method on ctx Widget).
// Make further configuration choices using Set* methods, which
// can be chained directly after the New call.
// Use Run call at the end to start the Stage running.
func NewMenuStage(sc *Scene, ctx Widget, pos image.Point) *Stage {
	if sc == nil || !sc.HasChildren() {
		return nil
	}
	st := NewPopupStage(MenuStage, sc, ctx)
	zp := image.Point{}
	if pos != zp {
		st.Pos = pos
	}
	return st
}

// NewMenu returns a new menu stage based on the given menu constructor
// function, in connection with given widget, which provides key context
// for constructing the menu at given RenderWin position
// (e.g., use ContextMenuPos or WinPos method on ctx Widget).
// Make further configuration choices using Set* methods, which
// can be chained directly after the New call.
// Use Run call at the end to start the Stage running.
func NewMenu(menu func(m *Scene), ctx Widget, pos image.Point) *Stage {
	return NewMenuStage(NewMenuScene(menu, ctx.Name()), ctx, pos)
}

func (wb *WidgetBase) ContextMenu(m *Scene) {
	// derived types put native menu code here
	if wb.CustomContextMenu != nil {
		wb.CustomContextMenu(m)
	}
	TheViewIFace.CtxtMenuView(wb.This(), wb.IsDisabled(), wb.Sc, m)
}

// ContextMenuPos returns the default position for the context menu
// upper left corner.  The event will be from a mouse ContextMenu
// event if non-nil: should handle both cases.
func (wb *WidgetBase) ContextMenuPos(e events.Event) image.Point {
	if e != nil {
		return e.Pos()
	}
	return wb.WinPos(.5, .5) // center
}

func (wb *WidgetBase) HandleWidgetContextMenu() {
	wb.On(events.ContextMenu, func(e events.Event) {
		wi := wb.This().(Widget)
		wi.ShowContextMenu(e)
	})
}

func (wb *WidgetBase) ShowContextMenu(e events.Event) {
	e.SetHandled() // always
	wi := wb.This().(Widget)
	nm := NewMenu(wi.ContextMenu, wi, wi.ContextMenuPos(e))
	if nm == nil { // no items
		return
	}
	nm.Run()
}

// NewMenuFromStrings constructs a new menu from given list of strings,
// calling the given function with the index of the selected string.
// if string == sel, that menu item is selected initially.
func NewMenuFromStrings(strs []string, sel string, fun func(idx int)) *Scene {
	return NewMenuScene(func(m *Scene) {
		for i, s := range strs {
			i := i
			s := s
			b := NewButton(m).SetText(s).OnClick(func(e events.Event) {
				fun(i)
			})
			if s == sel {
				b.SetSelected(true)
			}
		}
	})
}
