// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"log/slog"

	"cogentcore.org/core/abilities"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keyfun"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/states"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
)

// todo: autoRepeat, autoRepeatInterval, autoRepeatDelay

// Button is a pressable button with text, an icon, an indicator, a shortcut,
// and/or a menu. The standard behavior is to register a click event with OnClick(...).
type Button struct { //core:embedder
	Box

	// the type of button
	Type ButtonTypes

	// Text is the label text for the button.
	// If it is blank, no label is shown.
	Text string `set:"-"`

	// Icon is the icon for the button.
	// If it is "" or [icons.None], no icon is shown.
	Icon icons.Icon `xml:"icon" view:"show-name"`

	// Indicator is the menu indicator icon to present.
	// If it is "" or [icons.None],, no indicator is shown.
	// It is automatically set to [icons.KeyboardArrowDown]
	// when there is a Menu elements present unless it is
	// set to [icons.None].
	Indicator icons.Icon `xml:"indicator" view:"show-name"`

	// Shortcut is an optional shortcut keyboard chord to trigger this button,
	// active in window-wide scope. Avoid conflicts with other shortcuts
	// (a log message will be emitted if so). Shortcuts are processed after
	// all other processing of keyboard input. Use Command for
	// Control / Meta (Mac Command key) per platform.
	Shortcut key.Chord `xml:"shortcut"`

	// Menu is a menu constructor function used to build and display
	// a menu whenever the button is clicked. There will be no menu
	// if it is nil. The constructor function should add buttons
	// to the Scene that it is passed.
	Menu func(m *Scene) `json:"-" xml:"-"`
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
	bt.Box.OnInit()
	bt.HandleEvents()
	bt.SetStyles()
}

func (bt *Button) SetStyles() {
	bt.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Activatable, abilities.Focusable, abilities.Hoverable)
		if !bt.IsDisabled() {
			s.Cursor = cursors.Pointer
		}
		s.Border.Radius = styles.BorderRadiusFull
		s.Padding.Set(units.Dp(10), units.Dp(24))
		if bt.Icon.IsSet() {
			s.Padding.Left.Dp(16)
		}
		if bt.Text == "" {
			s.Padding.Right.Dp(16)
		}
		s.Justify.Content = styles.Center
		s.MaxBoxShadow = styles.BoxShadow1()
		switch bt.Type {
		case ButtonFilled:
			s.Background = colors.C(colors.Scheme.Primary.Base)
			s.Color = colors.Scheme.Primary.On
			if s.Is(states.Focused) {
				s.Border.Color.Set(colors.Scheme.OnSurface) // primary is too hard to see
			}
		case ButtonTonal:
			s.Background = colors.C(colors.Scheme.Secondary.Container)
			s.Color = colors.Scheme.Secondary.OnContainer
		case ButtonElevated:
			s.Background = colors.C(colors.Scheme.SurfaceContainerLow)
			s.Color = colors.Scheme.Primary.Base
			s.MaxBoxShadow = styles.BoxShadow2()
			s.BoxShadow = styles.BoxShadow1()
		case ButtonOutlined:
			s.Color = colors.Scheme.Primary.Base
			s.Border.Style.Set(styles.BorderSolid)
			s.Border.Width.Set(units.Dp(1))
			// if focused then default primary
			if !s.Is(states.Focused) {
				s.Border.Color.Set(colors.Scheme.Outline)
			}
		case ButtonText:
			s.Color = colors.Scheme.Primary.Base
		case ButtonAction:
			s.MaxBoxShadow = styles.BoxShadow0()
			s.Justify.Content = styles.Start
		case ButtonMenu:
			s.Grow.Set(1, 0) // need to go to edge of menu
			s.Justify.Content = styles.Start
			s.Border.Radius = styles.BorderRadiusNone
			s.Padding.Set(units.Dp(6), units.Dp(12))
			s.MaxBoxShadow = styles.BoxShadow0()
		}
		if s.Is(states.Hovered) {
			s.BoxShadow = s.MaxBoxShadow
		}
		if bt.IsDisabled() {
			s.MaxBoxShadow = styles.BoxShadow0()
			s.BoxShadow = s.MaxBoxShadow
		}
	})
	bt.OnWidgetAdded(func(w Widget) {
		switch w.PathFrom(bt) {
		case "parts":
			w.Style(func(s *styles.Style) {
				s.Gap.Zero()
				s.Align.Content = styles.Center
				s.Align.Items = styles.Center
				s.Text.AlignV = styles.Center
			})
		case "parts/icon":
			w.Style(func(s *styles.Style) {
				s.Min.X.Dp(18)
				s.Min.Y.Dp(18)
				s.Margin.Zero()
				s.Padding.Zero()
				if bt.Text != "" {
					s.Margin.Right.Ch(1)
				}
			})
		case "parts/label":
			label := w.(*Label)
			if bt.Type == ButtonMenu {
				label.Type = LabelBodyMedium
			} else {
				label.Type = LabelLabelLarge
			}
			w.Style(func(s *styles.Style) {
				s.SetNonSelectable()
				s.SetTextWrap(false)
				s.Margin.Zero()
				s.Padding.Zero()
				s.Max.X.Zero()
				s.FillMargin = false
			})
		case "parts/ind-stretch":
			w.Style(func(s *styles.Style) {
				s.Min.X.Em(0.2)
			})
		case "parts/indicator":
			w.Style(func(s *styles.Style) {
				s.Min.X.Dp(18)
				s.Min.Y.Dp(18)
				s.Margin.Zero()
				s.Padding.Zero()
				s.Align.Self = styles.End
			})
		case "parts/shortcut":
			sc := w.(*Label)
			if bt.Type == ButtonMenu {
				sc.Type = LabelBodyMedium
			} else {
				sc.Type = LabelLabelLarge
			}
			w.Style(func(s *styles.Style) {
				s.SetNonSelectable()
				s.SetTextWrap(false)
			})
		}
	})
}

