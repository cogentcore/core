// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"log"

	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

// Action is a button widget that can display a text label and / or an icon
// and / or a keyboard shortcut -- this is what is put in menus, menubars, and
// toolbars, and also for any standalone simple action.  The default styling
// differs depending on whether it is in a Menu versus a MenuBar or ToolBar --
// this is controlled by the Class which is automatically set to menu-action
// or bar-action.
type Action struct {
	ButtonBase
	Data       interface{}       `json:"-" xml:"-" view:"-" desc:"optional data that is sent with the ActionSig when it is emitted"`
	ActionSig  ki.Signal         `json:"-" xml:"-" view:"-" desc:"signal for action -- does not have a signal type, as there is only one type: Action triggered -- data is Data of this action"`
	UpdateFunc func(act *Action) `json:"-" xml:"-" view:"-" desc:"optional function that is called to update state of action (typically updating Active state) -- called automatically for menus prior to showing"`
}

var KiT_Action = kit.Types.AddType(&Action{}, ActionProps)

var ActionProps = ki.Props{
	"border-width":     units.NewValue(0, units.Px), // todo: should be default
	"border-radius":    units.NewValue(0, units.Px),
	"border-color":     &Prefs.Colors.Border,
	"border-style":     BorderSolid,
	"box-shadow.color": &Prefs.Colors.Shadow,
	"text-align":       AlignCenter,
	"background-color": &Prefs.Colors.Control,
	"color":            &Prefs.Colors.Font,
	"padding":          units.NewValue(2, units.Px),
	"margin":           units.NewValue(2, units.Px),
	"min-width":        units.NewValue(1, units.Ch),
	"min-height":       units.NewValue(1, units.Em),
	"#icon": ki.Props{
		"width":   units.NewValue(1, units.Em),
		"height":  units.NewValue(1, units.Em),
		"margin":  units.NewValue(0, units.Px),
		"padding": units.NewValue(0, units.Px),
		"fill":    &Prefs.Colors.Icon,
		"stroke":  &Prefs.Colors.Font,
	},
	"#label": ki.Props{
		"margin":  units.NewValue(0, units.Px),
		"padding": units.NewValue(0, units.Px),
	},
	"#indicator": ki.Props{
		"width":          units.NewValue(1.5, units.Ex),
		"height":         units.NewValue(1.5, units.Ex),
		"margin":         units.NewValue(0, units.Px),
		"padding":        units.NewValue(0, units.Px),
		"vertical-align": AlignBottom,
		"fill":           &Prefs.Colors.Icon,
		"stroke":         &Prefs.Colors.Font,
	},
	"#ind-stretch": ki.Props{
		"width": units.NewValue(1, units.Em),
	},
	"#shortcut": ki.Props{
		"margin":  units.NewValue(0, units.Px),
		"padding": units.NewValue(0, units.Px),
	},
	"#sc-stretch": ki.Props{
		"min-width": units.NewValue(2, units.Em),
	},
	".menu-action": ki.Props{ // class of actions as menu items
		"padding":   units.NewValue(2, units.Px),
		"margin":    units.NewValue(0, units.Px),
		"max-width": -1,
		"indicator": "widget-wedge-right",
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
			"border-width":     units.NewValue(2, units.Px),
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
	".bar-action": ki.Props{ // class of actions as bar items (MenuBar, ToolBar)
		"padding":   units.NewValue(4, units.Px), // we go to edge of bar
		"margin":    units.NewValue(0, units.Px),
		"indicator": "none",
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
			"border-width":     units.NewValue(2, units.Px),
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
			"border-width":     units.NewValue(2, units.Px),
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

func (ac *Action) ButtonAsBase() *ButtonBase {
	return &(ac.ButtonBase)
}

// Trigger triggers the action signal -- for external activation of action --
// only works if action is not inactive
func (ac *Action) Trigger() {
	if ac.IsInactive() {
		return
	}
	ac.ActionSig.Emit(ac.This, 0, ac.Data)
}

// trigger action signal
func (ac *Action) ButtonRelease() {
	if ac.IsInactive() {
		return
	}
	wasPressed := (ac.State == ButtonDown)
	updt := ac.UpdateStart()
	ac.SetButtonState(ButtonActive)
	ac.ButtonSig.Emit(ac.This, int64(ButtonReleased), nil)
	menOpen := false
	if wasPressed {
		ac.ActionSig.Emit(ac.This, 0, ac.Data)
		ac.ButtonSig.Emit(ac.This, int64(ButtonClicked), ac.Data)
		menOpen = ac.OpenMenu()
	}
	if !menOpen && ac.IsMenu() && ac.Viewport != nil {
		win := ac.Viewport.Win
		if win != nil {
			win.ClosePopup(ac.Viewport) // in case we are a menu popup -- no harm if not
		}
	}
	ac.UpdateEnd(updt)
}

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

func (ac *Action) ConfigPartsShortcut(scIdx int) {
	if scIdx < 0 {
		return
	}
	sc := ac.Parts.KnownChild(scIdx).(*Label)
	sclbl := ac.Shortcut.Shortcut()
	if sc.Text != sclbl {
		sc.Text = sclbl
		ac.StylePart(Node2D(sc))
		ac.StylePart(ac.Parts.KnownChild(scIdx - 1).(Node2D)) // also get the stretch
	}
}

func (ac *Action) ConfigPartsButton() {
	config, icIdx, lbIdx := ac.ConfigPartsIconLabel(string(ac.Icon), ac.Text)
	indIdx := ac.ConfigPartsAddIndicator(&config, false) // default off
	mods, updt := ac.Parts.ConfigChildren(config, false) // not unique names
	ac.ConfigPartsSetIconLabel(string(ac.Icon), ac.Text, icIdx, lbIdx)
	ac.ConfigPartsIndicator(indIdx)
	if ac.Tooltip == "" {
		if ac.Shortcut != "" {
			ac.Tooltip = fmt.Sprintf("Shortcut: %v", ac.Shortcut)
		}
	}
	if mods {
		ac.UpdateEnd(updt)
	}
}

func (ac *Action) ConfigPartsMenuItem() {
	config, icIdx, lbIdx := ac.ConfigPartsIconLabel(string(ac.Icon), ac.Text)
	indIdx := ac.ConfigPartsAddIndicator(&config, false) // default off
	scIdx := -1
	if indIdx < 0 && ac.Shortcut != "" {
		scIdx = ac.ConfigPartsAddShortcut(&config)
	} else if ac.Shortcut != "" {
		log.Printf("gi.Action shortcut cannot be used on a sub-menu for action: %v\n", ac.Text)
	}
	mods, updt := ac.Parts.ConfigChildren(config, false) // not unique names
	if mods {
	}
	ac.ConfigPartsSetIconLabel(string(ac.Icon), ac.Text, icIdx, lbIdx)
	ac.ConfigPartsIndicator(indIdx)
	ac.ConfigPartsShortcut(scIdx)
	if mods {
		ac.UpdateEnd(updt)
	}
}

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
		fallthrough
	case istbar:
		if ac.Class == "" {
			ac.Class = "bar-action"
		}
		ac.ConfigPartsButton()
	case ac.IsMenu():
		if ac.Class == "" {
			ac.Class = "menu-action"
		}
		if ac.Indicator == "" {
			ac.Indicator = "widget-wedge-right"
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
