// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"reflect"

	"github.com/goki/gi/gist"
	"github.com/goki/gi/units"
	"github.com/goki/ki/bitflag"
	"github.com/goki/ki/ints"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/ki/sliceclone"
)

// ButtonBox is a widget for containing a set of CheckBox buttons.
// It can optionally enforce mutual excusivity (i.e., Radio Buttons).
// The buttons are all in the Parts of the widget and the Parts layout
// determines how they are displayed.
type ButtonBox struct {
	PartsWidgetBase
	Items     []string  `desc:"the list of items (checbox button labels)"`
	Mutex     bool      `desc:"make the items mutually exclusive -- checking one turns off all the others"`
	ButtonSig ki.Signal `copy:"-" json:"-" xml:"-" view:"-" desc:"signal for button box, when any button is updated -- the signal type is the index of the selected item, and the data is the label"`
}

var KiT_ButtonBox = kit.Types.AddType(&ButtonBox{}, ButtonBoxProps)

// AddNewButtonBox adds a new button to given parent node, with given name.
func AddNewButtonBox(parent ki.Ki, name string) *ButtonBox {
	return parent.AddNewChild(KiT_ButtonBox, name).(*ButtonBox)
}

func (bb *ButtonBox) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*ButtonBox)
	bb.PartsWidgetBase.CopyFieldsFrom(&fr.PartsWidgetBase)
	bb.Items = sliceclone.String(fr.Items)
}

func (bb *ButtonBox) Disconnect() {
	bb.PartsWidgetBase.Disconnect()
	bb.ButtonSig.DisconnectAll()
}

var ButtonBoxProps = ki.Props{
	"EnumType:Flag":    KiT_NodeFlags,
	"border-width":     units.NewPx(1),
	"border-radius":    units.NewPx(2),
	"border-color":     &Prefs.Colors.Border,
	"padding":          units.NewPx(2),
	"margin":           units.NewPx(2),
	"text-align":       gist.AlignCenter,
	"background-color": &Prefs.Colors.Control,
	"color":            &Prefs.Colors.Font,
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

// ItemsFromEnumList sets the Items list from a list of enum values (see
// kit.EnumRegistry)
func (bb *ButtonBox) ItemsFromEnumList(el []kit.EnumValue) {
	sz := len(el)
	if sz == 0 {
		return
	}
	bb.Items = make([]string, sz)
	for i, enum := range el {
		bb.Items[i] = enum.Name
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
	mx := ints.MaxInt(len(els), bb.Parts.NumChildren())
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
	mx := ints.MaxInt(len(els), bb.Parts.NumChildren())
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

func (bb *ButtonBox) ConfigItems() {
	for i, cbi := range *bb.Parts.Children() {
		cb := cbi.(*CheckBox)
		lbl := bb.Items[i]
		cb.SetText(lbl)
		cb.SetProp("index", i)
		cb.ButtonSig.Connect(bb.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			if sig != int64(ButtonToggled) {
				return
			}
			bbb, _ := recv.Embed(KiT_ButtonBox).(*ButtonBox)
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
	config := kit.TypeAndNameList{}
	for _, lb := range bb.Items {
		config.Add(KiT_CheckBox, lb)
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

func (bb *ButtonBox) Init2D() {
	bb.Init2DWidget()
	bb.ConfigParts()
}

func (bb *ButtonBox) Style2D() {
	bb.StyMu.Lock()
	bb.Style2DWidget()
	bb.LayState.SetFromStyle(&bb.Sty.Layout) // also does reset
	bb.StyMu.Unlock()
	bb.ConfigParts()
}

func (bb *ButtonBox) Layout2D(parBBox image.Rectangle, iter int) bool {
	bb.ConfigPartsIfNeeded()
	bb.Layout2DBase(parBBox, true, iter) // init style
	bb.Layout2DParts(parBBox, iter)
	return bb.Layout2DChildren(iter)
}

func (bb *ButtonBox) RenderButtonBox() {
	rs, _, st := bb.RenderLock()
	bb.RenderStdBox(st)
	bb.RenderUnlock(rs)
}

func (bb *ButtonBox) Render2D() {
	if bb.FullReRenderIfNeeded() {
		return
	}
	if bb.PushBounds() {
		bb.This().(Node2D).ConnectEvents2D()
		bb.RenderButtonBox()
		bb.Render2DParts()
		bb.Render2DChildren()
		bb.PopBounds()
	} else {
		bb.DisconnectAllEvents(RegPri)
	}
}
