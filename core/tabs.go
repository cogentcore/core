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
	Frame

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

func (ts *Tabs) Init() {
	ts.Frame.Init()
	ts.MaxChars = 16
	ts.CloseIcon = icons.Close
	ts.Styler(func(s *styles.Style) {
		s.Color = colors.C(colors.Scheme.OnBackground)
		s.Grow.Set(1, 1)
		if ts.Type.Effective(ts).IsColumn() {
			s.Direction = styles.Row
		} else {
			s.Direction = styles.Column
		}
	})
	ts.OnWidgetAdded(func(w Widget) {
		if w.AsTree().Parent == ts.ChildByName("frame") { // TODO(config): figure out how to get this to work with new config paradigm
			w.AsWidget().Styler(func(s *styles.Style) {
				// tab frames must scroll independently and grow
				s.Overflow.Set(styles.OverflowAuto)
				s.Grow.Set(1, 1)
			})
		}
	})

	ts.Maker(func(p *Plan) {
		AddAt(p, "tabs", func(w *Frame) {
			w.Styler(func(s *styles.Style) {
				s.Overflow.Set(styles.OverflowHidden) // no scrollbars!
				s.Gap.Set(units.Dp(4))

				if ts.Type.Effective(ts).IsColumn() {
					s.Direction = styles.Column
					s.Grow.Set(0, 1)
				} else {
					s.Direction = styles.Row
					s.Grow.Set(1, 0)
					s.Wrap = true
				}
			})
			w.Maker(func(p *Plan) {
				if ts.NewTabButton {
					AddAt(p, "new-tab", func(w *Button) { // TODO(config)
						w.SetIcon(icons.Add).SetType(ButtonAction)
						w.OnClick(func(e events.Event) {
							ts.NewTab("New tab")
							ts.SelectTabIndex(ts.NumTabs() - 1)
						})
					})
				}
			})
		})
		AddAt(p, "frame", func(w *Frame) {
			w.LayoutStackTopOnly = true // key for allowing each tab to have its own size
			w.Styler(func(s *styles.Style) {
				s.Display = styles.Stacked
				s.Min.Set(units.Dp(160), units.Dp(96))
				s.Grow.Set(1, 1)
			})
		})
		// frame comes before tabs in bottom navigation bar
		if ts.Type.Effective(ts) == NavigationBar {
			(*p)[0], (*p)[1] = (*p)[1], (*p)[0]
		}
	})
}

// NumTabs returns the number of tabs.
func (ts *Tabs) NumTabs() int {
	fr := ts.FrameWidget()
	if fr == nil {
		return 0
	}
	return len(fr.Children)
}

// CurrentTab returns currently selected tab and its index; returns nil if none.
func (ts *Tabs) CurrentTab() (Widget, int) {
	if ts.NumTabs() == 0 {
		return nil, -1
	}
	ts.Mu.Lock()
	defer ts.Mu.Unlock()
	fr := ts.FrameWidget()
	if fr.StackTop < 0 {
		return nil, -1
	}
	w := fr.Child(fr.StackTop).(Widget)
	return w, fr.StackTop
}

// NewTab adds a new tab with the given label and returns the resulting tab frame.
// It is the main end-user API for creating new tabs. An optional icon can also
// be passed for the tab button.
func (ts *Tabs) NewTab(label string, icon ...icons.Icon) *Frame {
	fr := ts.FrameWidget()
	idx := len(fr.Children)
	frame := ts.InsertNewTab(label, idx, icon...)
	return frame
}

// InsertNewTab inserts a new tab with the given label at the given index position
// within the list of tabs and returns the resulting tab frame. An optional icon
// can also be passed for the tab button.
func (ts *Tabs) InsertNewTab(label string, idx int, icon ...icons.Icon) *Frame {
	tfr := ts.FrameWidget()
	frame := tree.InsertNewChild[*Frame](tfr, idx)
	frame.SetName(label)
	frame.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	ts.InsertTabOnlyAt(frame, label, idx, icon...)
	ts.Update()
	return frame
}

// AddTab adds an already existing frame as a new tab with the given tab label
// and returns the index of that tab.
func (ts *Tabs) AddTab(frame *Frame, label string) int {
	fr := ts.FrameWidget()
	idx := len(fr.Children)
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
		ts.SelectTabByName(tab.Name)
	})
	fr := ts.FrameWidget()
	if len(fr.Children) == 1 {
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

	fr := ts.FrameWidget()
	fr.InsertChild(frame, idx)
	ts.InsertTabOnlyAt(frame, label, idx, icon...)
	ts.NeedsLayout()
}

// TabAtIndex returns content frame and tab button at given index, false if
// index out of range (emits log message)
func (ts *Tabs) TabAtIndex(idx int) (*Frame, *Tab, bool) {
	ts.Mu.Lock()
	defer ts.Mu.Unlock()

	fr := ts.FrameWidget()
	tb := ts.Tabs()
	sz := len(fr.Children)
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
	fr := ts.FrameWidget()
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
	fr := ts.FrameWidget()
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
	return tab.AsTree().IndexInParent()
}

