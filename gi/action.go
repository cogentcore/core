// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"log"

	"github.com/goki/gi/gist"
	"github.com/goki/gi/icons"
	"github.com/goki/gi/oswin/cursor"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// Action is a button widget that can display a text label and / or an icon
// and / or a keyboard shortcut -- this is what is put in menus, menubars, and
// toolbars, and also for any standalone simple action.  The default styling
// differs depending on whether it is in a Menu versus a MenuBar or ToolBar --
// this is controlled by the Class which is automatically set to
// menu, menubar, or toolbar
type Action struct {
	ButtonBase

	// [view: -] optional data that is sent with the ActionSig when it is emitted
	Data any `json:"-" xml:"-" view:"-" desc:"optional data that is sent with the ActionSig when it is emitted"`

	// [view: -] signal for action -- does not have a signal type, as there is only one type: Action triggered -- data is Data of this action
	ActionSig ki.Signal `json:"-" xml:"-" view:"-" desc:"signal for action -- does not have a signal type, as there is only one type: Action triggered -- data is Data of this action"`

	// [view: -] optional function that is called to update state of action (typically updating Active state) -- called automatically for menus prior to showing
	UpdateFunc func(act *Action) `json:"-" xml:"-" view:"-" desc:"optional function that is called to update state of action (typically updating Active state) -- called automatically for menus prior to showing"`

	// the type of action
	Type ActionTypes `desc:"the type of action"`
}

var TypeAction = kit.Types.AddType(&Action{}, ActionProps)

// ActionTypes is an enum representing
// the different possible types of actions
type ActionTypes int

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

	ActionTypesN
)

var TypeActionTypes = kit.Enums.AddEnumAltLower(ActionTypesN, kit.NotBitFlag, gist.StylePropProps, "Action")

// AddNewAction adds a new action to given parent node, with given name.
func AddNewAction(parent ki.Ki, name string) *Action {
	return parent.AddNewChild(TypeAction, name).(*Action)
}

func (ac *Action) OnInit() {
	ac.AddStyleFunc(StyleFuncDefault, func() {
		s := &ac.Style
		s.Cursor = cursor.HandPointing
		s.Border.Style.Set(gist.BorderNone)
		s.Text.Align = gist.AlignCenter
		s.BackgroundColor.SetColor(ColorScheme.SurfaceContainerLow)
		s.Color = ColorScheme.OnSurface
		switch ac.Type {
		case ActionStandalone:
			s.Border.Radius = gist.BorderRadiusFull
			s.Margin.Set(units.Px(2 * Prefs.DensityMul()))
			s.Padding.Set(units.Px(6*Prefs.DensityMul()), units.Px(12*Prefs.DensityMul()))
			s.BackgroundColor.SetColor(ColorScheme.SecondaryContainer)
			s.Color = ColorScheme.OnSecondaryContainer
		case ActionParts:
			s.Border.Radius.Set()
			// s.Margin.Set(units.Px(2 * Prefs.DensityMul()))
			// s.Padding.Set(units.Px(2 * Prefs.DensityMul()))
			s.BackgroundColor = ac.ParentBackgroundColor()
		case ActionMenu:
			s.Margin.Set()
			s.Padding.Set(units.Px(6*Prefs.DensityMul()), units.Px(12*Prefs.DensityMul()))
			s.MaxWidth.SetPx(-1)
			s.BackgroundColor.SetColor(ColorScheme.SurfaceContainer)
		case ActionMenuBar:
			s.Padding.Set(units.Em(0.25*Prefs.DensityMul()), units.Em(0.5*Prefs.DensityMul()))
			s.Margin.Set()
			ac.Indicator = icons.None
		case ActionToolBar:
			s.Padding.Set(units.Em(0.25*Prefs.DensityMul()), units.Em(0.5*Prefs.DensityMul()))
			s.Margin.Set()
			ac.Indicator = icons.None
		}
		// switch ac.State {
		// case ButtonActive:
		// 	s.BackgroundColor.SetColor(s.BackgroundColor.Color.Highlight(7))
		// case ButtonInactive:
		// 	s.BackgroundColor.SetColor(s.BackgroundColor.Color.Highlight(20))
		// 	s.Color = ColorScheme.OnBackground.Highlight(20)
		// case ButtonFocus, ButtonSelected:
		// 	s.BackgroundColor.SetColor(s.BackgroundColor.Color.Highlight(15))
		// case ButtonHover:
		// 	s.BackgroundColor.SetColor(s.BackgroundColor.Color.Highlight(20))
		// case ButtonDown:
		// 	s.BackgroundColor.SetColor(s.BackgroundColor.Color.Highlight(25))
		// }
	})
}

