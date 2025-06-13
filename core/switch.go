// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"reflect"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/text"
	"cogentcore.org/core/tree"
)

// Switch is a widget that can toggle between an on and off state.
// It can be displayed as a switch, chip, checkbox, radio button,
// or segmented button.
type Switch struct {
	Frame

	// Type is the styling type of switch.
	// It must be set using [Switch.SetType].
	Type SwitchTypes `set:"-"`

	// Text is the optional text of the switch.
	Text string

	// IconOn is the icon to use for the on, checked state of the switch.
	IconOn icons.Icon

	// Iconoff is the icon to use for the off, unchecked state of the switch.
	IconOff icons.Icon

	// IconIndeterminate is the icon to use for the indeterminate (unknown) state.
	IconIndeterminate icons.Icon
}

// SwitchTypes contains the different types of [Switch]es.
type SwitchTypes int32 //enums:enum -trim-prefix Switch -transform kebab

const (
	// SwitchSwitch indicates to display a switch as a switch (toggle slider).
	SwitchSwitch SwitchTypes = iota

	// SwitchChip indicates to display a switch as chip (like Material Design's
	// filter chip), which is typically only used in the context of [Switches].
	SwitchChip

	// SwitchCheckbox indicates to display a switch as a checkbox.
	SwitchCheckbox

	// SwitchRadioButton indicates to display a switch as a radio button.
	SwitchRadioButton

	// SwitchSegmentedButton indicates to display a segmented button, which is
	// typically only used in the context of [Switches].
	SwitchSegmentedButton
)

func (sw *Switch) WidgetValue() any { return sw.IsChecked() }

func (sw *Switch) SetWidgetValue(value any) error {
	b, err := reflectx.ToBool(value)
	if err != nil {
		return err
	}
	sw.SetChecked(b)
	return nil
}

func (sw *Switch) OnBind(value any, tags reflect.StructTag) {
	if d, ok := tags.Lookup("display"); ok {
		errors.Log(sw.Type.SetString(d))
		sw.SetType(sw.Type)
	}
}

func (sw *Switch) Init() {
	sw.Frame.Init()
	sw.Styler(func(s *styles.Style) {
		if !sw.IsReadOnly() {
			s.SetAbilities(true, abilities.Activatable, abilities.Focusable, abilities.Hoverable, abilities.Checkable)
			s.Cursor = cursors.Pointer
		}
		s.Text.Align = text.Start
		s.Text.AlignV = text.Center
		s.Padding.SetVertical(units.Dp(4))
		s.Padding.SetHorizontal(units.Dp(ConstantSpacing(4))) // needed for layout issues
		s.Border.Radius = styles.BorderRadiusSmall
		s.Gap.Zero()
		s.CenterAll()
		s.IconSize.Set(units.Em(1.5))

		switch sw.Type {
		case SwitchChip:
			if s.Is(states.Checked) {
				s.Background = colors.Scheme.SurfaceVariant
				s.Color = colors.Scheme.OnSurfaceVariant
			} else if !s.Is(states.Focused) {
				s.Border.Width.Set(units.Dp(1))
			}
		case SwitchSegmentedButton:
			if !s.Is(states.Focused) {
				s.Border.Width.Set(units.Dp(1))
			}
			if s.Is(states.Checked) {
				s.Background = colors.Scheme.SurfaceVariant
				s.Color = colors.Scheme.OnSurfaceVariant
			}
		case SwitchSwitch:
			s.IconSize.Set(units.Em(2), units.Em(1.5)) // switches need to be bigger
		}

		if s.Is(states.Selected) {
			s.Background = colors.Scheme.Select.Container
		}
	})

	sw.SendClickOnEnter()
	sw.OnFinal(events.Click, func(e events.Event) {
		if sw.IsReadOnly() {
			return
		}
		sw.SetChecked(sw.IsChecked())
		if sw.Type == SwitchChip || sw.Type == SwitchSegmentedButton {
			sw.updateStackTop() // must update here
			sw.NeedsLayout()
		} else {
			sw.NeedsRender()
		}
		sw.SendChange(e)
	})

	sw.Maker(func(p *tree.Plan) {
		if sw.IconOn == "" {
			sw.IconOn = icons.ToggleOnFill // fallback
		}
		if sw.IconOff == "" {
			sw.IconOff = icons.ToggleOff // fallback
		}

		tree.AddAt(p, "stack", func(w *Frame) {
			w.Styler(func(s *styles.Style) {
				s.Display = styles.Stacked
				s.Gap.Zero()
			})
			w.Updater(func() {
				sw.updateStackTop() // need to update here
			})
			w.Maker(func(p *tree.Plan) {
				tree.AddAt(p, "icon-on", func(w *Icon) {
					w.Styler(func(s *styles.Style) {
						if sw.Type == SwitchChip {
							s.Color = colors.Scheme.OnSurfaceVariant
						} else {
							s.Color = colors.Scheme.Primary.Base
						}
					})
					w.Updater(func() {
						w.SetIcon(sw.IconOn)
					})
				})
				// same styles for off and indeterminate
				iconStyle := func(s *styles.Style) {
					if sw.IconOff == icons.None && sw.IconIndeterminate == icons.None {
						s.IconSize.Zero() // nothing to render
					}
				}
				tree.AddAt(p, "icon-off", func(w *Icon) {
					w.Styler(iconStyle)
					w.Updater(func() {
						w.SetIcon(sw.IconOff)
					})
				})
				tree.AddAt(p, "icon-indeterminate", func(w *Icon) {
					w.Styler(iconStyle)
					w.Updater(func() {
						w.SetIcon(sw.IconIndeterminate)
					})
				})
			})
		})

		if sw.Text != "" {
			tree.AddAt(p, "space", func(w *Space) {
				w.Styler(func(s *styles.Style) {
					s.Min.X.Ch(0.1)
				})
			})
			tree.AddAt(p, "text", func(w *Text) {
				w.Styler(func(s *styles.Style) {
					s.SetNonSelectable()
					s.SetTextWrap(false)
					s.FillMargin = false
				})
				w.Updater(func() {
					w.SetText(sw.Text)
				})
			})
		}
	})
}

