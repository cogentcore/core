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
	"goki.dev/fi/uri"
	"goki.dev/gi/v2/keyfun"
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
	WidgetBase

	// the type of combo box
	Type ChooserTypes

	// optional icon
	Icon icons.Icon `view:"show-name"`

	// name of the indicator icon to present.
	Indicator icons.Icon `view:"show-name"`

	// provide a text field for editing the value, or just a button for selecting items?  Set the editable property
	Editable bool

	// TODO(kai): implement AllowNew button

	// whether to allow the user to add new items to the combo box through the editable textfield (if Editable is set to true) and a button at the end of the combo box menu
	AllowNew bool

	// CurLabel is the string label for the current value
	CurLabel string `set:"-"`

	// current selected value
	CurVal any `json:"-" xml:"-" set:"-"`

	// current index in list of possible items
	CurIndex int `json:"-" xml:"-" set:"-"`

	// items available for selection
	Items []any `json:"-" xml:"-"`

	// an optional list of icons displayed for Chooser items;
	// the indices for the icons correspond to those for the items
	Icons []icons.Icon `json:"-" xml:"-"`

	// an optional list of tooltips displayed on hover for Chooser items;
	// the indices for the tooltips correspond to those for the items
	Tooltips []string `json:"-" xml:"-"`

	// if Editable is set to true, text that is displayed in the text field when it is empty, in a lower-contrast manner
	Placeholder string `set:"-"`

	// maximum label length (in runes)
	MaxLength int

	// ItemsFunc, if non-nil, is a function to call before showing the items
	// of the chooser, which is typically used to configure them (eg: if they
	// are based on dynamic data)
	ItemsFunc func()
}

func (ch *Chooser) CopyFieldsFrom(frm any) {
	fr := frm.(*Chooser)
	ch.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
	ch.Editable = fr.Editable
	ch.CurVal = fr.CurVal
	ch.CurIndex = fr.CurIndex
	ch.Items = fr.Items
	ch.MaxLength = fr.MaxLength
}

// ChooserTypes is an enum containing the
// different possible types of combo boxes
type ChooserTypes int32 //enums:enum -trim-prefix Chooser

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
	ch.HandleChooserEvents()
	ch.ChooserStyles()
}

func (ch *Chooser) ChooserStyles() {
	ch.Icon = icons.None
	ch.Indicator = icons.KeyboardArrowDown
	ch.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Activatable, abilities.FocusWithinable, abilities.Hoverable, abilities.LongHoverable)
		if !ch.Editable {
			s.SetAbilities(true, abilities.Focusable)
		}
		s.Cursor = cursors.Pointer
		s.Text.Align = styles.Center
		s.Border.Radius = styles.BorderRadiusSmall
		s.Padding.Set(units.Dp(8), units.Dp(16))
		switch ch.Type {
		case ChooserFilled:
			s.BackgroundColor.SetSolid(colors.Scheme.Secondary.Container)
			s.Color = colors.Scheme.Secondary.OnContainer
			if ch.Editable {
				s.Border.Style.Set(styles.BorderNone).SetBottom(styles.BorderSolid)
				s.Border.Width.Zero().SetBottom(units.Dp(1))
				s.Border.Color.Zero().SetBottom(colors.Scheme.OnSurfaceVariant)
				s.Border.Radius = styles.BorderRadiusExtraSmallTop
				if s.Is(states.Focused) {
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
				if s.Is(states.Focused) {
					s.Border.Width.Set(units.Dp(2))
					s.Border.Color.Set(colors.Scheme.Primary.Base)
				}
			}
		}
	})
	ch.OnWidgetAdded(func(w Widget) {
		switch w.PathFrom(ch) {
		case "parts":
			w.Style(func(s *styles.Style) {
				s.Align.Content = styles.Center
				s.Align.Items = styles.Center
			})
		case "parts/icon":
			w.Style(func(s *styles.Style) {
				s.Margin.Zero()
				s.Padding.Zero()
			})
		case "parts/label":
			w.Style(func(s *styles.Style) {
				s.SetNonSelectable()
				s.SetTextWrap(false)
				s.Margin.Zero()
				s.Padding.Zero()
				// TODO(kai): figure out what to do with MaxLength
				// if ch.MaxLength > 0 {
				// 	s.Min.X.Ch(float32(ch.MaxLength))
				// }
			})
		case "parts/text":
			text := w.(*TextField)
			text.Placeholder = ch.Placeholder
			if ch.Type == ChooserFilled {
				text.Type = TextFieldFilled
			} else {
				text.Type = TextFieldOutlined
			}
			ch.HandleChooserTextFieldEvents(text)
			text.Style(func(s *styles.Style) {
				s.Grow = ch.Styles.Grow // we grow like our parent
				// parent handles everything
				s.Min.Y.Em(1.2) // note: this is essential
				// TODO(kai): figure out what to do with MaxLength
				// if ch.MaxLength > 0 {
				// 	s.Min.X.Ch(float32(ch.MaxLength))
				// }
				s.Padding.Zero()
				s.Border.Style.Set(styles.BorderNone)
				s.Border.Width.Zero()
				// must stay consistent with parent
				s.StateLayer = ch.Styles.StateLayer
				s.StateColor = ch.Styles.Color
				s.BackgroundColor.SetSolid(colors.Transparent)
				// if ch.MaxLength > 0 {
				// 	s.Min.X.Ch(float32(ch.MaxLength))
				// }
			})
		case "parts/indicator":
			w.Style(func(s *styles.Style) {
				s.Font.Size.Dp(16)
				s.Min.X.Em(1)
				s.Min.Y.Em(1)
				s.Justify.Self = styles.End
				s.Align.Self = styles.Center
			})
		}
	})
}

