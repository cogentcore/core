// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"strings"
	"unicode"

	"cogentcore.org/core/enums"
	"cogentcore.org/core/events"
	"cogentcore.org/core/gti"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/states"
	"cogentcore.org/core/strcase"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
)

// Switches is a widget for containing a set of switches.
// It can optionally enforce mutual exclusivity (i.e., Radio Buttons).
// The buttons are all in the Parts of the widget and the Parts layout
// determines how they are displayed.
type Switches struct { //core:embedder
	Frame

	// the type of switches that will be made
	Type SwitchTypes

	// Items are the items displayed to the user.
	Items []SwitchesItem

	// whether to make the items mutually exclusive (checking one turns off all the others)
	Mutex bool
}

// SwitchesItem contains the properties of one item in a [Switches].
type SwitchesItem struct {

	// Label is the label displayed to the user for this item.
	Label string

	// Tooltip is the tooltip displayed to the user for this item.
	Tooltip string
}

func (sw *Switches) OnInit() {
	sw.Frame.OnInit()
	sw.SetStyles()
}

func (sw *Switches) SetStyles() {
	sw.Style(func(s *styles.Style) {
		s.Padding.Set(units.Dp(2))
		s.Margin.Set(units.Dp(2))
		s.Grow.Set(1, 0)
		if sw.Type == SwitchSegmentedButton {
			s.Gap.Zero()
		} else {
			s.Wrap = true
		}
	})
	sw.OnWidgetAdded(func(w Widget) {
		if w.Parent() != sw {
			return
		}
		sw.HandleSwitchEvents(w.(*Switch))
		w.Style(func(s *styles.Style) {
			if sw.Type != SwitchSegmentedButton {
				return
			}
			ip := w.IndexInParent()
			brf := styles.BorderRadiusFull.Top
			ps := &sw.Styles
			if ip == 0 {
				if ps.Direction == styles.Row {
					s.Border.Radius.Set(brf, units.Zero(), units.Zero(), brf)
				} else {
					s.Border.Radius.Set(brf, brf, units.Zero(), units.Zero())
				}
			} else if ip == sw.NumChildren()-1 {
				if ps.Direction == styles.Row {
					if !s.Is(states.Focused) {
						s.Border.Width.SetLeft(units.Zero())
						s.MaxBorder.Width = s.Border.Width
					}
					s.Border.Radius.Set(units.Zero(), brf, brf, units.Zero())
				} else {
					if !s.Is(states.Focused) {
						s.Border.Width.SetTop(units.Zero())
						s.MaxBorder.Width = s.Border.Width
					}
					s.Border.Radius.Set(units.Zero(), units.Zero(), brf, brf)
				}
			} else {
				if !s.Is(states.Focused) {
					if ps.Direction == styles.Row {
						s.Border.Width.SetLeft(units.Zero())
					} else {
						s.Border.Width.SetTop(units.Zero())
					}
					s.MaxBorder.Width = s.Border.Width
				}
				s.Border.Radius.Zero()
			}
		})
	})
}

// SelectItem activates a given item but does NOT send a change event.
// See SelectItemAction for event emitting version.
// returns error if index is out of range.
func (sw *Switches) SelectItem(idx int) error {
	if idx >= sw.NumChildren() || idx < 0 {
		return fmt.Errorf("gi.Switches: SelectItem, index out of range: %v", idx)
	}
	if sw.Mutex {
		sw.UnCheckAllBut(idx)
	}
	cs := sw.Child(idx).(*Switch)
	cs.SetChecked(true)
	sw.NeedsRender()
	return nil
}

// SelectItemAction activates a given item and emits a change event.
// This is mainly for Mutex use.
// returns error if index is out of range.
func (sw *Switches) SelectItemAction(idx int) error {
	err := sw.SelectItem(idx)
	if err != nil {
		return err
	}
	sw.SendChange()
	return nil
}

// SelectedItem returns the first selected (checked) switch. It is only
// useful when [Switches.Mutex] is true. If no switches are selected,
// it returns "".
func (sw *Switches) SelectedItem() string {
	for _, swi := range sw.Kids {
		sw := swi.(*Switch)
		if sw.IsChecked() {
			return sw.Text
		}
	}
	return ""
}

