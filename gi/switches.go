// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"slices"

	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/ki/v2"
)

// Switches is a widget for containing a set of switches.
// It can optionally enforce mutual exclusivity (i.e., Radio Buttons).
// The buttons are all in the Parts of the widget and the Parts layout
// determines how they are displayed.
//
//goki:embedder
type Switches struct {
	WidgetBase

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
	sw.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
	sw.Items = slices.Clone(fr.Items)
}

func (sw *Switches) OnInit() {
	sw.HandleWidgetEvents()
	sw.SwitchesStyles()
}

func (sw *Switches) SwitchesStyles() {
	sw.AddStyles(func(s *styles.Style) {
		s.Padding.Set(units.Dp(2))
		s.Margin.Set(units.Dp(2))
	})
}

// SelectItem activates a given item but does NOT emit the ButtonSig signal.
// See SelectItemAction for signal emitting version.
// returns error if index is out of range.
func (sw *Switches) SelectItem(idx int) error {
	if idx >= sw.Parts.NumChildren() || idx < 0 {
		return fmt.Errorf("gi.Switches: SelectItem, index out of range: %v", idx)
	}
	updt := sw.UpdateStart()
	if sw.Mutex {
		sw.UnCheckAllBut(idx)
	}
	cb := sw.Parts.Child(idx).(*Switch)
	cb.SetState(true, states.Checked)
	sw.UpdateEnd(updt)
	return nil
}

// SelectItemAction activates a given item and emits the ButtonSig signal.
// This is mainly for Mutex use.
// returns error if index is out of range.
func (sw *Switches) SelectItemAction(idx int) error {
	updt := sw.UpdateStart()
	defer sw.UpdateEnd(updt)

	err := sw.SelectItem(idx)
	if err != nil {
		return err
	}
	// cb := sw.Parts.Child(idx).(*CheckBox)
	// sw.ButtonSig.Emit(sw.This(), int64(idx), cb.Text)
	return nil
}

// UnCheckAll unchecks all switches
func (sw *Switches) UnCheckAll() {
	updt := sw.UpdateStart()
	for _, cbi := range *sw.Parts.Children() {
		cb := cbi.(*Switch)
		cb.SetState(false, states.Checked)
	}
	sw.UpdateEnd(updt)
}

// UnCheckAllBut unchecks all switches except given one
func (sw *Switches) UnCheckAllBut(idx int) {
	updt := sw.UpdateStart()
	for i, cbi := range *sw.Parts.Children() {
		if i == idx {
			continue
		}
		cb := cbi.(*Switch)
		cb.SetState(false, states.Checked)
	}
	sw.UpdateEnd(updt)
}

// ItemsFromStringList sets the Items list from a list of string values -- if
// setFirst then set current item to the first item in the list, and maxLen if
// > 0 auto-sets the width of the button to the contents, with the given upper
// limit
func (sw *Switches) ItemsFromStringList(el []string) {
	sz := len(el)
	if sz == 0 {
		return
	}
	sw.Items = make([]string, sz)
	copy(sw.Items, el)
}

// todo:

