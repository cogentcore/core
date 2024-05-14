// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"errors"
	"fmt"
	"image"
	"log/slog"
	"slices"
	"strings"
	"unicode"

	"cogentcore.org/core/base/strcase"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/enums"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/parse/complete"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/types"
)

// Chooser is a drop down selection widget that allows users to choose
// one option among a list of items.
type Chooser struct {
	Frame

	// Type is the styling type of the chooser.
	Type ChooserTypes

	// Items are the chooser items available for selection.
	Items []ChooserItem

	// Icon is an optional icon displayed on the left side of the chooser.
	Icon icons.Icon `view:"show-name"`

	// Indicator is the icon to use for the indicator displayed on the
	// right side of the chooser.
	Indicator icons.Icon `view:"show-name"`

	// Editable is whether provide a text field for editing the value,
	// or just a button for selecting items.
	Editable bool

	// AllowNew is whether to allow the user to add new items to the
	// chooser through the editable textfield (if Editable is set to
	// true) and a button at the end of the chooser menu. See also [DefaultNew].
	AllowNew bool

	// DefaultNew configures the chooser to accept new items, as in
	// [AllowNew], and also turns off completion popups and always
	// adds new items to the list of items, without prompting.
	// Use this for cases where the typical use-case is to enter new values,
	// but the history of prior values can also be useful.
	DefaultNew bool

	// Placeholder, if Editable is set to true, is the text that is
	// displayed in the text field when it is empty. It must be set
	// using [Chooser.SetPlaceholder].
	Placeholder string `set:"-"`

	// ItemsFuncs is a slice of functions to call before showing the items
	// of the chooser, which is typically used to configure them
	// (eg: if they are based on dynamic data). The functions are called
	// in ascending order such that the items added in the first function
	// will appear before those added in the last function. Use
	// [Chooser.AddItemsFunc] to add a new items function. If at least
	// one ItemsFunc is specified, the items of the chooser will be
	// cleared before calling the functions.
	ItemsFuncs []func() `copier:"-" json:"-" xml:"-" set:"-"`

	// CurrentItem is the currently selected item.
	CurrentItem ChooserItem `json:"-" xml:"-" set:"-"`

	// CurrentIndex is the index of the currently selected item
	// in [Chooser.Items].
	CurrentIndex int `json:"-" xml:"-" set:"-"`
}

// ChooserItem is an item that can be used in a [Chooser].
type ChooserItem struct { //types:add

	// Value is the underlying value of the chooser item.
	Value any

	// Text is the text displayed to the user for this item.
	// If it is empty, then [ToLabel] of [ChooserItem.Value] is
	// used instead.
	Text string

	// Icon is the icon displayed to the user for this item.
	Icon icons.Icon

	// Tooltip is the tooltip displayed to the user for this item.
	Tooltip string

	// Func, if non-nil, is a function to call whenever this
	// item is selected as the current value of the chooser.
	Func func() `json:"-" xml:"-"`

	// SeparatorBefore is whether to add a separator before
	// this item in the chooser menu.
	SeparatorBefore bool
}

// GetLabel returns the effective label of this chooser item.
// If [ChooserItem.Label] is set, it returns that. Otherwise,
// it uses [ToLabel] of [ChooserItem.Value]
func (ci *ChooserItem) GetLabel() string {
	if ci.Text != "" {
		return ci.Text
	}
	if ci.Value == nil {
		return ""
	}
	return ToLabel(ci.Value)
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

func (ch *Chooser) WidgetValue() any { return &ch.CurrentItem.Value }

func (ch *Chooser) SetWidgetValue(value any) error {
	ch.SetCurrentValue(value)
	return nil
}

func (ch *Chooser) OnBind(value any) {
	if e, ok := value.(enums.Enum); ok {
		ch.SetEnum(e)
	}
}

func (ch *Chooser) OnInit() {
	ch.WidgetBase.OnInit()
	ch.HandleEvents()
	ch.SetStyles()
}

func (ch *Chooser) SetStyles() {
	ch.Icon = icons.None
	ch.Indicator = icons.KeyboardArrowDown
	ch.CurrentIndex = -1
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
			s.Color = colors.C(colors.Scheme.Secondary.OnContainer)
		case ChooserOutlined:
			if !s.Is(states.Focused) {
				s.Border.Style.Set(styles.BorderSolid)
				s.Border.Width.Set(units.Dp(1))
				s.Border.Color.Set(colors.C(colors.Scheme.OnSurfaceVariant))
			}
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
		case "text-field/trail-icon": // TODO(config): need some way to do this with the new config paradigm
			w.Style(func(s *styles.Style) {
				// indicator does not need to be focused
				s.SetAbilities(false, abilities.Focusable)
			})
		}
	})
}

