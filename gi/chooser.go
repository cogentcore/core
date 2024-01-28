// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"log/slog"
	"sort"
	"strconv"

	"cogentcore.org/core/abilities"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/enums"
	"cogentcore.org/core/events"
	"cogentcore.org/core/glop/sentence"
	"cogentcore.org/core/gti"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keyfun"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/pi/complete"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
)

// Chooser is for selecting items from a dropdown list, with an optional
// edit TextField for typing directly.
// The items can be of any type, including enum values -- they are converted
// to strings for the display.  If the items are of type [icons.Icon], then they
// are displayed using icons instead.
type Chooser struct {
	Box

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

	// items available for selection
	Items []any `json:"-" xml:"-"`

	// an optional list of labels displayed for Chooser items;
	// the indices for the labels correspond to those for the items
	Labels []string `json:"-" xml:"-"`

	// an optional list of icons displayed for Chooser items;
	// the indices for the icons correspond to those for the items
	Icons []icons.Icon `json:"-" xml:"-"`

	// an optional list of tooltips displayed on hover for Chooser items;
	// the indices for the tooltips correspond to those for the items
	Tooltips []string `json:"-" xml:"-"`

	// if Editable is set to true, text that is displayed in the text field when it is empty, in a lower-contrast manner
	Placeholder string `set:"-"`

	// ItemsFuncs is a slice of functions to call before showing the items
	// of the chooser, which is typically used to configure them
	// (eg: if they are based on dynamic data). The functions are called
	// in ascending order such that the items added in the first function
	// will appear before those added in the last function. Use
	// [Chooser.AddItemsFunc] to add a new items function. If at least
	// one ItemsFunc is specified, the items, labels, icons, and tooltips
	// of the chooser will be cleared before calling the functions.
	ItemsFuncs []func() `copier:"-" set:"-"`

	// CurLabel is the string label for the current value
	CurLabel string `set:"-"`

	// current selected value
	CurVal any `json:"-" xml:"-" set:"-"`

	// current index in list of possible items
	CurIndex int `json:"-" xml:"-" set:"-"`
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
	ch.Box.OnInit()
	ch.HandleEvents()
	ch.SetStyles()
}

