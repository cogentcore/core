// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"log/slog"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/system"
)

// Button is an interactive button with text, an icon, an indicator, a shortcut,
// and/or a menu. The standard behavior is to register a click event handler with
// OnClick.
type Button struct { //core:embedder
	Frame

	// Type is the type of button.
	Type ButtonTypes

	// Text is the text for the button.
	// If it is blank, no text is shown.
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
	// all other processing of keyboard input. Command is automatically translated
	// into Meta on macOS and Control on all other platforms.
	Shortcut key.Chord `xml:"shortcut"`

	// Menu is a menu constructor function used to build and display
	// a menu whenever the button is clicked. There will be no menu
	// if it is nil. The constructor function should add buttons
	// to the Scene that it is passed.
	Menu func(m *Scene) `json:"-" xml:"-"`
}

// ButtonTypes is an enum containing the
// different possible types of buttons.
type ButtonTypes int32 //enums:enum -trim-prefix Button

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
	// surrounding context sufficiently. It is equivalent to Material Design's
	// icon button, but it can also contain text and other things (and frequently does).
	ButtonAction

	// ButtonMenu is similar to [ButtonAction], but it is only
	// for buttons located in popup menus.
	ButtonMenu
)

func (bt *Button) OnInit() {
	bt.WidgetBase.OnInit()
	bt.HandleEvents()
	bt.SetStyles()
}

func (bt *Button) SetStyles() {
	bt.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Activatable, abilities.Focusable, abilities.Hoverable, abilities.DoubleClickable, abilities.TripleClickable)
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
		s.Gap.Zero()
		// s.CenterAll() // TODO(kai): fix button layout

		s.MaxBoxShadow = styles.BoxShadow1()
		switch bt.Type {
		case ButtonFilled:
			s.Background = colors.C(colors.Scheme.Primary.Base)
			s.Color = colors.C(colors.Scheme.Primary.On)
			s.Border.Offset.Set(units.Dp(2))
		case ButtonTonal:
			s.Background = colors.C(colors.Scheme.Secondary.Container)
			s.Color = colors.C(colors.Scheme.Secondary.OnContainer)
		case ButtonElevated:
			s.Background = colors.C(colors.Scheme.SurfaceContainerLow)
			s.Color = colors.C(colors.Scheme.Primary.Base)
			s.MaxBoxShadow = styles.BoxShadow2()
			s.BoxShadow = styles.BoxShadow1()
		case ButtonOutlined:
			s.Color = colors.C(colors.Scheme.Primary.Base)
			s.Border.Style.Set(styles.BorderSolid)
			s.Border.Width.Set(units.Dp(1))
			// if focused then default primary
			if !s.Is(states.Focused) {
				s.Border.Color.Set(colors.C(colors.Scheme.Outline))
			}
		case ButtonText:
			s.Color = colors.C(colors.Scheme.Primary.Base)
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
}

// SetKey sets the shortcut of the button from the given [keymap.Functions]
func (bt *Button) SetKey(kf keymap.Functions) *Button {
	bt.SetShortcut(kf.Chord())
	return bt
}

// NOTE: Button.SetText must be defined manually so that [views.FuncButton]
// can define its own SetText method that updates the tooltip

// SetText sets the [Button.Text]:
// Text is the text for the button.
// If it is blank, no text is shown.
func (bt *Button) SetText(v string) *Button {
	bt.Text = v
	return bt
}

func (bt *Button) Label() string {
	if bt.Text != "" {
		return bt.Text
	}
	return bt.Nm
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
	if indic := bt.ChildByName("indicator", 3); indic != nil {
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
		bt.Scene.Stage.ClosePopupAndBelow()
	})
}

func (bt *Button) WidgetTooltip(pos image.Point) (string, image.Point) {
	res := bt.Tooltip
	if bt.Shortcut != "" && (!TheApp.SystemPlatform().IsMobile() || TheApp.Platform() == system.Offscreen) {
		res = "[" + bt.Shortcut.Label() + "]"
		if bt.Tooltip != "" {
			res += " " + bt.Tooltip
		}
	}
	return res, bt.DefaultTooltipPos()
}

func (bt *Button) HandleEvents() {
	bt.HandleClickOnEnterSpace()
	bt.OnClick(func(e events.Event) {
		if bt.OpenMenu(e) {
			e.SetHandled()
		}
	})
	bt.OnDoubleClick(func(e events.Event) {
		bt.Send(events.Click, e)
	})
	bt.On(events.TripleClick, func(e events.Event) {
		bt.Send(events.Click, e)
	})
}

func (bt *Button) Config(c *Plan) {
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

	if bt.Icon.IsSet() {
		Configure(c, "icon", func(w *Icon) {
			w.Style(func(s *styles.Style) {
				s.Font.Size.Dp(18)
			})
		}, func(w *Icon) {
			w.SetIcon(bt.Icon)
		})
		if bt.Text != "" {
			Configure[*Space](c, "space")
		}
	}
	if bt.Text != "" {
		Configure(c, "text", func(w *Text) {
			w.Style(func(s *styles.Style) {
				s.SetNonSelectable()
				s.SetTextWrap(false)
				s.FillMargin = false
			})
		}, func(w *Text) {
			if bt.Type == ButtonMenu {
				w.SetType(TextBodyMedium)
			} else {
				w.SetType(TextLabelLarge)
			}
			w.SetText(bt.Text)
		})
	}

	if bt.Indicator.IsSet() {
		Configure(c, "indicator-stretch", func(w *Stretch) {
			w.Style(func(s *styles.Style) {
				s.Min.Set(units.Em(0.2))
				if bt.Type == ButtonMenu {
					s.Grow.Set(1, 0)
				} else {
					s.Grow.Set(0, 0)
				}
			})
		})
		Configure(c, "indicator", func(w *Icon) {
			w.Style(func(s *styles.Style) {
				s.Min.X.Dp(18)
				s.Min.Y.Dp(18)
				s.Margin.Zero()
				s.Padding.Zero()
			})
		}, func(w *Icon) {
			w.SetIcon(bt.Indicator)
		})
	}

	if bt.Type == ButtonMenu && (!TheApp.SystemPlatform().IsMobile() || TheApp.Platform() == system.Offscreen) {
		if !bt.Indicator.IsSet() && bt.Shortcut != "" {
			Configure[*Stretch](c, "shortcut-stretch")
			Configure(c, "shortcut", func(w *Text) {
				w.Style(func(s *styles.Style) {
					s.SetNonSelectable()
					s.SetTextWrap(false)
					s.Color = colors.C(colors.Scheme.OnSurfaceVariant)
				})
			}, func(w *Text) {
				if bt.Type == ButtonMenu {
					w.SetType(TextBodyMedium)
				} else {
					w.SetType(TextLabelLarge)
				}
				w.SetText(bt.Shortcut.Label())
			})
		} else if bt.Shortcut != "" {
			slog.Error("programmer error: core.Button: shortcut cannot be used on a sub-menu for", "button", bt)
		}
	}
}
