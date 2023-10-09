// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"log"

	"log/slog"

	"goki.dev/colors"
	"goki.dev/cursors"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/goosi/events/key"
	"goki.dev/icons"
	"goki.dev/ki/v2"
)

// todo: autoRepeat, autoRepeatInterval, autoRepeatDelay

// Button is a pressable button with text, an icon, an indicator, a shortcut,
// and/or a menu. The standard behavior is to register a click event with OnClick(...).
//
//goki:embedder
type Button struct {
	WidgetBase

	// the type of button
	Type ButtonTypes `desc:"the type of button"`

	// label for the button -- if blank then no label is presented
	Text string `xml:"text" desc:"label for the button -- if blank then no label is presented"`

	// [view: show-name] optional icon for the button -- different buttons can configure this in different ways relative to the text if both are present
	Icon icons.Icon `xml:"icon" view:"show-name" desc:"optional icon for the button -- different buttons can configure this in different ways relative to the text if both are present"`

	// [view: show-name] name of the menu indicator icon to present, or blank or 'nil' or 'none' -- shown automatically when there are Menu elements present unless 'none' is set
	Indicator icons.Icon `xml:"indicator" view:"show-name" desc:"name of the menu indicator icon to present, or blank or 'nil' or 'none' -- shown automatically when there are Menu elements present unless 'none' is set"`

	// optional shortcut keyboard chord to trigger this action -- always window-wide in scope, and should generally not conflict other shortcuts (a log message will be emitted if so).  Shortcuts are processed after all other processing of keyboard input.  Use Command for Control / Meta (Mac Command key) per platform.  These are only set automatically for Menu items, NOT for items in ToolBar or buttons somewhere, but the tooltip for buttons will show the shortcut if set.
	Shortcut key.Chord `xml:"shortcut" desc:"optional shortcut keyboard chord to trigger this action -- always window-wide in scope, and should generally not conflict other shortcuts (a log message will be emitted if so).  Shortcuts are processed after all other processing of keyboard input.  Use Command for Control / Meta (Mac Command key) per platform.  These are only set automatically for Menu items, NOT for items in ToolBar or buttons somewhere, but the tooltip for buttons will show the shortcut if set."`

	// the menu items for this menu -- typically add Action elements for menus, along with separators
	Menu MenuActions `desc:"the menu items for this menu -- typically add Action elements for menus, along with separators"`

	// [view: -] set this to make a menu on demand -- if set then this button acts like a menu button
	MakeMenuFunc MakeMenuFunc `copy:"-" json:"-" xml:"-" view:"-" desc:"set this to make a menu on demand -- if set then this button acts like a menu button"`

	// [view: -] optional data that is sent with events to identify the button
	Data any `json:"-" xml:"-" view:"-" desc:"optional data that is sent with events to identify the button"`

	// [view: -] optional function that is called to update state of button (typically updating Active state); called automatically for menus prior to showing
	UpdateFunc func(bt *Button) `json:"-" xml:"-" view:"-" desc:"optional function that is called to update state of button (typically updating Active state); called automatically for menus prior to showing"`
}

func (bt *Button) CopyFieldsFrom(frm any) {
	fr, ok := frm.(*Button)
	if !ok {
		log.Printf("GoGi node of type: %v needs a CopyFieldsFrom method defined -- currently falling back on earlier Button one\n", bt.KiType().Name)
		return
	}
	bt.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
	bt.Type = fr.Type
	bt.Text = fr.Text
	bt.Icon = fr.Icon
	bt.Indicator = fr.Indicator
	bt.Shortcut = fr.Shortcut
	bt.Menu = fr.Menu // note: can't use CopyFrom: need closure funcs in actions; todo: could do more elaborate copy etc but is it worth it?
	bt.MakeMenuFunc = fr.MakeMenuFunc
	bt.Data = fr.Data
}

// ButtonTypes is an enum containing the
// different possible types of buttons
type ButtonTypes int //enums:enum

