// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"cogentcore.org/core/base/labels"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/base/strcase"
	"cogentcore.org/core/enums"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
)

// Switches is a widget for containing a set of [Switch]es.
// It can optionally enforce mutual exclusivity (ie: radio buttons)
// through the [Switches.Mutex] field. It supports binding to
// [enums.Enum] and [enums.BitFlag] values with appropriate properties
// automatically set.
type Switches struct {
	Frame

	// Type is the type of switches that will be made.
	Type SwitchTypes

	// Items are the items displayed to the user.
	Items []SwitchItem

	// Mutex is whether to make the items mutually exclusive
	// (checking one turns off all the others).
	Mutex bool

	// AllowNone is whether to allow the user to deselect all items.
	// It is on by default.
	AllowNone bool `default:"true"`

	// selectedIndexes are the indexes in [Switches.Items] of the currently
	// selected switch items.
	selectedIndexes []int

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

// getText returns the effective text for this switch item.
// If [SwitchItem.Text] is set, it returns that. Otherwise,
// it returns [labels.ToLabel] of [SwitchItem.Value].
func (si *SwitchItem) getText() string {
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
		sw.bitFlagFromSelected(sw.bitFlagValue)
		// We must return a non-pointer value to prevent [ResetWidgetValue]
		// from clearing the bit flag value (since we only ever have one
		// total pointer to it, so it is uniquely vulnerable to being destroyed).
		return reflectx.Underlying(reflect.ValueOf(sw.bitFlagValue)).Interface()
	}
	item := sw.SelectedItem()
	if item == nil {
		return nil
	}
	return item.Value
}

func (sw *Switches) SetWidgetValue(value any) error {
	up := reflectx.UnderlyingPointer(reflect.ValueOf(value))
	if bf, ok := up.Interface().(enums.BitFlagSetter); ok {
		sw.selectFromBitFlag(bf)
		return nil
	}
	return sw.SelectValue(up.Elem().Interface())
}

func (sw *Switches) OnBind(value any, tags reflect.StructTag) {
	if e, ok := value.(enums.Enum); ok {
		sw.SetEnum(e).SetType(SwitchSegmentedButton).SetMutex(true)
	}
	if bf, ok := value.(enums.BitFlagSetter); ok {
		sw.bitFlagValue = bf
		sw.SetType(SwitchChip).SetMutex(false)
	} else {
		sw.bitFlagValue = nil
		sw.AllowNone = false
	}
}

func (sw *Switches) Init() {
	sw.Frame.Init()
	sw.AllowNone = true
	sw.Styler(func(s *styles.Style) {
		s.Padding.Set(units.Dp(ConstantSpacing(2)))
		s.Margin.Set(units.Dp(ConstantSpacing(2)))
		if sw.Type == SwitchSegmentedButton {
			s.Gap.Zero()
		} else {
			s.Wrap = true
		}
		if sw.Type == SwitchSwitch {
			s.IconSize.Set(units.Em(2), units.Em(1.5))
		} else {
			s.IconSize.Set(units.Em(1.5))
		}
	})
	sw.FinalStyler(func(s *styles.Style) {
		if s.Direction != styles.Row {
			// if we wrap, it just goes in the x direction
			s.Wrap = false
		}
	})

	sw.Maker(func(p *tree.Plan) {
		for i, item := range sw.Items {
			tree.AddAt(p, strconv.Itoa(i), func(w *Switch) {
				w.OnChange(func(e events.Event) {
					if w.IsChecked() {
						if sw.Mutex {
							sw.selectedIndexes = []int{i}
						} else {
							sw.selectedIndexes = append(sw.selectedIndexes, i)
						}
					} else if sw.AllowNone || len(sw.selectedIndexes) > 1 {
						sw.selectedIndexes = slices.DeleteFunc(sw.selectedIndexes, func(v int) bool { return v == i })
					}
					sw.SendChange(e)
					sw.UpdateRender()
				})
				w.Styler(func(s *styles.Style) {
					ps := &sw.Styles
					s.IconSize = ps.IconSize // Directly inherit

					// Remaining styles are only for segmented buttons
					if sw.Type != SwitchSegmentedButton {
						return
					}
					ip := w.IndexInParent()
					brf := styles.BorderRadiusFull.Top
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
					w.SetType(sw.Type).SetText(item.getText()).SetTooltip(item.Tooltip)
					if sw.Type == SwitchSegmentedButton && sw.Styles.Direction == styles.Column {
						// need a blank icon to create a cohesive segmented button
						w.SetIconOff(icons.Blank).SetIconIndeterminate(icons.Blank)
					}
					if !w.StateIs(states.Indeterminate) {
						w.SetChecked(slices.Contains(sw.selectedIndexes, i))
					}
				})
			})
		}
	})
}

// SelectedItem returns the first selected (checked) switch item. It is only
// useful when [Switches.Mutex] is true; if it is not, use [Switches.SelectedItems].
// If no switches are selected, it returns nil.
func (sw *Switches) SelectedItem() *SwitchItem {
	if len(sw.selectedIndexes) == 0 {
		return nil
	}
	return &sw.Items[sw.selectedIndexes[0]]
}

// SelectedItems returns all of the currently selected (checked) switch items.
// If [Switches.Mutex] is true, you should use [Switches.SelectedItem] instead.
func (sw *Switches) SelectedItems() []SwitchItem {
	res := []SwitchItem{}
	for _, i := range sw.selectedIndexes {
		res = append(res, sw.Items[i])
	}
	return res
}

// SelectValue sets the item with the given [SwitchItem.Value]
// to be the only selected item.
func (sw *Switches) SelectValue(value any) error {
	for i, item := range sw.Items {
		if item.Value == value {
			sw.selectedIndexes = []int{i}
			return nil
		}
	}
	return fmt.Errorf("Switches.SelectValue: item not found: (value: %v, items: %v)", value, sw.Items)
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

// selectFromBitFlag sets which switches are selected based on the given bit flag value.
func (sw *Switches) selectFromBitFlag(bitflag enums.BitFlagSetter) {
	values := bitflag.Values()
	sw.selectedIndexes = []int{}
	for i, value := range values {
		if bitflag.HasFlag(value.(enums.BitFlag)) {
			sw.selectedIndexes = append(sw.selectedIndexes, i)
		}
	}
}

// bitFlagFromSelected sets the given bit flag value based on which switches are selected.
func (sw *Switches) bitFlagFromSelected(bitflag enums.BitFlagSetter) {
	bitflag.SetInt64(0)
	values := bitflag.Values()
	for _, i := range sw.selectedIndexes {
		bitflag.SetFlag(true, values[i].(enums.BitFlag))
	}
}
