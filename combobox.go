// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"reflect"
	"sort"
	"unicode/utf8"

	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

////////////////////////////////////////////////////////////////////////////////////////
// ComboBox for selecting items from a list

type ComboBox struct {
	ButtonBase
	Editable  bool          `desc:"provide a text field for editing the value, or just a button for selecting items?"`
	CurVal    interface{}   `json:"-" xml:"-" desc:"current selected value"`
	CurIndex  int           `json:"-" xml:"-" desc:"current index in list of possible items"`
	Items     []interface{} `json:"-" xml:"-" desc:"items available for selection"`
	ItemsMenu Menu          `json:"-" xml:"-" desc:"the menu of actions for selecting items -- automatically generated from Items"`
	ComboSig  ki.Signal     `json:"-" xml:"-" view:"-" desc:"signal for combo box, when a new value has been selected -- the signal type is the index of the selected item, and the data is the value"`
	MaxLength int           `desc:"maximum label length (in runes)"`
}

var KiT_ComboBox = kit.Types.AddType(&ComboBox{}, ComboBoxProps)

var ComboBoxProps = ki.Props{
	"border-width":     units.NewValue(1, units.Px),
	"border-radius":    units.NewValue(4, units.Px),
	"border-color":     &Prefs.BorderColor,
	"border-style":     BorderSolid,
	"padding":          units.NewValue(4, units.Px),
	"margin":           units.NewValue(4, units.Px),
	"text-align":       AlignCenter,
	"background-color": &Prefs.ControlColor,
	"color":            &Prefs.FontColor,
	"#icon": ki.Props{
		"width":   units.NewValue(1, units.Em),
		"height":  units.NewValue(1, units.Em),
		"margin":  units.NewValue(0, units.Px),
		"padding": units.NewValue(0, units.Px),
		"fill":    &Prefs.IconColor,
		"stroke":  &Prefs.FontColor,
	},
	"#label": ki.Props{
		"margin":  units.NewValue(0, units.Px),
		"padding": units.NewValue(0, units.Px),
	},
	"#indicator": ki.Props{
		"width":          units.NewValue(1.5, units.Ex),
		"height":         units.NewValue(1.5, units.Ex),
		"margin":         units.NewValue(0, units.Px),
		"padding":        units.NewValue(0, units.Px),
		"vertical-align": AlignBottom,
		"fill":           &Prefs.IconColor,
		"stroke":         &Prefs.FontColor,
	},
	"#ind-stretch": ki.Props{
		"width": units.NewValue(1, units.Em),
	},
	ButtonSelectors[ButtonActive]: ki.Props{
		"background-color": "linear-gradient(lighter-0, highlight-10)",
	},
	ButtonSelectors[ButtonInactive]: ki.Props{
		"border-color": "highlight-50",
		"color":        "highlight-50",
	},
	ButtonSelectors[ButtonHover]: ki.Props{
		"background-color": "linear-gradient(highlight-10, highlight-10)",
	},
	ButtonSelectors[ButtonFocus]: ki.Props{
		"border-width":     units.NewValue(2, units.Px),
		"background-color": "linear-gradient(samelight-50, highlight-10)",
	},
	ButtonSelectors[ButtonDown]: ki.Props{
		"color":            "highlight-90",
		"background-color": "linear-gradient(highlight-30, highlight-10)",
	},
	ButtonSelectors[ButtonSelected]: ki.Props{
		"background-color": "linear-gradient(pref(SelectColor), highlight-10)",
		"color":            "highlight-90",
	},
}

// ButtonWidget interface

func (g *ComboBox) ButtonAsBase() *ButtonBase {
	return &(g.ButtonBase)
}

