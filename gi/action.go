// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"log"

	"github.com/goki/gi/gist"
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
	Data       interface{}       `json:"-" xml:"-" view:"-" desc:"optional data that is sent with the ActionSig when it is emitted"`
	ActionSig  ki.Signal         `json:"-" xml:"-" view:"-" desc:"signal for action -- does not have a signal type, as there is only one type: Action triggered -- data is Data of this action"`
	UpdateFunc func(act *Action) `json:"-" xml:"-" view:"-" desc:"optional function that is called to update state of action (typically updating Active state) -- called automatically for menus prior to showing"`
}

var KiT_Action = kit.Types.AddType(&Action{}, ActionProps)

// AddNewAction adds a new action to given parent node, with given name.
func AddNewAction(parent ki.Ki, name string) *Action {
	return parent.AddNewChild(KiT_Action, name).(*Action)
}

func (ac *Action) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*Action)
	ac.ButtonBase.CopyFieldsFrom(&fr.ButtonBase)
	ac.Data = fr.Data
}

func (ac *Action) Disconnect() {
	ac.ButtonBase.Disconnect()
	ac.ActionSig.DisconnectAll()
}

var ActionProps = ki.Props{
	"EnumType:Flag":    KiT_ButtonFlags,
	"border-width":     units.NewPx(0), // todo: should be default
	"border-radius":    units.NewPx(0),
	"border-color":     &Prefs.Colors.Border,
	"text-align":       gist.AlignCenter,
	"background-color": &Prefs.Colors.Control,
	"color":            &Prefs.Colors.Font,
	"padding":          units.NewPx(2),
	"margin":           units.NewPx(2),
	"min-width":        units.NewEm(1),
	"min-height":       units.NewEm(1),
	"#icon": ki.Props{
		"width":   units.NewEm(1),
		"height":  units.NewEm(1),
		"margin":  units.NewPx(0),
		"padding": units.NewPx(0),
		"fill":    &Prefs.Colors.Icon,
		"stroke":  &Prefs.Colors.Font,
	},
	"#space": ki.Props{
		"width":     units.NewCh(.5),
		"min-width": units.NewCh(.5),
	},
	"#label": ki.Props{
		"margin":  units.NewPx(0),
		"padding": units.NewPx(0),
	},
	"#indicator": ki.Props{
		"width":          units.NewEx(1.5),
		"height":         units.NewEx(1.5),
		"margin":         units.NewPx(0),
		"padding":        units.NewPx(0),
		"vertical-align": gist.AlignBottom,
		"fill":           &Prefs.Colors.Icon,
		"stroke":         &Prefs.Colors.Font,
	},
	"#ind-stretch": ki.Props{
		"width": units.NewEm(1),
	},
	"#shortcut": ki.Props{
		"margin":  units.NewPx(0),
		"padding": units.NewPx(0),
	},
	"#sc-stretch": ki.Props{
		"min-width": units.NewCh(2),
	},
	".menu-action": ki.Props{ // class of actions as menu items
		"padding":   units.NewPx(2),
		"margin":    units.NewPx(0),
		"max-width": -1,
		"indicator": "wedge-right",
		ButtonSelectors[ButtonActive]: ki.Props{
			"background-color": "lighter-0",
		},
		ButtonSelectors[ButtonInactive]: ki.Props{
			"border-color": "highlight-50",
			"color":        "highlight-50",
		},
		ButtonSelectors[ButtonHover]: ki.Props{
			"background-color": "highlight-10",
		},
		ButtonSelectors[ButtonFocus]: ki.Props{
			"border-width":     units.NewPx(2),
			"background-color": "samelight-50",
		},
		ButtonSelectors[ButtonDown]: ki.Props{
			"color":            "highlight-90",
			"background-color": "highlight-30",
		},
		ButtonSelectors[ButtonSelected]: ki.Props{
			"background-color": &Prefs.Colors.Select,
		},
	},
	".menubar-action": ki.Props{ // class of actions in MenuBar
		"padding":      units.NewPx(4), // we go to edge of bar
		"margin":       units.NewPx(0),
		"indicator":    "none",
		"border-width": units.NewPx(0),
		ButtonSelectors[ButtonActive]: ki.Props{
			"background-color": "linear-gradient(lighter-0, highlight-10)",
		},
		ButtonSelectors[ButtonInactive]: ki.Props{
			"border-color": "lighter-50",
			"color":        "lighter-50",
		},
		ButtonSelectors[ButtonHover]: ki.Props{
			"background-color": "linear-gradient(highlight-10, highlight-10)",
		},
		ButtonSelectors[ButtonFocus]: ki.Props{
			"border-width":     units.NewPx(2),
			"background-color": "linear-gradient(samelight-50, highlight-10)",
		},
		ButtonSelectors[ButtonDown]: ki.Props{
			"color":            "lighter-90",
			"background-color": "linear-gradient(highlight-30, highlight-10)",
		},
		ButtonSelectors[ButtonSelected]: ki.Props{
			"background-color": "linear-gradient(pref(Select), highlight-10)",
		},
	},
	".toolbar-action": ki.Props{ // class of actions in ToolBar
		"padding":      units.NewPx(4), // we go to edge of bar
		"margin":       units.NewPx(0),
		"indicator":    "none",
		"border-width": units.NewPx(0.5),
		ButtonSelectors[ButtonActive]: ki.Props{
			"background-color": "linear-gradient(lighter-0, highlight-10)",
		},
		ButtonSelectors[ButtonInactive]: ki.Props{
			"border-color": "lighter-50",
			"color":        "lighter-50",
		},
		ButtonSelectors[ButtonHover]: ki.Props{
			"background-color": "linear-gradient(highlight-10, highlight-10)",
		},
		ButtonSelectors[ButtonFocus]: ki.Props{
			"border-width":     units.NewPx(2),
			"background-color": "linear-gradient(samelight-50, highlight-10)",
		},
		ButtonSelectors[ButtonDown]: ki.Props{
			"color":            "lighter-90",
			"background-color": "linear-gradient(highlight-30, highlight-10)",
		},
		ButtonSelectors[ButtonSelected]: ki.Props{
			"background-color": "linear-gradient(pref(Select), highlight-10)",
		},
	},
	".": ki.Props{ // default class -- stand-alone buttons presumably
		ButtonSelectors[ButtonActive]: ki.Props{
			"background-color": "linear-gradient(lighter-0, highlight-10)",
		},
		ButtonSelectors[ButtonInactive]: ki.Props{
			"border-color": "lighter-50",
			"color":        "lighter-50",
		},
		ButtonSelectors[ButtonHover]: ki.Props{
			"background-color": "linear-gradient(highlight-10, highlight-10)",
		},
		ButtonSelectors[ButtonFocus]: ki.Props{
			"border-width":     units.NewPx(2),
			"background-color": "linear-gradient(samelight-50, highlight-10)",
		},
		ButtonSelectors[ButtonDown]: ki.Props{
			"color":            "lighter-90",
			"background-color": "linear-gradient(highlight-30, highlight-10)",
		},
		ButtonSelectors[ButtonSelected]: ki.Props{
			"background-color": "linear-gradient(pref(Select), highlight-10)",
		},
	},
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
	ac.SetButtonState(ButtonActive)
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
	config.Add(KiT_Stretch, "sc-stretch")
	scIdx := len(*config)
	config.Add(KiT_Label, "shortcut")
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
	icIdx, lbIdx := ac.ConfigPartsIconLabel(&config, string(ac.Icon), ac.Text)
	indIdx := ac.ConfigPartsAddIndicator(&config, false) // default off
	mods, updt := ac.Parts.ConfigChildren(config)
	ac.ConfigPartsSetIconLabel(string(ac.Icon), ac.Text, icIdx, lbIdx)
	ac.ConfigPartsIndicator(indIdx)
	if mods {
		ac.UpdateEnd(updt)
	}
}