func (ch *Chooser) Config(c *Config) {
	// automatically select the first item if we have nothing selected and no placeholder
	if !ch.Editable && ch.CurrentIndex < 0 && ch.CurrentItem.Text == "" {
		ch.SetCurrentIndex(0)
	}

	// editable handles through TextField
	if ch.Icon.IsSet() && !ch.Editable {
		AddConfig(c, "icon", func() *Icon {
			return NewIcon()
		}, func(w *Icon) {
			w.SetIcon(ch.Icon)
		})
	}
	if ch.Editable {
		AddConfig(c, "text-field",
			func() *TextField {
				w := NewTextField().SetPlaceholder(ch.Placeholder)
				ch.HandleChooserTextFieldEvents(w)
				w.Style(func(s *styles.Style) {
					s.Grow = ch.Styles.Grow // we grow like our parent
					s.Max.X.Zero()          // constrained by parent
					s.SetTextWrap(false)
				})
				return w
			}, func(w *TextField) {
				w.SetText(ch.CurrentItem.GetLabel()).SetLeadingIcon(ch.Icon).
					SetTrailingIcon(ch.Indicator, func(e events.Event) {
						ch.OpenMenu(e)
					})
				if ch.Type == ChooserFilled {
					w.SetType(TextFieldFilled)
				} else {
					w.SetType(TextFieldOutlined)
				}
				w.ConfigWidget() // this is actually essential (TODO: figure out a way to get rid of this?)
				if !ch.DefaultNew {
					w.SetCompleter(w, ch.CompleteMatch, ch.CompleteEdit)
				}
			})
	} else {
		AddConfig(c, "text",
			func() *Text {
				w := NewText()
				w.Style(func(s *styles.Style) {
					s.SetNonSelectable()
					s.SetTextWrap(false)
				})
				return w
			}, func(w *Text) {
				w.SetText(ch.CurrentItem.GetLabel())
			})
	}
	if ch.Indicator == "" {
		ch.Indicator = icons.KeyboardArrowRight
	}
	// editable handles through TextField
	if !ch.Editable {
		AddConfig(c, "indicator",
			func() *Icon {
				w := NewIcon()
				w.Style(func(s *styles.Style) {
					s.Justify.Self = styles.End
				})
				return w
			}, func(w *Icon) {
				w.SetIcon(ch.Indicator)
			})
	}
}

// TextWidget returns the text widget if present.
func (ch *Chooser) TextWidget() *Text {
	text, _ := ch.ChildByName("text").(*Text)
	return text
}

// TextField returns the text field widget of an editable Chooser if present.
func (ch *Chooser) TextField() *TextField {
	tf, _ := ch.ChildByName("text-field").(*TextField)
	return tf
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
	for _, f := range ch.ItemsFuncs {
		f()
	}
}

// SetTypes sets the [Chooser.Items] from the given types.
func (ch *Chooser) SetTypes(ts ...*types.Type) *Chooser {
	ch.Items = make([]ChooserItem, len(ts))
	for i, typ := range ts {
		ch.Items[i] = ChooserItem{Value: typ}
	}
	return ch
}

// SetStrings sets the [Chooser.Items] from the given strings.
func (ch *Chooser) SetStrings(ss ...string) *Chooser {
	ch.Items = make([]ChooserItem, len(ss))
	for i, s := range ss {
		ch.Items[i] = ChooserItem{Value: s}
	}
	return ch
}

// SetEnums sets the [Chooser.Items] from the given enums.
func (ch *Chooser) SetEnums(es ...enums.Enum) *Chooser {
	ch.Items = make([]ChooserItem, len(es))
	for i, enum := range es {
		str := enum.String()
		lbl := strcase.ToSentence(str)
		desc := enum.Desc()
		// If the documentation does not start with the transformed name, but it does
		// start with an uppercase letter, then we assume that the first word of the
		// documentation is the correct untransformed name. This fixes
		// https://github.com/cogentcore/core/issues/774 (also for Switches).
		if !strings.HasPrefix(desc, str) && len(desc) > 0 && unicode.IsUpper(rune(desc[0])) {
			str, _, _ = strings.Cut(desc, " ")
		}
		tip := types.FormatDoc(desc, str, lbl)
		ch.Items[i] = ChooserItem{Value: enum, Text: lbl, Tooltip: tip}
	}
	return ch
}

// SetEnum sets the [Chooser.Items] from the [enums.Enum.Values] of the given enum.
func (ch *Chooser) SetEnum(enum enums.Enum) *Chooser {
	return ch.SetEnums(enum.Values()...)
}

// FindItem finds the given item value on the list of items and returns its index
func (ch *Chooser) FindItem(it any) int {
	for i, v := range ch.Items {
		if it == v.Value {
			return i
		}
	}
	return -1
}