func (g *ComboBox) ButtonRelease() {
	if g.IsInactive() {
		return
	}
	wasPressed := (g.State == ButtonDown)
	updt := g.UpdateStart()
	g.MakeItemsMenu()
	g.SetButtonState(ButtonActive)
	g.ButtonSig.Emit(g.This, int64(ButtonReleased), nil)
	if wasPressed {
		g.ButtonSig.Emit(g.This, int64(ButtonClicked), nil)
	}
	g.UpdateEnd(updt)
	pos := g.WinBBox.Max
	indic, ok := g.Parts.ChildByName("indicator", 3)
	if ok {
		pos = KiToNode2DBase(indic).WinBBox.Min
	} else {
		pos.Y -= 10
		pos.X -= 10
	}
	PopupMenu(g.ItemsMenu, pos.X, pos.Y, g.Viewport, g.Text)
}

func (g *ComboBox) ConfigParts() {
	config, icIdx, lbIdx := g.ConfigPartsIconLabel(string(g.Icon), g.Text)
	indIdx := g.ConfigPartsAddIndicator(&config, true)  // default on
	mods, updt := g.Parts.ConfigChildren(config, false) // not unique names
	g.ConfigPartsSetIconLabel(string(g.Icon), g.Text, icIdx, lbIdx)
	g.ConfigPartsIndicator(indIdx)
	if g.MaxLength > 0 && lbIdx >= 0 {
		lbl := g.Parts.KnownChild(lbIdx).(*Label)
		lbl.SetMinPrefWidth(units.NewValue(float32(g.MaxLength), units.Ex))
	}
	if mods {
		g.UpdateEnd(updt)
	}
}

// MakeItems makes sure the Items list is made, and if not, or reset is true,
// creates one with the given capacity
func (g *ComboBox) MakeItems(reset bool, capacity int) {
	if g.Items == nil || reset {
		g.Items = make([]interface{}, 0, capacity)
	}
}

// SortItems sorts the items according to their labels
func (g *ComboBox) SortItems(ascending bool) {
	sort.Slice(g.Items, func(i, j int) bool {
		if ascending {
			return ToLabel(g.Items[i]) < ToLabel(g.Items[j])
		} else {
			return ToLabel(g.Items[i]) > ToLabel(g.Items[j])
		}
	})
}

// SetToMaxLength gets the maximum label length so that the width of the
// button label is automatically set according to the max length of all items
// in the list -- if maxLen > 0 then it is used as an upper do-not-exceed
// length
func (g *ComboBox) SetToMaxLength(maxLen int) {
	ml := 0
	for _, it := range g.Items {
		ml = kit.MaxInt(ml, utf8.RuneCountInString(ToLabel(it)))
	}
	if maxLen > 0 {
		ml = kit.MinInt(ml, maxLen)
	}
	g.MaxLength = ml
}

// ItemsFromTypes sets the Items list from a list of types -- see e.g.,
// AllImplementersOf or AllEmbedsOf in kit.TypeRegistry -- if setFirst then
// set current item to the first item in the list, sort sorts the list in
// ascending order, and maxLen if > 0 auto-sets the width of the button to the
// contents, with the given upper limit
func (g *ComboBox) ItemsFromTypes(tl []reflect.Type, setFirst, sort bool, maxLen int) {
	sz := len(tl)
	g.Items = make([]interface{}, sz)
	for i, typ := range tl {
		g.Items[i] = typ
	}
	if sort {
		g.SortItems(true)
	}
	if maxLen > 0 {
		g.SetToMaxLength(maxLen)
	}
	if setFirst {
		g.SetCurIndex(0)
	}
}

// ItemsFromStringList sets the Items list from a list of string values -- if
// setFirst then set current item to the first item in the list, and maxLen if
// > 0 auto-sets the width of the button to the contents, with the given upper
// limit
func (g *ComboBox) ItemsFromStringList(el []string, setFirst bool, maxLen int) {
	sz := len(el)
	g.Items = make([]interface{}, sz)
	for i, str := range el {
		g.Items[i] = str
	}
	if maxLen > 0 {
		g.SetToMaxLength(maxLen)
	}
	if setFirst {
		g.SetCurIndex(0)
	}
}

