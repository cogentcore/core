// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"log/slog"
	"sync"

	"cogentcore.org/core/base/elide"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
)

// Tabs divide widgets into logical groups and give users the ability
// to freely navigate between them using tab buttons.
type Tabs struct {
	Layout

	// Type is the styling type of the tabs. If it is changed after
	// the tabs are first configured, Update needs to be called on
	// the tabs.
	Type TabTypes

	// NewTabButton is whether to show a new tab button at the end of the list of tabs.
	NewTabButton bool

	// MaxChars is the maximum number of characters to include in the tab text.
	// It elides text that are longer than that.
	MaxChars int

	// CloseIcon is the icon used for tab close buttons.
	// If it is "" or [icons.None], the tab is not closeable.
	// The default value is [icons.Close].
	// Only [FunctionalTabs] can be closed; all other types of
	// tabs will not render a close button and can not be closed.
	CloseIcon icons.Icon

	// PrevEffectiveType is the previous effective type of the tabs
	// as computed by [TabTypes.Effective].
	PrevEffectiveType TabTypes `copier:"-" json:"-" xml:"-" set:"-"`

	// Mu is a mutex protecting updates to tabs. Tabs can be driven
	// programmatically and via user input so need extra protection.
	Mu sync.Mutex `copier:"-" json:"-" xml:"-" view:"-" set:"-"`
}

// TabTypes are the different styling types of tabs.
type TabTypes int32 //enums:enum

const (
	// StandardTabs indicates to render the standard type
	// of Material Design style tabs.
	StandardTabs TabTypes = iota

	// FunctionalTabs indicates to render functional tabs
	// like those in Google Chrome. These tabs take up less
	// space and are the only kind that can be closed.
	// They can also be moved.
	FunctionalTabs

	// NavigationAuto indicates to render the tabs as either
	// [NavigationBar], [NavigationRail], or [NavigationDrawer],
	// if [WidgetBase.SizeClass] is [SizeCompact], [SizeMedium],
	// or [SizeExpanded], respectively. NavigationAuto should
	// typically be used instead of one of the specific navigation
	// types for better cross-platform compatability.
	NavigationAuto

	// NavigationBar indicates to render the tabs as a
	// bottom navigation bar with text and icons.
	NavigationBar

	// NavigationRail indicates to render the tabs as a
	// side navigation rail, which only has icons.
	NavigationRail

	// NavigationDrawer indicates to render the tabs as a
	// side navigation drawer, which has full text and icons.
	NavigationDrawer
)

// EffectiveType returns the effective tab type in the context
// of the given widget, handling [NavigationAuto] based on
// [WidgetBase.SizeClass].
func (tt TabTypes) Effective(w Widget) TabTypes {
	if tt != NavigationAuto {
		return tt
	}
	switch w.AsWidget().SizeClass() {
	case SizeCompact:
		return NavigationBar
	case SizeMedium:
		return NavigationRail
	default:
		return NavigationDrawer
	}
}

// IsColumn returns whether the tabs should be arranged in a column.
func (tt TabTypes) IsColumn() bool {
	return tt == NavigationRail || tt == NavigationDrawer
}

func (ts *Tabs) OnInit() {
	ts.WidgetBase.OnInit()
	ts.Layout.HandleEvents()
	ts.SetStyles()
}

func (ts *Tabs) SetStyles() {
	ts.MaxChars = 16
	ts.CloseIcon = icons.Close
	ts.Style(func(s *styles.Style) {
		s.Color = colors.C(colors.Scheme.OnBackground)
		s.Grow.Set(1, 1)
		if ts.Type.Effective(ts).IsColumn() {
			s.Direction = styles.Row
		} else {
			s.Direction = styles.Column
		}
	})
	ts.OnWidgetAdded(func(w Widget) {
		switch w.PathFrom(ts) {
		case "tabs":
			w.Style(func(s *styles.Style) {
				s.Overflow.Set(styles.OverflowHidden) // no scrollbars!
				s.Margin.Zero()
				s.Padding.Zero()
				s.Gap.Set(units.Dp(4))

				if ts.Type.Effective(ts).IsColumn() {
					s.Direction = styles.Column
					s.Grow.Set(0, 1)
				} else {
					s.Direction = styles.Row
					s.Grow.Set(1, 0)
					s.Wrap = true
				}

				// s.Border.Style.Set(styles.BorderNone)
				// s.Border.Style.Bottom = styles.BorderSolid
				// s.Border.Width.Bottom.SetDp(1)
				// s.Border.Color.Bottom = colors.Scheme.OutlineVariant
			})
		case "frame":
			frame := w.(*Frame)
			w.Style(func(s *styles.Style) {
				s.Display = styles.Stacked
				frame.SetFlag(true, LayoutStackTopOnly) // key for allowing each tab to have its own size
				s.Min.X.Dp(160)
				s.Min.Y.Dp(96)
				s.Grow.Set(1, 1)
			})
		}
		if w.Parent() == ts.ChildByName("frame") {
			// tab frames must scroll independently
			w.Style(func(s *styles.Style) {
				s.Overflow.Set(styles.OverflowAuto)
			})
		}
	})
}