// SetPlaceholder sets the given placeholder text and
// indicates that nothing has been selected.
func (ch *Chooser) SetPlaceholder(text string) *Chooser {
	ch.Placeholder = text
	if !ch.Editable {
		ch.CurrentItem.Text = text
		ch.ShowCurrentItem()
	}
	ch.CurrentIndex = -1
	return ch
}

// SetCurrentValue sets the current item and index to those associated with the given value.
// If the given item is not found, it adds it to the items list. It also sets the text
// of the chooser to the label of the item.
func (ch *Chooser) SetCurrentValue(v any) *Chooser {
	ch.CurrentIndex = ch.FindItem(v)
	if ch.CurrentIndex < 0 { // add to list if not found
		ch.CurrentIndex = len(ch.Items)
		ch.Items = append(ch.Items, ChooserItem{Value: v})
	}
	ch.CurrentItem = ch.Items[ch.CurrentIndex]
	ch.ShowCurrentItem()
	return ch
}

// SetCurrentIndex sets the current index and the item associated with it.
func (ch *Chooser) SetCurrentIndex(index int) *Chooser {
	if index < 0 || index >= len(ch.Items) {
		return ch
	}
	ch.CurrentIndex = index
	ch.CurrentItem = ch.Items[index]
	ch.ShowCurrentItem()
	return ch
}

// SetCurrentText sets the current index and item based on the given text string.
// It can only be used for editable choosers.
func (ch *Chooser) SetCurrentText(text string) error {
	for i, item := range ch.Items {
		if text == item.GetLabel() {
			ch.SetCurrentIndex(i)
			return nil
		}
	}
	if !(ch.AllowNew || ch.DefaultNew) {
		return errors.New("unknown value")
	}
	ch.Items = append(ch.Items, ChooserItem{Value: text})
	ch.SetCurrentIndex(len(ch.Items) - 1)
	return nil
}

// ShowCurrentItem updates the display to present the current item.
func (ch *Chooser) ShowCurrentItem() *Chooser {
	if ch.Editable {
		tf := ch.TextField()
		if tf != nil {
			tf.SetText(ch.CurrentItem.GetLabel())
		}
	} else {
		text := ch.TextWidget()
		if text != nil {
			text.SetText(ch.CurrentItem.GetLabel()).ConfigWidget()
		}
	}
	if ch.CurrentItem.Icon.IsSet() {
		picon := ch.Icon
		ch.SetIcon(ch.CurrentItem.Icon)
		if ch.Icon != picon {
			ch.Update()
		}
	}
	ch.NeedsRender()
	return ch
}

// SelectItem selects the item at the given index and updates the chooser to display it.
func (ch *Chooser) SelectItem(index int) *Chooser {
	if ch.This() == nil {
		return ch
	}
	ch.SetCurrentIndex(index)
	ch.NeedsLayout()
	return ch
}

// SelectItemAction selects the item at the given index and updates the chooser to display it.
// It also sends an [events.Change] event to indicate that the value has changed.
func (ch *Chooser) SelectItemAction(index int) *Chooser {
	if ch.This() == nil {
		return ch
	}
	ch.SelectItem(index)
	ch.SendChange()
	return ch
}

// ClearText clears the text field, for editable choosers.  Also clears any errors.
func (ch *Chooser) ClearText() {
	tf := ch.TextField()
	if tf == nil {
		return
	}
	tf.ClearError()
	tf.Clear()
}

// ClearError clears any existing validation error, for editable choosers.
func (ch *Chooser) ClearError() {
	tf := ch.TextField()
	if tf == nil {
		return
	}
	tf.ClearError()
}

// MakeItemsMenu constructs a menu of all the items. It is used when the chooser is clicked.
func (ch *Chooser) MakeItemsMenu(m *Scene) {
	ch.CallItemsFuncs()
	for i, it := range ch.Items {
		if it.SeparatorBefore {
			NewSeparator(m)
		}
		bt := NewButton(m).SetText(it.GetLabel()).SetIcon(it.Icon).SetTooltip(it.Tooltip)
		bt.SetSelected(i == ch.CurrentIndex)
		bt.OnClick(func(e events.Event) {
			ch.SelectItemAction(i)
		})
	}
	if ch.AllowNew {
		NewSeparator(m)
		NewButton(m).SetText("New item").SetIcon(icons.Add).
			SetTooltip("Add a new item to the chooser").
			OnClick(func(e events.Event) {
				d := NewBody().AddTitle("New item").AddText("Add a new item to the chooser")
				tf := NewTextField(d)
				d.AddBottomBar(func(parent Widget) {
					d.AddCancel(parent)
					d.AddOK(parent).SetText("Add").SetIcon(icons.Add).OnClick(func(e events.Event) {
						ch.Items = append(ch.Items, ChooserItem{Value: tf.Text()})
						ch.SelectItemAction(len(ch.Items) - 1)
					})
				})
				d.RunDialog(ch)
			})
	}
}