func (ch *Chooser) ConfigWidget() {
	ch.ConfigParts()
}

func (ch *Chooser) ConfigParts() {
	parts := ch.NewParts()
	config := ki.Config{}

	icIdx := -1
	var lbIdx, txIdx, indIdx int
	if ch.Icon.IsValid() {
		config.Add(IconType, "icon")
		config.Add(SpaceType, "space")
		icIdx = 0
	}
	if ch.Editable {
		lbIdx = -1
		txIdx = len(config)
		config.Add(TextFieldType, "text")
	} else {
		txIdx = -1
		lbIdx = len(config)
		config.Add(LabelType, "label")
	}
	if !ch.Indicator.IsValid() {
		ch.Indicator = icons.KeyboardArrowRight
	}
	indIdx = len(config)
	config.Add(IconType, "indicator")

	mods, updt := parts.ConfigChildren(config)

	if icIdx >= 0 {
		ic := ch.Parts.Child(icIdx).(*Icon)
		ic.SetIcon(ch.Icon)
	}
	if ch.Editable {
		tx := ch.Parts.Child(txIdx).(*TextField)
		tx.SetText(ch.CurLabel)
		tx.Config() // this is essential
		tx.SetCompleter(tx, ch.CompleteMatch, ch.CompleteEdit)
	} else {
		lbl := ch.Parts.Child(lbIdx).(*Label)
		lbl.SetText(ch.CurLabel)
		lbl.Config() // this is essential
	}
	{ // indicator
		ic := ch.Parts.Child(indIdx).(*Icon)
		ic.SetIcon(ch.Indicator)
	}
	if mods {
		parts.Update()
		parts.UpdateEnd(updt)
		ch.SetNeedsLayout(updt)
	}
}

// LabelWidget returns the label widget if present
func (ch *Chooser) LabelWidget() *Label {
	if ch.Parts == nil {
		return nil
	}
	lbi := ch.Parts.ChildByName("label")
	if lbi == nil {
		return nil
	}
	return lbi.(*Label)
}

// IconWidget returns the icon widget if present
func (ch *Chooser) IconWidget() *Icon {
	if ch.Parts == nil {
		return nil
	}
	ici := ch.Parts.ChildByName("icon")
	if ici == nil {
		return nil
	}
	return ici.(*Icon)
}

