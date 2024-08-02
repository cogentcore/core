// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"errors"
	"fmt"
	"image"
	"log/slog"
	"reflect"
	"slices"
	"strings"
	"unicode"

	"cogentcore.org/core/base/labels"
	"cogentcore.org/core/base/reflectx"
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
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
)

// Chooser is a dropdown selection widget that allows users to choose
// one option among a list of items.
type Chooser struct {
	Frame

	// Type is the styling type of the chooser.
	Type ChooserTypes

	// Items are the chooser items available for selection.
	Items []ChooserItem

	// Icon is an optional icon displayed on the left side of the chooser.
	Icon icons.Icon

	// Indicator is the icon to use for the indicator displayed on the
	// right side of the chooser.
	Indicator icons.Icon

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

	// placeholder, if Editable is set to true, is the text that is
	// displayed in the text field when it is empty. It must be set
	// using [Chooser.SetPlaceholder].
	placeholder string `set:"-"`

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

	text      *Text
	textField *TextField
}

// ChooserItem is an item that can be used in a [Chooser].
type ChooserItem struct {

	// Value is the underlying value the chooser item represents.
	Value any

	// Text is the text displayed to the user for this item.
	// If it is empty, then [labels.ToLabel] of [ChooserItem.Value]
	// is used instead.
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

// GetText returns the effective text for this chooser item.
// If [ChooserItem.Text] is set, it returns that. Otherwise,
// it returns [labels.ToLabel] of [ChooserItem.Value].
func (ci *ChooserItem) GetText() string {
	if ci.Text != "" {
		return ci.Text
	}
	if ci.Value == nil {
		return ""
	}
	return labels.ToLabel(ci.Value)
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

func (ch *Chooser) WidgetValue() any { return ch.CurrentItem.Value }

func (ch *Chooser) SetWidgetValue(value any) error {
	rv := reflect.ValueOf(value)
	// If the first item is a pointer, we assume that our value should
	// be a pointer. Otherwise, it should be a non-pointer value.
	if len(ch.Items) > 0 && reflect.TypeOf(ch.Items[0].Value).Kind() == reflect.Pointer {
		rv = reflectx.UnderlyingPointer(rv)
	} else {
		rv = reflectx.Underlying(rv)
	}
	ch.SetCurrentValue(rv.Interface())
	return nil
}

func (ch *Chooser) OnBind(value any, tags reflect.StructTag) {
	if e, ok := value.(enums.Enum); ok {
		ch.SetEnum(e)
	}
}

func (ch *Chooser) Init() {
	ch.Frame.Init()
	ch.SetIcon(icons.None).SetIndicator(icons.KeyboardArrowDown)
	ch.CurrentIndex = -1
	ch.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Activatable, abilities.Hoverable, abilities.LongHoverable)
		if !ch.Editable {
			s.SetAbilities(true, abilities.Focusable)
		}
		s.Cursor = cursors.Pointer
		s.Text.Align = styles.Center
		s.Border.Radius = styles.BorderRadiusSmall
		s.Padding.Set(units.Dp(8), units.Dp(16))
		s.CenterAll()
		switch ch.Type {
		case ChooserFilled:
			s.Background = colors.Scheme.Secondary.Container
			s.Color = colors.Scheme.Secondary.OnContainer
		case ChooserOutlined:
			if !s.Is(states.Focused) {
				s.Border.Style.Set(styles.BorderSolid)
				s.Border.Width.Set(units.Dp(1))
				s.Border.Color.Set(colors.Scheme.OnSurfaceVariant)
			}
		}
		// textfield handles everything
		if ch.Editable {
			s.RenderBox = false
			s.Border = styles.Border{}
			s.MaxBorder = s.Border
			s.Background = nil
			s.StateLayer = 0
			s.Padding.Zero()
			s.Border.Radius.Zero()
		}
	})

	ch.OnClick(func(e events.Event) {
		if ch.openMenu(e) {
			e.SetHandled()
		}
	})
	ch.OnChange(func(e events.Event) {
		if ch.CurrentItem.Func != nil {
			ch.CurrentItem.Func()
		}
	})
	ch.OnFinal(events.KeyChord, func(e events.Event) {
		tf := ch.textField
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
				ch.selectItemEvent(index)
			}
		case kf == keymap.MoveDown:
			e.SetHandled()
			if len(ch.Items) > 0 {
				index := ch.CurrentIndex + 1
				if index >= len(ch.Items) {
					index -= len(ch.Items)
				}
				ch.selectItemEvent(index)
			}
		case kf == keymap.PageUp:
			e.SetHandled()
			if len(ch.Items) > 10 {
				index := ch.CurrentIndex - 10
				for index < 0 {
					index += len(ch.Items)
				}
				ch.selectItemEvent(index)
			}
		case kf == keymap.PageDown:
			e.SetHandled()
			if len(ch.Items) > 10 {
				index := ch.CurrentIndex + 10
				for index >= len(ch.Items) {
					index -= len(ch.Items)
				}
				ch.selectItemEvent(index)
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

	ch.Maker(func(p *tree.Plan) {
		// automatically select the first item if we have nothing selected and no placeholder
		if !ch.Editable && ch.CurrentIndex < 0 && ch.CurrentItem.Text == "" {
			ch.SetCurrentIndex(0)
		}

		// editable handles through TextField
		if ch.Icon.IsSet() && !ch.Editable {
			tree.AddAt(p, "icon", func(w *Icon) {
				w.Updater(func() {
					w.SetIcon(ch.Icon)
				})
			})
		}
		if ch.Editable {
			tree.AddAt(p, "text-field", func(w *TextField) {
				ch.textField = w
				ch.text = nil
				w.SetPlaceholder(ch.placeholder)
				w.Styler(func(s *styles.Style) {
					s.Grow = ch.Styles.Grow // we grow like our parent
					s.Max.X.Zero()          // constrained by parent
					s.SetTextWrap(false)
				})
				w.SetValidator(func() error {
					err := ch.setCurrentText(w.Text())
					if err == nil {
						ch.SendChange()
					}
					return err
				})
				w.OnFocus(func(e events.Event) {
					if ch.IsReadOnly() {
						return
					}
					ch.CallItemsFuncs()
				})
				w.OnClick(func(e events.Event) {
					ch.CallItemsFuncs()
					w.offerComplete()
				})
				w.OnKeyChord(func(e events.Event) {
					kf := keymap.Of(e.KeyChord())
					if kf == keymap.Abort {
						if w.error != nil {
							w.clear()
							w.clearError()
							e.SetHandled()
						}
					}
				})
				w.Updater(func() {
					if w.error != nil {
						return // don't override anything when we have an invalid value
					}
					w.SetText(ch.CurrentItem.GetText()).SetLeadingIcon(ch.Icon).
						SetTrailingIcon(ch.Indicator, func(e events.Event) {
							ch.openMenu(e)
						})
					if ch.Type == ChooserFilled {
						w.SetType(TextFieldFilled)
					} else {
						w.SetType(TextFieldOutlined)
					}
					if ch.DefaultNew && w.complete != nil {
						w.complete = nil
					} else if !ch.DefaultNew && w.complete == nil {
						w.SetCompleter(w, ch.completeMatch, ch.completeEdit)
					}
				})
				w.Maker(func(p *tree.Plan) {
					tree.AddInit(p, "trail-icon", func(w *Button) {
						w.Styler(func(s *styles.Style) {
							// indicator does not need to be focused
							s.SetAbilities(false, abilities.Focusable)
						})
					})
				})
			})
		} else {
			tree.AddAt(p, "text", func(w *Text) {
				ch.text = w
				ch.textField = nil
				w.Styler(func(s *styles.Style) {
					s.SetNonSelectable()
					s.SetTextWrap(false)
				})
				w.Updater(func() {
					w.SetText(ch.CurrentItem.GetText())
				})
			})
		}
		if ch.Indicator == "" {
			ch.Indicator = icons.KeyboardArrowRight
		}
		// editable handles through TextField
		if !ch.Editable {
			tree.AddAt(p, "indicator", func(w *Icon) {
				w.Styler(func(s *styles.Style) {
					s.Justify.Self = styles.End
				})
				w.Updater(func() {
					w.SetIcon(ch.Indicator)
				})
			})
		}
	})
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

// findItem finds the given item value on the list of items and returns its index.
func (ch *Chooser) findItem(it any) int {
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
	ch.placeholder = text
	if !ch.Editable {
		ch.CurrentItem.Text = text
		ch.showCurrentItem()
	}
	ch.CurrentIndex = -1
	return ch
}

// SetCurrentValue sets the current item and index to those associated with the given value.
// If the given item is not found, it adds it to the items list if it is not "". It also
// sets the text of the chooser to the label of the item.
func (ch *Chooser) SetCurrentValue(value any) *Chooser {
	ch.CurrentIndex = ch.findItem(value)
	if value != "" && ch.CurrentIndex < 0 { // add to list if not found
		ch.CurrentIndex = len(ch.Items)
		ch.Items = append(ch.Items, ChooserItem{Value: value})
	}
	if ch.CurrentIndex >= 0 {
		ch.CurrentItem = ch.Items[ch.CurrentIndex]
	}
	ch.showCurrentItem()
	return ch
}

// SetCurrentIndex sets the current index and the item associated with it.
func (ch *Chooser) SetCurrentIndex(index int) *Chooser {
	if index < 0 || index >= len(ch.Items) {
		return ch
	}
	ch.CurrentIndex = index
	ch.CurrentItem = ch.Items[index]
	ch.showCurrentItem()
	return ch
}

// setCurrentText sets the current index and item based on the given text string.
// It can only be used for editable choosers.
func (ch *Chooser) setCurrentText(text string) error {
	for i, item := range ch.Items {
		if text == item.GetText() {
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

// showCurrentItem updates the display to present the current item.
func (ch *Chooser) showCurrentItem() *Chooser {
	if ch.Editable {
		tf := ch.textField
		if tf != nil {
			tf.SetText(ch.CurrentItem.GetText())
		}
	} else {
		text := ch.text
		if text != nil {
			text.SetText(ch.CurrentItem.GetText()).UpdateWidget()
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

// selectItem selects the item at the given index and updates the chooser to display it.
func (ch *Chooser) selectItem(index int) *Chooser {
	if ch.This == nil {
		return ch
	}
	ch.SetCurrentIndex(index)
	ch.NeedsLayout()
	return ch
}

// selectItemEvent selects the item at the given index and updates the chooser to display it.
// It also sends an [events.Change] event to indicate that the value has changed.
func (ch *Chooser) selectItemEvent(index int) *Chooser {
	if ch.This == nil {
		return ch
	}
	ch.selectItem(index)
	if ch.textField != nil {
		ch.textField.validate()
	}
	ch.SendChange()
	return ch
}

// ClearError clears any existing validation error for an editable chooser.
func (ch *Chooser) ClearError() {
	tf := ch.textField
	if tf == nil {
		return
	}
	tf.clearError()
}

// makeItemsMenu constructs a menu of all the items.
// It is used when the chooser is clicked.
func (ch *Chooser) makeItemsMenu(m *Scene) {
	ch.CallItemsFuncs()
	for i, it := range ch.Items {
		if it.SeparatorBefore {
			NewSeparator(m)
		}
		bt := NewButton(m).SetText(it.GetText()).SetIcon(it.Icon).SetTooltip(it.Tooltip)
		bt.SetSelected(i == ch.CurrentIndex)
		bt.OnClick(func(e events.Event) {
			ch.selectItemEvent(i)
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
						ch.selectItemEvent(len(ch.Items) - 1)
					})
				})
				d.RunDialog(ch)
			})
	}
}

// openMenu opens the chooser menu that displays all of the items.
// It returns false if there are no items.
func (ch *Chooser) openMenu(e events.Event) bool {
	pos := ch.ContextMenuPos(e)
	if indicator, ok := ch.ChildByName("indicator").(Widget); ok {
		pos = indicator.ContextMenuPos(nil) // use the pos
	}
	m := NewMenu(ch.makeItemsMenu, ch.This.(Widget), pos)
	if m == nil {
		return false
	}
	m.Run()
	return true
}

func (ch *Chooser) WidgetTooltip(pos image.Point) (string, image.Point) {
	if ch.CurrentItem.Tooltip != "" {
		return ch.CurrentItem.Tooltip, ch.DefaultTooltipPos()
	}
	return ch.Tooltip, ch.DefaultTooltipPos()
}

// completeMatch is the [complete.MatchFunc] used for the
// editable text field part of the Chooser (if it exists).
func (ch *Chooser) completeMatch(data any, text string, posLine, posChar int) (md complete.Matches) {
	md.Seed = text
	comps := make(complete.Completions, len(ch.Items))
	for i, item := range ch.Items {
		comps[i] = complete.Completion{
			Text: item.GetText(),
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

// completeEdit is the [complete.EditFunc] used for the
// editable textfield part of the Chooser (if it exists).
func (ch *Chooser) completeEdit(data any, text string, cursorPos int, completion complete.Completion, seed string) (ed complete.Edit) {
	return complete.Edit{
		NewText:       completion.Text,
		ForwardDelete: len([]rune(text)),
	}
}