// NumTabs returns number of tabs
func (ts *Tabs) NumTabs() int {
	fr := ts.Frame()
	if fr == nil {
		return 0
	}
	return len(fr.Kids)
}

// CurTab returns currently selected tab, and its index -- returns false none
func (ts *Tabs) CurTab() (Widget, int, bool) {
	if ts.NumTabs() == 0 {
		return nil, -1, false
	}
	ts.Mu.Lock()
	defer ts.Mu.Unlock()
	fr := ts.Frame()
	if fr.StackTop < 0 {
		return nil, -1, false
	}
	w := fr.Child(fr.StackTop).(Widget)
	return w, fr.StackTop, true
}

// NewTab adds a new tab with the given label and returns the resulting tab frame.
// It is the main end-user API for creating new tabs. An optional icon can also
// be passed for the tab button.
func (ts *Tabs) NewTab(label string, icon ...icons.Icon) *Frame {
	fr := ts.Frame()
	idx := len(*fr.Children())
	frame := ts.InsertNewTab(label, idx, icon...)
	return frame
}

// InsertNewTab inserts a new tab with the given label at the given index position
// within the list of tabs and returns the resulting tab frame. An optional icon
// can also be passed for the tab button.
func (ts *Tabs) InsertNewTab(label string, idx int, icon ...icons.Icon) *Frame {
	tfr := ts.Frame()
	frame := tree.InsertNewChild[*Frame](tfr, idx)
	frame.SetName(label)
	frame.Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	ts.InsertTabOnlyAt(frame, label, idx, icon...)
	ts.Update()
	return frame
}

// AddTab adds an already existing frame as a new tab with the given tab label
// and returns the index of that tab.
func (ts *Tabs) AddTab(frame *Frame, label string) int {
	fr := ts.Frame()
	idx := len(*fr.Children())
	ts.InsertTab(frame, label, idx)
	return idx
}

// InsertTabOnlyAt inserts just the tab button at given index, after the panel has
// already been added to the frame; assumed to be wrapped in update. Generally
// for internal use only. An optional icon can also be passed for the tab button.
func (ts *Tabs) InsertTabOnlyAt(frame *Frame, label string, idx int, icon ...icons.Icon) {
	tb := ts.Tabs()
	tab := tree.InsertNewChild[*Tab](tb, idx)
	tab.SetName(label)
	tab.SetText(label).SetType(ts.Type).SetCloseIcon(ts.CloseIcon).SetMaxChars(ts.MaxChars).SetTooltip(label)
	if len(icon) > 0 {
		tab.SetIcon(icon[0])
	}
	tab.OnClick(func(e events.Event) {
		ts.SelectTabByName(tab.Nm)
	})
	fr := ts.Frame()
	if len(fr.Kids) == 1 {
		fr.StackTop = 0
		tab.SetSelected(true)
		// } else {
		// 	frame.SetState(true, states.Invisible) // new tab is invisible until selected
	}
}

// InsertTab inserts a frame into given index position within list of tabs.
// An optional icon can also be passed for the tab button.
func (ts *Tabs) InsertTab(frame *Frame, label string, idx int, icon ...icons.Icon) {
	ts.Mu.Lock()
	defer ts.Mu.Unlock()

	fr := ts.Frame()
	fr.InsertChild(frame, idx)
	ts.InsertTabOnlyAt(frame, label, idx, icon...)
	ts.NeedsLayout()
}