// TextField returns the text field of an editable Chooser
// if present
func (ch *Chooser) TextField() (*TextField, bool) {
	if ch.Parts == nil {
		return nil, false
	}
	tf := ch.Parts.ChildByName("text", 2)
	if tf == nil {
		return nil, false
	}
	return tf.(*TextField), true
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

// SetTypes sets the Items list from a list of types, e.g., from gti.AllEmbedersOf.
// If setFirst then set current item to the first item in the list,
// and maxLen if > 0 auto-sets the width of the button to the
// contents, with the given upper limit.
func (ch *Chooser) SetTypes(tl []*gti.Type, setFirst, sort bool, maxLen int) *Chooser {
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

// SetStrings sets the Items list from a list of string values.
// If setFirst then set current item to the first item in the list,
// and maxLen if > 0 auto-sets the width of the button to the
// contents, with the given upper limit.
func (ch *Chooser) SetStrings(el []string, setFirst bool, maxLen int) *Chooser {
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

// SetIcons sets the Items list from a list of icons.Icon values.
// If setFirst then set current item to the first item in the list,
// and maxLen if > 0 auto-sets the width of the button to the
// contents, with the given upper limit.
func (ch *Chooser) SetIcons(el []icons.Icon, setFirst bool, maxLen int) *Chooser {
	n := len(el)
	if n == 0 {
		return ch
	}
	ch.Items = make([]any, n)
	for i, ic := range el {
		ch.Items[i] = ic
		ch.Icons[i] = ic
	}
	if maxLen > 0 {
		ch.SetToMaxLength(maxLen)
	}
	if setFirst {
		ch.SetCurIndex(0)
	}
	return ch
}

// SetEnums sets the Items list from a list of enums.Enum values.
// If setFirst then set current item to the first item in the list,
// and maxLen if > 0 auto-sets the width of the button to the
// contents, with the given upper limit.
func (ch *Chooser) SetEnums(el []enums.Enum, setFirst bool, maxLen int) *Chooser {
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

// SetEnum sets the Items list from given enums.Enum Values().
// If setFirst then set current item to the first item in the list,
// and maxLen if > 0 auto-sets the width of the button to the
// contents, with the given upper limit.
func (ch *Chooser) SetEnum(enum enums.Enum, setFirst bool, maxLen int) *Chooser {
	return ch.SetEnums(enum.Values(), setFirst, maxLen)
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

// SetPlaceholder sets the given placeholder text and
// CurIndex = -1, indicating that nothing has not been selected.
func (ch *Chooser) SetPlaceholder(text string) *Chooser {
	ch.Placeholder = text
	if !ch.Editable {
		ch.ShowCurVal(text)
	}
	ch.CurIndex = -1
	return ch
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
	ch.ShowCurVal(ToLabel(ch.CurVal))
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
		ch.ShowCurVal(fmt.Sprintf("idx %v > len", idx))
	} else {
		ch.CurVal = ch.Items[idx]
		ch.ShowCurVal(ToLabel(ch.CurVal))
	}
	return ch.CurVal
}

// GetCurTextAction is for Editable choosers only: sets the current index (CurIndex)
// and the corresponding CurVal based on current user-entered Text value,
// and triggers a Change event
func (ch *Chooser) GetCurTextAction() any {
	tf, ok := ch.TextField()
	if !ok {
		slog.Error("gi.Chooser: GetCurTextAction only available for Editable Chooser")
		return ch.CurVal
	}
	ch.SetCurTextAction(tf.Text())
	return ch.CurVal
}

// SetCurTextAction is for Editable choosers only: sets the current index (CurIndex)
// and the corresponding CurVal based on given text string,
// and triggers a Change event
func (ch *Chooser) SetCurTextAction(text string) any {
	ch.SetCurText(text)
	ch.SendChange(nil)
	return ch.CurVal
}

// SetCurText is for Editable choosers only: sets the current index (CurIndex)
// and the corresponding CurVal based on given text string
func (ch *Chooser) SetCurText(text string) any {
	for idx, item := range ch.Items {
		if text == ToLabel(item) {
			ch.SetCurIndex(idx)
			return ch.CurVal
		}
	}
	if !ch.AllowNew {
		// TODO: use validation
		slog.Error("invalid Chooser value", "value", text)
		return ch.CurVal
	}
	ch.Items = append(ch.Items, text)
	ch.SetCurIndex(len(ch.Items) - 1)
	return ch.CurVal
}

// ShowCurVal updates the display to present the
// currently-selected value (CurVal)
func (ch *Chooser) ShowCurVal(label string) {
	updt := ch.UpdateStart()
	defer ch.UpdateEndRender(updt)

	ch.CurLabel = label
	if ch.Editable {
		tf, ok := ch.TextField()
		if ok {
			tf.SetTextUpdate(ch.CurLabel)
		}
	} else {
		lbl := ch.LabelWidget()
		if lbl != nil {
			lbl.SetTextUpdate(ch.CurLabel)
		}
	}
	if ch.CurIndex < len(ch.Icons) {
		picon := ch.Icon
		ch.SetIcon(ch.Icons[ch.CurIndex])
		if ch.Icon != picon {
			ch.Update()
			ch.SetNeedsLayout(true)
		}
	}
}

// SelectItem selects a given item and updates the display to it
func (ch *Chooser) SelectItem(idx int) *Chooser {
	if ch.This() == nil {
		return ch
	}
	updt := ch.UpdateStart()
	ch.SetCurIndex(idx)
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

// MakeItemsMenu constructs a menu of all the items.
// It is automatically set as the [Button.Menu] for the Chooser.
func (ch *Chooser) MakeItemsMenu(m *Scene) {
	if ch.ItemsFunc != nil {
		ch.ItemsFunc()
	}
	for i, it := range ch.Items {
		nm := "item-" + strconv.Itoa(i)
		bt := NewButton(m, nm).SetType(ButtonMenu)
		bt.SetText(ToLabel(it))
		if len(ch.Icons) > i {
			bt.SetIcon(ch.Icons[i])
		}
		if len(ch.Tooltips) > i {
			bt.SetTooltip(ch.Tooltips[i])
		}
		bt.Data = i // index is the data
		bt.SetSelected(i == ch.CurIndex)
		idx := i
		bt.OnClick(func(e events.Event) {
			ch.SelectItemAction(idx)
		})
	}
}

func (ch *Chooser) HandleChooserEvents() {
	ch.HandleWidgetEvents()
	ch.HandleSelectToggle()
	ch.HandleClickMenu()
	ch.HandleChooserKeys()
}

func (ch *Chooser) HandleClickMenu() {
	ch.OnClick(func(e events.Event) {
		if ch.OpenMenu(e) {
			e.SetHandled()
		}
	})
}

// OpenMenu will open any menu associated with this element.
// Returns true if menu opened, false if not.
func (ch *Chooser) OpenMenu(e events.Event) bool {
	pos := ch.ContextMenuPos(e)
	if ch.Parts != nil {
		if indic := ch.Parts.ChildByName("indicator", 3); indic != nil {
			pos = indic.(Widget).ContextMenuPos(nil) // use the pos
		}
	}
	m := NewMenu(ch.MakeItemsMenu, ch.This().(Widget), pos)
	if m == nil {
		return false
	}
	m.Run()
	return true
}

func (ch *Chooser) HandleChooserKeys() {
	ch.OnKeyChord(func(e events.Event) {
		if KeyEventTrace {
			fmt.Printf("Chooser KeyChordEvent: %v\n", ch.Path())
		}
		kf := keyfun.Of(e.KeyChord())
		switch {
		case kf == keyfun.MoveUp:
			e.SetHandled()
			if len(ch.Items) > 0 {
				idx := ch.CurIndex - 1
				if idx < 0 {
					idx += len(ch.Items)
				}
				ch.SelectItemAction(idx)
			}
		case kf == keyfun.MoveDown:
			e.SetHandled()
			if len(ch.Items) > 0 {
				idx := ch.CurIndex + 1
				if idx >= len(ch.Items) {
					idx -= len(ch.Items)
				}
				ch.SelectItemAction(idx)
			}
		case kf == keyfun.PageUp:
			e.SetHandled()
			if len(ch.Items) > 10 {
				idx := ch.CurIndex - 10
				for idx < 0 {
					idx += len(ch.Items)
				}
				ch.SelectItemAction(idx)
			}
		case kf == keyfun.PageDown:
			e.SetHandled()
			if len(ch.Items) > 10 {
				idx := ch.CurIndex + 10
				for idx >= len(ch.Items) {
					idx -= len(ch.Items)
				}
				ch.SelectItemAction(idx)
			}
		case kf == keyfun.Menu:
			if len(ch.Items) > 0 {
				e.SetHandled()
				ch.OpenMenu(e)
			}
		case kf == keyfun.Enter || (!ch.Editable && e.KeyRune() == ' '):
			// if !(kt.Rune == ' ' && chb.Sc.Type == ScCompleter) {
			e.SetHandled()
			ch.Send(events.Click, e)
		// }
		default:
			tf, ok := ch.TextField()
			if !ok {
				break
			}
			// if we don't have anything special to do,
			// we just give our key event to our textfield
			tf.HandleEvent(e)
		}
	})
}

func (ch *Chooser) HandleChooserTextFieldEvents(tf *TextField) {
	tf.OnChange(func(e events.Event) {
		ch.SetCurText(tf.Text())
		ch.SendChange(e)
	})
	tf.OnFocus(func(e events.Event) {
		if ch.ItemsFunc != nil {
			ch.ItemsFunc()
			if ch.CurIndex <= 0 {
				ch.SetCurIndex(0)
			}
		}
		tf.CursorStart()
	})
	tf.OnClick(func(e events.Event) {
		if ch.IsReadOnly() {
			return
		}
		tf.FocusClear()
		ch.SetFocusEvent()
		ch.Send(events.Focus, e)
	})
	// Chooser gives its textfield focus styling but not actual focus
	ch.OnFocus(func(e events.Event) {
		tf.SetState(true, states.Focused)
	})
	ch.OnFocusLost(func(e events.Event) {
		tf.SetState(false, states.Focused)
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
		if u, ok := item.(uri.URI); ok {
			comps[idx].Icon = string(u.Icon)
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

func (ch *Chooser) RenderChooser() {
	_, st := ch.RenderLock()
	ch.RenderStdBox(st)
	ch.RenderUnlock()
}

func (ch *Chooser) Render() {
	if ch.PushBounds() {
		ch.RenderChooser()
		ch.RenderParts()
		ch.RenderChildren()
		ch.PopBounds()
	}
}
