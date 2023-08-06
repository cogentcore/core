// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"log"

	"github.com/goki/gi/gist"
	"github.com/goki/gi/icons"
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

var TypeAction = kit.Types.AddType(&Action{}, nil)

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

//go:generate stringer -type=ActionTypes

// AddNewAction adds a new action to given parent node, with given name.
func AddNewAction(parent ki.Ki, name string) *Action {
	return parent.AddNewChild(TypeAction, name).(*Action)
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

// // DefaultStyle implements the [DefaultStyler] interface
// func (ac *Action) DefaultStyle() {
// 	s := &ac.Style

// 	s.Border.Style.Set(gist.BorderNone)
// 	s.Border.Width.Set()
// 	s.Border.Radius.Set()
// 	s.Border.Color.Set()
// 	s.Text.Align = gist.AlignCenter
// 	s.BackgroundColor.SetColor(TheColorScheme.Secondary)
// 	s.Color.SetColor(TheColorScheme.Secondary.ContrastColor())

// 	s.Padding.Set(units.Px(2))
// 	s.Margin.Set(units.Px(2))
// 	s.MinWidth.SetEm(1)
// 	s.MinHeight.SetEm(1)
// }

var ActionProps = ki.Props{
	"EnumType:Flag": TypeButtonFlags,
	// "border-width":     units.Px(0), // todo: should be default
	// "border-radius":    units.Px(0),
	// "border-color":     &Prefs.Colors.Border,
	// "text-align":       gist.AlignCenter,
	// "background-color": &Prefs.Colors.Control,
	// "color":            &Prefs.Colors.Font,
	// "padding":          units.Px(2),
	// "margin":           units.Px(2),
	// "min-width":        units.Em(1),
	// "min-height":       units.Em(1),
	// "#icon": ki.Props{
	// 	"width":   units.Em(1),
	// 	"height":  units.Em(1),
	// 	"margin":  units.Px(0),
	// 	"padding": units.Px(0),
	// 	"fill":    &Prefs.Colors.Icon,
	// 	"stroke":  &Prefs.Colors.Font,
	// },
	// "#space": ki.Props{
	// 	"width":     units.Ch(.5),
	// 	"min-width": units.Ch(.5),
	// },
	// "#label": ki.Props{
	// 	"margin":  units.Px(0),
	// 	"padding": units.Px(0),
	// },
	// "#indicator": ki.Props{
	// 	"width":          units.Ex(1.5),
	// 	"height":         units.Ex(1.5),
	// 	"margin":         units.Px(0),
	// 	"padding":        units.Px(0),
	// 	"vertical-align": gist.AlignBottom,
	// 	"fill":           &Prefs.Colors.Icon,
	// 	"stroke":         &Prefs.Colors.Font,
	// },
	// "#ind-stretch": ki.Props{
	// 	"width": units.Em(1),
	// },
	// "#shortcut": ki.Props{
	// 	"margin":  units.Px(0),
	// 	"padding": units.Px(0),
	// },
	// "#sc-stretch": ki.Props{
	// 	"min-width": units.Ch(2),
	// },
	// ".menu-action": ki.Props{ // class of actions as menu items
	// 	"padding":   units.Px(2),
	// 	"margin":    units.Px(0),
	// 	"max-width": -1,
	// 	"indicator": icons.KeyboardArrowRight,
	// 	ButtonSelectors[ButtonActive]: ki.Props{
	// 		"background-color": "lighter-0",
	// 	},
	// 	ButtonSelectors[ButtonInactive]: ki.Props{
	// 		"border-color": "highlight-50",
	// 		"color":        "highlight-50",
	// 	},
	// 	ButtonSelectors[ButtonHover]: ki.Props{
	// 		"background-color": "highlight-10",
	// 	},
	// 	ButtonSelectors[ButtonFocus]: ki.Props{
	// 		"border-width":     units.Px(2),
	// 		"background-color": "samelight-50",
	// 	},
	// 	ButtonSelectors[ButtonDown]: ki.Props{
	// 		"color":            "highlight-90",
	// 		"background-color": "highlight-30",
	// 	},
	// 	ButtonSelectors[ButtonSelected]: ki.Props{
	// 		"background-color": &Prefs.Colors.Select,
	// 	},
	// },
	// ".menubar-action": ki.Props{ // class of actions in MenuBar
	// 	"padding":      units.Px(4), // we go to edge of bar
	// 	"margin":       units.Px(0),
	// 	"indicator":    icons.None,
	// 	"border-width": units.Px(0),
	// 	ButtonSelectors[ButtonActive]: ki.Props{
	// 		"background-color": "linear-gradient(lighter-0, highlight-10)",
	// 	},
	// 	ButtonSelectors[ButtonInactive]: ki.Props{
	// 		"border-color": "lighter-50",
	// 		"color":        "lighter-50",
	// 	},
	// 	ButtonSelectors[ButtonHover]: ki.Props{
	// 		"background-color": "linear-gradient(highlight-10, highlight-10)",
	// 	},
	// 	ButtonSelectors[ButtonFocus]: ki.Props{
	// 		"border-width":     units.Px(2),
	// 		"background-color": "linear-gradient(samelight-50, highlight-10)",
	// 	},
	// 	ButtonSelectors[ButtonDown]: ki.Props{
	// 		"color":            "lighter-90",
	// 		"background-color": "linear-gradient(highlight-30, highlight-10)",
	// 	},
	// 	ButtonSelectors[ButtonSelected]: ki.Props{
	// 		"background-color": "linear-gradient(pref(Select), highlight-10)",
	// 	},
	// },
	// ".toolbar-action": ki.Props{ // class of actions in ToolBar
	// 	"padding":      units.Px(4), // we go to edge of bar
	// 	"margin":       units.Px(0),
	// 	"indicator":    icons.None,
	// 	"border-width": units.Px(0.5),
	// 	ButtonSelectors[ButtonActive]: ki.Props{
	// 		"background-color": "linear-gradient(lighter-0, highlight-10)",
	// 	},
	// 	ButtonSelectors[ButtonInactive]: ki.Props{
	// 		"border-color": "lighter-50",
	// 		"color":        "lighter-50",
	// 	},
	// 	ButtonSelectors[ButtonHover]: ki.Props{
	// 		"background-color": "linear-gradient(highlight-10, highlight-10)",
	// 	},
	// 	ButtonSelectors[ButtonFocus]: ki.Props{
	// 		"border-width":     units.Px(2),
	// 		"background-color": "linear-gradient(samelight-50, highlight-10)",
	// 	},
	// 	ButtonSelectors[ButtonDown]: ki.Props{
	// 		"color":            "lighter-90",
	// 		"background-color": "linear-gradient(highlight-30, highlight-10)",
	// 	},
	// 	ButtonSelectors[ButtonSelected]: ki.Props{
	// 		"background-color": "linear-gradient(pref(Select), highlight-10)",
	// 	},
	// },
	// ".": ki.Props{ // default class -- stand-alone buttons presumably
	// 	ButtonSelectors[ButtonActive]: ki.Props{
	// 		"background-color": "linear-gradient(lighter-0, highlight-10)",
	// 	},
	// 	ButtonSelectors[ButtonInactive]: ki.Props{
	// 		"border-color": "lighter-50",
	// 		"color":        "lighter-50",
	// 	},
	// 	ButtonSelectors[ButtonHover]: ki.Props{
	// 		"background-color": "linear-gradient(highlight-10, highlight-10)",
	// 	},
	// 	ButtonSelectors[ButtonFocus]: ki.Props{
	// 		"border-width":     units.Px(2),
	// 		"background-color": "linear-gradient(samelight-50, highlight-10)",
	// 	},
	// 	ButtonSelectors[ButtonDown]: ki.Props{
	// 		"color":            "lighter-90",
	// 		"background-color": "linear-gradient(highlight-30, highlight-10)",
	// 	},
	// 	ButtonSelectors[ButtonSelected]: ki.Props{
	// 		"background-color": "linear-gradient(pref(Select), highlight-10)",
	// 	},
	// },
}

// ButtonWidget interface

// Trigger triggers the action signal -- for external activation of action --
// only works if action is not inactive
func (ac *Action) Trigger() {
	if ac.IsInactive() {
		return
	}
	ac.ActionSig.Emit(ac.This(), 0, ac.Data)
}

// ButtonRelease triggers action signal
func (ac *Action) ButtonRelease() {
	if ac.IsInactive() {
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
	ac.ConfigStyles()
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
		if ac.Indicator == "" {
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

func (ac *Action) ConfigStyles() {
	ac.AddStyleFunc(StyleFuncDefault, func() {
		ac.Style.Border.Style.Set(gist.BorderNone)
		ac.Style.Text.Align = gist.AlignCenter
		ac.Style.BackgroundColor.SetColor(ColorScheme.Surface)
		ac.Style.Color = ColorScheme.OnSurface
		switch ac.Type {
		case ActionStandalone:
			ac.Style.Border.Radius.Set(gist.BorderRadiusFull)
			ac.Style.Margin.Set(units.Px(2 * Prefs.DensityMul()))
			ac.Style.Padding.Set(units.Px(6*Prefs.DensityMul()), units.Px(12*Prefs.DensityMul()))
			ac.Style.BackgroundColor.SetColor(ColorScheme.SecondaryContainer)
			ac.Style.Color = ColorScheme.OnSecondaryContainer
		case ActionParts:
			ac.Style.Border.Radius.Set()
			ac.Style.Margin.Set(units.Px(2 * Prefs.DensityMul()))
			ac.Style.Padding.Set(units.Px(2 * Prefs.DensityMul()))
			ac.Style.BackgroundColor.SetColor(ColorScheme.Background)
		case ActionMenu:
			ac.Style.Margin.Set()
			ac.Style.Padding.Set(units.Px(2 * Prefs.DensityMul()))
			ac.Style.MaxWidth.SetPx(-1)
			ac.Indicator = icons.KeyboardArrowRight
		case ActionMenuBar:
			ac.Style.Padding.Set(units.Em(0.25*Prefs.DensityMul()), units.Em(0.5*Prefs.DensityMul()))
			ac.Style.Margin.Set()
			ac.Indicator = icons.None
		case ActionToolBar:
			ac.Style.Padding.Set(units.Em(0.25*Prefs.DensityMul()), units.Em(0.5*Prefs.DensityMul()))
			ac.Style.Margin.Set()
			ac.Indicator = icons.None
		}
		// switch ac.State {
		// case ButtonActive:
		// 	ac.Style.BackgroundColor.SetColor(ac.Style.BackgroundColor.Color.Highlight(7))
		// case ButtonInactive:
		// 	ac.Style.BackgroundColor.SetColor(ac.Style.BackgroundColor.Color.Highlight(20))
		// 	ac.Style.Color = ColorScheme.OnBackground.Highlight(20)
		// case ButtonFocus, ButtonSelected:
		// 	ac.Style.BackgroundColor.SetColor(ac.Style.BackgroundColor.Color.Highlight(15))
		// case ButtonHover:
		// 	ac.Style.BackgroundColor.SetColor(ac.Style.BackgroundColor.Color.Highlight(20))
		// case ButtonDown:
		// 	ac.Style.BackgroundColor.SetColor(ac.Style.BackgroundColor.Color.Highlight(25))
		// }
	})
	ac.Parts.AddChildStyleFunc("icon", ki.StartMiddle, StyleFuncParts(ac), func(icon *WidgetBase) {
		icon.Style.Width.SetEm(1)
		icon.Style.Height.SetEm(1)
		icon.Style.Margin.Set()
		icon.Style.Padding.Set()
	})
	ac.Parts.AddChildStyleFunc("space", ki.StartMiddle, StyleFuncParts(ac), func(space *WidgetBase) {
		space.Style.Width.SetCh(0.5)
		space.Style.MinWidth.SetCh(0.5)
	})
	ac.Parts.AddChildStyleFunc("label", ki.StartMiddle, StyleFuncParts(ac), func(label *WidgetBase) {
		label.Style.Margin.Set()
		label.Style.Padding.Set()
	})
	ac.Parts.AddChildStyleFunc("indicator", ki.StartMiddle, StyleFuncParts(ac), func(ind *WidgetBase) {
		ind.Style.Width.SetEx(1.5)
		ind.Style.Height.SetEx(1.5)
		ind.Style.Margin.Set()
		ind.Style.Padding.Set()
		ind.Style.AlignV = gist.AlignBottom
	})
	ac.Parts.AddChildStyleFunc("ind-stretch", ki.StartMiddle, StyleFuncParts(ac), func(ins *WidgetBase) {
		ins.Style.Width.SetEm(1)
	})
	ac.Parts.AddChildStyleFunc("shortcut", ki.StartMiddle, StyleFuncParts(ac), func(short *WidgetBase) {
		short.Style.Margin.Set()
		short.Style.Padding.Set()
	})
	ac.Parts.AddChildStyleFunc("sc-stretch", ki.StartMiddle, StyleFuncParts(ac), func(scs *WidgetBase) {
		scs.Style.MinWidth.SetCh(2)
	})
}