// TabAtIndex returns content frame and tab button at given index, false if
// index out of range (emits log message)
func (ts *Tabs) TabAtIndex(idx int) (*Frame, *Tab, bool) {
	ts.Mu.Lock()
	defer ts.Mu.Unlock()

	fr := ts.Frame()
	tb := ts.Tabs()
	sz := len(*fr.Children())
	if idx < 0 || idx >= sz {
		slog.Error("core.Tabs: index out of range for number of tabs", "index", idx, "numTabs", sz)
		return nil, nil, false
	}
	tab := tb.Child(idx).(*Tab)
	frame := fr.Child(idx).(*Frame)
	return frame, tab, true
}

// SelectTabIndex selects tab at given index, returning it.
// Returns false if index is invalid.  This is the final
// tab selection path.
func (ts *Tabs) SelectTabIndex(idx int) (*Frame, bool) {
	frame, tab, ok := ts.TabAtIndex(idx)
	if !ok {
		return nil, false
	}
	fr := ts.Frame()
	if fr.StackTop == idx {
		return frame, true
	}
	ts.Mu.Lock()
	ts.UnselectOtherTabs(idx)
	tab.SetSelected(true)
	fr.StackTop = idx
	fr.Update()
	ts.Mu.Unlock()
	return frame, true
}

// TabByName returns tab Frame with given widget name
// (nil if not found)
// The widget name is the original full tab label, prior to any eliding.
func (ts *Tabs) TabByName(name string) *Frame {
	ts.Mu.Lock()
	defer ts.Mu.Unlock()
	fr := ts.Frame()
	frame, _ := fr.ChildByName(name).(*Frame)
	return frame
}

// TabIndexByName returns the tab index for the given tab widget name
// and -1 if it can not be found.
// The widget name is the original full tab label, prior to any eliding.
func (ts *Tabs) TabIndexByName(name string) int {
	ts.Mu.Lock()
	defer ts.Mu.Unlock()

	tb := ts.Tabs()
	tab := tb.ChildByName(name)
	if tab == nil {
		return -1
	}
	return tab.IndexInParent()
}

// TabLabel returns tab label at given index
func (ts *Tabs) TabLabel(idx int) string {
	ts.Mu.Lock()
	defer ts.Mu.Unlock()

	tb := ts.Tabs()
	tbut := tb.Child(idx)
	if tbut == nil {
		return ""
	}
	return tbut.Name()
}

// SelectTabByName selects tab by widget name, returning it.
// The widget name is the original full tab label, prior to any eliding.
func (ts *Tabs) SelectTabByName(name string) *Frame {
	idx := ts.TabIndexByName(name)
	if idx < 0 {
		return nil
	}
	ts.SelectTabIndex(idx)
	fr := ts.Frame()
	return fr.Child(idx).(*Frame)
}

// RecycleTab returns a tab with given name, first by looking for an existing one,
// and if not found, making a new one. If sel, then select it. It returns the
// frame for the tab.
func (ts *Tabs) RecycleTab(name string, sel bool) *Frame {
	frame := ts.TabByName(name)
	if frame != nil {
		if sel {
			ts.SelectTabByName(name)
		}
		return frame
	}
	frame = ts.NewTab(name)
	if sel {
		ts.SelectTabByName(name)
	}
	return frame
}

// RecycleTabWidget returns a tab with given widget type in the tab frame,
// first by looking for an existing one, with given name, and if not found,
// making and configuring a new one.
// If sel, then select it. It returns the Widget item for the tab.
func (ts *Tabs) RecycleTabWidget(name string, sel bool, typ *types.Type) Widget {
	fr := ts.RecycleTab(name, sel)
	if fr.HasChildren() {
		return fr.Child(0).(Widget)
	}
	wi := fr.NewChild(typ).(Widget)
	wi.Config()
	return wi
}

