// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"log"

	"goki.dev/colors"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events/key"
	"goki.dev/icons"
	"goki.dev/ki/v2"
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
	UpdateFunc  func(act *Action)
}

// Action is a button widget that can display a text label and / or an icon
// and / or a keyboard shortcut -- this is what is put in menus, menubars, and
// toolbars, and also for any standalone simple action.  The default styling
// differs depending on whether it is in a Menu versus a MenuBar or ToolBar --
// this is controlled by the Class which is automatically set to
// menu, menubar, or toolbar
//
//goki:embedder
type Action struct {
	ButtonBase

	// [view: -] optional data that is sent with the ActionSig when it is emitted
	Data any `json:"-" xml:"-" view:"-" desc:"optional data that is sent with the ActionSig when it is emitted"`

	// [view: -] signal for action -- does not have a signal type, as there is only one type: Action triggered -- data is Data of this action
	// ActionSig ki.Signal `json:"-" xml:"-" view:"-" desc:"signal for action -- does not have a signal type, as there is only one type: Action triggered -- data is Data of this action"`

	// [view: -] optional function that is called to update state of action (typically updating Active state) -- called automatically for menus prior to showing
	UpdateFunc func(act *Action) `json:"-" xml:"-" view:"-" desc:"optional function that is called to update state of action (typically updating Active state) -- called automatically for menus prior to showing"`

	// the type of action
	Type ActionTypes `desc:"the type of action"`
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
	ac.ButtonBaseHandlers()
	ac.ActionStyles()
}

func (ac *Action) ActionStyles() {
	ac.AddStyles(func(s *styles.Style) {
		// s.Cursor = cursor.HandPointing
		s.Border.Style.Set(styles.BorderNone)
		s.Text.Align = styles.AlignCenter
		s.BackgroundColor.SetSolid(colors.Scheme.SurfaceContainerLow)
		s.Color = colors.Scheme.OnSurface
		switch ac.Type {
		case ActionStandalone:
			s.Border.Radius = styles.BorderRadiusFull
			s.Margin.Set(units.Px(2 * Prefs.DensityMul()))
			s.Padding.Set(units.Px(6*Prefs.DensityMul()), units.Px(12*Prefs.DensityMul()))
			s.BackgroundColor.SetSolid(colors.Scheme.Secondary.Container)
			s.Color = colors.Scheme.Secondary.OnContainer
		case ActionParts:
			s.Border.Radius.Set()
			s.BackgroundColor.SetSolid(colors.Transparent)
			// s.Margin.Set(units.Px(2 * Prefs.DensityMul()))
			// s.Padding.Set(units.Px(2 * Prefs.DensityMul()))
		case ActionMenu:
			s.Margin.Set()
			s.Padding.Set(units.Px(6*Prefs.DensityMul()), units.Px(12*Prefs.DensityMul()))
			s.MaxWidth.SetPx(-1)
			s.BackgroundColor.SetSolid(colors.Scheme.SurfaceContainer)
		case ActionMenuBar:
			s.Padding.Set(units.Em(0.25*Prefs.DensityMul()), units.Em(0.5*Prefs.DensityMul()))
			s.Margin.Set()
			ac.Indicator = icons.None
		case ActionToolBar:
			s.Padding.Set(units.Em(0.25*Prefs.DensityMul()), units.Em(0.5*Prefs.DensityMul()))
			s.Margin.Set()
			ac.Indicator = icons.None
		}
		if s.Is(states.Hovered) {
			s.BackgroundColor.SetSolid(colors.Scheme.SurfaceContainerHighest)
		}
		if s.Is(states.Focused) {
			s.Border.Style.Set(styles.BorderSolid)
			s.Border.Width.Set(units.Px(2))
			s.Border.Color.Set(colors.Scheme.Outline)
		}
		// switch ac.State {
		// case ButtonActive:
		// 	s.BackgroundColor.SetSolid(s.BackgroundColor.Solid.Highlight(7))
		// case ButtonInactive:
		// 	s.BackgroundColor.SetSolid(s.BackgroundColor.Solid.Highlight(20))
		// 	s.Color = colors.Scheme.OnBackground.Highlight(20)
		// case ButtonFocus, ButtonSelected:
		// 	s.BackgroundColor.SetSolid(s.BackgroundColor.Solid.Highlight(15))
		// case ButtonHover:
		// 	s.BackgroundColor.SetSolid(s.BackgroundColor.Solid.Highlight(20))
		// case ButtonDown:
		// 	s.BackgroundColor.SetSolid(s.BackgroundColor.Solid.Highlight(25))
		// }
	})
}

