// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"unicode/utf8"

	"goki.dev/colors"
	"goki.dev/cursors"
	"goki.dev/enums"
	"goki.dev/girl/abilities"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/gti"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/pi/v2/complete"
)

// Chooser is for selecting items from a dropdown list, with an optional
// edit TextField for typing directly.
// The items can be of any type, including enum values -- they are converted
// to strings for the display.  If the items are of type [icons.Icon], then they
// are displayed using icons instead.
type Chooser struct {
	Button

	// the type of combo box
	Type ChooserTypes `desc:"the type of combo box"`

	// provide a text field for editing the value, or just a button for selecting items?  Set the editable property
	Editable bool `xml:"editable" desc:"provide a text field for editing the value, or just a button for selecting items?  Set the editable property"`

	// TODO(kai): implement AllowNew button

	// whether to allow the user to add new items to the combo box through the editable textfield (if Editable is set to true) and a button at the end of the combo box menu
	AllowNew bool `desc:"whether to allow the user to add new items to the combo box through the editable textfield (if Editable is set to true) and a button at the end of the combo box menu"`

	// current selected value
	CurVal any `json:"-" xml:"-" desc:"current selected value"`

	// current index in list of possible items
	CurIndex int `json:"-" xml:"-" desc:"current index in list of possible items"`

	// items available for selection
	Items []any `json:"-" xml:"-" desc:"items available for selection"`

	// an optional list of tooltips displayed on hover for Chooser items; the indices for tooltips correspond to those for items
	Tooltips []string `json:"-" xml:"-" desc:"an optional list of tooltips displayed on hover for Chooser items; the indices for tooltips correspond to those for items"`

	// if Editable is set to true, text that is displayed in the text field when it is empty, in a lower-contrast manner
	Placeholder string `desc:"if Editable is set to true, text that is displayed in the text field when it is empty, in a lower-contrast manner"`

	// maximum label length (in runes)
	MaxLength int `desc:"maximum label length (in runes)"`
}

func (ch *Chooser) CopyFieldsFrom(frm any) {
	fr := frm.(*Chooser)
	ch.Button.CopyFieldsFrom(&fr.Button)
	ch.Editable = fr.Editable
	ch.CurVal = fr.CurVal
	ch.CurIndex = fr.CurIndex
	ch.Items = fr.Items
	ch.MaxLength = fr.MaxLength
}

// ChooserTypes is an enum containing the
// different possible types of combo boxes
type ChooserTypes int //enums:enum

const (
	// ChooserFilled represents a filled
	// Chooser with a background color
	// and a bottom border
	ChooserFilled ChooserTypes = iota
	// ChooserOutlined represents an outlined
	// Chooser with a border on all sides
	// and no background color
	ChooserOutlined
)

func (ch *Chooser) OnInit() {
	ch.ChooserHandlers()
	ch.ChooserStyles()
}

func (ch *Chooser) ChooserHandlers() {
	ch.ButtonHandlers()
	ch.ChooserKeys()
}

func (ch *Chooser) ChooserStyles() {
	ch.AddStyles(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Activatable, abilities.Focusable, abilities.FocusWithinable, abilities.Hoverable, abilities.LongHoverable)
		s.Cursor = cursors.Pointer
		s.Text.Align = styles.AlignCenter
		if ch.Editable {
			s.Padding.Set()
			s.Padding.Right.SetDp(16)
		} else {
			s.Border.Radius = styles.BorderRadiusExtraSmall
			s.Padding.Set(units.Dp(8), units.Dp(16))
		}
		s.Color = colors.Scheme.OnSurface
		switch ch.Type {
		case ChooserFilled:
			s.BackgroundColor.SetSolid(colors.Scheme.SurfaceContainerHighest)
			if ch.Editable {
				s.Border.Style.Set(styles.BorderNone)
				s.Border.Style.Bottom = styles.BorderSolid
				s.Border.Width.Set()
				s.Border.Width.Bottom = units.Dp(1)
				s.Border.Color.Set()
				s.Border.Color.Bottom = colors.Scheme.OnSurfaceVariant
				s.Border.Radius = styles.BorderRadiusExtraSmallTop
				if s.Is(states.FocusedWithin) {
					s.Border.Width.Bottom = units.Dp(2)
					s.Border.Color.Bottom = colors.Scheme.Primary.Base
				}
			}
		case ChooserOutlined:
			s.Border.Style.Set(styles.BorderSolid)
			s.Border.Width.Set(units.Dp(1))
			s.Border.Color.Set(colors.Scheme.OnSurfaceVariant)
			if ch.Editable {
				s.Border.Radius = styles.BorderRadiusExtraSmall
				if s.Is(states.FocusedWithin) {
					s.Border.Width.Set(units.Dp(2))
					s.Border.Color.Set(colors.Scheme.Primary.Base)
				}
			}
		}
		if s.Is(states.Selected) {
			s.BackgroundColor.SetSolid(colors.Scheme.Select.Container)
		}
		if s.Is(states.Disabled) {
			s.Color = colors.Scheme.SurfaceContainer
		}
	})
}