const (
	// ButtonFilled is a filled button with a
	// contrasting background color. It should be
	// used for prominent actions, typically those
	// that are the final in a sequence. It is equivalent
	// to Material Design's filled button.
	ButtonFilled ButtonTypes = iota
	// ButtonTonal is a filled button, similar
	// to [ButtonFilled]. It is used for the same purposes,
	// but it has a lighter background color and less emphasis.
	// It is equivalent to Material Design's filled tonal button.
	ButtonTonal
	// ButtonElevated is an elevated button with
	// a light background color and a shadow.
	// It is equivalent to Material Design's elevated button.
	ButtonElevated
	// ButtonOutlined is an outlined button that is
	// used for secondary actions that are still important.
	// It is equivalent to Material Design's outlined button.
	ButtonOutlined
	// ButtonText is a low-importance button with only
	// text and/or an icon and no border, background color,
	// or shadow. They should only be used for low emphasis
	// actions, and you must ensure they stand out from the
	// surrounding context sufficiently. It is equivalent
	// to Material Design's text and icon buttons.
	ButtonText
)

// TODO(kai): the difference between ButtonFlagMenu and HasMenu is documented
// inconsistently, so we need to reach a clear decision on what they are an
// whether we need ButtonFlags

// ButtonFlags extend WidgetFlags to hold button state
type ButtonFlags WidgetFlags //enums:bitflag

const (
	// Menu flag means that the button is a menu item itself
	// (not that it has a menu; see [Button.HasMenu])
	ButtonFlagMenu ButtonFlags = ButtonFlags(WidgetFlagsN) + iota
)

func (bt *Button) OnInit() {
	bt.ButtonHandlers()
	bt.ButtonStyles()
}

func (bt *Button) ButtonStyles() {
	bt.AddStyles(func(s *styles.Style) {
		s.SetAbilities(true, states.Activatable, states.Focusable, states.Hoverable)
		s.SetAbilities(bt.ShortcutTooltip() != "", states.LongHoverable)
		s.Cursor = cursors.Pointer
		s.Border.Radius = styles.BorderRadiusFull
		s.Padding.Set(units.Em(0.625*Prefs.DensityMul()), units.Em(1.5*Prefs.DensityMul()))
		if !bt.Icon.IsNil() {
			s.Padding.Left.SetEm(1 * Prefs.DensityMul())
		}
		if bt.Text == "" {
			s.Padding.Right.SetEm(1 * Prefs.DensityMul())
		}
		s.Text.Align = styles.AlignCenter
		s.MaxBoxShadow = styles.BoxShadow1()
		switch bt.Type {
		case ButtonFilled:
			s.BackgroundColor.SetSolid(colors.Scheme.Primary.Base)
			s.Color = colors.Scheme.Primary.On
			if s.Is(states.Focused) {
				s.Border.Color.Set(colors.Scheme.OnSurface) // primary is too hard to see
			}
		case ButtonTonal:
			s.BackgroundColor.SetSolid(colors.Scheme.Secondary.Container)
			s.Color = colors.Scheme.Secondary.OnContainer
		case ButtonElevated:
			s.BackgroundColor.SetSolid(colors.Scheme.SurfaceContainerLow)
			s.Color = colors.Scheme.Primary.Base
			s.MaxBoxShadow = styles.BoxShadow2()
			s.BoxShadow = styles.BoxShadow1()
		case ButtonOutlined:
			s.BackgroundColor.SetSolid(colors.Scheme.Surface)
			s.Color = colors.Scheme.Primary.Base
			s.Border.Style.Set(styles.BorderSolid)
			s.Border.Color.Set(colors.Scheme.Outline)
			s.Border.Width.Set(units.Dp(1))
		case ButtonText:
			s.Color = colors.Scheme.Primary.Base
		}
		if s.Is(states.Hovered) {
			s.BoxShadow = s.MaxBoxShadow
		}
	})
}

func (bt *Button) OnChildAdded(child ki.Ki) {
	w, _ := AsWidget(child)
	switch w.Name() {
	case "icon":
		w.AddStyles(func(s *styles.Style) {
			s.Width.SetEm(1.125)
			s.Height.SetEm(1.125)
			s.Margin.Set()
			s.Padding.Set()
		})
	case "space":
		w.AddStyles(func(s *styles.Style) {
			s.Width.SetEm(0.5)
			s.MinWidth.SetEm(0.5)
		})
	case "label":
		label := w.(*Label)
		label.Type = LabelLabelLarge
		w.AddStyles(func(s *styles.Style) {
			s.SetAbilities(false, states.Selectable, states.DoubleClickable)
			s.Cursor = cursors.None
			s.Text.WhiteSpace = styles.WhiteSpaceNowrap
			s.Margin.Set()
			s.Padding.Set()
			s.AlignV = styles.AlignMiddle
		})
	case "ind-stretch":
		w.AddStyles(func(s *styles.Style) {
			s.Width.SetEm(0.5)
		})
	case "indicator":
		w.AddStyles(func(s *styles.Style) {
			s.Width.SetEm(1.125)
			s.Height.SetEm(1.125)
			s.Margin.Set()
			s.Padding.Set()
			s.AlignV = styles.AlignBottom
		})
	}
}

