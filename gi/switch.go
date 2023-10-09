// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"log/slog"

	"goki.dev/colors"
	"goki.dev/cursors"
	"goki.dev/girl/abilities"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/icons"
	"goki.dev/ki/v2"
)

// Switch is a widget that can toggle between an on and off state.
// It can be displayed as a switch, checkbox, or radio button.
type Switch struct {
	WidgetBase

	// the type of switch that this is
	Type SwitchTypes `desc:"the type of switch that this is"`

	// the label text for the switch
	Text string `desc:"the label text for the switch"`

	// [view: show-name] icon to use for the on, checked state of the switch
	IconOn icons.Icon `view:"show-name" desc:"icon to use for the on, checked state of the switch"`

	// [view: show-name] icon to use for the off, unchecked state of the switch
	IconOff icons.Icon `view:"show-name" desc:"icon to use for the off, unchecked state of the switch"`
}

// SwitchTypes contains the different types of [Switch]es
type SwitchTypes int32 //enums:enum -trimprefix Switch

const (
	// SwitchSwitch indicates to display a switch as a switch (toggle slider)
	SwitchSwitch SwitchTypes = iota
	// SwitchCheckbox indicates to display a switch as a checkbox
	SwitchCheckbox
	// SwitchRadioButton indicates to display a switch as a radio button
	SwitchRadioButton
)

func (sw *Switch) CopyFieldsFrom(frm any) {
	fr := frm.(*Switch)
	sw.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
	sw.Type = fr.Type
	sw.Text = fr.Text
	sw.IconOn = fr.IconOn
	sw.IconOff = fr.IconOff
}

func (sw *Switch) OnInit() {
	sw.SwitchHandlers()
	sw.SwitchStyles()
}

func (sw *Switch) SwitchHandlers() {
	sw.WidgetHandlers()
	sw.OnClick(func(e events.Event) {
		if sw.StateIs(states.Disabled) {
			return
		}
		e.SetHandled()
		sw.SetState(!sw.StateIs(states.Checked), states.Checked)
		if sw.Parts != nil && sw.Parts.HasChildren() {
			ist := sw.Parts.ChildByName("stack", 0).(*Layout)
			if sw.StateIs(states.Checked) {
				ist.StackTop = 0
			} else {
				ist.StackTop = 1
			}
		}
		sw.Send(events.Change, e)
	})
	sw.On(events.KeyChord, func(e events.Event) {
		if sw.StateIs(states.Disabled) {
			return
		}
		if KeyEventTrace {
			slog.Info("Switch KeyChordEvent", "switch", sw)
		}
		kf := KeyFun(e.KeyChord())
		if kf == KeyFunEnter || e.KeyRune() == ' ' {
			// if !(kt.Rune == ' ' && bbb.Sc.Type == ScCompleter) {
			e.SetHandled()
			sw.Send(events.Click, e)
			// }
		}
	})
}

func (sw *Switch) SwitchStyles() {
	sw.AddStyles(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Activatable, abilities.Focusable, abilities.Hoverable, abilities.Checkable)
		s.Cursor = cursors.Pointer
		s.Text.Align = styles.AlignLeft
		s.Color = colors.Scheme.OnBackground
		s.Margin.Set(units.Dp(1 * Prefs.DensityMul()))
		s.Padding.Set(units.Dp(1 * Prefs.DensityMul()))
		s.Border.Radius = styles.BorderRadiusExtraSmall

		if s.Is(states.Selected) {
			s.BackgroundColor.SetSolid(colors.Scheme.Select.Container)
		}
		if s.Is(states.Disabled) {
			s.Color = colors.Scheme.SurfaceContainer
		}
	})
}

func (sw *Switch) OnChildAdded(child ki.Ki) {
	w, _ := AsWidget(child)
	switch w.Name() {
	case "icon0": // on
		w.AddStyles(func(s *styles.Style) {
			s.Color = colors.Scheme.Primary.Base
			// switches need to be bigger
			if sw.Type == SwitchSwitch {
				s.Width.SetEm(2.5)
				s.Height.SetEm(2.5)
			} else {
				s.Width.SetEm(1.5)
				s.Height.SetEm(1.5)
			}
		})
	case "icon1": // off
		w.AddStyles(func(s *styles.Style) {
			// switches need to be bigger
			if sw.Type == SwitchSwitch {
				s.Width.SetEm(2.5)
				s.Height.SetEm(2.5)
			} else {
				s.Width.SetEm(1.5)
				s.Height.SetEm(1.5)
			}
		})
	case "space":
		w.AddStyles(func(s *styles.Style) {
			s.Width.SetCh(0.1)
		})
	case "label":
		w.AddStyles(func(s *styles.Style) {
			s.SetAbilities(false, abilities.Selectable, abilities.DoubleClickable)
			s.Cursor = cursors.None
			s.Margin.Set()
			s.Padding.Set()
			s.AlignV = styles.AlignMiddle
		})
	}
}

