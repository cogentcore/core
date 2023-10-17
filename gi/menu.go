// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"

	"goki.dev/colors"
	"goki.dev/girl/styles"
)

// MenuSceneConfigStyles configures the default styles
// for the given pop-up menu frame with the given parent.
// It should be called on menu frames when they are created.
func MenuSceneConfigStyles(msc *Scene) {
	msc.Style(func(s *styles.Style) {
		s.Border.Style.Set(styles.BorderNone)
		s.Border.Radius = styles.BorderRadiusExtraSmall
		s.BackgroundColor.SetSolid(colors.Scheme.SurfaceContainer)
		s.BoxShadow = styles.BoxShadow2()
	})
}

// MenuMaxHeight is the maximum height of any menu popup panel in units of font height
// scroll bars are enforced beyond that size.
var MenuMaxHeight = 30

// NewMenuScene constructs a [Scene] for displaying a menu, using the
// given menu constructor function. If no name is provided, it defaults
// to "menu".
func NewMenuScene(menu func(menu *Scene), name ...string) *Scene {
	nm := "menu"
	if len(name) > 0 {
		nm = name[0] + "-menu"
	}
	msc := NewScene(nm)
	MenuSceneConfigStyles(msc)
	menu(msc)

	// TODO(kai/menu): do we need this?
	/*
		 hasSelected := false
			for _, ac := range menu {
				wi, wb := AsWidget(ac)
				if wi == nil {
					continue
				}
				cl := wi.Clone().This().(Widget)
				cb := cl.AsWidget()
				if bt, ok := cl.(*Button); ok {
					bt.Type = ButtonMenu
					if bt.Menu == nil {
						cb.Listeners[events.Click] = wb.Listeners[events.Click]
						bt.HandleClickDismissMenu()
					}
				}
				cb.Sc = msc
				msc.AddChild(cl)
				if !hasSelected && cb.StateIs(states.Selected) {
					msc.EventMgr.SetStartFocus(cl)
					hasSelected = true
				}
			}
			if !hasSelected && msc.HasChildren() {
				msc.EventMgr.SetStartFocus(msc.Child(0).(Widget))
			}
	*/
	return msc
}

// NewMenuFromScene returns a new Menu stage with given scene contents,
// in connection with given widget, which provides key context
// for constructing the menu, at given RenderWin position
// (e.g., use ContextMenuPos or WinPos method on ctx Widget).
// Typically use NewMenu which takes a standard [Menu].
// Make further configuration choices using Set* methods, which
// can be chained directly after the New call.
// Use Run call at the end to start the Stage running.
func NewMenuFromScene(sc *Scene, ctx Widget, pos image.Point) *PopupStage {
	sc.Geom.Pos = pos
	return NewPopupStage(MenuStage, sc, ctx)
}

// NewMenu returns a new menu stage based on the given menu constructor
// function, in connection with given widget, which provides key context
// for constructing the menu at given RenderWin position
// (e.g., use ContextMenuPos or WinPos method on ctx Widget).
// Make further configuration choices using Set* methods, which
// can be chained directly after the New call.
// Use Run call at the end to start the Stage running.
func NewMenu(menu func(menu *Scene), ctx Widget, pos image.Point) *PopupStage {
	return NewMenuFromScene(NewMenuScene(menu, ctx.Name()), ctx, pos)
}