func (ch *Chooser) OnChildAdded(child ki.Ki) {
	w, _ := AsWidget(child)
	switch w.Name() {
	case "icon":
		w.AddStyles(func(s *styles.Style) {
			s.Margin.Set()
			s.Padding.Set()
		})
	case "label":
		w.AddStyles(func(s *styles.Style) {
			s.SetAbilities(false, abilities.Selectable, abilities.DoubleClickable)
			s.Cursor = cursors.None
			s.Margin.Set()
			s.Padding.Set()
			s.AlignV = styles.AlignMiddle
			if ch.MaxLength > 0 {
				s.SetMinPrefWidth(units.Ch(float32(ch.MaxLength)))
			}
		})
	case "text":
		text := w.(*TextField)
		text.Placeholder = ch.Placeholder
		if ch.Type == ChooserFilled {
			text.Type = TextFieldFilled
		} else {
			text.Type = TextFieldOutlined
		}
		ch.TextFieldHandlers(text)
		text.AddStyles(func(s *styles.Style) {
			s.Border.Style.Set(styles.BorderNone)
			s.Border.Width.Set()
			if ch.MaxLength > 0 {
				s.SetMinPrefWidth(units.Ch(float32(ch.MaxLength)))
			}
		})
	case "ind-stretch":
		w.AddStyles(func(s *styles.Style) {
			if ch.Editable {
				s.Width.SetDp(0)
			} else {
				s.Width.SetDp(16)
			}
		})
	case "indicator":
		w.AddStyles(func(s *styles.Style) {
			s.Font.Size.SetDp(16)
			s.AlignV = styles.AlignMiddle
		})
	}
}

// SetType sets the styling type of the combo box
func (ch *Chooser) SetType(typ ChooserTypes) *Chooser {
	updt := ch.UpdateStart()
	ch.Type = typ
	ch.UpdateEndLayout(updt)
	return ch
}

// SetType sets whether the combo box is editable
func (ch *Chooser) SetEditable(editable bool) *Chooser {
	updt := ch.UpdateStart()
	ch.Editable = editable
	ch.UpdateEndLayout(updt)
	return ch
}

// SetAllowNew sets whether to allow the user to add new values
func (ch *Chooser) SetAllowNew(allowNew bool) *Chooser {
	updt := ch.UpdateStart()
	ch.AllowNew = allowNew
	ch.UpdateEndLayout(updt)
	return ch
}

// ConfigPartsIconText returns a standard config for creating parts, of icon
// and text left-to right in a row -- always makes text
func (ch *Chooser) ConfigPartsIconText(config *ki.Config, icnm icons.Icon) (icIdx, txIdx int) {
	// todo: add some styles for button layout
	icIdx = -1
	txIdx = -1
	if icnm.IsValid() {
		icIdx = len(*config)
		config.Add(IconType, "icon")
		config.Add(SpaceType, "space")
	}
	txIdx = len(*config)
	config.Add(TextFieldType, "text")
	return
}

// ConfigPartsSetText sets part style props, using given props if not set in
// object props
func (ch *Chooser) ConfigPartsSetText(txt string, txIdx, icIdx, indIdx int) {
	if txIdx >= 0 {
		tx := ch.Parts.Child(txIdx).(*TextField)
		tx.SetText(txt)
		tx.SetCompleter(tx, ch.CompleteMatch, ch.CompleteEdit)
	}
}

// ConfigPartsAddIndicatorSpace adds indicator with a space instead of a stretch
// for editable Chooser, where textfield then takes up the rest of the space
func (ch *Chooser) ConfigPartsAddIndicatorSpace(config *ki.Config, defOn bool) int {
	needInd := (ch.HasMenu() || defOn) && ch.Indicator != icons.None
	if !needInd {
		return -1
	}
	indIdx := -1
	config.Add(SpaceType, "ind-stretch")
	indIdx = len(*config)
	config.Add(IconType, "indicator")
	return indIdx
}