func (ch *Chooser) SetStyles() {
	ch.Icon = icons.None
	ch.Indicator = icons.KeyboardArrowDown
	ch.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Activatable, abilities.Hoverable, abilities.LongHoverable)
		if !ch.Editable {
			s.SetAbilities(true, abilities.Focusable)
		}
		s.Cursor = cursors.Pointer
		s.Text.Align = styles.Center
		s.Border.Radius = styles.BorderRadiusSmall
		s.Padding.Set(units.Dp(8), units.Dp(16))
		switch ch.Type {
		case ChooserFilled:
			s.Background = colors.C(colors.Scheme.Secondary.Container)
			s.Color = colors.Scheme.Secondary.OnContainer
		case ChooserOutlined:
			s.Border.Style.Set(styles.BorderSolid)
			s.Border.Width.Set(units.Dp(1))
			s.Border.Color.Set(colors.Scheme.OnSurfaceVariant)
		}
		// textfield handles everything
		if ch.Editable {
			s.Border = styles.Border{}
			s.MaxBorder = s.Border
			s.Background = nil
			s.StateLayer = 0
			s.Padding.Zero()
			if ch.Type == ChooserFilled {
				s.Border.Radius = styles.BorderRadiusExtraSmallTop
			} else {
				s.Border.Radius = styles.BorderRadiusExtraSmall
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
				s.Margin.Right.Ch(1)
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
	config := ki.Config{}

	ici := -1
	var lbi, txi, indi int
	// editable handles through textfield
	if ch.Icon.IsSet() && !ch.Editable {
		config.Add(IconType, "icon")
		ici = 0
	}
	if ch.Editable {
		lbi = -1
		txi = len(config)
		config.Add(TextFieldType, "text")
	} else {
		txi = -1
		lbi = len(config)
		config.Add(LabelType, "label")
	}
	if !ch.Indicator.IsSet() {
		ch.Indicator = icons.KeyboardArrowRight
	}
	// editable handles through textfield
	if !ch.Editable {
		indi = len(config)
		config.Add(IconType, "indicator")
	}

	ch.ConfigParts(config, func() {
		if ici >= 0 {
			ic := ch.Parts.Child(ici).(*Icon)
			ic.SetIcon(ch.Icon)
		}
		if ch.Editable {
			tx := ch.Parts.Child(txi).(*TextField)
			tx.SetText(ch.CurLabel)
			tx.SetLeadingIcon(ch.Icon)
			tx.SetTrailingIcon(ch.Indicator, func(e events.Event) {
				ch.OpenMenu(e)
			})
			tx.Config() // this is essential
			tx.SetCompleter(tx, ch.CompleteMatch, ch.CompleteEdit)
		} else {
			lbl := ch.Parts.Child(lbi).(*Label)
			lbl.SetText(ch.CurLabel)
			lbl.Config() // this is essential

			ic := ch.Parts.Child(indi).(*Icon)
			ic.SetIcon(ch.Indicator)
		}
	})
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

// SetIconUpdate sets the icon and drives an update, for the already-displayed case.
func (ch *Chooser) SetIconUpdate(ic icons.Icon) *Chooser {
	updt := ch.UpdateStart()
	defer ch.UpdateEndRender(updt)

	ch.Icon = ic
	if ch.Editable {
		tf := ch.TextField()
		if tf != nil {
			tf.SetLeadingIconUpdate(ic)
		}
	} else {
		iw := ch.IconWidget()
		if iw != nil {
			iw.SetIconUpdate(ic)
		}
	}
	return ch
}

// TextField returns the text field of an editable Chooser
// if present
func (ch *Chooser) TextField() *TextField {
	if ch.Parts == nil {
		return nil
	}
	tf := ch.Parts.ChildByName("text", 2)
	if tf == nil {
		return nil
	}
	return tf.(*TextField)
}

// AddItemsFunc adds the given function to [Chooser.ItemsFuncs].
// These functions are called before showing the items of the chooser,
// and they are typically used to configure them (eg: if they are based
// on dynamic data). The functions are called in ascending order such
// that the items added in the first function will appear before those
// added in the last function. If at least one ItemsFunc is specified,
// the items, labels, icons, and tooltips of the chooser will be cleared
// before calling the functions.
func (ch *Chooser) AddItemsFunc(f func()) *Chooser {
	ch.ItemsFuncs = append(ch.ItemsFuncs, f)
	return ch
}

// CallItemsFuncs calls [Chooser.ItemsFuncs].
func (ch *Chooser) CallItemsFuncs() {
	if len(ch.ItemsFuncs) == 0 {
		return
	}
	ch.Items = nil
	ch.Labels = nil
	ch.Icons = nil
	ch.Tooltips = nil
	for _, f := range ch.ItemsFuncs {
		f()
	}
}

// MakeItems makes sure the Items list is made, and if not, or reset is true,
// creates one with the given capacity
func (ch *Chooser) MakeItems(reset bool, capacity int) {
	if ch.Items == nil || reset {
		ch.Items = make([]any, 0, capacity)
	}
}

// SortItems sorts the items according to their labels
func (ch *Chooser) SortItems(ascending bool) *Chooser {
	sort.Slice(ch.Items, func(i, j int) bool {
		if ascending {
			return ch.LabelFor(i, ch.Items[i]) < ch.LabelFor(j, ch.Items[j])
		} else {
			return ch.LabelFor(i, ch.Items[i]) > ch.LabelFor(j, ch.Items[j])
		}
	})
	return ch
}

// LabelFor converts the given item at the given index into
// a user-facing label. It first tries [Chooser.Labels] and
// falls back on [ToLabel].
func (ch *Chooser) LabelFor(i int, item any) string {
	if len(ch.Labels) > i {
		return ch.Labels[i]
	}
	return ToLabel(item)
}

// SetTypes sets the Items list from a list of types, e.g., from gti.AllEmbedersOf.
// If setFirst then set current item to the first item in the list, and if sort
// then sort the list items in ascending order according to their labels.
func (ch *Chooser) SetTypes(tl []*gti.Type, setFirst, sort bool) *Chooser {
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
	if setFirst {
		ch.SetCurIndex(0)
	}
	return ch
}

// SetStrings sets the Items list from a list of string values.
// If setFirst then set current item to the first item in the list.
func (ch *Chooser) SetStrings(el []string, setFirst bool) *Chooser {
	n := len(el)
	if n == 0 {
		return ch
	}
	ch.Items = make([]any, n)
	for i, str := range el {
		ch.Items[i] = str
	}
	if setFirst {
		ch.SetCurIndex(0)
	}
	return ch
}

// SetIconItems sets the Items list from a list of icons.Icon values.
// If setFirst then set current item to the first item in the list.
func (ch *Chooser) SetIconItems(el []icons.Icon, setFirst bool) *Chooser {
	n := len(el)
	if n == 0 {
		return ch
	}
	ch.Items = make([]any, n)
	ch.Labels = make([]string, n)
	ch.Icons = make([]icons.Icon, n)
	for i, ic := range el {
		ch.Items[i] = ic
		ch.Labels[i] = sentence.Case(string(ic))
		ch.Icons[i] = ic
	}
	if setFirst {
		ch.SetCurIndex(0)
	}
	return ch
}

// SetEnums sets the Items list from a list of enums.Enum values.
// If setFirst then set current item to the first item in the list.
func (ch *Chooser) SetEnums(el []enums.Enum, setFirst bool) *Chooser {
	n := len(el)
	if n == 0 {
		return ch
	}
	ch.Items = make([]any, n)
	ch.Labels = make([]string, n)
	ch.Tooltips = make([]string, n)
	for i, enum := range el {
		ch.Items[i] = enum
		str := enum.String()
		lbl := sentence.Case(str)
		ch.Labels[i] = lbl
		// TODO(kai): this desc is not always correct because we
		// don't have the name of the enum value pre-generator-transformation
		// (same as with Switches) (#774)
		ch.Tooltips[i] = sentence.Doc(enum.Desc(), str, lbl)
	}
	if setFirst {
		ch.SetCurIndex(0)
	}
	return ch
}

// SetEnum sets the Items list from given enums.Enum Values().
// If setFirst then set current item to the first item in the list.
func (ch *Chooser) SetEnum(enum enums.Enum, setFirst bool) *Chooser {
	return ch.SetEnums(enum.Values(), setFirst)
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
	ch.ShowCurVal(ch.LabelFor(ch.CurIndex, ch.CurVal))
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
		ch.ShowCurVal(ch.LabelFor(ch.CurIndex, ch.CurVal))
	}
	return ch.CurVal
}

// GetCurTextAction is for Editable choosers only: sets the current index (CurIndex)
// and the corresponding CurVal based on current user-entered Text value,
// and triggers a Change event
func (ch *Chooser) GetCurTextAction() any {
	tf := ch.TextField()
	if tf == nil {
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
	for i, item := range ch.Items {
		if text == ch.LabelFor(i, item) {
			ch.SetCurIndex(i)
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
		tf := ch.TextField()
		if tf != nil {
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
	if ch.CurIndex < len(ch.Tooltips) {
		ch.SetTooltip(ch.Tooltips[ch.CurIndex])
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
	ch.CallItemsFuncs()
	for i, it := range ch.Items {
		nm := "item-" + strconv.Itoa(i)
		bt := NewButton(m, nm).SetType(ButtonMenu)
		bt.SetText(ch.LabelFor(i, it))
		if len(ch.Icons) > i {
			bt.SetIcon(ch.Icons[i])
		}
		if len(ch.Tooltips) > i {
			bt.SetTooltip(ch.Tooltips[i])
		}
		bt.SetSelected(i == ch.CurIndex)
		idx := i
		bt.OnClick(func(e events.Event) {
			ch.SelectItemAction(idx)
		})
	}
}

func (ch *Chooser) HandleEvents() {
	ch.HandleSelectToggle()
	ch.HandleKeys()

	ch.OnClick(func(e events.Event) {
		if ch.OpenMenu(e) {
			e.SetHandled()
		}
	})
}

// OpenMenu will open any menu associated with this chooser.
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

func (ch *Chooser) HandleKeys() {
	ch.OnFinal(events.KeyChord, func(e events.Event) {
		if DebugSettings.KeyEventTrace {
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
		case kf == keyfun.Enter || (!ch.Editable && e.KeyRune() == ' '):
			// if !(kt.Rune == ' ' && chb.Sc.Type == ScCompleter) {
			e.SetHandled()
			ch.Send(events.Click, e)
		// }
		default:
			tf := ch.TextField()
			if tf == nil {
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
		if ch.IsReadOnly() {
			return
		}
		ch.CallItemsFuncs()
		tf.OfferComplete(dontForce)
	})
}

// CompleteMatch is the [complete.MatchFunc] used for the
// editable textfield part of the Chooser (if it exists).
func (ch *Chooser) CompleteMatch(data any, text string, posLn, posCh int) (md complete.Matches) {
	md.Seed = text
	comps := make(complete.Completions, len(ch.Items))
	for i, item := range ch.Items {
		tooltip := ""
		if len(ch.Tooltips) > i {
			tooltip = ch.Tooltips[i]
		}
		icon := ""
		if len(ch.Icons) > i {
			icon = string(ch.Icons[i])
		}
		comps[i] = complete.Completion{
			Text: ch.LabelFor(i, item),
			Desc: tooltip,
			Icon: icon,
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