// DeleteTabIndex deletes tab at given index, returning whether it was successful.
func (ts *Tabs) DeleteTabIndex(idx int) bool {
	_, _, ok := ts.TabAtIndex(idx)
	if !ok {
		return false
	}

	ts.Mu.Lock()
	fr := ts.Frame()
	sz := len(*fr.Children())
	tb := ts.Tabs()
	nidx := -1
	if fr.StackTop == idx {
		if idx > 0 {
			nidx = idx - 1
		} else if idx < sz-1 {
			nidx = idx
		}
	}
	// if we didn't delete the current tab and have at least one
	// other tab, we go to the next tab over
	if nidx < 0 && ts.NumTabs() > 1 {
		nidx = max(idx-1, 0)
	}
	fr.DeleteChildAtIndex(idx)
	tb.DeleteChildAtIndex(idx)
	ts.Mu.Unlock()

	if nidx >= 0 {
		ts.SelectTabIndex(nidx)
	}
	ts.NeedsLayout()
	return true
}

// ConfigNewTabButton configures the new tab button at the end of the list of tabs, if applicable.
func (ts *Tabs) ConfigNewTabButton() bool {
	ntabs := ts.NumTabs()
	tb := ts.Tabs()
	nkids := len(tb.Kids)
	if ts.NewTabButton {
		if nkids == ntabs+1 {
			return false
		}
		nt := NewButton(tb).SetIcon(icons.Add).SetType(ButtonAction)
		nt.SetName("new-tab")
		nt.OnClick(func(e events.Event) {
			ts.NewTab("New tab")
			ts.SelectTabIndex(ts.NumTabs() - 1)
		})
		return true
	} else {
		if nkids == ntabs {
			return false
		}
		tb.DeleteChildAtIndex(nkids - 1)
		return true
	}
}

// Config configures the tabs widget children if necessary.
// Only the 2 primary children (Frames) need to be configured.
// Re-config is needed when the type of tabs changes, but not
// when a new tab is added, which only requires a new layout pass.
func (ts *Tabs) Config() {
	config := tree.Config{}
	// frame only comes before tabs in bottom nav bar
	if ts.Type.Effective(ts) == NavigationBar {
		config.Add(FrameType, "frame")
		config.Add(FrameType, "tabs")
	} else {
		config.Add(FrameType, "tabs")
		config.Add(FrameType, "frame")
	}
	if ts.ConfigChildren(config) {
		ts.NeedsLayout()
	}
	ts.ConfigNewTabButton()
}

// Tabs returns the layout containing the tabs (the first element within us).
// It configures the Tabs if necessary.
func (ts *Tabs) Tabs() *Frame {
	if ts.ChildByName("tabs", 0) == nil {
		ts.Config()
	}
	return ts.ChildByName("tabs", 0).(*Frame)
}

// Frame returns the stacked frame layout (the second element within us).
// It configures the Tabs if necessary.
func (ts *Tabs) Frame() *Frame {
	if ts.ChildByName("frame", 1) == nil {
		ts.Config()
	}
	return ts.ChildByName("frame", 1).(*Frame)
}

// UnselectOtherTabs turns off all the tabs except given one
func (ts *Tabs) UnselectOtherTabs(idx int) {
	sz := ts.NumTabs()
	tbs := ts.Tabs()
	for i := 0; i < sz; i++ {
		if i == idx {
			continue
		}
		tb := tbs.Child(i).(*Tab)
		if tb.StateIs(states.Selected) {
			tb.SetSelected(false)
		}
	}
}

////////////////////////////////////////////////////////////////////////////////////////
// Tab

// Tab is a tab button that contains any, all, or none of a label, an icon,
// and a close icon. Tabs should be made using the [Tabs.NewTab] function.
type Tab struct { //core:no-new
	WidgetBase

	// Type is the styling type of the tab. This property
	// must be set on the parent [Tabs] for it to work correctly.
	Type TabTypes

	// Text is the text for the tab.
	// If it is nil, no text is shown.
	// Text is never shown for [NavigationRail] tabs.
	Text string

	// Icon is the icon for the tab.
	// If it is "" or [icons.None], no icon is shown.
	Icon icons.Icon

	// CloseIcon is the icon used as a close button for the tab.
	// If it is "" or [icons.None], the tab is not closeable.
	// The default value is [icons.Close].
	// Only [FunctionalTabs] can be closed; all other types of
	// tabs will not render a close button and can not be closed.
	CloseIcon icons.Icon

	// TODO(kai): replace this with general text overflow property (#778)

	// MaxChars is the maximum number of characters to include in tab text.
	// It elides text that is longer than that.
	MaxChars int
}