func (ch *Chooser) HandleEvents() {
	ch.HandleSelectToggle()

	ch.OnClick(func(e events.Event) {
		if ch.OpenMenu(e) {
			e.SetHandled()
		}
	})
	ch.OnChange(func(e events.Event) {
		if ch.CurrentItem.Func != nil {
			ch.CurrentItem.Func()
		}
	})
	ch.OnFinal(events.KeyChord, func(e events.Event) {
		tf := ch.TextField()
		kf := keymap.Of(e.KeyChord())
		if DebugSettings.KeyEventTrace {
			slog.Info("Chooser KeyChordEvent", "widget", ch, "keyFunction", kf)
		}
		switch {
		case kf == keymap.MoveUp:
			e.SetHandled()
			if len(ch.Items) > 0 {
				index := ch.CurrentIndex - 1
				if index < 0 {
					index += len(ch.Items)
				}
				ch.SelectItemAction(index)
			}
		case kf == keymap.MoveDown:
			e.SetHandled()
			if len(ch.Items) > 0 {
				index := ch.CurrentIndex + 1
				if index >= len(ch.Items) {
					index -= len(ch.Items)
				}
				ch.SelectItemAction(index)
			}
		case kf == keymap.PageUp:
			e.SetHandled()
			if len(ch.Items) > 10 {
				index := ch.CurrentIndex - 10
				for index < 0 {
					index += len(ch.Items)
				}
				ch.SelectItemAction(index)
			}
		case kf == keymap.PageDown:
			e.SetHandled()
			if len(ch.Items) > 10 {
				index := ch.CurrentIndex + 10
				for index >= len(ch.Items) {
					index -= len(ch.Items)
				}
				ch.SelectItemAction(index)
			}
		case kf == keymap.Enter || (!ch.Editable && e.KeyRune() == ' '):
			// if !(kt.Rune == ' ' && chb.Sc.Type == ScCompleter) {
			e.SetHandled()
			ch.Send(events.Click, e)
		// }
		default:
			if tf == nil {
				break
			}
			// if we don't have anything special to do,
			// we just give our key event to our textfield
			tf.HandleEvent(e)
		}
	})
}

// OpenMenu opens the chooser menu that displays all of the items.
// It returns false if there are no items.
func (ch *Chooser) OpenMenu(e events.Event) bool {
	pos := ch.ContextMenuPos(e)
	if indicator, ok := ch.ChildByName("indicator").(Widget); ok {
		pos = indicator.ContextMenuPos(nil) // use the pos
	}
	m := NewMenu(ch.MakeItemsMenu, ch.This().(Widget), pos)
	if m == nil {
		return false
	}
	m.Run()
	return true
}

func (ch *Chooser) HandleChooserTextFieldEvents(tf *TextField) {
	tf.SetValidator(func() error {
		err := ch.SetCurrentText(tf.Text())
		if err == nil {
			ch.SendChange()
		}
		return err
	})
	tf.OnFocus(func(e events.Event) {
		if ch.IsReadOnly() {
			return
		}
		ch.CallItemsFuncs()
	})
	tf.OnClick(func(e events.Event) {
		ch.CallItemsFuncs()
		tf.OfferComplete()
	})
	tf.OnKeyChord(func(e events.Event) {
		kf := keymap.Of(e.KeyChord())
		if kf == keymap.Abort {
			if tf.Error != nil {
				tf.Clear()
				tf.ClearError()
				e.SetHandled()
			}
		}
	})
}

func (ch *Chooser) WidgetTooltip(pos image.Point) (string, image.Point) {
	if ch.CurrentItem.Tooltip != "" {
		return ch.CurrentItem.Tooltip, ch.DefaultTooltipPos()
	}
	return ch.Tooltip, ch.DefaultTooltipPos()
}

// CompleteMatch is the [complete.MatchFunc] used for the
// editable text field part of the Chooser (if it exists).
func (ch *Chooser) CompleteMatch(data any, text string, posLine, posChar int) (md complete.Matches) {
	md.Seed = text
	comps := make(complete.Completions, len(ch.Items))
	for i, item := range ch.Items {
		comps[i] = complete.Completion{
			Text: item.GetLabel(),
			Desc: item.Tooltip,
			Icon: item.Icon,
		}
	}
	md.Matches = complete.MatchSeedCompletion(comps, md.Seed)
	if ch.AllowNew && text != "" && !slices.ContainsFunc(md.Matches, func(c complete.Completion) bool {
		return c.Text == text
	}) {
		md.Matches = append(md.Matches, complete.Completion{
			Text:  text,
			Label: "Add " + text,
			Icon:  icons.Add,
			Desc:  fmt.Sprintf("Add %q to the chooser", text),
		})
	}
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