// ItemsFromEnumList sets the Items list from a list of enum values (see
// kit.EnumRegistry) -- if setFirst then set current item to the first item in
// the list, and maxLen if > 0 auto-sets the width of the button to the
// contents, with the given upper limit
func (g *ComboBox) ItemsFromEnumList(el []kit.EnumValue, setFirst bool, maxLen int) {
	sz := len(el)
	g.Items = make([]interface{}, sz)
	for i, enum := range el {
		g.Items[i] = enum
	}
	if maxLen > 0 {
		g.SetToMaxLength(maxLen)
	}
	if setFirst {
		g.SetCurIndex(0)
	}
}

// ItemsFromEnum sets the Items list from an enum type, which must be
// registered on kit.EnumRegistry -- if setFirst then set current item to the
// first item in the list, and maxLen if > 0 auto-sets the width of the button
// to the contents, with the given upper limit -- see kit.EnumRegistry, and
// maxLen if > 0 auto-sets the width of the button to the contents, with the
// given upper limit
func (g *ComboBox) ItemsFromEnum(enumtyp reflect.Type, setFirst bool, maxLen int) {
	g.ItemsFromEnumList(kit.Enums.TypeValues(enumtyp, true), setFirst, maxLen)
}

// FindItem finds an item on list of items and returns its index
func (g *ComboBox) FindItem(it interface{}) int {
	if g.Items == nil {
		return -1
	}
	for i, v := range g.Items {
		if v == it {
			return i
		}
	}
	return -1
}

// SetCurVal sets the current value (CurVal) and the corresponding CurIndex
// for that item on the current Items list (adds to items list if not found)
// -- returns that index -- and sets the text to the string value of that
// value (using standard Stringer string conversion)
func (g *ComboBox) SetCurVal(it interface{}) int {
	g.CurVal = it
	g.CurIndex = g.FindItem(it)
	if g.CurIndex < 0 { // add to list if not found..
		g.CurIndex = len(g.Items)
		g.Items = append(g.Items, it)
	}
	g.SetText(ToLabel(it))
	return g.CurIndex
}

// SetCurIndex sets the current index (CurIndex) and the corresponding CurVal
// for that item on the current Items list (-1 if not found) -- returns value
// -- and sets the text to the string value of that value (using standard
// Stringer string conversion)
func (g *ComboBox) SetCurIndex(idx int) interface{} {
	g.CurIndex = idx
	if idx < 0 || idx >= len(g.Items) {
		g.CurVal = nil
		g.SetText(fmt.Sprintf("idx %v > len", idx))
	} else {
		g.CurVal = g.Items[idx]
		g.SetText(ToLabel(g.CurVal))
	}
	return g.CurVal
}

// SelectItem selects a given item and emits the index as the ComboSig signal
// and the selected item as the data
func (g *ComboBox) SelectItem(idx int) {
	updt := g.UpdateStart()
	g.SetCurIndex(idx)
	g.ComboSig.Emit(g.This, int64(g.CurIndex), g.CurVal)
	g.UpdateEnd(updt)
}

// MakeItemsMenu makes menu of all the items
func (g *ComboBox) MakeItemsMenu() {
	if g.ItemsMenu == nil {
		g.ItemsMenu = make(Menu, 0, len(g.Items))
	}
	sz := len(g.ItemsMenu)
	for i, it := range g.Items {
		var ac *Action
		if sz > i {
			ac = g.ItemsMenu[i].(*Action)
		} else {
			ac = &Action{}
			ac.Init(ac)
			g.ItemsMenu = append(g.ItemsMenu, ac.This.(Node2D))
		}
		txt := ToLabel(it)
		nm := fmt.Sprintf("Item_%v", i)
		ac.SetName(nm)
		ac.Text = txt
		ac.Data = i // index is the data
		ac.SetSelectedState(i == g.CurIndex)
		ac.SetAsMenu()
		ac.ActionSig.ConnectOnly(g.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			idx := data.(int)
			cb := recv.(*ComboBox)
			cb.SelectItem(idx)
		})
	}
}
