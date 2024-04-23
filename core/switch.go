// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"cogentcore.org/core/abilities"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/states"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/units"
)

// Switch is a widget that can toggle between an on and off state.
// It can be displayed as a switch, checkbox, or radio button.
type Switch struct {
	WidgetBase

	// Type is the styling type of switch.
	Type SwitchTypes `set:"-"`

	// Text is the text for the switch.
	Text string

	// IconOn is the icon to use for the on, checked state of the switch.
	IconOn icons.Icon `view:"show-name"`

	// Iconoff is the icon to use for the off, unchecked state of the switch.
	IconOff icons.Icon `view:"show-name"`

	// IconIndeterminate is the icon to use for the indeterminate (unknown) state.
	IconIndeterminate icons.Icon `view:"show-name"`
}

// SwitchTypes contains the different types of [Switch]es
type SwitchTypes int32 //enums:enum -trim-prefix Switch

const (
	// SwitchSwitch indicates to display a switch as a switch (toggle slider).
	SwitchSwitch SwitchTypes = iota

	// SwitchChip indicates to display a switch as chip (like Material Design's filter chip),
	// which is typically only used in the context of [Switches].
	SwitchChip

	// SwitchCheckbox indicates to display a switch as a checkbox.
	SwitchCheckbox

	// SwitchRadioButton indicates to display a switch as a radio button.
	SwitchRadioButton

	// SwitchSegmentedButton indicates to display a segmented button, which is typically only used in
	// the context of [Switches].
	SwitchSegmentedButton
)

func (sw *Switch) OnInit() {
	sw.WidgetBase.OnInit()
	sw.HandleEvents()
	sw.SetStyles()
}

// IsChecked tests if this switch is checked
func (sw *Switch) IsChecked() bool {
	return sw.StateIs(states.Checked)
}

// SetChecked sets the checked state and updates the icon accordingly
func (sw *Switch) SetChecked(on bool) *Switch {
	sw.SetState(on, states.Checked)
	sw.SetState(false, states.Indeterminate)
	sw.SetIconFromState()
	return sw
}

// SetIconFromState updates icon state based on checked status
func (sw *Switch) SetIconFromState() {
	if sw.Parts == nil {
		return
	}
	ist := sw.Parts.ChildByName("stack", 0)
	if ist == nil {
		return
	}
	st := ist.(*Layout)
	switch {
	case sw.StateIs(states.Indeterminate):
		st.StackTop = 2
	case sw.IsChecked():
		st.StackTop = 0
	default:
		if sw.Type == SwitchChip {
			// chips render no icon when off
			st.StackTop = -1
			return
		}
		st.StackTop = 1
	}
}

func (sw *Switch) HandleEvents() {
	sw.HandleSelectToggle()
	sw.HandleClickOnEnterSpace()
	sw.OnFinal(events.Click, func(e events.Event) {
		sw.SetChecked(sw.IsChecked())
		if sw.Type == SwitchChip {
			sw.NeedsLayout()
		} else {
			sw.NeedsRender()
		}
		sw.SendChange(e)
	})
}

func (sw *Switch) SetStyles() {
	sw.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Activatable, abilities.Focusable, abilities.Hoverable, abilities.Checkable)
		if !sw.IsReadOnly() {
			s.Cursor = cursors.Pointer
		}
		s.Text.Align = styles.Start
		s.Text.AlignV = styles.Center
		s.Padding.Set(units.Dp(4))
		s.Border.Radius = styles.BorderRadiusSmall

		if sw.Type == SwitchChip {
			if s.Is(states.Checked) {
				s.Background = colors.C(colors.Scheme.SurfaceVariant)
				s.Color = colors.C(colors.Scheme.OnSurfaceVariant)
			} else if !s.Is(states.Focused) {
				s.Border.Color.Set(colors.C(colors.Scheme.Outline))
				s.Border.Width.Set(units.Dp(1))
			}
		}
		if sw.Type == SwitchSegmentedButton {
			if !s.Is(states.Focused) {
				s.Border.Color.Set(colors.C(colors.Scheme.Outline))
				s.Border.Width.Set(units.Dp(1))
			}
			if s.Is(states.Checked) {
				s.Background = colors.C(colors.Scheme.SurfaceVariant)
				s.Color = colors.C(colors.Scheme.OnSurfaceVariant)
			}
		}

		if s.Is(states.Selected) {
			s.Background = colors.C(colors.Scheme.Select.Container)
		}
	})
	sw.OnWidgetAdded(func(w Widget) {
		switch w.PathFrom(sw) {
		case "parts":
			w.Style(func(s *styles.Style) {
				s.Gap.Zero()
				s.Align.Content = styles.Center
				s.Align.Items = styles.Center
				s.Text.AlignV = styles.Center
			})
		case "parts/stack":
			w.Style(func(s *styles.Style) {
				s.Display = styles.Stacked
				s.Grow.Set(0, 0)
				s.Gap.Zero()
			})
		case "parts/stack/icon0": // on
			w.Style(func(s *styles.Style) {
				if sw.Type == SwitchChip {
					s.Color = colors.C(colors.Scheme.OnSurfaceVariant)
				} else {
					s.Color = colors.C(colors.Scheme.Primary.Base)
				}
				// switches need to be bigger
				if sw.Type == SwitchSwitch {
					s.Min.X.Em(2)
					s.Min.Y.Em(1.5)
				} else {
					s.Min.X.Em(1.5)
					s.Min.Y.Em(1.5)
				}
			})
		case "parts/stack/icon1": // off
			w.Style(func(s *styles.Style) {
				switch sw.Type {
				case SwitchSwitch:
					// switches need to be bigger
					s.Min.X.Em(2)
					s.Min.Y.Em(1.5)
				case SwitchChip:
					// chips render no icon when off
					s.Min.X.Zero()
					s.Min.Y.Zero()
				default:
					s.Min.X.Em(1.5)
					s.Min.Y.Em(1.5)
				}
			})
		case "parts/stack/icon2": // indeterminate
			w.Style(func(s *styles.Style) {
				switch sw.Type {
				case SwitchSwitch:
					// switches need to be bigger
					s.Min.X.Em(2)
					s.Min.Y.Em(1.5)
				case SwitchChip:
					// chips render no icon when off
					s.Min.X.Zero()
					s.Min.Y.Zero()
				default:
					s.Min.X.Em(1.5)
					s.Min.Y.Em(1.5)
				}
			})
		case "parts/space":
			w.Style(func(s *styles.Style) {
				s.Min.X.Ch(0.1)
			})
		case "parts/text":
			w.Style(func(s *styles.Style) {
				s.SetNonSelectable()
				s.SetTextWrap(false)
				s.Margin.Zero()
				s.Padding.Zero()
				s.Text.AlignV = styles.Center
				s.FillMargin = false
			})
		}
	})
}

