// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"

	"goki.dev/girl/gist"
	"goki.dev/girl/units"
	"goki.dev/ki/v2"
)

// ButtonBox is a widget for containing a set of CheckBox buttons.
// It can optionally enforce mutual excusivity (i.e., Radio Buttons).
// The buttons are all in the Parts of the widget and the Parts layout
// determines how they are displayed.
type ButtonBox struct {
	WidgetBase

	// the list of items (checbox button labels)
	Items []string `desc:"the list of items (checbox button labels)"`

	// an optional list of tooltips displayed on hover for checkbox items; the indices for tooltips correspond to those for items
	Tooltips []string `desc:"an optional list of tooltips displayed on hover for checkbox items; the indices for tooltips correspond to those for items"`

	// make the items mutually exclusive -- checking one turns off all the others
	Mutex bool `desc:"make the items mutually exclusive -- checking one turns off all the others"`

	// [view: -] signal for button box, when any button is updated -- the signal type is the index of the selected item, and the data is the label
	ButtonSig ki.Signal `copy:"-" json:"-" xml:"-" view:"-" desc:"signal for button box, when any button is updated -- the signal type is the index of the selected item, and the data is the label"`
}

func (bb *ButtonBox) OnInit() {
	bb.AddStyler(func(w *WidgetBase, s *gist.Style) {
		s.Border.Style.Set(gist.BorderNone)
		s.Border.Radius.Set(units.Px(2))
		s.Padding.Set(units.Px(2 * Prefs.DensityMul()))
		s.Margin.Set(units.Px(2 * Prefs.DensityMul()))
		s.Text.Align = gist.AlignCenter
		s.BackgroundColor.SetSolid(ColorScheme.Surface)
		s.Color = ColorScheme.OnSurface
	})
}

func (bb *ButtonBox) CopyFieldsFrom(frm any) {
	fr := frm.(*ButtonBox)
	bb.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
	bb.Items = slice.Clone(fr.Items)
}

func (bb *ButtonBox) Disconnect() {
	bb.WidgetBase.Disconnect()
	bb.ButtonSig.DisconnectAll()
}

var ButtonBoxProps = ki.Props{
	ki.EnumTypeFlag: TypeNodeFlags,
}

// SelectItem activates a given item but does NOT emit the ButtonSig signal.
// See SelectItemAction for signal emitting version.
// returns error if index is out of range.
func (bb *ButtonBox) SelectItem(idx int) error {
	if idx >= bb.Parts.NumChildren() || idx < 0 {
		return fmt.Errorf("gi.ButtonBox: SelectItem, index out of range: %v", idx)
	}
	updt := bb.UpdateStart()
	if bb.Mutex {
		bb.UnCheckAllBut(idx)
	}
	cb := bb.Parts.Child(idx).(*CheckBox)
	cb.SetChecked(true)
	bb.UpdateEnd(updt)
	return nil
}

// SelectItemAction activates a given item and emits the ButtonSig signal.
// This is mainly for Mutex use.
// returns error if index is out of range.
func (bb *ButtonBox) SelectItemAction(idx int) error {
	updt := bb.UpdateStart()
	defer bb.UpdateEnd(updt)

	err := bb.SelectItem(idx)
	if err != nil {
		return err
	}
	cb := bb.Parts.Child(idx).(*CheckBox)
	bb.ButtonSig.Emit(bb.This(), int64(idx), cb.Text)
	return nil
}

// UnCheckAll unchecks all buttons
func (bb *ButtonBox) UnCheckAll() {
	updt := bb.UpdateStart()
	for _, cbi := range *bb.Parts.Children() {
		cb := cbi.(*CheckBox)
		cb.SetChecked(false)
	}
	bb.UpdateEnd(updt)
}

// UnCheckAllBut unchecks all buttons except given one
func (bb *ButtonBox) UnCheckAllBut(idx int) {
	updt := bb.UpdateStart()
	for i, cbi := range *bb.Parts.Children() {
		if i == idx {
			continue
		}
		cb := cbi.(*CheckBox)
		cb.SetChecked(false)
	}
	bb.UpdateEnd(updt)
}

// ItemsFromStringList sets the Items list from a list of string values -- if
// setFirst then set current item to the first item in the list, and maxLen if
// > 0 auto-sets the width of the button to the contents, with the given upper
// limit
func (bb *ButtonBox) ItemsFromStringList(el []string) {
	sz := len(el)
	if sz == 0 {
		return
	}
	bb.Items = make([]string, sz)
	copy(bb.Items, el)
}

// todo:

// ItemsFromEnumList sets the Items list from a list of enum values (see
// kit.EnumRegistry)
/*
func (bb *ButtonBox) ItemsFromEnumList(el []kit.EnumValue) {
	sz := len(el)
	if sz == 0 {
		return
	}
	bb.Items = make([]string, sz)
	bb.Tooltips = make([]string, sz)
	for i, enum := range el {
		bb.Items[i] = enum.Name
		bb.Tooltips[i] = enum.Desc
	}
}

// ItemsFromEnum sets the Items list from an enum type, which must be
// registered on kit.EnumRegistry.
func (bb *ButtonBox) ItemsFromEnum(enumtyp reflect.Type) {
	bb.ItemsFromEnumList(kit.Enums.TypeValues(enumtyp, true))
}

// UpdateFromBitFlags sets the button checked state from a registered
// BitFlag Enum type (see kit.EnumRegistry) with given value
func (bb *ButtonBox) UpdateFromBitFlags(enumtyp reflect.Type, val int64) {
	els := kit.Enums.TypeValues(enumtyp, true)
	mx := max(len(els), bb.Parts.NumChildren())
	for i := 0; i < mx; i++ {
		ev := els[i]
		cbi := bb.Parts.Child(i)
		cb := cbi.(*CheckBox)
		on := bitflag.Has(val, int(ev.Value))
		cb.SetChecked(on)
	}
}

// BitFlagsValue returns the int64 value for all checkboxes from given
// BitFlag Enum type (see kit.EnumRegistry) with given value
func (bb *ButtonBox) BitFlagsValue(enumtyp reflect.Type) int64 {
	val := int64(0)
	els := kit.Enums.TypeValues(enumtyp, true)
	mx := max(len(els), bb.Parts.NumChildren())
	for i := 0; i < mx; i++ {
		ev := els[i]
		cbi := bb.Parts.Child(i)
		cb := cbi.(*CheckBox)
		if cb.IsChecked() {
			bitflag.Set(&val, int(ev.Value))
		}
	}
	return val
}
*/

func (bb *ButtonBox) ConfigItems() {
	for i, cbi := range *bb.Parts.Children() {
		cb := cbi.(*CheckBox)
		lbl := bb.Items[i]
		cb.SetText(lbl)
		if len(bb.Tooltips) > i {
			cb.Tooltip = bb.Tooltips[i]
		}
		if bb.Mutex {
			cb.Icon = gicons.RadioButtonChecked
			cb.IconOff = gicons.RadioButtonUnchecked
		}
		cb.SetProp("index", i)
		cb.ButtonSig.Connect(bb.This(), func(recv, send ki.Ki, sig int64, data any) {
			if sig != int64(ButtonToggled) {
				return
			}
			bbb, _ := recv.Embed(TypeButtonBox).(*ButtonBox)
			cbb := send.(*CheckBox)
			idx := cbb.Prop("index").(int)
			ischk := cbb.IsChecked()
			if bbb.Mutex && ischk {
				bbb.UnCheckAllBut(idx)
			}
			bbb.ButtonSig.Emit(bbb.This(), int64(idx), cbb.Text)
		})
	}
}

func (bb *ButtonBox) ConfigParts() {
	if len(bb.Items) == 0 {
		bb.Parts.DeleteChildren(ki.DestroyKids)
		return
	}
	config := ki.TypeAndNameList{}
	for _, lb := range bb.Items {
		config.Add(TypeCheckBox, lb)
	}
	mods, updt := bb.Parts.ConfigChildren(config)
	if mods || gist.RebuildDefaultStyles {
		bb.ConfigItems()
		bb.UpdateEnd(updt)
	}
}

func (bb *ButtonBox) ConfigPartsIfNeeded() {
	if bb.NumChildren() == len(bb.Items) {
		return
	}
	bb.ConfigParts()
}

func (bb *ButtonBox) Config() {
	bb.ConfigWidget()
	bb.ConfigParts()
}

func (bb *ButtonBox) SetStyle() {
	bb.StyMu.Lock()
	bb.SetStyleWidget()
	bb.LayState.SetFromStyle(&bb.Style) // also does reset
	bb.StyMu.Unlock()
	bb.ConfigParts()
}

func (bb *ButtonBox) DoLayout(vp *Viewport, parBBox image.Rectangle, iter int) bool {
	bb.ConfigPartsIfNeeded()
	bb.DoLayoutBase(parBBox, true, iter) // init style
	bb.DoLayoutParts(parBBox, iter)
	return bb.DoLayoutChildren(iter)
}

func (bb *ButtonBox) RenderButtonBox() {
	rs, _, st := bb.RenderLock()
	bb.RenderStdBox(st)
	bb.RenderUnlock(rs)
}

func (bb *ButtonBox) Render(vp *Viewport) {
	if bb.FullReRenderIfNeeded() {
		return
	}
	if bb.PushBounds() {
		bb.This().(Node2D).ConnectEvents()
		bb.RenderButtonBox()
		bb.RenderParts()
		bb.RenderChildren()
		bb.PopBounds()
	} else {
		bb.DisconnectAllEvents(RegPri)
	}
}