// ConfigPartsMenuItem sets the label, icon, etc for action menu item
func (ac *Action) ConfigPartsMenuItem() {
	config := kit.TypeAndNameList{}
	icIdx, lbIdx := ac.ConfigPartsIconLabel(&config, string(ac.Icon), ac.Text)
	indIdx := ac.ConfigPartsAddIndicator(&config, false) // default off
	scIdx := -1
	if indIdx < 0 && ac.Shortcut != "" {
		scIdx = ac.ConfigPartsAddShortcut(&config)
	} else if ac.Shortcut != "" {
		log.Printf("gi.Action shortcut cannot be used on a sub-menu for action: %v\n", ac.Text)
	}
	mods, updt := ac.Parts.ConfigChildren(config)
	ac.ConfigPartsSetIconLabel(string(ac.Icon), ac.Text, icIdx, lbIdx)
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
		ac.Indicator = "none" // menu-bar specifically
		if ac.Class == "" {
			ac.Class = "menubar-action"
		}
		ac.ConfigPartsButton()
	case istbar:
		if ac.Class == "" {
			ac.Class = "toolbar-action"
		}
		ac.ConfigPartsButton()
	case ac.IsMenu():
		if ac.Class == "" {
			ac.Class = "menu-action"
		}
		if ac.Indicator == "" {
			ac.Indicator = "wedge-right"
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
