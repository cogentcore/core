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
	Button

	// [view: show-name] icon to use for the off, unchecked state of the icon -- plain Icon holds the On state -- can be set with icon-off property
	IconOff icons.Icon `xml:"icon-off" view:"show-name" desc:"icon to use for the off, unchecked state of the icon -- plain Icon holds the On state -- can be set with icon-off property"`
}

func (sw *Switch) CopyFieldsFrom(frm any) {
	fr := frm.(*Switch)
	sw.Button.CopyFieldsFrom(&fr.Button)
	sw.IconOff = fr.IconOff
}

func (sw *Switch) OnInit() {
	sw.ButtonHandlers()
	sw.CheckBoxStyles()
}

func (sw *Switch) CheckBoxStyles() {
	sw.AddStyles(func(s *styles.Style) {
		s.SetAbilities(true, states.Activatable, states.Focusable, states.Hoverable, states.Checkable)
		s.SetAbilities(sw.ShortcutTooltip() != "", states.LongHoverable)
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

// SetIcons sets the Icons (by name) for the On (checked) and Off (unchecked)
// states, and updates button
func (sw *Switch) SetIcons(icOn, icOff icons.Icon) {
	updt := sw.UpdateStart()
	sw.Icon = icOn
	sw.IconOff = icOff
	sw.ConfigParts(sw.Sc)
	// todo: better config logic -- do layout
	sw.UpdateEnd(updt)
}

func (sw *Switch) ConfigWidget(sc *Scene) {
	sw.SetAbilities(true, states.Checkable)
	sw.ConfigParts(sc)
}

func (sw *Switch) ConfigParts(sc *Scene) {
	parts := sw.NewParts(LayoutHoriz)
	sw.SetAbilities(true, states.Checkable)
	if !sw.Icon.IsValid() {
		sw.Icon = icons.CheckBox // fallback
	}
	if !sw.IconOff.IsValid() {
		sw.IconOff = icons.CheckBoxOutlineBlank
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
		icon.SetIcon(sw.Icon)
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