func (tb *Tab) OnInit() {
	tb.WidgetBase.OnInit()
	tb.HandleClickOnEnterSpace()
	tb.SetStyles()
}

func (tb *Tab) SetStyles() {
	tb.MaxChars = 16
	tb.CloseIcon = icons.Close
	tb.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Activatable, abilities.Focusable, abilities.Hoverable)

		if !tb.IsReadOnly() {
			s.Cursor = cursors.Pointer
		}

		if tb.Type.Effective(tb).IsColumn() {
			s.Grow.X = 1
			s.Border.Radius = styles.BorderRadiusFull
			s.Padding.Set(units.Dp(16))
		} else {
			s.Border.Radius = styles.BorderRadiusSmall
			s.Padding.Set(units.Dp(10))
		}

		if tb.StateIs(states.Selected) {
			s.Color = colors.C(colors.Scheme.Select.OnContainer)
		} else {
			s.Color = colors.C(colors.Scheme.OnSurfaceVariant)
			if tb.Type.Effective(tb) == FunctionalTabs {
				s.Background = colors.C(colors.Scheme.SurfaceContainer)
			}
		}
	})
	tb.OnWidgetAdded(func(w Widget) {
		switch w.PathFrom(tb) {
		case "parts":
			w.Style(func(s *styles.Style) {
				s.Gap.Zero()
				s.Align.Content = styles.Center
				s.Align.Items = styles.Center
			})
		case "parts/icon":
			w.Style(func(s *styles.Style) {
				s.Font.Size.Dp(18)
				s.Margin.Zero()
				s.Padding.Zero()
			})
		case "parts/text":
			text := w.(*Text)
			if tb.Type.Effective(tb) == FunctionalTabs {
				text.Type = TextBodyMedium
			} else {
				text.Type = TextLabelLarge
			}
			w.Style(func(s *styles.Style) {
				s.SetNonSelectable()
				s.SetTextWrap(false)
				s.Margin.Zero()
				s.Padding.Zero()
			})
		case "parts/close.parts/icon":
			w.Style(func(s *styles.Style) {
				s.Font.Size.Dp(16)
			})
		case "parts/close":
			w.Style(func(s *styles.Style) {
				s.Padding.Zero()
				s.Border.Radius = styles.BorderRadiusFull
			})
			w.OnClick(func(e events.Event) {
				ts := tb.Tabs()
				if ts == nil {
					return
				}
				idx := ts.TabIndexByName(tb.Nm)
				// if OnlyCloseActiveTab is on, only process delete when already selected
				if SystemSettings.OnlyCloseActiveTab && !tb.StateIs(states.Selected) {
					ts.SelectTabIndex(idx)
				} else {
					ts.DeleteTabIndex(idx)
				}
			})
		}
	})
}

// Tabs returns the parent [Tabs] of this [Tab].
func (tb *Tab) Tabs() *Tabs {
	return tb.Parent().Parent().(*Tabs)
}

func (tb *Tab) Config() {
	config := tree.Config{}
	if tb.MaxChars > 0 {
		tb.Text = elide.Middle(tb.Text, tb.MaxChars)
	}

	ici := -1
	lbi := -1
	clsi := -1
	if tb.Icon.IsSet() {
		ici = len(config)
		config.Add(IconType, "icon")
		if tb.Text != "" {
			config.Add(SpaceType, "space")
		}
	}
	if tb.Text != "" {
		lbi = len(config)
		config.Add(TextType, "text")
	}
	if tb.Type.Effective(tb) == FunctionalTabs && tb.CloseIcon.IsSet() {
		config.Add(SpaceType, "close-space")
		clsi = len(config)
		config.Add(ButtonType, "close")
	}

	tb.ConfigParts(config, func() {
		if ici >= 0 {
			ic := tb.Parts.Child(ici).(*Icon)
			ic.SetIcon(tb.Icon)
		}
		if lbi >= 0 {
			text := tb.Parts.Child(lbi).(*Text)
			text.SetText(tb.Text)
		}
		if clsi >= 0 {
			cls := tb.Parts.Child(clsi).(*Button)
			cls.SetType(ButtonAction).SetIcon(tb.CloseIcon)
		}
	})
}