// IsChecked returns whether the switch is checked.
func (sw *Switch) IsChecked() bool {
	return sw.StateIs(states.Checked)
}

// SetChecked sets whether the switch it checked.
func (sw *Switch) SetChecked(on bool) *Switch {
	sw.SetState(on, states.Checked)
	sw.SetState(false, states.Indeterminate)
	return sw
}

// updateStackTop updates the [Frame.StackTop] of the stack in the switch
// according to the current icon. It is called automatically to keep the
// switch up-to-date.
func (sw *Switch) updateStackTop() {
	st, ok := sw.ChildByName("stack", 0).(*Frame)
	if !ok {
		return
	}
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

// SetType sets the styling type of the switch.
func (sw *Switch) SetType(typ SwitchTypes) *Switch {
	sw.Type = typ
	sw.IconIndeterminate = icons.Blank
	switch sw.Type {
	case SwitchSwitch:
		// TODO: material has more advanced switches with a checkmark
		// if they are turned on; we could implement that at some point
		sw.IconOn = icons.ToggleOnFill
		sw.IconOff = icons.ToggleOff
		sw.IconIndeterminate = icons.ToggleMid
	case SwitchChip, SwitchSegmentedButton:
		sw.IconOn = icons.Check
		sw.IconOff = icons.None
		sw.IconIndeterminate = icons.None
	case SwitchCheckbox:
		sw.IconOn = icons.CheckBoxFill
		sw.IconOff = icons.CheckBoxOutlineBlank
		sw.IconIndeterminate = icons.IndeterminateCheckBox
	case SwitchRadioButton:
		sw.IconOn = icons.RadioButtonChecked
		sw.IconOff = icons.RadioButtonUnchecked
		sw.IconIndeterminate = icons.RadioButtonPartial
	}
	return sw
}

func (sw *Switch) Render() {
	sw.updateStackTop() // important: make sure we're always up-to-date on render
	st, ok := sw.ChildByName("stack", 0).(*Frame)
	if ok {
		st.UpdateStackedVisibility()
	}
	sw.WidgetBase.Render()
}
