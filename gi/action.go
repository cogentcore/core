// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"goki.dev/goosi/events/key"
	"goki.dev/icons"
)

// ActOpts provides named and partial parameters for AddAction method
type ActOpts struct {
	Name        string
	Label       string
	Icon        icons.Icon
	Tooltip     string
	Shortcut    key.Chord
	ShortcutKey KeyFuns
	Data        any
	UpdateFunc  func(bt *Button)
}

/*

// Action is a button widget that can display a text label and / or an icon
// and / or a keyboard shortcut -- this is what is put in menus, menubars, and
// toolbars, and also for any standalone simple action.  The default styling
// differs depending on whether it is in a Menu versus a MenuBar or ToolBar --
// this is controlled by the Class which is automatically set to
// menu, menubar, or toolbar.
// Action functions provide the *Action that generated them, which has
// a Data value that can be used to determine the proper action to take,
// in the case of automatically-generated chooser-type menus.
// The Action(s) are called via the On(events.Click) action,
// that wraps the func(act *Action) call.
//
//goki:embedder
type Action struct {
	Button

	// [view: -] optional data that is sent with the ActionSig when it is emitted
	Data any `json:"-" xml:"-" view:"-" desc:"optional data that is sent with the ActionSig when it is emitted"`

	// [view: -] optional function that is called to update state of action (typically updating Active state) -- called automatically for menus prior to showing
	UpdateFunc func(act *Action) `json:"-" xml:"-" view:"-" desc:"optional function that is called to update state of action (typically updating Active state) -- called automatically for menus prior to showing"`

	// the type of action
	Type ActionTypes `desc:"the type of action"`
}

func (ac *Action) CopyFieldsFrom(frm any) {
	fr := frm.(*Action)
	ac.Button.CopyFieldsFrom(&fr.Button)
	ac.Data = fr.Data
	ac.Type = fr.Type
}

// ActionTypes is an enum representing
// the different possible types of actions
type ActionTypes int //enums:enum

const (
	// ActionStandalone is a default, standalone
	// action that is not part of a menu,
	// menubar, toolbar, or other element
	ActionStandalone ActionTypes = iota
	// ActionParts is an action that is part of
	// another element (like a clear button in a textfield)
	ActionParts
	// ActionMenu is an action contained
	// within a popup menu
	ActionMenu
	// ActionMenuBar is an action contained
	// within a menu bar
	ActionMenuBar
	// ActionToolBar is an action contained
	// within a toolbar
	ActionToolBar
)

func (ac *Action) OnInit() {
	ac.ButtonHandlers()
	ac.ActionStyles()
}

func (ac *Action) ActionStyles() {
	ac.AddStyles(func(s *styles.Style) {
		s.SetAbilities(true, states.Activatable, states.Focusable, states.Hoverable)
		s.SetAbilities(ac.ShortcutTooltip() != "", states.LongHoverable)
		s.Cursor = cursors.Pointer
		s.Border.Style.Set(styles.BorderNone)
		s.Text.Align = styles.AlignCenter
		s.BackgroundColor.SetSolid(colors.Scheme.SurfaceContainerLow)
		s.Color = colors.Scheme.OnSurface
		switch ac.Type {
		case ActionStandalone:
			s.Border.Radius = styles.BorderRadiusFull
			s.Margin.Set(units.Dp(2 * Prefs.DensityMul()))
			s.Padding.Set(units.Dp(6*Prefs.DensityMul()), units.Dp(12*Prefs.DensityMul()))
			s.BackgroundColor.SetSolid(colors.Scheme.Secondary.Container)
			s.Color = colors.Scheme.Secondary.OnContainer
		case ActionParts:
			s.Border.Radius.Set()
			s.BackgroundColor.SetSolid(colors.Transparent)
			// s.Margin.Set(units.Dp(2 * Prefs.DensityMul()))
			// s.Padding.Set(units.Dp(2 * Prefs.DensityMul()))
		case ActionMenu:
			s.Margin.Set()
			s.Padding.Set(units.Dp(6*Prefs.DensityMul()), units.Dp(12*Prefs.DensityMul()))
			s.MaxWidth.SetDp(-1)
			s.BackgroundColor.SetSolid(colors.Scheme.SurfaceContainer)
		case ActionMenuBar:
			s.Padding.Set(units.Em(0.25*Prefs.DensityMul()), units.Em(0.5*Prefs.DensityMul()))
			s.Margin.Set()
			ac.Indicator = icons.None
		case ActionToolBar:
			s.Padding.Set(units.Em(0.25*Prefs.DensityMul()), units.Em(0.5*Prefs.DensityMul()))
			s.Margin.Set()
			ac.Indicator = icons.None
			s.BackgroundColor.SetSolid(colors.Transparent)
			s.Border.Radius = styles.BorderRadiusSmall
		}
		if s.Is(states.Selected) {
			s.BackgroundColor.SetSolid(colors.Scheme.Select.Container)
		}
		if s.Is(states.Disabled) {
			s.Color = colors.Scheme.SurfaceContainer
		}
	})
}

func (ac *Action) OnChildAdded(child ki.Ki) {
	if _, w := AsWidget(child); w != nil {
		switch w.Name() {
		case "icon":
			w.AddStyles(func(s *styles.Style) {
				if ac.Type == ActionMenu {
					s.Font.Size.SetDp(16)
				}
				s.Margin.Set()
				s.Padding.Set()
			})
		case "space":
			w.AddStyles(func(s *styles.Style) {
				s.Width.SetCh(0.5)
				s.MinWidth.SetCh(0.5)
			})
		case "label":
			w.AddStyles(func(s *styles.Style) {
				s.SetAbilities(false, states.Selectable, states.DoubleClickable)
				s.Cursor = cursors.None
				s.Margin.Set()
				s.Padding.Set()
			})
		case "indicator":
			w.AddStyles(func(s *styles.Style) {
				if ac.Type == ActionMenu {
					s.Font.Size.SetDp(16)
				}
				s.Margin.Set()
				s.Padding.Set()
				s.AlignV = styles.AlignMiddle
			})
		case "ind-stretch":
			w.AddStyles(func(s *styles.Style) {
				s.Width.SetEm(1)
			})
		case "shortcut":
			w.AddStyles(func(s *styles.Style) {
				s.Margin.Set()
				s.Padding.Set()
			})
		case "sc-stretch":
			w.AddStyles(func(s *styles.Style) {
				s.MinWidth.SetCh(2)
			})
		}
	}

}

// SetType sets the styling type of the action
func (ac *Action) SetType(typ ActionTypes) *Action {
	updt := ac.UpdateStart()
	ac.Type = typ
	ac.UpdateEndLayout(updt)
	return ac
}

// Config calls functions to initialize widget and parts
func (ac *Action) ConfigWidget(sc *Scene) {
	ac.ConfigParts(sc)
}


// ConfigPartsButton sets the label, icon etc for the button
func (ac *Action) ConfigPartsButton(sc *Scene) {
	parts := ac.NewParts(LayoutHoriz)
	config := ki.Config{}
	icIdx, lbIdx := ac.ConfigPartsIconLabel(&config, ac.Icon, ac.Text)
	indIdx := ac.ConfigPartsAddIndicator(&config, false) // default off
	mods, updt := parts.ConfigChildren(config)
	ac.ConfigPartsSetIconLabel(ac.Icon, ac.Text, icIdx, lbIdx)
	ac.ConfigPartsIndicator(indIdx)
	if mods {
		parts.UpdateEnd(updt)
		ac.SetNeedsLayout(sc, updt)
	}
}

// ConfigPartsMenuItem sets the label, icon, etc for action menu item
func (ac *Action) ConfigPartsMenuItem(sc *Scene) {
	parts := ac.NewParts(LayoutHoriz)
	config := ki.Config{}
	icIdx, lbIdx := ac.ConfigPartsIconLabel(&config, ac.Icon, ac.Text)
	indIdx := ac.ConfigPartsAddIndicator(&config, false) // default off
	scIdx := -1
	if indIdx < 0 && ac.Shortcut != "" {
		scIdx = ac.ConfigPartsAddShortcut(&config)
	} else if ac.Shortcut != "" {
		log.Printf("gi.Action shortcut cannot be used on a sub-menu for action: %v\n", ac.Text)
	}
	mods, updt := parts.ConfigChildren(config)
	ac.ConfigPartsSetIconLabel(ac.Icon, ac.Text, icIdx, lbIdx)
	ac.ConfigPartsIndicator(indIdx)
	ac.ConfigPartsShortcut(scIdx)
	if mods {
		parts.UpdateEnd(updt)
		ac.SetNeedsLayout(sc, updt)
	}
}

// ConfigParts switches on part type on calls specific config
func (ac *Action) ConfigParts(sc *Scene) {
	ismbar := false
	istbar := false
	if ac.Par != nil {
		_, ismbar = ac.Par.(*MenuBar)
		_, istbar = ac.Par.(*ToolBar)
	}
	switch {
	case ismbar:
		ac.Indicator = icons.None // menu-bar specifically
		ac.Type = ActionMenuBar
		if ac.Class == "" {
			ac.Class = "menubar-action"
		}
		ac.ConfigPartsButton(sc)
	case istbar:
		ac.Type = ActionToolBar
		if ac.Class == "" {
			ac.Class = "toolbar-action"
		}
		ac.ConfigPartsButton(sc)
	case ac.Is(ButtonFlagMenu):
		ac.Type = ActionMenu
		if ac.Class == "" {
			ac.Class = "menu-action"
		}
		if ac.Indicator == "" && ac.HasMenu() {
			ac.Indicator = icons.KeyboardArrowRight
		}
		ac.ConfigPartsMenuItem(sc)
	default:
		ac.ConfigPartsButton(sc)
	}
}

// UpdateActions calls UpdateFunc on me and any of my menu items
func (ac *Action) UpdateActions() {
	if ac.UpdateFunc != nil {
		ac.UpdateFunc(ac)
	}
	if ac.Menu != nil {
		ac.Menu.UpdateActions()
	}
}

func (ac *Action) ClickDismissMenu() {
	ac.On(events.Click, func(e events.Event) {
		if ac.StateIs(states.Disabled) {
			return
		}
		if ac.Sc != nil && ac.Sc.Stage != nil {
			pst := ac.Sc.Stage.AsPopup()
			if pst != nil && pst.Type == Menu {
				pst.Close()
			}
		} else {
			if ac.Sc == nil {
				fmt.Println("ac.Sc == nil")
			} else if ac.Sc.Stage == nil {
				fmt.Println("ac.Sc.Stage == nil")
			}
		}
	})
}
*/
