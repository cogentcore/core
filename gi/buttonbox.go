// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"slices"

	"goki.dev/colors"
	"goki.dev/girl/gist"
	"goki.dev/girl/units"
	"goki.dev/icons"
	"goki.dev/ki/v2"
)

type ButtonBoxEmbedder interface {
	AsButtonBox() *ButtonBox
}

func AsButtonBox(k ki.Ki) *ButtonBox {
	if ac, ok := k.(ButtonBoxEmbedder); ok {
		return ac.AsButtonBox()
	}
	return nil
}

func (bb *ButtonBox) AsButtonBox() *ButtonBox {
	return bb
}

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

// event functions for this type
var ButtonBoxEventFuncs WidgetEvents

func (bb *ButtonBox) OnInit() {
	bb.AddEvents(&ButtonBoxEventFuncs)
	bb.AddStyler(func(w *WidgetBase, s *gist.Style) {
		s.Border.Style.Set(gist.BorderNone)
		s.Border.Radius.Set(units.Px(2))
		s.Padding.Set(units.Px(2 * Prefs.DensityMul()))
		s.Margin.Set(units.Px(2 * Prefs.DensityMul()))
		s.Text.Align = gist.AlignCenter
		s.BackgroundColor.SetSolid(colors.Scheme.Surface)
		s.Color = colors.Scheme.OnSurface
	})
}

func (bb *ButtonBox) CopyFieldsFrom(frm any) {
	fr := frm.(*ButtonBox)
	bb.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
	bb.Items = slices.Clone(fr.Items)
}

func (bb *ButtonBox) Disconnect() {
	bb.WidgetBase.Disconnect()
	bb.ButtonSig.DisconnectAll()
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
			cb.Icon = icons.RadioButtonChecked
			cb.IconOff = icons.RadioButtonUnchecked
		}
		cb.SetProp("index", i)
		cb.ButtonSig.Connect(bb.This(), func(recv, send ki.Ki, sig int64, data any) {
			if sig != int64(ButtonToggled) {
				return
			}
			bbb := AsButtonBox(recv)
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

func (bb *ButtonBox) ConfigParts(sc *Scene) {
	if len(bb.Items) == 0 {
		bb.Parts.DeleteChildren(ki.DestroyKids)
		return
	}
	config := ki.Config{}
	for _, lb := range bb.Items {
		config.Add(CheckBoxType, lb)
	}
	mods, updt := bb.Parts.ConfigChildren(config)
	if mods || gist.RebuildDefaultStyles {
		bb.ConfigItems()
		bb.UpdateEnd(updt)
	}
}

func (bb *ButtonBox) ConfigWidget(sc *Scene) {
	bb.ConfigParts(sc)
}

func (bb *ButtonBox) ApplyStyle(sc *Scene) {
	bb.StyMu.Lock()
	bb.ApplyStyleWidget(sc)
	bb.StyMu.Unlock()
	// bb.ConfigParts(sc) // todo: no config in styling!?
}

func (bb *ButtonBox) DoLayout(sc *Scene, parBBox image.Rectangle, iter int) bool {
	bb.DoLayoutBase(sc, parBBox, true, iter) // init style
	bb.DoLayoutParts(sc, parBBox, iter)
	return bb.DoLayoutChildren(sc, iter)
}

func (bb *ButtonBox) RenderButtonBox(sc *Scene) {
	rs, _, st := bb.RenderLock(sc)
	bb.RenderStdBox(sc, st)
	bb.RenderUnlock(rs)
}

func (bb *ButtonBox) Render(sc *Scene) {
	wi := bb.This().(Widget)
	if bb.PushBounds(sc) {
		wi.FilterEvents()
		bb.RenderButtonBox(sc)
		bb.RenderParts(sc)
		bb.RenderChildren(sc)
		bb.PopBounds(sc)
	}
}