// SetType sets the styling type of the switch
func (sw *Switch) SetType(typ SwitchTypes) *Switch {
	updt := sw.UpdateStart()
	sw.Type = typ
	switch sw.Type {
	case SwitchSwitch:
		// TODO: material has more advanced switches with a checkmark
		// if they are turned on; we could implement that at some point
		sw.IconOn = icons.ToggleOn.Fill()
		sw.IconOff = icons.ToggleOff
	case SwitchCheckbox:
		sw.IconOn = icons.CheckBox.Fill()
		sw.IconOff = icons.CheckBoxOutlineBlank
	case SwitchRadioButton:
		sw.IconOn = icons.RadioButtonChecked
		sw.IconOff = icons.RadioButtonUnchecked
	}
	sw.UpdateEndLayout(updt)
	return sw
}

// LabelWidget returns the label widget if present
func (sw *Switch) LabelWidget() *Label {
	lbi := sw.Parts.ChildByName("label")
	if lbi == nil {
		return nil
	}
	return lbi.(*Label)
}

// SetText sets the text and updates the switch
func (sw *Switch) SetText(txt string) *Switch {
	updt := sw.UpdateStart()
	sw.Text = txt
	sw.UpdateEndLayout(updt)
	return sw
}

// SetIcons sets the icons for the on (checked)
// and off (unchecked) states, and updates the switch
func (sw *Switch) SetIcons(on, off icons.Icon) *Switch {
	updt := sw.UpdateStart()
	sw.IconOn = on
	sw.IconOff = off
	sw.UpdateEndLayout(updt)
	return sw
}

func (sw *Switch) ConfigWidget(sc *Scene) {
	sw.ConfigParts(sc)
}

func (sw *Switch) ConfigParts(sc *Scene) {
	parts := sw.NewParts(LayoutHoriz)
	if !sw.IconOn.IsValid() {
		sw.IconOn = icons.ToggleOn.Fill() // fallback
	}
	if !sw.IconOff.IsValid() {
		sw.IconOff = icons.ToggleOff // fallback
	}
	config := ki.Config{}
	icIdx := 0 // always there
	lbIdx := -1
	config.Add(LayoutType, "stack")
	if sw.Text != "" {
		config.Add(SpaceType, "space")
		lbIdx = len(config)
		config.Add(LabelType, "label")
	}
	mods, updt := parts.ConfigChildren(config)
	ist := parts.Child(icIdx).(*Layout)
	if mods || sw.NeedsRebuild() {
		ist.Lay = LayoutStacked
		ist.SetNChildren(2, IconType, "icon") // covered by above config update
		icon := ist.Child(0).(*Icon)
		icon.SetIcon(sw.IconOn)
		icoff := ist.Child(1).(*Icon)
		icoff.SetIcon(sw.IconOff)
	}
	if sw.StateIs(states.Checked) {
		ist.StackTop = 0
	} else {
		ist.StackTop = 1
	}
	if lbIdx >= 0 {
		lbl := parts.Child(lbIdx).(*Label)
		if lbl.Text != sw.Text {
			lbl.SetText(sw.Text)
		}
	}
	if mods {
		parts.UpdateEnd(updt)
		sw.SetNeedsLayout(sc, updt)
	}
}

func (sw *Switch) RenderSwitch(sc *Scene) {
	rs, _, st := sw.RenderLock(sc)
	sw.RenderStdBox(sc, st)
	sw.RenderUnlock(rs)
}

func (sw *Switch) Render(sc *Scene) {
	if sw.PushBounds(sc) {
		sw.RenderSwitch(sc)
		sw.RenderParts(sc)
		sw.RenderChildren(sc)
		sw.PopBounds(sc)
	}
}