// ItemsFromEnumList sets the Items list from a list of enum values (see
// kit.EnumRegistry)
/*
func (sw *Switches) ItemsFromEnumList(el []kit.EnumValue) {
	sz := len(el)
	if sz == 0 {
		return
	}
	sw.Items = make([]string, sz)
	sw.Tooltips = make([]string, sz)
	for i, enum := range el {
		sw.Items[i] = enum.Name
		sw.Tooltips[i] = enum.Desc
	}
}

// ItemsFromEnum sets the Items list from an enum type, which must be
// registered on kit.EnumRegistry.
func (sw *Switches) ItemsFromEnum(enumtyp reflect.Type) {
	sw.ItemsFromEnumList(kit.Enums.TypeValues(enumtyp, true))
}

// UpdateFromBitFlags sets the button checked state from a registered
// BitFlag Enum type (see kit.EnumRegistry) with given value
func (sw *Switches) UpdateFromBitFlags(enumtyp reflect.Type, val int64) {
	els := kit.Enums.TypeValues(enumtyp, true)
	mx := max(len(els), sw.Parts.NumChildren())
	for i := 0; i < mx; i++ {
		ev := els[i]
		cbi := sw.Parts.Child(i)
		cb := cbi.(*CheckBox)
		on := bitflag.Has(val, int(ev.Value))
		cb.SetState(on, states.Checked)
	}
}

// BitFlagsValue returns the int64 value for all checkboxes from given
// BitFlag Enum type (see kit.EnumRegistry) with given value
func (sw *Switches) BitFlagsValue(enumtyp reflect.Type) int64 {
	val := int64(0)
	els := kit.Enums.TypeValues(enumtyp, true)
	mx := max(len(els), sw.Parts.NumChildren())
	for i := 0; i < mx; i++ {
		ev := els[i]
		cbi := sw.Parts.Child(i)
		cb := cbi.(*CheckBox)
		if cb.StateIs(states.Checked) {
			bitflag.Set(&val, int(ev.Value))
		}
	}
	return val
}
*/

func (sw *Switches) ConfigItems() {
	for i, cbi := range *sw.Parts.Children() {
		s := cbi.(*Switch)
		s.SetType(sw.Type)
		lbl := sw.Items[i]
		s.SetText(lbl)
		if len(sw.Tooltips) > i {
			s.Tooltip = sw.Tooltips[i]
		}
		s.SetProp("index", i)
		// cb.ButtonSig.Connect(sw.This(), func(recv, send ki.Ki, sig int64, data any) {
		// 	if sig != int64(ButtonToggled) {
		// 		return
		// 	}
		// 	swb := AsSwitches(recv)
		// 	csw := send.(*CheckBox)
		// 	idx := csw.Prop("index").(int)
		// 	ischk := csw.StateIs(states.Checked)
		// 	if swb.Mutex && ischk {
		// 		swb.UnCheckAllBut(idx)
		// 	}
		// 	swb.ButtonSig.Emit(swb.This(), int64(idx), csw.Text)
		// })
	}
}

func (sw *Switches) ConfigParts(sc *Scene) {
	parts := sw.NewParts(LayoutVert)
	if len(sw.Items) == 0 {
		parts.DeleteChildren(ki.DestroyKids)
		return
	}
	config := ki.Config{}
	for _, lb := range sw.Items {
		config.Add(SwitchType, lb)
	}
	mods, updt := parts.ConfigChildren(config)
	if mods || sw.NeedsRebuild() {
		sw.ConfigItems()
		parts.UpdateEnd(updt)
		sw.SetNeedsLayout(sc, updt)
	}
}

func (sw *Switches) ConfigWidget(sc *Scene) {
	sw.ConfigParts(sc)
}

func (sw *Switches) ApplyStyle(sc *Scene) {
	sw.StyMu.Lock()
	sw.ApplyStyleWidget(sc)
	sw.StyMu.Unlock()
	// sw.ConfigParts(sc) // todo: no config in styling!?
}

func (sw *Switches) DoLayout(sc *Scene, parswox image.Rectangle, iter int) bool {
	sw.DoLayoutBase(sc, parswox, iter)
	sw.DoLayoutParts(sc, parswox, iter)
	return sw.DoLayoutChildren(sc, iter)
}

func (sw *Switches) RenderSwitches(sc *Scene) {
	rs, _, st := sw.RenderLock(sc)
	sw.RenderStdBox(sc, st)
	sw.RenderUnlock(rs)
}

func (sw *Switches) Render(sc *Scene) {
	if sw.PushBounds(sc) {
		sw.RenderSwitches(sc)
		sw.RenderParts(sc)
		sw.RenderChildren(sc)
		sw.PopBounds(sc)
	}
}