// UnCheckAll unchecks all switches
func (sw *Switches) UnCheckAll() {
	for _, cbi := range sw.Kids {
		cs := cbi.(*Switch)
		cs.SetChecked(false)
	}
	sw.NeedsRender()
}

// UnCheckAllBut unchecks all switches except given one
func (sw *Switches) UnCheckAllBut(idx int) {
	for i, cbi := range sw.Kids {
		if i == idx {
			continue
		}
		cs := cbi.(*Switch)
		cs.SetChecked(false)
		cs.Update()
	}
	sw.NeedsRender()
}

// SetStrings sets the [Switches.Items] from the given strings.
func (sw *Switches) SetStrings(ss []string) *Switches {
	sw.Items = make([]SwitchesItem, len(ss))
	for i, s := range ss {
		sw.Items[i] = SwitchesItem{Label: s}
	}
	return sw
}

// SetEnums sets the [Switches.Items] from the given enums.
func (sw *Switches) SetEnums(es []enums.Enum) *Switches {
	sw.Items = make([]SwitchesItem, len(es))
	for i, enum := range es {
		str := ""
		if bf, ok := enum.(enums.BitFlag); ok {
			str = bf.BitIndexString()
		} else {
			str = enum.String()
		}
		lbl := strcase.ToSentence(str)
		desc := enum.Desc()
		// If the documentation does not start with the transformed name, but it does
		// start with an uppercase letter, then we assume that the first word of the
		// documentation is the correct untransformed name. This fixes
		// https://github.com/cogentcore/core/issues/774 (also for Chooser).
		if !strings.HasPrefix(desc, str) && len(desc) > 0 && unicode.IsUpper(rune(desc[0])) {
			str, _, _ = strings.Cut(desc, " ")
		}
		tip := gti.FormatDoc(desc, str, lbl)
		sw.Items[i] = SwitchesItem{Label: lbl, Tooltip: tip}
	}
	return sw
}

// SetEnum sets the [Switches.Items] from the [enums.Enum.Values] of the given enum.
func (sw *Switches) SetEnum(enum enums.Enum) *Switches {
	return sw.SetEnums(enum.Values())
}

// UpdateFromBitFlags sets the checked state of the switches from the
// given bit flag enum value.
func (sw *Switches) UpdateFromBitFlag(bitflag enums.BitFlag) {
	els := bitflag.Values()
	mn := min(len(els), sw.NumChildren())
	for i := 0; i < mn; i++ {
		ev := els[i]
		swi := sw.Child(i)
		sw := swi.(*Switch)
		on := bitflag.HasFlag(ev.(enums.BitFlag))
		sw.SetChecked(on)
	}
	sw.NeedsRender()
}

// UpdateBitFlag sets the given bitflag value to the value specified
// by the checked state of the switches.
func (sw *Switches) UpdateBitFlag(bitflag enums.BitFlagSetter) {
	bitflag.SetInt64(0)

	els := bitflag.Values()
	mn := min(len(els), sw.NumChildren())
	for i := 0; i < mn; i++ {
		ev := els[i]
		swi := sw.Child(i)
		sw := swi.(*Switch)
		if sw.IsChecked() {
			bitflag.SetFlag(true, ev.(enums.BitFlag))
		}
	}
}

// HandleSwitchEvents handles the events for the given switch.
func (sw *Switches) HandleSwitchEvents(swi *Switch) {
	swi.OnChange(func(e events.Event) {
		if sw.Mutex && swi.IsChecked() {
			sw.UnCheckAllBut(swi.IndexInParent())
		}
		sw.SendChange(e)
	})
}

func (sw *Switches) Config() {
	config := ki.Config{}
	for _, item := range sw.Items {
		config.Add(SwitchType, item.Label)
	}
	if sw.ConfigChildren(config) || sw.NeedsRebuild() {
		for i, swi := range sw.Kids {
			s := swi.(*Switch)
			item := sw.Items[i]
			s.SetType(sw.Type).SetText(item.Label).SetTooltip(item.Tooltip)
		}
		sw.NeedsLayout()
	}
}