func (ch *Chooser) ConfigWidget(sc *Scene) {
	ch.ConfigParts(sc)
}

func (ch *Chooser) ConfigParts(sc *Scene) {
	ch.MakeMenuFunc = ch.MakeItemsMenu
	parts := ch.NewParts(LayoutHoriz)
	config := ki.Config{}
	var icIdx, lbIdx, txIdx, indIdx int
	if ch.Editable {
		lbIdx = -1
		icIdx, txIdx = ch.ConfigPartsIconText(&config, ch.Icon)
		indIdx = ch.ConfigPartsAddIndicatorSpace(&config, true) // use space instead of stretch
	} else {
		txIdx = -1
		icIdx, lbIdx = ch.ConfigPartsIconLabel(&config, ch.Icon, ch.Text)
		indIdx = ch.ConfigPartsAddIndicator(&config, true) // default on
	}
	mods, updt := parts.ConfigChildren(config)
	ch.ConfigPartsSetIconLabel(ch.Icon, ch.Text, icIdx, lbIdx)
	ch.ConfigPartsIndicator(indIdx)
	if txIdx >= 0 {
		ch.ConfigPartsSetText(ch.Text, txIdx, icIdx, indIdx)
	}
	if mods {
		parts.UpdateEnd(updt)
		ch.SetNeedsLayout(sc, updt)
	}
}

// TextField returns the text field of an editable Chooser, and false if not made
func (ch *Chooser) TextField() (*TextField, bool) {
	tff := ch.Parts.ChildByName("text", 2)
	if tff == nil {
		return nil, false
	}
	return tff.(*TextField), true
}

// MakeItems makes sure the Items list is made, and if not, or reset is true,
// creates one with the given capacity
func (ch *Chooser) MakeItems(reset bool, capacity int) {
	if ch.Items == nil || reset {
		ch.Items = make([]any, 0, capacity)
	}
}

// SortItems sorts the items according to their labels
func (ch *Chooser) SortItems(ascending bool) {
	sort.Slice(ch.Items, func(i, j int) bool {
		if ascending {
			return ToLabel(ch.Items[i]) < ToLabel(ch.Items[j])
		} else {
			return ToLabel(ch.Items[i]) > ToLabel(ch.Items[j])
		}
	})
}

// SetToMaxLength gets the maximum label length so that the width of the
// button label is automatically set according to the max length of all items
// in the list -- if maxLen > 0 then it is used as an upper do-not-exceed
// length
func (ch *Chooser) SetToMaxLength(maxLen int) {
	ml := 0
	for _, it := range ch.Items {
		ml = max(ml, utf8.RuneCountInString(ToLabel(it)))
	}
	if maxLen > 0 {
		ml = min(ml, maxLen)
	}
	ch.MaxLength = ml
}

// ItemsFromTypes sets the Items list from a list of types, e.g., from gti.AllEmbedersOf.
// If setFirst then set current item to the first item in the list,
// and maxLen if > 0 auto-sets the width of the button to the
// contents, with the given upper limit.
func (ch *Chooser) ItemsFromTypes(tl []*gti.Type, setFirst, sort bool, maxLen int) *Chooser {
	n := len(tl)
	if n == 0 {
		return ch
	}
	ch.Items = make([]any, n)
	for i, typ := range tl {
		ch.Items[i] = typ
	}
	if sort {
		ch.SortItems(true)
	}
	if maxLen > 0 {
		ch.SetToMaxLength(maxLen)
	}
	if setFirst {
		ch.SetCurIndex(0)
	}
	return ch
}

// ItemsFromStringList sets the Items list from a list of string values.
// If setFirst then set current item to the first item in the list,
// and maxLen if > 0 auto-sets the width of the button to the
// contents, with the given upper limit.
func (ch *Chooser) ItemsFromStringList(el []string, setFirst bool, maxLen int) *Chooser {
	n := len(el)
	if n == 0 {
		return ch
	}
	ch.Items = make([]any, n)
	for i, str := range el {
		ch.Items[i] = str
	}
	if maxLen > 0 {
		ch.SetToMaxLength(maxLen)
	}
	if setFirst {
		ch.SetCurIndex(0)
	}
	return ch
}

