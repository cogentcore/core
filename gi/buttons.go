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
}

func (bb *Button) CopyFieldsFrom(frm any) {
	fr, ok := frm.(*Button)
	if !ok {
		log.Printf("GoGi node of type: %v needs a CopyFieldsFrom method defined -- currently falling back on earlier ButtonBase one\n", bb.KiType().Name)
		return
	}
	bb.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
	bb.Type = fr.Type
	bb.Text = fr.Text
	bb.Icon = fr.Icon
	bb.Indicator = fr.Indicator
	bb.Shortcut = fr.Shortcut
	bb.Menu = fr.Menu // note: can't use CopyFrom: need closure funcs in actions; todo: could do more elaborate copy etc but is it worth it?
	bb.MakeMenuFunc = fr.MakeMenuFunc
}

// ButtonFlags extend WidgetFlags to hold button state
type ButtonFlags WidgetFlags //enums:bitflag

const (
	// Menu flag means that the button is a menu item
	ButtonFlagMenu ButtonFlags = ButtonFlags(WidgetFlagsN) + iota
)

// see menus.go for MakeMenuFunc, etc

// SetCheckable sets whether this button is checkable
func (bb *Button) SetCheckable(checkable bool) {
	bb.Style.SetAbilities(checkable, states.Checkable)
}

// SetAsMenu ensures that this functions as a menu even before menu items are added
func (bb *Button) SetAsMenu() {
	bb.SetFlag(true, ButtonFlagMenu)
}

// SetAsButton clears the explicit ButtonFlagMenu -- if there are menu items
// or a menu function then it will still behave as a menu
func (bb *Button) SetAsButton() {
	bb.SetFlag(false, ButtonFlagMenu)
}

// LabelWidget returns the label widget if present
func (bb *Button) LabelWidget() *Label {
	lbi := bb.Parts.ChildByName("label")
	if lbi == nil {
		return nil
	}
	return lbi.(*Label)
}

// IconWidget returns the iconl widget if present
func (bb *Button) IconWidget() *Icon {
	ici := bb.Parts.ChildByName("icon")
	if ici == nil {
		return nil
	}
	return ici.(*Icon)
}

// SetText sets the text and updates the button.
// Use this for optimized auto-updating based on nature of changes made.
// Otherwise, can set Text directly followed by ReConfig()
func (bb *Button) SetText(txt string) ButtonWidget {
	if bb.Text == txt {
		return bb.This().(ButtonWidget)
	}
	updt := bb.UpdateStart()
	recfg := bb.Parts == nil || (bb.Text == "" && txt != "") || (bb.Text != "" && txt == "")
	bb.Text = txt
	if recfg {
		bb.This().(ButtonWidget).ConfigParts(bb.Sc)
	} else {
		lbl := bb.LabelWidget()
		if lbl != nil {
			lbl.SetText(bb.Text)
		}
	}
	bb.UpdateEndLayout(updt) // todo: could optimize to not re-layout every time but..
	return bb.This().(ButtonWidget)
}

// SetIcon sets the Icon to given icon name (could be empty or 'none') and
// updates the button.
// Use this for optimized auto-updating based on nature of changes made.
// Otherwise, can set Icon directly followed by ReConfig()
func (bb *Button) SetIcon(iconName icons.Icon) ButtonWidget {
	if bb.Icon == iconName {
		return bb.This().(ButtonWidget)
	}
	updt := bb.UpdateStart()
	recfg := (bb.Icon == "" && iconName != "") || (bb.Icon != "" && iconName == "")
	bb.Icon = iconName
	if recfg {
		bb.This().(ButtonWidget).ConfigParts(bb.Sc)
	} else {
		ic := bb.IconWidget()
		if ic != nil {
			ic.SetIcon(bb.Icon)
		}
	}
	bb.UpdateEndLayout(updt)
	return bb.This().(ButtonWidget)
}

// HasMenu returns true if there is a menu or menu-making function set, or the
// explicit ButtonFlagMenu has been set
func (bb *Button) HasMenu() bool {
	return bb.MakeMenuFunc != nil || len(bb.Menu) > 0
}