// see menus.go for MakeMenuFunc, etc

// SetAsMenu ensures that this functions as a menu even before menu items are added
func (bt *Button) SetAsMenu() {
	bt.SetFlag(true, ButtonFlagMenu)
}

// SetAsButton clears the explicit ButtonFlagMenu -- if there are menu items
// or a menu function then it will still behave as a menu
func (bt *Button) SetAsButton() {
	bt.SetFlag(false, ButtonFlagMenu)
}

// SetType sets the styling type of the button
func (bt *Button) SetType(typ ButtonTypes) *Button {
	updt := bt.UpdateStart()
	bt.Type = typ
	bt.UpdateEndLayout(updt)
	return bt
}

// LabelWidget returns the label widget if present
func (bt *Button) LabelWidget() *Label {
	lbi := bt.Parts.ChildByName("label")
	if lbi == nil {
		return nil
	}
	return lbi.(*Label)
}

// IconWidget returns the icon widget if present
func (bt *Button) IconWidget() *Icon {
	ici := bt.Parts.ChildByName("icon")
	if ici == nil {
		return nil
	}
	return ici.(*Icon)
}

// SetText sets the text and updates the button.
// Use this for optimized auto-updating based on nature of changes made.
// Otherwise, can set Text directly followed by ReConfig()
func (bt *Button) SetText(txt string) *Button {
	if bt.Text == txt {
		return bt
	}
	updt := bt.UpdateStart()
	recfg := bt.Parts == nil || (bt.Text == "" && txt != "") || (bt.Text != "" && txt == "")
	bt.Text = txt
	if recfg {
		bt.ConfigParts(bt.Sc)
	} else {
		lbl := bt.LabelWidget()
		if lbl != nil {
			lbl.SetText(bt.Text)
		}
	}
	bt.UpdateEndLayout(updt) // todo: could optimize to not re-layout every time but..
	return bt
}

// SetIcon sets the Icon to given icon name (could be empty or 'none') and
// updates the button.
// Use this for optimized auto-updating based on nature of changes made.
// Otherwise, can set Icon directly followed by ReConfig()
func (bt *Button) SetIcon(iconName icons.Icon) *Button {
	if bt.Icon == iconName {
		return bt
	}
	updt := bt.UpdateStart()
	recfg := (bt.Icon == "" && iconName != "") || (bt.Icon != "" && iconName == "")
	bt.Icon = iconName
	if recfg {
		bt.ConfigParts(bt.Sc)
	} else {
		ic := bt.IconWidget()
		if ic != nil {
			ic.SetIcon(bt.Icon)
		}
	}
	bt.UpdateEndLayout(updt)
	return bt
}

// HasMenu returns true if there is a menu or menu-making function set, or the
// explicit ButtonFlagMenu has been set
func (bt *Button) HasMenu() bool {
	// we're not even using ButtonFlagMenu!
	return bt.MakeMenuFunc != nil || len(bt.Menu) > 0
}

// OpenMenu will open any menu associated with this element -- returns true if
// menu opened, false if not
func (bt *Button) OpenMenu() bool {
	if !bt.HasMenu() {
		return false
	}
	if bt.MakeMenuFunc != nil {
		bt.MakeMenuFunc(bt.This().(Widget), &bt.Menu)
	}
	pos := bt.ContextMenuPos()
	if bt.Parts != nil {
		if indic := bt.Parts.ChildByName("indicator", 3); indic != nil {
			pos = indic.(Widget).ContextMenuPos()
		}
	} else {
		slog.Error("Button: parts nil", "button", bt)
	}
	NewMenu(bt.Menu, bt.This().(Widget), pos).Run()
	return true
}