// ItemsFromIconList sets the Items list from a list of icons.Icon values.
// If setFirst then set current item to the first item in the list,
// and maxLen if > 0 auto-sets the width of the button to the
// contents, with the given upper limit.
func (ch *Chooser) ItemsFromIconList(el []icons.Icon, setFirst bool, maxLen int) *Chooser {
	n := len(el)
	if n == 0 {
		return ch
	}
	ch.Items = make([]any, n)
	for i, str := range el {
		ch.Items[i] = str
	}
	if maxLen > 0 {
		ch.SetToMaxLength(maxLen)
	}
	if setFirst {
		ch.SetCurIndex(0)
	}
	return ch
}

// ItemsFromEnumListsets the Items list from a list of enums.Enum values.
// If setFirst then set current item to the first item in the list,
// and maxLen if > 0 auto-sets the width of the button to the
// contents, with the given upper limit.
func (ch *Chooser) ItemsFromEnums(el []enums.Enum, setFirst bool, maxLen int) *Chooser {
	n := len(el)
	if n == 0 {
		return ch
	}
	ch.Items = make([]any, n)
	ch.Tooltips = make([]string, n)
	for i, enum := range el {
		ch.Items[i] = enum
		ch.Tooltips[i] = enum.Desc()
	}
	if maxLen > 0 {
		ch.SetToMaxLength(maxLen)
	}
	if setFirst {
		ch.SetCurIndex(0)
	}
	return ch
}

// ItemsFromEnum sets the Items list from given enums.Enum Values().
// If setFirst then set current item to the first item in the list,
// and maxLen if > 0 auto-sets the width of the button to the
// contents, with the given upper limit.
func (ch *Chooser) ItemsFromEnum(enum enums.Enum, setFirst bool, maxLen int) *Chooser {
	return ch.ItemsFromEnums(enum.Values(), setFirst, maxLen)
}