// SetType sets the styling type of the switch
func (sw *Switch) SetType(typ SwitchTypes) *Switch {
	sw.Type = typ
	sw.IconIndeterminate = icons.Blank
	switch sw.Type {
	case SwitchSwitch:
		// TODO: material has more advanced switches with a checkmark
		// if they are turned on; we could implement that at some point
		sw.IconOn = icons.ToggleOn.Fill()
		sw.IconOff = icons.ToggleOff
		sw.IconIndeterminate = icons.ToggleMid
	case SwitchChip, SwitchSegmentedButton:
		sw.IconOn = icons.Check
		sw.IconOff = icons.None
		sw.IconIndeterminate = icons.None
	case SwitchCheckbox:
		sw.IconOn = icons.CheckBox.Fill()
		sw.IconOff = icons.CheckBoxOutlineBlank
		sw.IconIndeterminate = icons.IndeterminateCheckBox
	case SwitchRadioButton:
		sw.IconOn = icons.RadioButtonChecked
		sw.IconOff = icons.RadioButtonUnchecked
		sw.IconIndeterminate = icons.RadioButtonPartial
	}
	sw.NeedsLayout()
	return sw
}

// SetIcons sets the icons for the on (checked), off (unchecked)
// and indeterminate (unknown) states.  See [SetIconsUpdate] for
// a version that updates the icon rendering
func (sw *Switch) SetIcons(on, off, unk icons.Icon) *Switch {
	sw.IconOn = on
	sw.IconOff = off
	sw.IconIndeterminate = unk
	return sw
}

// ClearIcons sets all of the switch icons to [icons.None]
func (sw *Switch) ClearIcons() *Switch {
	sw.IconOn = icons.None
	sw.IconOff = icons.None
	sw.IconIndeterminate = icons.None
	return sw
}

func (sw *Switch) Config() {
	config := tree.Config{}
	if sw.IconOn == "" {
		sw.IconOn = icons.ToggleOn.Fill() // fallback
	}
	if sw.IconOff == "" {
		sw.IconOff = icons.ToggleOff // fallback
	}
	ici := 0 // always there
	lbi := -1
	config.Add(LayoutType, "stack")
	if sw.Text != "" {
		config.Add(SpaceType, "space")
		lbi = len(config)
		config.Add(TextType, "text")
	}
	sw.ConfigParts(config, func() {
		ist := sw.Parts.Child(ici).(*Layout)
		ist.SetNChildren(3, IconType, "icon")
		icon := ist.Child(0).(*Icon)
		icon.SetIcon(sw.IconOn)
		icoff := ist.Child(1).(*Icon)
		icoff.SetIcon(sw.IconOff)
		icunk := ist.Child(2).(*Icon)
		icunk.SetIcon(sw.IconIndeterminate)
		sw.SetIconFromState()
		if lbi >= 0 {
			text := sw.Parts.Child(lbi).(*Text)
			if text.Text != sw.Text {
				text.SetText(sw.Text)
			}
		}
	})
}

func (sw *Switch) Render() {
	sw.SetIconFromState() // make sure we're always up-to-date on render
	if sw.Parts != nil {
		ist := sw.Parts.ChildByName("stack", 0)
		if ist != nil {
			ist.(*Layout).UpdateStackedVisibility()
		}
	}
	sw.WidgetBase.Render()
}
