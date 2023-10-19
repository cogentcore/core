// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"log"

	"log/slog"

	"goki.dev/colors"
	"goki.dev/cursors"
	"goki.dev/girl/abilities"
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
	//goki:setter
	Type ButtonTypes

	// label for the button -- if blank then no label is presented
	Text string `xml:"text"`

	// optional icon for the button -- different buttons can configure this in different ways relative to the text if both are present
	Icon icons.Icon `xml:"icon" view:"show-name"`

	// name of the menu indicator icon to present, or blank or 'nil' or 'none' -- shown automatically when there are Menu elements present unless 'none' is set
	Indicator icons.Icon `xml:"indicator" view:"show-name"`

	// optional shortcut keyboard chord to trigger this button -- always window-wide in scope, and should generally not conflict other shortcuts (a log message will be emitted if so).  Shortcuts are processed after all other processing of keyboard input.  Use Command for Control / Meta (Mac Command key) per platform.  These are only set automatically for Menu items, NOT for items in Toolbar or buttons somewhere, but the tooltip for buttons will show the shortcut if set.
	Shortcut key.Chord `xml:"shortcut"`

	// If non-nil, a menu constructor function used to build and display a menu whenever the button is clicked.
	// The constructor function should add buttons to the scene that it is passed.
	Menu func(m *Scene)

	// optional data that is sent with events to identify the button
	Data any `json:"-" xml:"-" view:"-"`

	// optional function that is called to update state of button (typically updating Active state); called automatically for menus prior to showing
	UpdateFunc func(bt *Button) `json:"-" xml:"-" view:"-"`
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
	bt.Menu = fr.Menu // TODO(kai/menu): is it safe to copy this?
	bt.Data = fr.Data
}

// ButtonTypes is an enum containing the
// different possible types of buttons
type ButtonTypes int32 //enums:enum -trimprefix Button

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
	// ButtonText is a low-importance button with no border,
	// background color, or shadow when not being interacted with.
	// It renders primary-colored text, and it renders a background
	// color and shadow when hovered/focused/active.
	// It should only be used for low emphasis
	// actions, and you must ensure it stands out from the
	// surrounding context sufficiently. It is equivalent
	// to Material Design's text button, but it can also
	// contain icons and other things.
	ButtonText
	// ButtonAction is a simple button that typically serves
	// as a simple action among a series of other buttons
	// (eg: in a toolbar), or as a part of another widget,
	// like a spinner or snackbar. It has no border, background color,
	// or shadow when not being interacted with. It inherits the text
	// color of its parent, and it renders a background when
	// hovered/focused/active. you must ensure it stands out from the
	// surrounding context  sufficiently. It is equivalent to Material Design's
	// icon button, but it can also contain text and other things (and frequently does).
	ButtonAction
	// ButtonMenu is similar to [ButtonAction], but it is only
	// for buttons located in popup menus.
	ButtonMenu
)

func (bt *Button) OnInit() {
	bt.HandleButtonEvents()
	bt.ButtonStyles()
}

func (bt *Button) ButtonStyles() {
	bt.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Activatable, abilities.Focusable, abilities.Hoverable)
		s.SetAbilities(bt.ShortcutTooltip() != "", abilities.LongHoverable)
		s.Cursor = cursors.Pointer
		s.Border.Radius = styles.BorderRadiusFull
		s.Padding.Set(units.Em(0.625), units.Em(1.5))
		if !bt.Icon.IsNil() {
			s.Padding.Left.SetEm(1)
		}
		if bt.Text == "" {
			s.Padding.Right.SetEm(1)
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
			s.Color = colors.Scheme.Primary.Base
			s.Border.Style.Set(styles.BorderSolid)
			s.Border.Color.Set(colors.Scheme.Outline)
			s.Border.Width.Set(units.Dp(1))
		case ButtonText:
			s.Color = colors.Scheme.Primary.Base
		case ButtonAction:
			s.MaxBoxShadow = styles.BoxShadow0()
		case ButtonMenu:
			s.SetStretchMaxWidth() // need to go to edge of menu
			s.Border.Radius = styles.BorderRadiusNone
			s.Padding.Set(units.Dp(6), units.Dp(12))
			s.MaxBoxShadow = styles.BoxShadow0()
		}
		if s.Is(states.Hovered) {
			s.BoxShadow = s.MaxBoxShadow
		}
		if s.Is(states.Disabled) {
			s.Cursor = cursors.NotAllowed
		}
	})
	bt.OnWidgetAdded(func(w Widget) {
		switch w.PathFrom(bt.This()) {
		case "parts/icon":
			w.Style(func(s *styles.Style) {
				s.Width.SetEm(1.125)
				s.Height.SetEm(1.125)
				s.Margin.Set()
				s.Padding.Set()
			})
		case "parts/space":
			w.Style(func(s *styles.Style) {
				s.Width.SetEm(0.5)
				s.MinWidth.SetEm(0.5)
			})
		case "parts/label":
			label := w.(*Label)
			label.Type = LabelLabelLarge
			w.Style(func(s *styles.Style) {
				s.SetAbilities(false, abilities.Selectable, abilities.DoubleClickable)
				s.Cursor = cursors.None
				s.Text.WhiteSpace = styles.WhiteSpaceNowrap
				s.Margin.Set()
				s.Padding.Set()
				s.AlignV = styles.AlignMiddle
			})
		case "parts/ind-stretch":
			w.Style(func(s *styles.Style) {
				s.Width.SetEm(0.5)
			})
		case "parts/indicator":
			w.Style(func(s *styles.Style) {
				s.Width.SetEm(1.125)
				s.Height.SetEm(1.125)
				s.Margin.Set()
				s.Padding.Set()
				s.AlignV = styles.AlignBottom
			})
		}
	})
}