func (ac *Action) OnChildAdded(child ki.Ki) {
	switch child.Name() {
	case "icon":
		icon := child.(*Icon)
		icon.AddStyleFunc(StyleFuncParent(ac), func() {
			if ac.Type == ActionMenu {
				icon.Style.Font.Size.SetEm(1.5)
			}
			icon.Style.Margin.Set()
			icon.Style.Padding.Set()
		})
	case "space":
		space := child.(*Space)
		space.AddStyleFunc(StyleFuncParent(ac), func() {
			space.Style.Width.SetCh(0.5)
			space.Style.MinWidth.SetCh(0.5)
		})
	case "label":
		label := child.(*Label)
		label.AddStyleFunc(StyleFuncParent(ac), func() {
			label.Style.Margin.Set()
			label.Style.Padding.Set()
		})
	case "indicator":
		ind := child.(*Icon)
		ind.AddStyleFunc(StyleFuncParent(ac), func() {
			if ac.Type == ActionMenu {
				ind.Style.Font.Size.SetEm(1.5)
			}
			ind.Style.Margin.Set()
			ind.Style.Padding.Set()
			ind.Style.AlignV = gist.AlignBottom
		})
	case "ind-stretch":
		ins := child.(*Stretch)
		ins.AddStyleFunc(StyleFuncParent(ac), func() {
			ins.Style.Width.SetEm(1)
		})
	case "shortcut":
		short := child.(*Label)
		short.AddStyleFunc(StyleFuncParent(ac), func() {
			short.Style.Margin.Set()
			short.Style.Padding.Set()
		})
	case "sc-stretch":
		scs := child.(*Stretch)
		scs.AddStyleFunc(StyleFuncParent(ac), func() {
			scs.Style.MinWidth.SetCh(2)
		})
	}
}

func (ac *Action) CopyFieldsFrom(frm any) {
	fr := frm.(*Action)
	ac.ButtonBase.CopyFieldsFrom(&fr.ButtonBase)
	ac.Data = fr.Data
}

func (ac *Action) Disconnect() {
	ac.ButtonBase.Disconnect()
	ac.ActionSig.DisconnectAll()
}

var ActionProps = ki.Props{
	ki.EnumTypeFlag: TypeButtonFlags,
}

// ButtonWidget interface

// Trigger triggers the action signal -- for external activation of action --
// only works if action is not inactive
func (ac *Action) Trigger() {
	if ac.IsDisabled() {
		return
	}
	ac.ActionSig.Emit(ac.This(), 0, ac.Data)
}

// ButtonRelease triggers action signal
func (ac *Action) ButtonRelease() {
	if ac.IsDisabled() {
		// fmt.Printf("action: %v inactive\n", ac.Nm)
		return
	}
	wasPressed := (ac.State == ButtonDown)
	updt := ac.UpdateStart()
	ac.SetButtonState(ButtonHover)
	ac.ButtonSig.Emit(ac.This(), int64(ButtonReleased), nil)
	menOpen := false
	if wasPressed {
		ac.ActionSig.Emit(ac.This(), 0, ac.Data)
		ac.ButtonSig.Emit(ac.This(), int64(ButtonClicked), ac.Data)
		menOpen = ac.OpenMenu()
		// } else {
		// 	fmt.Printf("action: %v not was pressed\n", ac.Nm)
	}
	if !menOpen && ac.IsMenu() && ac.Viewport != nil {
		win := ac.ParentWindow()
		if win != nil {
			win.ClosePopup(ac.Viewport) // in case we are a menu popup -- no harm if not
		}
	}
	ac.UpdateEnd(updt)
}

// Init2D calls functions to initialize widget and parts
func (ac *Action) Init2D() {
	ac.Init2DWidget()
	ac.ConfigParts()
}

// ConfigPartsAddShortcut adds a menu shortcut, with a stretch space -- only called when needed
func (ac *Action) ConfigPartsAddShortcut(config *kit.TypeAndNameList) int {
	config.Add(TypeStretch, "sc-stretch")
	scIdx := len(*config)
	config.Add(TypeLabel, "shortcut")
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
		ac.StylePart(Node2D(sc))
		ac.StylePart(ac.Parts.Child(scIdx - 1).(Node2D)) // also get the stretch
	}
}

// ConfigPartsButton sets the label, icon etc for the button
func (ac *Action) ConfigPartsButton() {
	config := kit.TypeAndNameList{}
	icIdx, lbIdx := ac.ConfigPartsIconLabel(&config, ac.Icon, ac.Text)
	indIdx := ac.ConfigPartsAddIndicator(&config, false) // default off
	mods, updt := ac.Parts.ConfigChildren(config)
	ac.ConfigPartsSetIconLabel(ac.Icon, ac.Text, icIdx, lbIdx)
	ac.ConfigPartsIndicator(indIdx)
	if mods {
		ac.UpdateEnd(updt)
	}
}

// ConfigPartsMenuItem sets the label, icon, etc for action menu item
func (ac *Action) ConfigPartsMenuItem() {
	config := kit.TypeAndNameList{}
	icIdx, lbIdx := ac.ConfigPartsIconLabel(&config, ac.Icon, ac.Text)
	indIdx := ac.ConfigPartsAddIndicator(&config, false) // default off
	scIdx := -1
	if indIdx < 0 && ac.Shortcut != "" {
		scIdx = ac.ConfigPartsAddShortcut(&config)
	} else if ac.Shortcut != "" {
		log.Printf("gi.Action shortcut cannot be used on a sub-menu for action: %v\n", ac.Text)
	}
	mods, updt := ac.Parts.ConfigChildren(config)
	ac.ConfigPartsSetIconLabel(ac.Icon, ac.Text, icIdx, lbIdx)
	ac.ConfigPartsIndicator(indIdx)
	ac.ConfigPartsShortcut(scIdx)
	if mods {
		ac.UpdateEnd(updt)
	}
}

// ConfigParts switches on part type on calls specific config
func (ac *Action) ConfigParts() {
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
	case ac.IsMenu():
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
