// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"slices"

	"goki.dev/enums"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
)

// Switches is a widget for containing a set of switches.
// It can optionally enforce mutual exclusivity (i.e., Radio Buttons).
// The buttons are all in the Parts of the widget and the Parts layout
// determines how they are displayed.
type Switches struct { //goki:embedder
	Frame

	// the type of switches that will be made
	Type SwitchTypes

	// the list of items (switch labels)
	Items []string

	// an optional list of tooltips displayed on hover for checkbox items; the indices for tooltips correspond to those for items
	Tooltips []string

	// whether to make the items mutually exclusive (checking one turns off all the others)
	Mutex bool
}

func (sw *Switches) CopyFieldsFrom(frm any) {
	fr := frm.(*Switches)
	sw.Frame.CopyFieldsFrom(&fr.Frame)
	sw.Items = slices.Clone(fr.Items)
}

func (sw *Switches) OnInit() {
	sw.HandleWidgetEvents()
	sw.SwitchesStyles()
}

func (sw *Switches) SwitchesStyles() {
	sw.Style(func(s *styles.Style) {
		s.Padding.Set(units.Dp(2))
		s.Margin.Set(units.Dp(2))
		if sw.Type == SwitchSegmentedButton {
			s.Gap.Zero()
		} else {
			s.Grow.Set(1, 0)
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
			ip, _ := w.IndexInParent()
			brf := styles.BorderRadiusFull.Top
			if ip == 0 {
				if s.MainAxis == mat32.X {
					s.Border.Radius.Set(brf, units.Zero(), units.Zero(), brf)
				} else {
					s.Border.Radius.Set(brf, brf, units.Zero(), units.Zero())
				}
			} else if ip == sw.NumChildren()-1 {
				if s.MainAxis == mat32.X {
					s.Border.Width.SetLeft(units.Zero())
					s.Border.Radius.Set(units.Zero(), brf, brf, units.Zero())
				} else {
					s.Border.Width.SetTop(units.Zero())
					s.Border.Radius.Set(units.Zero(), units.Zero(), brf, brf)
				}
			} else {
				if s.MainAxis == mat32.X {
					s.Border.Width.SetLeft(units.Zero())
				} else {
					s.Border.Width.SetTop(units.Zero())
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
	updt := sw.UpdateStart()
	if sw.Mutex {
		sw.UnCheckAllBut(idx)
	}
	cs := sw.Child(idx).(*Switch)
	cs.SetChecked(true)
	sw.UpdateEndRender(updt)
	return nil
}

// SelectItemAction activates a given item and emits a change event.
// This is mainly for Mutex use.
// returns error if index is out of range.
func (sw *Switches) SelectItemAction(idx int) error {
	updt := sw.UpdateStart()
	defer sw.UpdateEnd(updt)

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
		if sw.StateIs(states.Checked) {
			return sw.Text
		}
	}
	return ""
}

// UnCheckAll unchecks all switches
func (sw *Switches) UnCheckAll() {
	updt := sw.UpdateStart()
	for _, cbi := range *sw.Children() {
		cs := cbi.(*Switch)
		cs.SetChecked(false)
	}
	sw.UpdateEndRender(updt)
}

// UnCheckAllBut unchecks all switches except given one
func (sw *Switches) UnCheckAllBut(idx int) {
	updt := sw.UpdateStart()
	for i, cbi := range *sw.Children() {
		if i == idx {
			continue
		}
		cs := cbi.(*Switch)
		cs.SetChecked(false)
		cs.Update()
	}
	sw.UpdateEndRender(updt)
}

// SetStrings sets the Items list from a list of string values -- if
// setFirst then set current item to the first item in the list, and maxLen if
// > 0 auto-sets the width of the button to the contents, with the given upper
// limit
func (sw *Switches) SetStrings(el []string) *Switches {
	sz := len(el)
	if sz == 0 {
		return sw
	}
	sw.Items = make([]string, sz)
	copy(sw.Items, el)
	return sw
}

// SetEnums sets the Items list from a list of enum values
func (sw *Switches) SetEnums(el []enums.Enum) *Switches {
	sz := len(el)
	if sz == 0 {
		return sw
	}
	sw.Items = make([]string, sz)
	sw.Tooltips = make([]string, sz)
	for i, enum := range el {
		if bf, ok := enum.(enums.BitFlag); ok {
			sw.Items[i] = bf.BitIndexString()
		} else {
			sw.Items[i] = enum.String()
		}
		sw.Tooltips[i] = enum.Desc()
	}
	return sw
}

// SetEnum sets the Items list from an enum value by calling [Switches.SetEnums]
// using the result of [enums.Enum.Values] on the given value
func (sw *Switches) SetEnum(enum enums.Enum) *Switches {
	return sw.SetEnums(enum.Values())
}

// UpdateFromBitFlags sets the checked state of the switches from the
// given bit flag enum value.
func (sw *Switches) UpdateFromBitFlag(bitflag enums.BitFlag) {
	els := bitflag.Values()
	mn := min(len(els), sw.NumChildren())
	updt := sw.UpdateStart()
	for i := 0; i < mn; i++ {
		ev := els[i]
		swi := sw.Child(i)
		sw := swi.(*Switch)
		on := bitflag.HasFlag(ev.(enums.BitFlag))
		sw.SetChecked(on)
	}
	sw.UpdateEndRender(updt)
}

// BitFlagsValue sets the given bitflag value to the value specified
// by the switches.
func (sw *Switches) BitFlagValue(bitflag enums.BitFlagSetter) {
	bitflag.SetInt64(0)

	els := bitflag.Values()
	mn := min(len(els), sw.NumChildren())
	for i := 0; i < mn; i++ {
		ev := els[i]
		swi := sw.Child(i)
		sw := swi.(*Switch)
		if sw.StateIs(states.Checked) {
			bitflag.SetFlag(true, ev.(enums.BitFlag))
		}
	}
}

// HandleSwitchEvents handles the events for the given switch.
func (sw *Switches) HandleSwitchEvents(swi *Switch) {
	swi.OnChange(func(e events.Event) {
		if sw.Mutex && swi.StateIs(states.Checked) {
			ip, _ := swi.IndexInParent()
			sw.UnCheckAllBut(ip)
		}
		sw.Send(events.Change, e)
	})
}

func (sw *Switches) ConfigItems() {
	for i, swi := range *sw.Children() {
		s := swi.(*Switch)
		s.SetType(sw.Type)
		lbl := sw.Items[i]
		s.SetText(lbl)
		if len(sw.Tooltips) > i {
			s.Tooltip = sw.Tooltips[i]
		}
	}
}

func (sw *Switches) ConfigSwitches(sc *Scene) {
	if len(sw.Items) == 0 {
		sw.DeleteChildren(ki.DestroyKids)
		return
	}
	config := ki.Config{}
	for _, lb := range sw.Items {
		config.Add(SwitchType, lb)
	}
	mods, updt := sw.ConfigChildren(config)
	if mods || sw.NeedsRebuild() {
		sw.ConfigItems()
		sw.UpdateEndLayout(updt)
	}
}

func (sw *Switches) ConfigWidget(sc *Scene) {
	sw.ConfigSwitches(sc)
}