// OpenMenu will open any menu associated with this element -- returns true if
// menu opened, false if not
func (bb *Button) OpenMenu() bool {
	if !bb.HasMenu() {
		return false
	}
	if bb.MakeMenuFunc != nil {
		bb.MakeMenuFunc(bb.This().(Widget), &bb.Menu)
	}
	pos := bb.ContextMenuPos()
	if bb.Parts != nil {
		if indic := bb.Parts.ChildByName("indicator", 3); indic != nil {
			pos = indic.(Widget).ContextMenuPos()
		}
	} else {
		slog.Error("ButtonBase: parts nil", "button", bb)
	}
	NewMenu(bb.Menu, bb.This().(Widget), pos).Run()
	return true
}

// ResetMenu removes all items in the menu
func (bb *Button) ResetMenu() {
	bb.Menu = make(MenuActions, 0, 10)
}

// ConfigPartsAddIndicator adds a menu indicator if the Indicator field is set to an icon;
// if defOn is true, an indicator is added even if the Indicator field is unset
// (as long as it is not explicitly set to [icons.None]);
// returns the index in Parts of the indicator object, which is named "indicator";
// an "ind-stretch" is added as well to put on the right by default.
func (bb *Button) ConfigPartsAddIndicator(config *ki.Config, defOn bool) int {
	needInd := !bb.Indicator.IsNil() || (defOn && bb.Indicator != icons.None)
	if !needInd {
		return -1
	}
	indIdx := -1
	config.Add(StretchType, "ind-stretch")
	indIdx = len(*config)
	config.Add(IconType, "indicator")
	return indIdx
}

func (bb *Button) ConfigPartsIndicator(indIdx int) {
	if indIdx < 0 {
		return
	}
	ic := bb.Parts.Child(indIdx).(*Icon)
	icnm := bb.Indicator
	if icnm.IsNil() {
		icnm = icons.KeyboardArrowDown
	}
	ic.SetIcon(icnm)
}

//////////////////////////////////////////////////////////////////
//		Events

func (bb *Button) ClickMenu() {
	bb.On(events.Click, func(e events.Event) {
		if bb.StateIs(states.Disabled) {
			return
		}
		bb.OpenMenu()
	})
}

// ClickOnEnterSpace adds key event handler for Enter or Space
// to generate a Click action
func (bb *Button) ClickOnEnterSpace() {
	bb.On(events.KeyChord, func(e events.Event) {
		if bb.StateIs(states.Disabled) {
			return
		}
		if KeyEventTrace {
			fmt.Printf("Button KeyChordEvent: %v\n", bb.Path())
		}
		kf := KeyFun(e.KeyChord())
		if kf == KeyFunEnter || e.KeyRune() == ' ' {
			// if !(kt.Rune == ' ' && bbb.Sc.Type == ScCompleter) {
			e.SetHandled()
			bb.Send(events.Click, e)
			// }
		}
	})
}

// ShortcutTooltip returns the effective tooltip of the button
// with any keyboard shortcut included.
func (bb *Button) ShortcutTooltip() string {
	if bb.Tooltip == "" && bb.Shortcut == "" {
		return ""
	}
	res := bb.Tooltip
	if bb.Shortcut != "" {
		res = "[ " + bb.Shortcut.Shortcut() + " ]"
		if bb.Tooltip != "" {
			res += ": " + bb.Tooltip
		}
	}
	return res
}

func (bb *Button) LongHoverTooltip() {
	bb.On(events.LongHoverStart, func(e events.Event) {
		if bb.StateIs(states.Disabled) {
			return
		}
		tt := bb.ShortcutTooltip()
		if tt == "" {
			return
		}
		e.SetHandled()
		NewTooltipText(bb, tt, e.Pos()).Run()
	})
}

func (bb *Button) ButtonBaseHandlers() {
	bb.WidgetHandlers()
	bb.LongHoverTooltip()
	bb.ClickMenu()
	bb.ClickOnEnterSpace()
}

///////////////////////////////////////////////////////////
//   ButtonWidget

