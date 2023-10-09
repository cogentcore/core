// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"goki.dev/colors"
	"goki.dev/cursors"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
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
	sw.WidgetHandlers()
	sw.SwitchStyles()
}

func (sw *Switch) SwitchStyles() {
	sw.AddStyles(func(s *styles.Style) {
		s.SetAbilities(true, states.Activatable, states.Focusable, states.Hoverable, states.Checkable)
		s.Cursor = cursors.Pointer
		s.Text.Align = styles.AlignLeft
		s.Color = colors.Scheme.OnBackground
		s.Margin.Set(units.Dp(1 * Prefs.DensityMul()))
		s.Padding.Set(units.Dp(1 * Prefs.DensityMul()))
		s.Border.Style.Set(styles.BorderNone)

		if sw.Parts != nil && sw.Parts.HasChildren() {
			ist := sw.Parts.ChildByName("stack", 0).(*Layout)
			if sw.StateIs(states.Checked) {
				ist.StackTop = 0
			} else {
				ist.StackTop = 1
			}
		}
		if s.Is(states.Checked) {
			s.Color = colors.Scheme.Primary.Base
		}
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
			s.Width.SetEm(1.5)
			s.Height.SetEm(1.5)
		})
	case "icon1": // off
		w.AddStyles(func(s *styles.Style) {
			s.Width.SetEm(1.5)
			s.Height.SetEm(1.5)
		})
	case "space":
		w.AddStyles(func(s *styles.Style) {
			s.Width.SetCh(0.1)
		})
	case "label":
		w.AddStyles(func(s *styles.Style) {
			s.SetAbilities(false, states.Selectable, states.DoubleClickable)
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

// SetText sets the text and updates the switch.
// Use this for optimized auto-updating based on nature of changes made.
// Otherwise, can set Text directly followed by ReConfig()
func (sw *Switch) SetText(txt string) *Switch {
	if sw.Text == txt {
		return sw
	}
	updt := sw.UpdateStart()
	recfg := sw.Parts == nil || (sw.Text == "" && txt != "") || (sw.Text != "" && txt == "")
	sw.Text = txt
	if recfg {
		sw.ConfigParts(sw.Sc)
	} else {
		lbl := sw.LabelWidget()
		if lbl != nil {
			lbl.SetText(sw.Text)
		}
	}
	sw.UpdateEndLayout(updt) // todo: could optimize to not re-layout every time but..
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
		sw.IconOn = icons.CheckBox.Fill() // fallback
	}
	if !sw.IconOff.IsValid() {
		sw.IconOff = icons.CheckBoxOutlineBlank // fallback
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