// SetShortcut sets the shortcut of the button
func (bt *Button) SetShortcut(shortcut key.Chord) *Button {
	updt := bt.UpdateStart()
	bt.Shortcut = shortcut
	bt.UpdateEndLayout(updt)
	return bt
}

// SetShortcut sets the shortcut of the button from the given [KeyFuns]
func (bt *Button) SetShortcutKey(kf KeyFuns) *Button {
	updt := bt.UpdateStart()
	bt.Shortcut = ShortcutForFun(kf)
	bt.UpdateEndLayout(updt)
	return bt
}

// SetData sets the data of the button
func (bt *Button) SetData(data any) *Button {
	updt := bt.UpdateStart()
	bt.Data = data
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

// HasMenu returns true if the button has a menu that pops up when it is clicked
// (not that it is in a menu itself; see [ButtonMenu])
func (bt *Button) HasMenu() bool {
	return bt.Menu != nil
}

// OpenMenu will open any menu associated with this element.
// Returns true if menu opened, false if not.
func (bt *Button) OpenMenu(e events.Event) bool {
	if !bt.HasMenu() {
		return false
	}
	pos := bt.ContextMenuPos(e)
	if bt.Parts != nil {
		if indic := bt.Parts.ChildByName("indicator", 3); indic != nil {
			pos = indic.(Widget).ContextMenuPos(nil) // use the pos
		}
	} else {
		slog.Error("Button: parts nil", "button", bt)
	}
	NewMenu(bt.Menu, bt.This().(Widget), pos).Run()
	return true
}

// TODO(kai/menu): do we need ResetMenu?

// ResetMenu removes the menu constructor function
func (bt *Button) ResetMenu() {
	bt.Menu = nil
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
}

//////////////////////////////////////////////////////////////////
//		Events

func (bt *Button) ContextMenu(e events.Event) {
	bt.OpenMenu(e)
}

func (bt *Button) HandleClickMenu() {
	bt.OnClick(func(e events.Event) {
		if bt.OpenMenu(e) {
			e.SetHandled()
		}
	})
}

func (bt *Button) HandleClickDismissMenu() {
	bt.OnClick(func(e events.Event) {
		if bt.Sc != nil && bt.Sc.Stage != nil {
			pst := bt.Sc.Stage.AsPopup()
			if pst != nil && pst.Type == MenuStage {
				pst.Close()
			}
		} else {
			if bt.Sc == nil {
				slog.Error("bt.Sc == nil")
			} else if bt.Sc.Stage == nil {
				slog.Error("bt.Sc.Stage == nil")
			}
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

func (bt *Button) HandleLongHoverTooltip() {
	bt.On(events.LongHoverStart, func(e events.Event) {
		tt := bt.ShortcutTooltip()
		if tt == "" {
			return
		}
		e.SetHandled()
		NewTooltipText(bt, tt, e.Pos()).Run()
	})
}

func (bt *Button) HandleButtonEvents() {
	bt.HandleWidgetEvents()
	bt.HandleLongHoverTooltip()
	bt.HandleClickMenu()
	bt.HandleClickOnEnterSpace()
}

func (bt *Button) ConfigWidget(sc *Scene) {
	bt.ConfigParts(sc)
}

func (bt *Button) ConfigParts(sc *Scene) {
	parts := bt.NewParts(LayoutHoriz)
	// we check if the icons are unset, not if they are nil, so
	// that people can manually set it to [icons.None]
	if bt.HasMenu() && bt.Icon == "" && bt.Indicator == "" {
		if bt.Type == ButtonMenu {
			bt.Indicator = icons.KeyboardArrowRight
		} else {
			bt.Icon = icons.Menu
		}
	}
	config := ki.Config{}
	icIdx, lbIdx := bt.ConfigPartsIconLabel(&config, bt.Icon, bt.Text)
	indIdx := bt.ConfigPartsAddIndicator(&config, false) // default off
	scIdx := -1
	if bt.Type == ButtonMenu {
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
		bt.SetNeedsLayoutUpdate(sc, updt)
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
	// TODO(kai/menu): figure out how to handle menu shortcuts
	/*
		if bt.Menu != nil {
			bt.Menu.SetShortcuts(bt.EventMgr())
		}
	*/
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
	// TODO(kai/menu): figure out how to handle menu shortcuts
	/*
		if bt.Menu != nil {
			bt.Menu.DeleteShortcuts(bt.EventMgr())
		}
	*/
}

// UpdateButtons calls UpdateFunc on me and any of my menu items
func (bt *Button) UpdateButtons() {
	if bt.UpdateFunc != nil {
		bt.UpdateFunc(bt)
	}
	// TODO(kai/menu): figure out how to handle menu updating
	/*
		if bt.Menu != nil {
			bt.Menu.UpdateButtons()
		}
	*/
}

// TODO(kai/menu): figure out what to do about FindButtonMenu/NewButtonMenu
/*
// FindButtonMenu finds the button with the given path in the given parent,
// searching through both children and any [Button.Menu]s it finds. The
// path omits menus; for example, if A has a menu that contains B, which
// has a menu that contains C, FindButtonMenu(A, "B/C") is correct, but
// FindButtonMenu(A, "menu/B/menu/C") is not correct. If the result of
// FindButtonMenu is nil, that indicates that the button could not be found.
func FindButtonMenu(par ki.Ki, path string) *Button {
	parts := strings.Split(path, "/")
	bt, _ := findButtonMenuImpl(par, parts).(*Button)
	return bt
}

// findButtonMenuImpl is the implementation of FindButtonMenu
func findButtonMenuImpl(par ki.Ki, parts []string) ki.Ki {
	if len(parts) == 0 {
		return par
	}
	if par == nil {
		return nil
	}
	sl := par.Children()
	if bt, ok := par.(*Button); ok {
		sl = (*ki.Slice)(&bt.Menu)
	}
	return findButtonMenuImpl(sl.ElemByName(parts[0]), parts[1:])
}

// NewButtonMenu creates a new button at the given path in the given parent,
// searching for each element of the path through both children and any
// [Button.Menu]s, and creating it if it doesn't exist. It assumes that
// you want to create a button with a menu for each element
// of the path; for example, if you have a button A, and you call
// NewButtonMenu(A, "B/C"), it will make a button B as part of a menu for A,
// and then it will make and return a button C as part of a menu for B.
// If the given path is "" or "/" and par is not a button, it returns nil.
func NewButtonMenu(par ki.Ki, path string) *Button {
	parts := strings.Split(path, "/")
	return newButtonMenuImpl(par, parts)
}

// newButtonMenuImpl is the implementation of NewButtonMenu
func newButtonMenuImpl(par ki.Ki, parts []string) *Button {
	if len(parts) == 0 {
		bt, _ := par.(*Button)
		return bt
	}
	nm := parts[0]

	bt, isBt := par.(*Button)
	if isBt {
		elem := (*ki.Slice)(&bt.Menu).ElemByName(nm)
		if elem != nil {
			return newButtonMenuImpl(elem, parts[1:])
		}
		newbt := bt.Menu.AddButton(ActOpts{Label: nm}, nil)
		return newButtonMenuImpl(newbt, parts[1:])
	}

	child := par.ChildByName(nm)
	if child != nil {
		return newButtonMenuImpl(child, parts[1:])
	}
	newbt := NewButton(par).SetType(ButtonAction).SetText(nm)
	return newButtonMenuImpl(newbt, parts[1:])
}
*/
