// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode"

	"cogentcore.org/core/base/labels"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/base/strcase"
	"cogentcore.org/core/enums"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/types"
)

// Switches is a widget for containing a set of [Switch]es.
// It can optionally enforce mutual exclusivity (ie: radio buttons)
// through the [Switches.Mutex] field.
type Switches struct {
	Frame

	// Type is the type of switches that will be made.
	Type SwitchTypes

	// Items are the items displayed to the user.
	Items []SwitchItem

	// Mutex is whether to make the items mutually exclusive
	// (checking one turns off all the others).
	Mutex bool

	// bitFlagValue is the associated bit flag value if non-nil (for [Value]).
	bitFlagValue enums.BitFlagSetter
}

// SwitchItem contains the properties of one item in a [Switches].
type SwitchItem struct {

	// Value is the underlying value the switch item represents.
	Value any

	// Text is the text displayed to the user for this item.
	// If it is empty, then [labels.ToLabel] of [SwitchItem.Value]
	// is used instead.
	Text string

	// Tooltip is the tooltip displayed to the user for this item.
	Tooltip string
}

// GetText returns the effective text for this switch item.
// If [SwitchItem.Text] is set, it returns that. Otherwise,
// it returns [labels.ToLabel] of [SwitchItem.Value].
func (si *SwitchItem) GetText() string {
	if si.Text != "" {
		return si.Text
	}
	if si.Value == nil {
		return ""
	}
	return labels.ToLabel(si.Value)
}

func (sw *Switches) WidgetValue() any {
	if sw.bitFlagValue != nil {
		sw.UpdateBitFlag(sw.bitFlagValue)
		return sw.bitFlagValue
	}
	item := sw.SelectedItem()
	if item == nil {
		return nil
	}
	return item.Value
}

func (sw *Switches) SetWidgetValue(value any) error {
	value = reflectx.Underlying(reflect.ValueOf(value)).Interface()
	if bf, ok := value.(enums.BitFlag); ok {
		sw.UpdateFromBitFlag(bf)
		return nil
	}
	return sw.SetSelectedItem(value)
}

func (sw *Switches) OnBind(value any) {
	if e, ok := value.(enums.Enum); ok {
		sw.SetEnum(e).SetType(SwitchSegmentedButton).SetMutex(true)
	}
	if bf, ok := value.(enums.BitFlagSetter); ok {
		sw.bitFlagValue = bf
		sw.SetType(SwitchChip).SetMutex(false)
	} else {
		sw.bitFlagValue = nil
	}
}

func (sw *Switches) Init() {
	sw.Frame.Init()
	sw.Style(func(s *styles.Style) {
		s.Padding.Set(units.Dp(2))
		s.Margin.Set(units.Dp(2))
		if sw.Type == SwitchSegmentedButton {
			s.Gap.Zero()
		} else {
			s.Wrap = true
		}
	})
	sw.StyleFinal(func(s *styles.Style) {
		if s.Direction == styles.Row {
			s.Grow.Set(1, 0)
		} else {
			// if we wrap, it just goes in the x direction
			s.Wrap = false
		}
	})

	sw.Maker(func(p *Plan) {
		for i, item := range sw.Items {
			AddAt(p, strconv.Itoa(i), func(w *Switch) {
				w.OnChange(func(e events.Event) {
					if sw.Mutex && w.IsChecked() {
						sw.UnCheckAllBut(w.IndexInParent())
					}
					sw.SendChange(e)
				})
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
				w.Updater(func() {
					w.SetType(sw.Type).SetText(item.GetText()).SetTooltip(item.Tooltip)
				})
			})
		}
	})
}

// SelectItem activates a given item but does NOT send a change event.
// See SelectItemAction for event emitting version.
// It returns an error if the index is out of range.
func (sw *Switches) SelectItem(index int) error {
	if index >= sw.NumChildren() || index < 0 {
		return fmt.Errorf("core.Switches.SelectItem: index out of range: %v", index)
	}
	if sw.Mutex {
		sw.UnCheckAllBut(index)
	}
	cs := sw.Child(index).(*Switch)
	cs.SetChecked(true)
	sw.NeedsRender()
	return nil
}

// SelectItemAction activates a given item and emits a change event.
// This is mainly for Mutex use.
// returns error if index is out of range.
func (sw *Switches) SelectItemAction(index int) error {
	err := sw.SelectItem(index)
	if err != nil {
		return err
	}
	sw.SendChange()
	return nil
}

// SelectedItem returns the first selected (checked) switch item. It is only
// useful when [Switches.Mutex] is true; if it is not, use [Switches.SelectedItems].
// If no switches are selected, it returns nil.
func (sw *Switches) SelectedItem() *SwitchItem {
	for i, kswi := range sw.Kids {
		ksw := kswi.(*Switch)
		if ksw.IsChecked() {
			return &sw.Items[i]
		}
	}
	return nil
}

// SetSelectedItem selects the item with the given [SwitchItem.Value].
func (sw *Switches) SetSelectedItem(value any) error {
	for i, item := range sw.Items {
		if item.Value == value {
			return sw.SelectItem(i)
		}
	}
	return fmt.Errorf("core.Switches.SetSelectedItem: item not found: (value: %v, items: %v)", value, sw.Items)
}

// SelectedItems returns all of the currently selected (checked) switch items.
// If [Switches.Mutex] is true, you should use [Switches.SelectedItem] instead.
func (sw *Switches) SelectedItems() []SwitchItem {
	res := []SwitchItem{}
	for i, kswi := range sw.Kids {
		ksw := kswi.(*Switch)
		if ksw.IsChecked() {
			res = append(res, sw.Items[i])
		}
	}
	return res
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
func (sw *Switches) SetStrings(ss ...string) *Switches {
	sw.Items = make([]SwitchItem, len(ss))
	for i, s := range ss {
		sw.Items[i] = SwitchItem{Value: s}
	}
	return sw
}

// SetEnums sets the [Switches.Items] from the given enums.
func (sw *Switches) SetEnums(es ...enums.Enum) *Switches {
	sw.Items = make([]SwitchItem, len(es))
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
		tip := types.FormatDoc(desc, str, lbl)
		sw.Items[i] = SwitchItem{Value: enum, Text: lbl, Tooltip: tip}
	}
	return sw
}

// SetEnum sets the [Switches.Items] from the [enums.Enum.Values] of the given enum.
func (sw *Switches) SetEnum(enum enums.Enum) *Switches {
	return sw.SetEnums(enum.Values()...)
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