// SetKey sets the shortcut of the button from the given [keyfun.Funs]
func (bt *Button) SetKey(kf keyfun.Funs) *Button {
	bt.SetShortcut(keyfun.ShortcutFor(kf))
	return bt
}

// NOTE: Button.SetText must be defined manually so that [giv.FuncButton]
// can define its own SetText method that updates the tooltip

// SetText sets the [Button.Text]:
// label for the button -- if blank then no label is presented
func (bt *Button) SetText(v string) *Button {
	bt.Text = v
	return bt
}

// SetTextUpdate sets the Label text and does a Config and updates
// the actual button label render, so it will render next time
// with the updated text value, without flagging any unnecessary layouts.
// This is used for more efficient large-scale updating in views.
func (bt *Button) SetTextUpdate(text string) *Button {
	bt.Text = text
	bt.ConfigWidget()
	lb := bt.LabelWidget()
	if lb != nil {
		lb.SetTextUpdate(text)
	}
	return bt
}

// SetIconUpdate sets the Icon and does a Config and updates
// the actual icon render, so it will render next time
// with the updated text value, without flagging any unnecessary layouts.
// This is used for more efficient large-scale updating in views.
func (bt *Button) SetIconUpdate(icon icons.Icon) *Button {
	bt.Icon = icon
	bt.ConfigWidget()
	ic := bt.IconWidget()
	if ic != nil {
		ic.SetIcon(icon)
	}
	return bt
}

func (bt *Button) Label() string {
	if bt.Text != "" {
		return bt.Text
	}
	return bt.Nm
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
	if indic := bt.Parts.ChildByName("indicator", 3); indic != nil {
		pos = indic.(Widget).ContextMenuPos(nil) // use the pos
	}
	m := NewMenu(bt.Menu, bt.This().(Widget), pos)
	if m == nil {
		return false
	}
	m.Run()
	return true
}

//////////////////////////////////////////////////////////////////
//		Events

func (bt *Button) HandleClickDismissMenu() {
	// note: must be called last so widgets aren't deleted when the click arrives
	bt.OnFinal(events.Click, func(e events.Event) {
		pst := bt.Scene.Stage
		pst.ClosePopup()
	})
}

func (bt *Button) WidgetTooltip() string {
	res := bt.Tooltip
	if bt.Shortcut != "" {
		res = "[ " + bt.Shortcut.Shortcut() + " ]"
		if bt.Tooltip != "" {
			res += " " + bt.Tooltip
		}
	}
	return res
}

func (bt *Button) HandleEvents() {
	bt.HandleClickOnEnterSpace()
	bt.OnClick(func(e events.Event) {
		if bt.OpenMenu(e) {
			e.SetHandled()
		}
	})
}

func (bt *Button) ConfigWidget() {
	config := ki.Config{}

	// we check if the icons are unset, not if they are nil, so
	// that people can manually set it to [icons.None]
	if bt.HasMenu() {
		if bt.Type == ButtonMenu {
			if bt.Indicator == "" {
				bt.Indicator = icons.KeyboardArrowRight
			}
		} else if bt.Text != "" {
			if bt.Indicator == "" {
				bt.Indicator = icons.KeyboardArrowDown
			}
		} else {
			if bt.Icon == "" {
				bt.Icon = icons.Menu
			}
		}
	}

	ici := -1
	lbi := -1
	if bt.Icon.IsSet() {
		ici = len(config)
		config.Add(IconType, "icon")
	}
	if bt.Text != "" {
		lbi = len(config)
		config.Add(LabelType, "label")
	}

	indi := -1
	if bt.Indicator.IsSet() {
		config.Add(StretchType, "ind-stretch")
		indi = len(config)
		config.Add(IconType, "indicator")
	}

	sci := -1
	if bt.Type == ButtonMenu {
		if indi < 0 && bt.Shortcut != "" {
			config.Add(StretchType, "sc-stretch")
			sci = len(config)
			config.Add(LabelType, "shortcut")
		} else if bt.Shortcut != "" {
			slog.Error("programmer error: gi.Button: shortcut cannot be used on a sub-menu for", "button", bt)
		}
	}

	bt.ConfigParts(config, func() {
		if ici >= 0 {
			ic := bt.Parts.Child(ici).(*Icon)
			ic.SetIcon(bt.Icon)
		}
		if lbi >= 0 {
			lbl := bt.Parts.Child(lbi).(*Label)
			if lbl.Text != bt.Text {
				lbl.SetTextUpdate(bt.Text)
			}
		}

		if indi >= 0 {
			ic := bt.Parts.Child(indi).(*Icon)
			ic.SetIcon(bt.Indicator)
		}

		if sci >= 0 {
			sc := bt.Parts.Child(sci).(*Label)
			sctxt := bt.Shortcut.Shortcut()
			if sc.Text != sctxt {
				sc.SetTextUpdate(sctxt)
			}
		}
	})
}

// NOTE: all menus are dynamic.  This obviates the need for updating them.
// OTOH, it makes it impossible to find them..

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