// ResetMenu removes all items in the menu
func (bt *Button) ResetMenu() {
	bt.Menu = make(MenuActions, 0, 10)
}

// ConfigPartsAddIndicator adds a menu indicator if the Indicator field is set to an icon;
// if defOn is true, an indicator is added even if the Indicator field is unset
// (as long as it is not explicitly set to [icons.None]);
// returns the index in Parts of the indicator object, which is named "indicator";
// an "ind-stretch" is added as well to put on the right by default.
func (bt *Button) ConfigPartsAddIndicator(config *ki.Config, defOn bool) int {
	needInd := !bt.Indicator.IsNil() || (defOn && bt.Indicator != icons.None)
	if !needInd {
		return -1
	}
	indIdx := -1
	config.Add(StretchType, "ind-stretch")
	indIdx = len(*config)
	config.Add(IconType, "indicator")
	return indIdx
}

func (bt *Button) ConfigPartsIndicator(indIdx int) {
	if indIdx < 0 {
		return
	}
	ic := bt.Parts.Child(indIdx).(*Icon)
	icnm := bt.Indicator
	if icnm.IsNil() {
		icnm = icons.KeyboardArrowDown
	}
	ic.SetIcon(icnm)
}

//////////////////////////////////////////////////////////////////
//		Events

func (bt *Button) ClickMenu() {
	bt.On(events.Click, func(e events.Event) {
		if bt.StateIs(states.Disabled) {
			return
		}
		bt.OpenMenu()
		// dismiss menu if needed
		if bt.Sc != nil && bt.Sc.Stage != nil {
			pst := bt.Sc.Stage.AsPopup()
			if pst != nil && pst.Type == Menu {
				pst.Close()
			}
		} else {
			if bt.Sc == nil {
				slog.Error("ac.Sc == nil")
			} else if bt.Sc.Stage == nil {
				slog.Error("ac.Sc.Stage == nil")
			}
		}
	})
}

// ClickOnEnterSpace adds key event handler for Enter or Space
// to generate a Click action
func (bt *Button) ClickOnEnterSpace() {
	bt.On(events.KeyChord, func(e events.Event) {
		if bt.StateIs(states.Disabled) {
			return
		}
		if KeyEventTrace {
			fmt.Printf("Button KeyChordEvent: %v\n", bt.Path())
		}
		kf := KeyFun(e.KeyChord())
		if kf == KeyFunEnter || e.KeyRune() == ' ' {
			// if !(kt.Rune == ' ' && bbb.Sc.Type == ScCompleter) {
			e.SetHandled()
			bt.Send(events.Click, e)
			// }
		}
	})
}

// ShortcutTooltip returns the effective tooltip of the button
// with any keyboard shortcut included.
func (bt *Button) ShortcutTooltip() string {
	if bt.Tooltip == "" && bt.Shortcut == "" {
		return ""
	}
	res := bt.Tooltip
	if bt.Shortcut != "" {
		res = "[ " + bt.Shortcut.Shortcut() + " ]"
		if bt.Tooltip != "" {
			res += ": " + bt.Tooltip
		}
	}
	return res
}

func (bt *Button) LongHoverTooltip() {
	bt.On(events.LongHoverStart, func(e events.Event) {
		if bt.StateIs(states.Disabled) {
			return
		}
		tt := bt.ShortcutTooltip()
		if tt == "" {
			return
		}
		e.SetHandled()
		NewTooltipText(bt, tt, e.Pos()).Run()
	})
}

func (bt *Button) ButtonHandlers() {
	bt.WidgetHandlers()
	bt.LongHoverTooltip()
	bt.ClickMenu()
	bt.ClickOnEnterSpace()
}

func (bt *Button) ConfigWidget(sc *Scene) {
	bt.ConfigParts(sc)
}