// SelectTabByName selects tab by widget name, returning it.
// The widget name is the original full tab label, prior to any eliding.
func (ts *Tabs) SelectTabByName(name string) *Frame {
	idx := ts.TabIndexByName(name)
	if idx < 0 {
		return nil
	}
	ts.SelectTabIndex(idx)
	fr := ts.FrameWidget()
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
	wi.AsWidget().UpdateWidget()
	return wi
}

// DeleteTabIndex deletes tab at given index, returning whether it was successful.
func (ts *Tabs) DeleteTabIndex(idx int) bool {
	_, _, ok := ts.TabAtIndex(idx)
	if !ok {
		return false
	}

	ts.Mu.Lock()
	fr := ts.FrameWidget()
	sz := len(fr.Children)
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
	fr.DeleteChildAt(idx)
	tb.DeleteChildAt(idx)
	ts.Mu.Unlock()

	if nidx >= 0 {
		ts.SelectTabIndex(nidx)
	}
	ts.NeedsLayout()
	return true
}

// Tabs returns the layout containing the tabs (the first element within us).
// It configures the Tabs if necessary.
func (ts *Tabs) Tabs() *Frame {
	if ts.ChildByName("tabs", 0) == nil {
		ts.UpdateWidget()
	}
	return ts.ChildByName("tabs", 0).(*Frame)
}

// Frame returns the stacked frame layout (the second element within us).
// It configures the Tabs if necessary.
func (ts *Tabs) FrameWidget() *Frame {
	if ts.ChildByName("frame", 1) == nil {
		ts.UpdateWidget()
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

// Tab is a tab button that contains any, all, or none of a label, an icon,
// and a close icon. Tabs should be made using the [Tabs.NewTab] function.
type Tab struct { //core:no-new
	Frame

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

func (tb *Tab) Init() {
	tb.Frame.Init()
	tb.MaxChars = 16
	tb.CloseIcon = icons.Close
	tb.Styler(func(s *styles.Style) {
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

		s.Gap.Zero()
		s.Align.Content = styles.Center
		s.Align.Items = styles.Center

		if tb.StateIs(states.Selected) {
			s.Color = colors.C(colors.Scheme.Select.OnContainer)
		} else {
			s.Color = colors.C(colors.Scheme.OnSurfaceVariant)
			if tb.Type.Effective(tb) == FunctionalTabs {
				s.Background = colors.C(colors.Scheme.SurfaceContainer)
			}
		}
	})

	tb.HandleClickOnEnterSpace()

	tb.Maker(func(p *Plan) {
		if tb.MaxChars > 0 { // TODO(config): find a better time to do this?
			tb.Text = elide.Middle(tb.Text, tb.MaxChars)
		}

		if tb.Icon.IsSet() {
			AddAt(p, "icon", func(w *Icon) {
				w.Styler(func(s *styles.Style) {
					s.Font.Size.Dp(18)
				})
				w.Updater(func() {
					w.SetIcon(tb.Icon)
				})
			})
			if tb.Text != "" {
				AddAt(p, "space", func(w *Space) {})
			}
		}
		if tb.Text != "" {
			AddAt(p, "text", func(w *Text) {
				w.Styler(func(s *styles.Style) {
					s.SetNonSelectable()
					s.SetTextWrap(false)
				})
				w.Updater(func() {
					if tb.Type.Effective(tb) == FunctionalTabs {
						w.SetType(TextBodyMedium)
					} else {
						w.SetType(TextLabelLarge)
					}
					w.SetText(tb.Text)
				})
			})
		}
		if tb.Type.Effective(tb) == FunctionalTabs && tb.CloseIcon.IsSet() {
			AddAt(p, "close-space", func(w *Space) {})
			AddAt(p, "close", func(w *Button) {
				w.SetType(ButtonAction)
				w.Styler(func(s *styles.Style) {
					s.Padding.Zero()
					s.Border.Radius = styles.BorderRadiusFull
				})
				w.OnClick(func(e events.Event) {
					ts := tb.Tabs()
					idx := ts.TabIndexByName(tb.Name)
					// if OnlyCloseActiveTab is on, only process delete when already selected
					if SystemSettings.OnlyCloseActiveTab && !tb.StateIs(states.Selected) {
						ts.SelectTabIndex(idx)
					} else {
						ts.DeleteTabIndex(idx)
					}
				})
				w.Updater(func() {
					w.SetIcon(tb.CloseIcon)
				})
			})
		}
	})
}

// Tabs returns the parent [Tabs] of this [Tab].
func (tb *Tab) Tabs() *Tabs {
	return tb.Parent.AsTree().Parent.(*Tabs)
}
