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

// CheckBox toggles between a checked and unchecked state
type CheckBox struct {
	Button

	// [view: show-name] icon to use for the off, unchecked state of the icon -- plain Icon holds the On state -- can be set with icon-off property
	IconOff icons.Icon `xml:"icon-off" view:"show-name" desc:"icon to use for the off, unchecked state of the icon -- plain Icon holds the On state -- can be set with icon-off property"`
}

func (cb *CheckBox) CopyFieldsFrom(frm any) {
	fr := frm.(*CheckBox)
	cb.Button.CopyFieldsFrom(&fr.Button)
	cb.IconOff = fr.IconOff
}

func (cb *CheckBox) OnInit() {
	cb.ButtonHandlers()
	cb.CheckBoxStyles()
}

func (cb *CheckBox) CheckBoxStyles() {
	cb.AddStyles(func(s *styles.Style) {
		s.SetAbilities(true, states.Activatable, states.Focusable, states.Hoverable, states.Checkable)
		s.SetAbilities(cb.ShortcutTooltip() != "", states.LongHoverable)
		s.Cursor = cursors.Pointer
		s.Text.Align = styles.AlignLeft
		s.Color = colors.Scheme.OnBackground
		s.Margin.Set(units.Dp(1 * Prefs.DensityMul()))
		s.Padding.Set(units.Dp(1 * Prefs.DensityMul()))
		s.Border.Style.Set(styles.BorderNone)

		if cb.Parts != nil && cb.Parts.HasChildren() {
			ist := cb.Parts.ChildByName("stack", 0).(*Layout)
			if cb.StateIs(states.Checked) {
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

func (cb *CheckBox) OnChildAdded(child ki.Ki) {
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

// CheckBoxWidget interface

// todo: base widget will set checked state automatically, setstyle, updaterender

// // OnClicked calls the given function when the button is clicked,
// // which is the default / standard way of activating the button
// func (cb *CheckBox) OnClicked(fun func()) ButtonWidget {
// 	// cb.ButtonSig.Connect(cb.This(), func(recv, send ki.Ki, sig int64, data any) {
// 	// 	if sig == int64(ButtonToggled) {
// 	// 		fun()
// 	// 	}
// 	// })
// 	return cb.This().(ButtonWidget)
// }

// SetIcons sets the Icons (by name) for the On (checked) and Off (unchecked)
// states, and updates button
func (cb *CheckBox) SetIcons(icOn, icOff icons.Icon) {
	updt := cb.UpdateStart()
	cb.Icon = icOn
	cb.IconOff = icOff
	cb.This().(ButtonWidget).ConfigParts(cb.Sc)
	// todo: better config logic -- do layout
	cb.UpdateEnd(updt)
}

func (cb *CheckBox) ConfigWidget(sc *Scene) {
	cb.SetAbilities(true, states.Checkable)
	cb.This().(ButtonWidget).ConfigParts(sc)
}

func (cb *CheckBox) ConfigParts(sc *Scene) {
	parts := cb.NewParts(LayoutHoriz)
	cb.SetAbilities(true, states.Checkable)
	if !cb.Icon.IsValid() {
		cb.Icon = icons.CheckBox // fallback
	}
	if !cb.IconOff.IsValid() {
		cb.IconOff = icons.CheckBoxOutlineBlank
	}
	config := ki.Config{}
	icIdx := 0 // always there
	lbIdx := -1
	config.Add(LayoutType, "stack")
	if cb.Text != "" {
		config.Add(SpaceType, "space")
		lbIdx = len(config)
		config.Add(LabelType, "label")
	}
	mods, updt := parts.ConfigChildren(config)
	ist := parts.Child(icIdx).(*Layout)
	if mods || cb.NeedsRebuild() {
		ist.Lay = LayoutStacked
		ist.SetNChildren(2, IconType, "icon") // covered by above config update
		icon := ist.Child(0).(*Icon)
		icon.SetIcon(cb.Icon)
		icoff := ist.Child(1).(*Icon)
		icoff.SetIcon(cb.IconOff)
	}
	if cb.StateIs(states.Checked) {
		ist.StackTop = 0
	} else {
		ist.StackTop = 1
	}
	if lbIdx >= 0 {
		lbl := parts.Child(lbIdx).(*Label)
		if lbl.Text != cb.Text {
			lbl.SetText(cb.Text)
		}
	}
	if mods {
		parts.UpdateEnd(updt)
		cb.SetNeedsLayout(sc, updt)
	}
}