func (ac *Action) OnChildAdded(child ki.Ki) {
	if _, w := AsWidget(child); w != nil {
		switch w.Name() {
		case "icon":
			w.AddStyles(func(s *styles.Style) {
				if ac.Type == ActionMenu {
					s.Font.Size.SetEm(1.5)
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
				s.Margin.Set()
				s.Padding.Set()
			})
		case "indicator":
			w.AddStyles(func(s *styles.Style) {
				if ac.Type == ActionMenu {
					s.Font.Size.SetEm(1.5)
				}
				s.Margin.Set()
				s.Padding.Set()
				s.AlignV = styles.AlignBottom
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

func (ac *Action) CopyFieldsFrom(frm any) {
	fr := frm.(*Action)
	ac.ButtonBase.CopyFieldsFrom(&fr.ButtonBase)
	ac.Data = fr.Data
}

// Config calls functions to initialize widget and parts
func (ac *Action) ConfigWidget(sc *Scene) {
	ac.ConfigParts(sc)
}

// ConfigPartsAddShortcut adds a menu shortcut, with a stretch space -- only called when needed
func (ac *Action) ConfigPartsAddShortcut(config *ki.Config) int {
	config.Add(StretchType, "sc-stretch")
	scIdx := len(*config)
	config.Add(LabelType, "shortcut")
	return scIdx
}

// ConfigPartsShortcut sets the shortcut
func (ac *Action) ConfigPartsShortcut(scIdx int) {
	if scIdx < 0 {
		return
	}
	sc := ac.Parts.Child(scIdx).(*Label)
	sclbl := ac.Shortcut.Shortcut()
	if sc.Text != sclbl {
		sc.Text = sclbl
	}
}

// ConfigPartsButton sets the label, icon etc for the button
func (ac *Action) ConfigPartsButton() {
	parts := ac.NewParts(LayoutHoriz)
	config := ki.Config{}
	icIdx, lbIdx := ac.ConfigPartsIconLabel(&config, ac.Icon, ac.Text)
	indIdx := ac.ConfigPartsAddIndicator(&config, false) // default off
	mods, updt := parts.ConfigChildren(config)
	ac.ConfigPartsSetIconLabel(ac.Icon, ac.Text, icIdx, lbIdx)
	ac.ConfigPartsIndicator(indIdx)
	if mods {
		ac.UpdateEnd(updt)
	}
}

// ConfigPartsMenuItem sets the label, icon, etc for action menu item
func (ac *Action) ConfigPartsMenuItem() {
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
		ac.UpdateEnd(updt)
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
		ac.ConfigPartsButton()
	case istbar:
		ac.Type = ActionToolBar
		if ac.Class == "" {
			ac.Class = "toolbar-action"
		}
		ac.ConfigPartsButton()
	case ac.Is(ButtonFlagMenu):
		ac.Type = ActionMenu
		if ac.Class == "" {
			ac.Class = "menu-action"
		}
		if ac.Indicator == "" && ac.HasMenu() {
			ac.Indicator = icons.KeyboardArrowRight
		}
		ac.ConfigPartsMenuItem()
	default:
		ac.ConfigPartsButton()
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