// FindItem finds an item on list of items and returns its index
func (ch *Chooser) FindItem(it any) int {
	if ch.Items == nil {
		return -1
	}
	for i, v := range ch.Items {
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
func (ch *Chooser) SetCurVal(it any) int {
	ch.CurVal = it
	ch.CurIndex = ch.FindItem(it)
	if ch.CurIndex < 0 { // add to list if not found..
		ch.CurIndex = len(ch.Items)
		ch.Items = append(ch.Items, it)
	}
	ch.ShowCurVal()
	return ch.CurIndex
}

// SetCurIndex sets the current index (CurIndex) and the corresponding CurVal
// for that item on the current Items list (-1 if not found) -- returns value
// -- and sets the text to the string value of that value (using standard
// Stringer string conversion)
func (ch *Chooser) SetCurIndex(idx int) any {
	ch.CurIndex = idx
	if idx < 0 || idx >= len(ch.Items) {
		ch.CurVal = nil
		ch.SetText(fmt.Sprintf("idx %v > len", idx))
	} else {
		ch.CurVal = ch.Items[idx]
		ch.ShowCurVal()
	}
	return ch.CurVal
}

// ShowCurVal updates the display to present the
// currently-selected value (CurVal)
func (ch *Chooser) ShowCurVal() {
	if icnm, isic := ch.CurVal.(icons.Icon); isic {
		ch.SetIcon(icnm)
	} else {
		ch.SetText(ToLabel(ch.CurVal))
	}
}

// SelectItem selects a given item and updates the display to it
func (ch *Chooser) SelectItem(idx int) *Chooser {
	if ch.This() == nil {
		return ch
	}
	updt := ch.UpdateStart()
	ch.SetCurIndex(idx)
	tf, ok := ch.TextField()
	if ok {
		tf.SetText(ToLabel(ch.CurVal))
	}
	ch.UpdateEndLayout(updt)
	return ch
}

// SelectItemAction selects a given item and updates the display to it
// and sends a Changed event to indicate that the value has changed.
func (ch *Chooser) SelectItemAction(idx int) {
	if ch.This() == nil {
		return
	}
	ch.SelectItem(idx)
	ch.SendChange()
}

// MakeItemsMenu makes menu of all the items.  It is set as the
// MakeMenuFunc for this Chooser.
func (ch *Chooser) MakeItemsMenu(obj Widget, menu *Menu) {
	nitm := len(ch.Items)
	if ch.Menu == nil {
		ch.Menu = make(Menu, 0, nitm)
	}
	n := len(ch.Menu)
	if nitm < n {
		ch.Menu = ch.Menu[0:nitm]
	}
	if nitm == 0 {
		return
	}
	_, ics := ch.Items[0].(icons.Icon) // if true, we render as icons
	for i, it := range ch.Items {
		var bt *Button
		if n > i {
			bt = ch.Menu[i].(*Button)
		} else {
			bt = &Button{}
			ki.InitNode(bt)
			ch.Menu = append(ch.Menu, bt.This().(Widget))
		}
		nm := "Item_" + strconv.Itoa(i)
		bt.SetName(nm)
		bt.Type = ButtonAction
		if ics {
			bt.Icon = it.(icons.Icon)
			bt.Tooltip = string(bt.Icon)
		} else {
			bt.Text = ToLabel(it)
			if len(ch.Tooltips) > i {
				bt.Tooltip = ch.Tooltips[i]
			}
		}
		bt.Data = i // index is the data
		bt.SetSelected(i == ch.CurIndex)
		bt.SetAsMenu()
		idx := i
		bt.OnClick(func(e events.Event) {
			ch.SelectItemAction(idx)
		})
	}
}

func (ch *Chooser) ChooserKeys() {
	ch.OnKeyChord(func(e events.Event) {
		if ch.StateIs(states.Disabled) {
			return
		}
		if KeyEventTrace {
			fmt.Printf("Chooser KeyChordEvent: %v\n", ch.Path())
		}
		kf := KeyFun(e.KeyChord())
		switch {
		case kf == KeyFunMoveUp:
			e.SetHandled()
			if len(ch.Items) > 0 {
				idx := ch.CurIndex - 1
				if idx < 0 {
					idx += len(ch.Items)
				}
				ch.SelectItemAction(idx)
			}
		case kf == KeyFunMoveDown:
			e.SetHandled()
			if len(ch.Items) > 0 {
				idx := ch.CurIndex + 1
				if idx >= len(ch.Items) {
					idx -= len(ch.Items)
				}
				ch.SelectItemAction(idx)
			}
		case kf == KeyFunPageUp:
			e.SetHandled()
			if len(ch.Items) > 10 {
				idx := ch.CurIndex - 10
				for idx < 0 {
					idx += len(ch.Items)
				}
				ch.SelectItemAction(idx)
			}
		case kf == KeyFunPageDown:
			e.SetHandled()
			if len(ch.Items) > 10 {
				idx := ch.CurIndex + 10
				for idx >= len(ch.Items) {
					idx -= len(ch.Items)
				}
				ch.SelectItemAction(idx)
			}
		case kf == KeyFunEnter || (!ch.Editable && e.KeyRune() == ' '):
			// if !(kt.Rune == ' ' && chb.Sc.Type == ScCompleter) {
			e.SetHandled()
			ch.Send(events.Click, e)
			// }
		}
	})
}

func (ch *Chooser) TextFieldHandlers(tf *TextField) {
	tf.OnChange(func(e events.Event) {
		text := tf.Text()
		for idx, item := range ch.Items {
			if text == ToLabel(item) {
				ch.SetCurIndex(idx)
				ch.SendChange()
				return
			}
		}
		if !ch.AllowNew {
			// TODO: use validation
			slog.Error("invalid Chooser value", "value", text)
			return
		}
		ch.Items = append(ch.Items, text)
		ch.SetCurIndex(len(ch.Items) - 1)
		ch.SendChange()
	})
}

// CompleteMatch is the [complete.MatchFunc] used for the
// editable textfield part of the Chooser (if it exists).
func (ch *Chooser) CompleteMatch(data any, text string, posLn, posCh int) (md complete.Matches) {
	md.Seed = text
	comps := make(complete.Completions, len(ch.Items))
	for idx, item := range ch.Items {
		tooltip := ""
		if len(ch.Tooltips) > idx {
			tooltip = ch.Tooltips[idx]
		}
		comps[idx] = complete.Completion{
			Text: ToLabel(item),
			Desc: tooltip,
		}
	}
	md.Matches = complete.MatchSeedCompletion(comps, md.Seed)
	return md
}

// CompleteEdit is the [complete.EditFunc] used for the
// editable textfield part of the Chooser (if it exists).
func (ch *Chooser) CompleteEdit(data any, text string, cursorPos int, completion complete.Completion, seed string) (ed complete.Edit) {
	return complete.Edit{
		NewText:       completion.Text,
		ForwardDelete: len([]rune(text)),
	}
}
