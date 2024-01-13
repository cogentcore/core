// Copyright (c) 2018, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"log/slog"
	"sync"

	"goki.dev/abilities"
	"goki.dev/colors"
	"goki.dev/cursors"
	"goki.dev/events"
	"goki.dev/glop/elide"
	"goki.dev/gti"
	"goki.dev/icons"
	"goki.dev/ki"
	"goki.dev/states"
	"goki.dev/styles"
	"goki.dev/units"
)

// Tabs switches among child widgets via tabs.  The selected widget gets
// the full allocated space avail after the tabs are accounted for.  The
// Tabs is just a Vertical layout that manages two child widgets: a
// HorizFlow Layout for the tabs (which can flow across multiple rows as
// needed) and a Stacked Frame that actually contains all the children, and
// provides scrollbars as needed to any content within.  Typically should have
// max stretch and a set preferred size, so it expands.
type Tabs struct { //goki:embedder
	Layout

	// Type is the styling type of the tabs. It must be set
	// before the tabs are first configured.
	Type TabTypes

	// Maximum number of characters to include in tab label.
	// Elides labels that are longer than that
	MaxChars int

	// show a new tab button at right of list of tabs
	NewTabButton bool

	// CloseIcon is the icon used for tab close buttons.
	// If it is "" or [icons.None], the tab is not closeable.
	// The default value is [icons.Close].
	// Only [FunctionalTabs] can be closed; all other types of
	// tabs will not render a close button and can not be closed.
	CloseIcon icons.Icon

	// mutex protecting updates to tabs.
	// Tabs can be driven programmatically and via user input so need extra protection
	Mu sync.Mutex `copy:"-" json:"-" xml:"-" view:"-" set:"-"`
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
	// side navigation drawer, which has full text labels and icons.
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

func (ts *Tabs) CopyFieldsFrom(frm any) {
	fr := frm.(*Tabs)
	ts.Layout.CopyFieldsFrom(&fr.Layout)
	ts.MaxChars = fr.MaxChars
	ts.NewTabButton = fr.NewTabButton
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
		// need border for separators (see RenderTabSeps)
		// TODO: maybe better solution for tab sep styles?
		s.Border.Style.Set(styles.BorderSolid)
		s.Border.Width.Set(units.Dp(1))
		s.Border.Color.Set(colors.Scheme.OutlineVariant)
		s.Color = colors.Scheme.OnBackground
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
				s.Wrap = true
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

// NTabs returns number of tabs
func (ts *Tabs) NTabs() int {
	fr := ts.Frame()
	if fr == nil {
		return 0
	}
	return len(fr.Kids)
}

// CurTab returns currently-selected tab, and its index -- returns false none
func (ts *Tabs) CurTab() (Widget, int, bool) {
	if ts.NTabs() == 0 {
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
	updt := ts.UpdateStart()
	defer ts.UpdateEndLayout(updt)

	fr := ts.Frame()
	fr.SetChildAdded()
	frame := fr.InsertNewChild(FrameType, idx, label).(*Frame)
	frame.Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	ts.InsertTabOnlyAt(frame, label, idx, icon...)
	ts.Update()
	ts.SetNeedsLayout(true)
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
	tb.SetChildAdded()
	tab := tb.InsertNewChild(TabType, idx, label).(*Tab)
	tab.Tooltip = label
	tab.Type = ts.Type
	tab.CloseIcon = ts.CloseIcon
	tab.MaxChars = ts.MaxChars
	tab.SetText(label)
	if len(icon) > 0 {
		tab.SetIcon(icon[0])
	}
	tab.OnClick(func(e events.Event) {
		ts.SelectTabByLabel(tab.Nm)
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
	updt := ts.UpdateStart()
	defer ts.UpdateEndLayout(updt)

	fr := ts.Frame()
	fr.SetChildAdded()
	fr.InsertChild(frame, idx)
	ts.InsertTabOnlyAt(frame, label, idx, icon...)
	ts.Mu.Unlock()
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
		slog.Error("gi.Tabs: index out of range for number of tabs", "index", idx, "numTabs", sz)
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
	updt := ts.UpdateStart()
	defer ts.UpdateEndLayout(updt)
	ts.UnselectOtherTabs(idx)
	tab.SetSelected(true)
	fr.StackTop = idx
	fr.Update()
	ts.Mu.Unlock()
	return frame, true
}

// TabByLabel returns tab with given label (nil if not found)
func (ts *Tabs) TabByLabel(label string) *Frame {
	ts.Mu.Lock()
	defer ts.Mu.Unlock()
	fr := ts.Frame()
	frame, _ := fr.ChildByName(label).(*Frame)
	return frame
}

// TabIndexByLabel returns the tab index for the given tab label
// and -1 if it can not be found.
func (ts *Tabs) TabIndexByLabel(label string) int {
	ts.Mu.Lock()
	defer ts.Mu.Unlock()

	tb := ts.Tabs()
	tab := tb.ChildByName(label)
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

// SelectTabByLabel selects tab by label, returning it.
func (ts *Tabs) SelectTabByLabel(label string) *Frame {
	idx := ts.TabIndexByLabel(label)
	if idx < 0 {
		return nil
	}
	ts.SelectTabIndex(idx)
	fr := ts.Frame()
	return fr.Child(idx).(*Frame)
}

// RecycleTab returns a tab with given label, first by looking for an existing one,
// and if not found, making a new one. If sel, then select it. It returns the
// frame for the tab.
func (ts *Tabs) RecycleTab(label string, sel bool) *Frame {
	frame := ts.TabByLabel(label)
	if frame != nil {
		if sel {
			ts.SelectTabByLabel(label)
		}
		return frame
	}
	frame = ts.NewTab(label)
	if sel {
		ts.SelectTabByLabel(label)
	}
	return frame
}

// RecycleTabWidget returns a tab with given widget type in the tab frame,
// first by looking for an existing one, with given label, and if not found,
// making and configuring a new one.
// If sel, then select it. It returns the Widget item for the tab.
func (ts *Tabs) RecycleTabWidget(label string, sel bool, typ *gti.Type) Widget {
	fr := ts.RecycleTab(label, sel)
	if fr.HasChildren() {
		wi, _ := AsWidget(fr.Child(0))
		return wi
	}
	wi, _ := AsWidget(fr.NewChild(typ, fr.Nm))
	wi.Config()
	return wi
}

// DeleteTabIndex deletes tab at given index, optionally calling destroy on
// tab contents -- returns frame if destroy == false, tab label, and bool success
func (ts *Tabs) DeleteTabIndex(idx int, destroy bool) (*Frame, string, bool) {
	frame, _, ok := ts.TabAtIndex(idx)
	if !ok {
		return nil, "", false
	}

	tnm := ts.TabLabel(idx)
	ts.Mu.Lock()
	fr := ts.Frame()
	sz := len(*fr.Children())
	tb := ts.Tabs()
	updt := ts.UpdateStart()
	defer ts.UpdateEndLayout(updt)
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
	if nidx < 0 && ts.NTabs() > 1 {
		nidx = max(idx-1, 0)
	}
	fr.DeleteChildAtIndex(idx, destroy)
	tb.DeleteChildAtIndex(idx, ki.DestroyKids) // always destroy -- we manage
	ts.Mu.Unlock()

	if nidx >= 0 {
		ts.SelectTabIndex(nidx)
	}
	if destroy {
		return nil, tnm, true
	} else {
		return frame, tnm, true
	}
}

// ConfigNewTabButton configures the new tab + button at end of list of tabs
func (ts *Tabs) ConfigNewTabButton() bool {
	sz := ts.NTabs()
	tb := ts.Tabs()
	ntb := len(tb.Kids)
	if ts.NewTabButton {
		if ntb == sz+1 {
			return false
		}
		tab := tb.InsertNewChild(ButtonType, ntb, "new-tab").(*Button)
		tab.Data = -1
		tab.SetIcon(icons.Add).SetType(ButtonAction)
		tab.OnClick(func(e events.Event) {
			ts.NewTab("New Tab")
			ts.SelectTabIndex(len(*ts.Frame().Children()) - 1)
		})
		return true
	} else {
		if ntb == sz {
			return false
		}
		tb.DeleteChildAtIndex(ntb-1, ki.DestroyKids) // always destroy -- we manage
		return true
	}
}

// ConfigWidget initializes the tab widget children if it hasn't been done yet.
// only the 2 primary children (Frames) need to be configured.
// no re-config needed when adding / deleting tabs -- just new layout.
func (ts *Tabs) ConfigWidget() {
	if len(ts.Kids) != 0 {
		return
	}
	// frame only comes before tabs in bottom nav bar
	if ts.Type.Effective(ts) == NavigationBar {
		NewFrame(ts, "frame")
		NewFrame(ts, "tabs")
	} else {
		NewFrame(ts, "tabs")
		NewFrame(ts, "frame")
	}
	ts.ConfigNewTabButton()
}

// Tabs returns the layout containing the tabs (the first element within us).
// It configures the Tabs if necessary.
func (ts *Tabs) Tabs() *Frame {
	ts.ConfigWidget()
	return ts.ChildByName("tabs", 0).(*Frame)
}

// Frame returns the stacked frame layout (the second element within us).
// It configures the Tabs if necessary.
func (ts *Tabs) Frame() *Frame {
	ts.ConfigWidget()
	return ts.ChildByName("frame", 1).(*Frame)
}

// UnselectOtherTabs turns off all the tabs except given one
func (ts *Tabs) UnselectOtherTabs(idx int) {
	sz := ts.NTabs()
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
type Tab struct { //goki:no-new
	Box

	// Type is the styling type of the tab. This property
	// must be set on the parent [Tabs] for it to work correctly.
	Type TabTypes

	// Text is the label text for the tab.
	// If it is nil, no label is shown.
	// Labels are never shown for [NavigationRail] tabs.
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

	// Maximum number of characters to include in tab label.
	// Elides labels that are longer than that
	MaxChars int
}

func (tb *Tab) OnInit() {
	tb.Box.OnInit()
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
		s.Max.X.Ch(float32(tb.MaxChars))
		// s.Min.Y.Ch(6)

		// s.Border.Style.Right = styles.BorderSolid
		// s.Border.Color.Right = colors.Scheme.OutlineVariant
		// s.Border.Width.Right.Dp(1)

		s.Text.Align = styles.Center
		s.Margin.Zero()

		if tb.Type.Effective(tb).IsColumn() {
			s.Grow.X = 1
			s.Border.Radius = styles.BorderRadiusFull
			s.Padding.Set(units.Dp(16))
		} else {
			s.Border.Radius = styles.BorderRadiusSmall
			s.Padding.Set(units.Dp(10))
		}

		if tb.StateIs(states.Selected) {
			s.Color = colors.Scheme.Select.OnContainer
		} else {
			s.Color = colors.Scheme.OnSurfaceVariant
			if tb.Type.Effective(tb) == FunctionalTabs {
				s.Background = colors.C(colors.Scheme.SurfaceContainer)
			}
		}

		// s.Border.Style.Set(styles.BorderNone)
		// if tb.StateIs(states.Selected) {
		// 	s.Border.Style.Bottom = styles.BorderSolid
		// 	s.Border.Width.Bottom.SetDp(2)
		// 	s.Border.Color.Bottom = colors.Scheme.Primary
		// }
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
				s.Min.X.Dp(18)
				s.Min.Y.Dp(18)
				s.Margin.Zero()
				s.Padding.Zero()
				if tb.Text != "" {
					s.Margin.Right.Ch(1)
				}
			})
		case "parts/label":
			label := w.(*Label)
			if tb.Type.Effective(tb) == FunctionalTabs {
				label.Type = LabelBodyMedium
			} else {
				label.Type = LabelLabelLarge
			}
			w.Style(func(s *styles.Style) {
				s.SetNonSelectable()
				s.SetTextWrap(false)
				s.Margin.Zero()
				s.Padding.Zero()
			})
		case "parts/close-stretch":
			w.Style(func(s *styles.Style) {
				s.Min.X.Ch(1)
				s.Grow.Set(1, 0)
			})
		case "parts/close.parts/icon":
			w.Style(func(s *styles.Style) {
				s.Max.X.Dp(16)
				s.Max.Y.Dp(16)
			})
		case "parts/close":
			w.Style(func(s *styles.Style) {
				s.Padding.Zero()
				s.Margin.Left.Dp(16)
				s.Border.Radius = styles.BorderRadiusFull
			})
			w.OnClick(func(e events.Event) {
				ts := tb.Tabs()
				if ts == nil {
					return
				}
				idx := ts.TabIndexByLabel(tb.Text)
				// if OnlyCloseActiveTab is on, only process delete when already selected
				if SystemSettings.Behavior.OnlyCloseActiveTab && !tb.StateIs(states.Selected) {
					ts.SelectTabIndex(idx)
				} else {
					ts.DeleteTabIndex(idx, true)
				}
			})
		case "parts/sc-stretch":
			w.Style(func(s *styles.Style) {
				s.Min.X.Ch(2)
				s.Grow.Set(1, 0)
			})
		case "parts/shortcut":
			w.Style(func(s *styles.Style) {
				s.Margin.Zero()
				s.Padding.Zero()
			})
		}
	})
}

func (tb *Tab) Tabs() *Tabs {
	ts := tb.ParentByType(TabsType, ki.Embeds)
	if ts == nil {
		return nil
	}
	return AsTabs(ts)
}

func (tb *Tab) ConfigWidget() {
	config := ki.Config{}
	if tb.MaxChars > 0 {
		tb.Text = elide.Middle(tb.Text, tb.MaxChars)
	}

	ici := -1
	lbi := -1
	clsi := -1
	if tb.Icon.IsValid() {
		ici = len(config)
		config.Add(IconType, "icon")
	}
	if tb.Text != "" {
		lbi = len(config)
		config.Add(LabelType, "label")
	}
	if tb.Type.Effective(tb) == FunctionalTabs && tb.CloseIcon.IsValid() {
		clsi = len(config)
		config.Add(ButtonType, "close")
	}

	tb.ConfigParts(config, func() {
		if ici >= 0 {
			ic := tb.Parts.Child(ici).(*Icon)
			ic.SetIcon(tb.Icon)
		}
		if lbi >= 0 {
			lbl := tb.Parts.Child(lbi).(*Label)
			if lbl.Text != tb.Text {
				lbl.SetTextUpdate(tb.Text)
			}
		}
		if clsi >= 0 {
			cls := tb.Parts.Child(clsi).(*Button)
			cls.SetType(ButtonAction).SetIcon(tb.CloseIcon)
		}
	})
}
