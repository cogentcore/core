// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/keyfun"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/states"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
)

// MenuSceneConfigStyles configures the default styles
// for the given pop-up menu frame with the given parent.
// It should be called on menu frames when they are created.
func MenuSceneConfigStyles(msc *Scene) {
	msc.Style(func(s *styles.Style) {
		s.Grow.Set(0, 0)
		s.Padding.Set(units.Dp(2))
		s.Border.Radius = styles.BorderRadiusExtraSmall
		s.Background = colors.C(colors.Scheme.SurfaceContainer)
		s.BoxShadow = styles.BoxShadow2()
		s.Gap.Zero()
	})
	msc.OnWidgetAdded(func(w Widget) {
		if bt := AsButton(w); bt != nil {
			bt.Type = ButtonMenu
			bt.OnKeyChord(func(e events.Event) {
				kf := keyfun.Of(e.KeyChord())
				switch kf {
				case keyfun.MoveRight:
					if bt.OpenMenu(e) {
						e.SetHandled()
					}
				case keyfun.MoveLeft:
					// need to be able to use arrow keys to navigate in completer
					if msc.Stage.Type != CompleterStage {
						msc.Stage.ClosePopup()
						e.SetHandled()
					}
				}
			})
			return
		}
		if sp, ok := w.(*Separator); ok {
			sp.Style(func(s *styles.Style) {
				s.Direction = styles.Row
			})
		}
	})
}

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

// AddContextMenu adds the given context menu to [WidgetBase.ContextMenus].
// It is the main way that code should modify a widget's context menus.
// Context menu functions are run in reverse order, and separators are
// automatically added between each context menu function.
func (wb *WidgetBase) AddContextMenu(menu func(m *Scene)) *WidgetBase {
	wb.ContextMenus = append(wb.ContextMenus, menu)
	return wb
}

// ApplyContextMenus adds the [Widget.ContextMenus] to the given menu scene
// in reverse order. It also adds separators between each context menu function.
func (wb *WidgetBase) ApplyContextMenus(m *Scene) {
	for i := len(wb.ContextMenus) - 1; i >= 0; i-- {
		wb.ContextMenus[i](m)

		nc := m.NumChildren()
		// we delete any extra separator
		if nc > 0 && m.Child(nc-1).KiType() == SeparatorType {
			m.DeleteChildAtIndex(nc-1, true)
		}
		if i != 0 {
			NewSeparator(m)
		}
	}
}

// ContextMenuPos returns the default position for the context menu
// upper left corner.  The event will be from a mouse ContextMenu
// event if non-nil: should handle both cases.
func (wb *WidgetBase) ContextMenuPos(e events.Event) image.Point {
	if e != nil {
		return e.WindowPos()
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
	nm := NewMenu(wi.ApplyContextMenus, wi, wi.ContextMenuPos(e))
	if nm == nil { // no items
		return
	}
	nm.Run()
}

// NewMenuFromStrings constructs a new menu from given list of strings,
// calling the given function with the index of the selected string.
// if string == sel, that menu item is selected initially.
func NewMenuFromStrings(strs []string, sel string, fun func(index int)) *Scene {
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