// ButtonWidget is an interface for button widgets allowing ButtonBase
// defaults to handle most cases.
type ButtonWidget interface {
	Widget

	// AsButtonBase gets the button base for most basic functions -- reduces
	// interface size.
	AsButtonBase() *Button

	// ConfigParts configures the parts of the button -- called during init
	// and style.
	ConfigParts(sc *Scene)

	// SetText sets the text and updates the button.
	// Use this for optimized auto-updating based on nature of changes made.
	// Otherwise, can set Text directly followed by ReConfig()
	SetText(txt string) ButtonWidget

	// SetIcon sets the Icon to given icon name (could be empty or 'none') and
	// updates the button.
	// Use this for optimized auto-updating based on nature of changes made.
	// Otherwise, can set Icon directly followed by ReConfig()
	SetIcon(iconName icons.Icon) ButtonWidget
}

///////////////////////////////////////////////////////////
// ButtonBase Widget and ButtonwWidget interface

func AsButtonBase(k ki.Ki) *Button {
	if ac, ok := k.(ButtonWidget); ok {
		return ac.AsButtonBase()
	}
	return nil
}

func (bb *Button) AsButtonBase() *Button {
	return bb
}

func (bb *Button) ConfigWidget(sc *Scene) {
	bb.This().(ButtonWidget).ConfigParts(sc)
}

func (bb *Button) ConfigParts(sc *Scene) {
	parts := bb.NewParts(LayoutHoriz)
	if bb.HasMenu() && bb.Icon.IsNil() {
		bb.Icon = icons.Menu
	}
	config := ki.Config{}
	icIdx, lbIdx := bb.ConfigPartsIconLabel(&config, bb.Icon, bb.Text)
	indIdx := bb.ConfigPartsAddIndicator(&config, false) // default off

	mods, updt := parts.ConfigChildren(config)
	bb.ConfigPartsSetIconLabel(bb.Icon, bb.Text, icIdx, lbIdx)
	bb.ConfigPartsIndicator(indIdx)
	if mods {
		parts.UpdateEnd(updt)
		bb.SetNeedsLayout(sc, updt)
	}
}

// ConfigPartsIconLabel adds to config to create parts, of icon
// and label left-to right in a row, based on whether items are nil or empty
func (bb *Button) ConfigPartsIconLabel(config *ki.Config, icnm icons.Icon, txt string) (icIdx, lbIdx int) {
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
func (bb *Button) ConfigPartsSetIconLabel(icnm icons.Icon, txt string, icIdx, lbIdx int) {
	if icIdx >= 0 {
		ic := bb.Parts.Child(icIdx).(*Icon)
		ic.SetIcon(icnm)
	}
	if lbIdx >= 0 {
		lbl := bb.Parts.Child(lbIdx).(*Label)
		if lbl.Text != txt {
			lbl.SetText(txt)
			lbl.Config(bb.Sc) // this is essential
		}
	}
}

func (bb *Button) ApplyStyle(sc *Scene) {
	bb.ApplyStyleWidget(sc)
	if bb.Menu != nil {
		bb.Menu.SetShortcuts(bb.EventMgr())
	}
}

func (bb *Button) DoLayout(sc *Scene, parBBox image.Rectangle, iter int) bool {
	bb.DoLayoutBase(sc, parBBox, iter)
	bb.DoLayoutParts(sc, parBBox, iter)
	return bb.DoLayoutChildren(sc, iter)
}

func (bb *Button) RenderButton(sc *Scene) {
	rs, _, st := bb.RenderLock(sc)
	bb.RenderStdBox(sc, st)
	bb.RenderUnlock(rs)
}

func (bb *Button) Render(sc *Scene) {
	if bb.PushBounds(sc) {
		bb.RenderButton(sc)
		bb.RenderParts(sc)
		bb.RenderChildren(sc)
		bb.PopBounds(sc)
	}
}

func (bb *Button) Destroy() {
	if bb.Menu != nil {
		bb.Menu.DeleteShortcuts(bb.EventMgr())
	}
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

func (bt *Button) OnInit() {
	bt.ButtonBaseHandlers()
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

// SetType sets the styling type of the button
func (bt *Button) SetType(typ ButtonTypes) *Button {
	updt := bt.UpdateStart()
	bt.Type = typ
	bt.UpdateEndLayout(updt)
	return bt
}