func (bt *Button) ConfigParts(sc *Scene) {
	parts := bt.NewParts(LayoutHoriz)
	if bt.HasMenu() && bt.Icon.IsNil() && bt.Indicator.IsNil() {
		if bt.Is(ButtonFlagMenu) {
			bt.Indicator = icons.KeyboardArrowRight
		} else {
			bt.Icon = icons.Menu
		}
	}
	config := ki.Config{}
	icIdx, lbIdx := bt.ConfigPartsIconLabel(&config, bt.Icon, bt.Text)
	indIdx := bt.ConfigPartsAddIndicator(&config, false) // default off
	scIdx := -1
	if bt.Is(ButtonFlagMenu) {
		if indIdx < 0 && bt.Shortcut != "" {
			scIdx = bt.ConfigPartsAddShortcut(&config)
		} else if bt.Shortcut != "" {
			slog.Error("gi.Button: shortcut cannot be used on a sub-menu for", "button", bt)
		}
	}
	mods, updt := parts.ConfigChildren(config)
	bt.ConfigPartsSetIconLabel(bt.Icon, bt.Text, icIdx, lbIdx)
	bt.ConfigPartsIndicator(indIdx)
	bt.ConfigPartsShortcut(scIdx)
	if mods {
		parts.UpdateEnd(updt)
		bt.SetNeedsLayout(sc, updt)
	}
}

// ConfigPartsIconLabel adds to config to create parts, of icon
// and label left-to right in a row, based on whether items are nil or empty
func (bt *Button) ConfigPartsIconLabel(config *ki.Config, icnm icons.Icon, txt string) (icIdx, lbIdx int) {
	icIdx = -1
	lbIdx = -1
	if icnm.IsValid() {
		icIdx = len(*config)
		config.Add(IconType, "icon")
		if txt != "" {
			config.Add(SpaceType, "space")
		}
	}
	if txt != "" {
		lbIdx = len(*config)
		config.Add(LabelType, "label")
	}
	return
}

// ConfigPartsSetIconLabel sets the icon and text values in parts, and get
// part style props, using given props if not set in object props
func (bt *Button) ConfigPartsSetIconLabel(icnm icons.Icon, txt string, icIdx, lbIdx int) {
	if icIdx >= 0 {
		ic := bt.Parts.Child(icIdx).(*Icon)
		ic.SetIcon(icnm)
	}
	if lbIdx >= 0 {
		lbl := bt.Parts.Child(lbIdx).(*Label)
		if lbl.Text != txt {
			lbl.SetText(txt)
			lbl.Config(bt.Sc) // this is essential
		}
	}
}

// ConfigPartsShortcut sets the shortcut
func (bt *Button) ConfigPartsShortcut(scIdx int) {
	if scIdx < 0 {
		return
	}
	sc := bt.Parts.Child(scIdx).(*Label)
	sclbl := bt.Shortcut.Shortcut()
	if sc.Text != sclbl {
		sc.Text = sclbl
	}
}

// ConfigPartsAddShortcut adds a menu shortcut, with a stretch space -- only called when needed
func (bt *Button) ConfigPartsAddShortcut(config *ki.Config) int {
	config.Add(StretchType, "sc-stretch")
	scIdx := len(*config)
	config.Add(LabelType, "shortcut")
	return scIdx
}

func (bt *Button) ApplyStyle(sc *Scene) {
	bt.ApplyStyleWidget(sc)
	if bt.Menu != nil {
		bt.Menu.SetShortcuts(bt.EventMgr())
	}
}

func (bt *Button) DoLayout(sc *Scene, parBBox image.Rectangle, iter int) bool {
	bt.DoLayoutBase(sc, parBBox, iter)
	bt.DoLayoutParts(sc, parBBox, iter)
	return bt.DoLayoutChildren(sc, iter)
}

func (bt *Button) RenderButton(sc *Scene) {
	rs, _, st := bt.RenderLock(sc)
	bt.RenderStdBox(sc, st)
	bt.RenderUnlock(rs)
}

func (bt *Button) Render(sc *Scene) {
	if bt.PushBounds(sc) {
		bt.RenderButton(sc)
		bt.RenderParts(sc)
		bt.RenderChildren(sc)
		bt.PopBounds(sc)
	}
}

func (bt *Button) Destroy() {
	if bt.Menu != nil {
		bt.Menu.DeleteShortcuts(bt.EventMgr())
	}
}

// UpdateButtons calls UpdateFunc on me and any of my menu items
func (bt *Button) UpdateButtons() {
	if bt.UpdateFunc != nil {
		bt.UpdateFunc(bt)
	}
	if bt.Menu != nil {
		bt.Menu.UpdateActions()
	}
}
