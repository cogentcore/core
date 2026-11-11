// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"log/slog"
	"slices"
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
)

// Tabber is an interface for getting the parent Tabs of tab buttons.
type Tabber interface {
	// AsCoreTabs returns the underlying Tabs implementation.
	AsCoreTabs() *Tabs
}

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

	// NewTabFunc is a function that will be called when the NewTabButton
	// is clicked to create a new tab, with the index of this new tab.
	// This can be used to update the new tab contents.
	NewTabFunc func(index int)

	// CloseTabFunc is a function that will be called when a tab is closed.
	// with the index of the tab that was closed.
	// This can be used to manage any additional resources associated with the tab.
	CloseTabFunc func(index int)

	// maxChars is the maximum number of characters to include in the tab text.
	// It elides text that are longer than that.
	maxChars int

	// CloseIcon is the icon used for tab close buttons.
	// If it is "" or [icons.None], the tab is not closeable.
	// The default value is [icons.Close].
	// Only [FunctionalTabs] can be closed; all other types of
	// tabs will not render a close button and can not be closed.
	CloseIcon icons.Icon

	// DragToWindow, if set, makes a tab of [FunctionalTabs] that is
	// dragged out of its window move into a new window, which then moves
	// with the mouse until the drag ends, on desktop platforms, as in web
	// browsers. It is called with the body of the new window, which it
	// initializes as any new window of the app (as a new browser window),
	// and returns the tabs in it that the tab is moved into (at their end),
	// or nil to not move it. The last tab is not moved. Tabs of
	// [FunctionalTabs] can always be dragged to other places in their
	// tabs, and to other tabs of [FunctionalTabs] in the same window.
	// See also [Tabs.MoveTabFunc].
	DragToWindow func(b *Body) *Tabs

	// MoveTabFunc is a function that is called when a tab has been moved
	// from the given index, by dragging it or by [Tabs.MoveTabIndex], to
	// the given index in the given tabs: these tabs, or other tabs (as
	// those of a new window, for [Tabs.DragToWindow]). It can be used to
	// keep any additional resources associated with the tabs in their order,
	// and to move them to those of the other tabs.
	MoveTabFunc func(from int, to *Tabs, at int)

	// mu is a mutex protecting updates to tabs. Tabs can be driven
	// programmatically and via user input so need extra protection.
	mu sync.Mutex

	tabs, frame *Frame
}

// TabTypes are the different styling types of tabs.
type TabTypes int32 //enums:enum

const (
	// StandardTabs indicates to render the standard type
	// of Material Design style tabs.
	StandardTabs TabTypes = iota

	// FunctionalTabs indicates to render functional tabs
	// like those in Google Chrome. These tabs take up less
	// space and are the only kind that can be closed, and they
	// can be dragged to new positions (see also [Tabs.DragToWindow]).
	FunctionalTabs

	// NavigationAuto indicates to render the tabs as either
	// [NavigationBar] or [NavigationDrawer] if
	// [WidgetBase.SizeClass] is [SizeCompact] or not, respectively.
	// NavigationAuto should typically be used instead of one of the
	// specific navigation types for better cross-platform compatability.
	NavigationAuto

	// NavigationBar indicates to render the tabs as a
	// bottom navigation bar with text and icons.
	NavigationBar

	// NavigationDrawer indicates to render the tabs as a
	// side navigation drawer with text and icons.
	NavigationDrawer
)

// effective returns the effective tab type in the context
// of the given widget, handling [NavigationAuto] based on
// [WidgetBase.SizeClass].
func (tt TabTypes) effective(w Widget) TabTypes {
	if tt != NavigationAuto {
		return tt
	}
	switch w.AsWidget().SizeClass() {
	case SizeCompact:
		return NavigationBar
	default:
		return NavigationDrawer
	}
}

// isColumn returns whether the tabs should be arranged in a column.
func (tt TabTypes) isColumn() bool {
	return tt == NavigationDrawer
}

func (ts *Tabs) AsCoreTabs() *Tabs { return ts }

func (ts *Tabs) Init() {
	ts.Frame.Init()
	ts.maxChars = 16
	ts.CloseIcon = icons.Close
	ts.Styler(func(s *styles.Style) {
		s.Color = colors.Scheme.OnSurface
		s.Grow.Set(1, 1)
		if ts.Type.effective(ts).isColumn() {
			s.Direction = styles.Row
		} else {
			s.Direction = styles.Column
		}
	})

	ts.Maker(func(p *tree.Plan) {
		tree.AddAt(p, "tabs", func(w *Frame) {
			ts.tabs = w
			w.Styler(func(s *styles.Style) {
				s.Overflow.Set(styles.OverflowHidden) // no scrollbars!
				s.Gap.Set(units.Dp(4))

				if ts.Type.effective(ts).isColumn() {
					s.Direction = styles.Column
					s.Grow.Set(0, 1)
				} else {
					s.Direction = styles.Row
					s.Grow.Set(1, 0)
					s.Wrap = true
				}
			})
			// a tab dropped after the tabs goes to the end
			w.On(events.DragEnter, func(e events.Event) { e.SetHandled() })
			w.On(events.DragLeave, func(e events.Event) { e.SetHandled() })
			w.On(events.Drop, func(e events.Event) {
				td, ok := e.(*events.DragDrop).Data.(*tabDrag)
				if !ok {
					return
				}
				e.SetHandled()
				td.dropAt(ts, ts.NumTabs())
			})
			w.FinalStyler(func(s *styles.Style) {
				s.SetAbilities(ts.Type.effective(ts) == FunctionalTabs, abilities.Droppable)
			})
			w.Updater(func() {
				if !ts.NewTabButton {
					w.DeleteChildByName("new-tab-button")
					return
				}
				if w.ChildByName("new-tab-button") != nil {
					return
				}
				ntb := NewButton(w).SetType(ButtonAction).SetIcon(icons.Add)
				ntb.SetTooltip("Add a new tab").SetName("new-tab-button")
				ntb.OnClick(func(e events.Event) {
					ts.NewTab("New tab")
					idx := ts.NumTabs() - 1
					if ts.NewTabFunc != nil {
						ts.NewTabFunc(idx)
					}
					ts.SelectTabIndex(idx)
				})
			})
		})
		tree.AddAt(p, "frame", func(w *Frame) {
			ts.frame = w
			w.LayoutStackTopOnly = true // key for allowing each tab to have its own size
			w.Styler(func(s *styles.Style) {
				s.Display = styles.Stacked
				s.Min.Set(units.Dp(160), units.Dp(96))
				s.Grow.Set(1, 1)
			})
		})
		// frame comes before tabs in bottom navigation bar
		if ts.Type.effective(ts) == NavigationBar {
			p.Children[0], p.Children[1] = p.Children[1], p.Children[0]
		}
	})
}

// NumTabs returns the number of tabs.
func (ts *Tabs) NumTabs() int {
	fr := ts.getFrame()
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
	ts.mu.Lock()
	defer ts.mu.Unlock()
	fr := ts.getFrame()
	if fr.StackTop < 0 {
		return nil, -1
	}
	w := fr.Child(fr.StackTop).(Widget)
	return w, fr.StackTop
}

// NewTab adds a new tab with the given label and returns the resulting tab frame
// and associated tab button, which can be further customized as needed.
// It is the main end-user API for creating new tabs.
func (ts *Tabs) NewTab(label string) (*Frame, *Tab) {
	fr := ts.getFrame()
	idx := len(fr.Children)
	return ts.insertTab(label, idx, nil)
}

// NewLazyTab adds a new tab with the given label whose content is made
// by the given function when the tab is first shown, and returns the
// tab frame and button as [Tabs.NewTab] does. The content of the selected
// tab is made right away. This saves the time of making and styling the
// content of tabs that are not shown, such as at the startup of an app;
// once a tab has been shown, it is the same as a tab made with [Tabs.NewTab]
// and filled with the same function.
func (ts *Tabs) NewLazyTab(label string, build func(tab *Frame)) (*Frame, *Tab) {
	fr := ts.getFrame()
	return ts.insertTab(label, len(fr.Children), build)
}

// insertTab inserts a new tab with the given label at the given index position
// within the list of tabs and returns the resulting tab frame and button.
// If build is non-nil, it makes the content of the tab when the tab is first
// updated: updates skip the content of tabs that are not selected, so that is
// when the tab is first shown.
func (ts *Tabs) insertTab(label string, idx int, build func(tab *Frame)) (*Frame, *Tab) {
	tfr := ts.getFrame()
	alreadyExists := tfr.ChildByName(label) != nil
	frame := NewFrame()
	tfr.InsertChild(frame, idx)
	frame.SetName(label)
	frame.Styler(func(s *styles.Style) {
		// tab frames must scroll independently and grow
		s.Overflow.Set(styles.OverflowAuto)
		s.Grow.Set(1, 1)
		s.Direction = styles.Column
	})
	if build != nil {
		built := false
		frame.FirstUpdater(func() {
			if !built {
				built = true
				build(frame)
			}
		})
	}
	button := ts.insertTabButtonAt(label, idx)
	if alreadyExists {
		tree.SetUniqueName(frame)  // prevent duplicate names
		button.SetName(frame.Name) // must be the same name
	}
	ts.Update()
	return frame, button
}

// insertTabButtonAt inserts just the tab button at given index, after the panel has
// already been added to the frame; assumed to be wrapped in update. Generally
// for internal use only.
func (ts *Tabs) insertTabButtonAt(label string, idx int) *Tab {
	tb := ts.getTabs()
	tab := tree.New[Tab]()
	tb.InsertChild(tab, idx)
	tab.SetName(label)
	tab.SetText(label).SetType(ts.Type).SetCloseIcon(ts.CloseIcon).SetTooltip(label)
	tab.maxChars = ts.maxChars
	tab.OnClick(func(e events.Event) {
		ts.SelectTabByName(tab.Name)
	})
	fr := ts.getFrame()
	if len(fr.Children) == 1 {
		fr.StackTop = 0
		tab.SetSelected(true)
		// } else {
		// 	frame.SetState(true, states.Invisible) // new tab is invisible until selected
	}
	return tab
}

// TabAtIndex returns content frame and tab button at given index, nil if
// index out of range (emits log message).
func (ts *Tabs) TabAtIndex(idx int) (*Frame, *Tab) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	fr := ts.getFrame()
	tb := ts.getTabs()
	sz := len(fr.Children)
	if idx < 0 || idx >= sz {
		slog.Error("Tabs: index out of range for number of tabs", "index", idx, "numTabs", sz)
		return nil, nil
	}
	tab := tb.Child(idx).(*Tab)
	frame := fr.Child(idx).(*Frame)
	return frame, tab
}

// SelectTabIndex selects the tab at the given index, returning it or nil.
// This is the final tab selection path.
func (ts *Tabs) SelectTabIndex(idx int) *Frame {
	frame, tab := ts.TabAtIndex(idx)
	if frame == nil {
		return nil
	}
	fr := ts.getFrame()
	if fr.StackTop == idx {
		return frame
	}
	ts.mu.Lock()
	ts.unselectOtherTabs(idx)
	tab.SetSelected(true)
	fr.StackTop = idx
	ts.mu.Unlock()
	fr.Update()
	frame.DeferShown()
	return frame
}

// TabByName returns the tab [Frame] with the given widget name
// (nil if not found). The widget name is the original full tab label,
// prior to any eliding.
func (ts *Tabs) TabByName(name string) *Frame {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	fr := ts.getFrame()
	frame, _ := fr.ChildByName(name).(*Frame)
	return frame
}

// tabIndexByName returns the tab index for the given tab widget name
// and -1 if it can not be found.
// The widget name is the original full tab label, prior to any eliding.
func (ts *Tabs) tabIndexByName(name string) int {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	tb := ts.getTabs()
	tab := tb.ChildByName(name)
	if tab == nil {
		return -1
	}
	return tab.AsTree().IndexInParent()
}

// SelectTabByName selects the tab by widget name, returning it.
// The widget name is the original full tab label, prior to any eliding.
func (ts *Tabs) SelectTabByName(name string) *Frame {
	idx := ts.tabIndexByName(name)
	if idx < 0 {
		return nil
	}
	ts.SelectTabIndex(idx)
	fr := ts.getFrame()
	return fr.Child(idx).(*Frame)
}

// RecycleTab returns a tab with the given name, first by looking for an existing one,
// and if not found, making a new one. It returns the frame for the tab.
func (ts *Tabs) RecycleTab(name string) *Frame {
	frame := ts.TabByName(name)
	if frame == nil {
		frame, _ = ts.NewTab(name)
	}
	ts.SelectTabByName(name)
	return frame
}

// RecycleTabWidget returns a tab with the given widget type in the tab frame,
// first by looking for an existing one with the given name, and if not found,
// making and configuring a new one. It returns the resulting widget.
func RecycleTabWidget[T tree.NodeValue](ts *Tabs, name string) *T {
	fr := ts.RecycleTab(name)
	if fr.HasChildren() {
		return any(fr.Child(0)).(*T)
	}
	w := tree.New[T](fr)
	any(w).(Widget).AsWidget().UpdateWidget()
	return w
}

// DeleteTabIndex deletes the tab at the given index, returning whether it was successful.
func (ts *Tabs) DeleteTabIndex(idx int) bool {
	frame, _ := ts.TabAtIndex(idx)
	if frame == nil {
		return false
	}

	ts.mu.Lock()
	fr := ts.getFrame()
	sz := len(fr.Children)
	tb := ts.getTabs()
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
	ts.mu.Unlock()

	if nidx >= 0 {
		ts.SelectTabIndex(nidx)
	}
	ts.NeedsLayout()
	if ts.CloseTabFunc != nil {
		ts.CloseTabFunc(idx)
	}
	return true
}

// MoveTabIndex moves the tab at the given index to the given index (the
// one that it then has), returning whether it was successful. The
// selected tab stays selected.
func (ts *Tabs) MoveTabIndex(from, to int) bool {
	ts.mu.Lock()
	fr := ts.getFrame()
	tb := ts.getTabs()
	n := len(fr.Children)
	if from < 0 || from >= n || to < 0 || to >= n {
		ts.mu.Unlock()
		slog.Error("Tabs: index out of range for number of tabs", "from", from, "to", to, "numTabs", n)
		return false
	}
	if from == to {
		ts.mu.Unlock()
		return true
	}
	moveChild(fr, from, to)
	moveChild(tb, from, to)
	switch top := fr.StackTop; {
	case top == from:
		fr.StackTop = to
	case from < top && to >= top:
		fr.StackTop--
	case from > top && to <= top:
		fr.StackTop++
	}
	ts.mu.Unlock()
	ts.NeedsLayout()
	if ts.MoveTabFunc != nil {
		ts.MoveTabFunc(from, ts, to)
	}
	return true
}

// moveChild moves the child of the given node at the given index to the
// other given index, without deleting it.
func moveChild(n tree.Node, from, to int) {
	nb := n.AsTree()
	k := nb.Children[from]
	nb.Children = slices.Delete(nb.Children, from, from+1)
	nb.Children = slices.Insert(nb.Children, to, k)
}

// MoveTabTo moves the tab at the given index (its content and its button)
// into the given other tabs, at the given index there, where it is
// selected, returning the index that it has there, or -1 if it was not
// moved. The tab next to it is selected in these tabs, as when it is
// deleted, but it is not closed: [Tabs.CloseTabFunc] is not called, and
// [Tabs.MoveTabFunc] is.
func (ts *Tabs) MoveTabTo(idx int, other *Tabs, at int) int {
	frame, tab := ts.TabAtIndex(idx)
	if frame == nil || other == ts {
		return -1
	}
	label, icon, tooltip := tab.Name, tab.Icon, tab.Tooltip
	if f := ts.Scene.Events.focus; f != nil && (tree.Node(f) == frame || isParentOf(frame, f)) {
		ts.Scene.Events.focusClear() // (not to send events to it here)
	}

	ts.mu.Lock()
	fr := ts.getFrame()
	tb := ts.getTabs()
	nidx := -1
	if len(fr.Children) > 1 {
		nidx = idx
		if fr.StackTop == idx {
			nidx = max(idx-1, 0)
		} else if fr.StackTop > idx {
			nidx = fr.StackTop - 1
		} else {
			nidx = fr.StackTop
		}
	}
	tb.DeleteChildAt(idx)
	fr.StackTop = -1 // (selected again below)
	ofr := other.getFrame()
	at = min(max(at, 0), len(ofr.Children))
	tree.MoveToParent(frame, ofr) // (at the end)
	moveChild(ofr, len(ofr.Children)-1, at)
	ts.mu.Unlock()
	if nidx >= 0 {
		ts.SelectTabIndex(nidx)
	}
	ts.NeedsLayout()

	other.mu.Lock()
	nt := other.insertTabButtonAt(label, at)
	nt.SetIcon(icon)
	nt.SetTooltip(tooltip)
	if len(ofr.Children) > 1 && ofr.StackTop >= at {
		ofr.StackTop++ // (it is after the new tab)
	}
	other.mu.Unlock()
	other.SelectTabIndex(at)
	other.Update()
	if ts.MoveTabFunc != nil {
		ts.MoveTabFunc(idx, other, at)
	}
	return at
}

// isParentOf returns whether the given node is in the given parent node.
func isParentOf(parent tree.Node, n tree.Node) bool {
	for p := n.AsTree().Parent; p != nil; p = p.AsTree().Parent {
		if p == parent {
			return true
		}
	}
	return false
}

// tabDrag is the data of the drag of a tab.
type tabDrag struct {
	// tabs are the tabs of the dragged tab.
	tabs *Tabs

	// tab is the button of the dragged tab.
	tab *Tab
}

// dropAt moves the dragged tab to the given index in the given tabs, into
// which it was dropped: in its own tabs, it moves there (the index that it
// then has), and from other tabs, it is inserted there.
func (td *tabDrag) dropAt(ts *Tabs, at int) {
	from := td.tab.IndexInParent()
	if from < 0 {
		return
	}
	if td.tabs == ts {
		ts.MoveTabIndex(from, min(at, ts.NumTabs()-1))
		return
	}
	if ts.Type.effective(ts) == FunctionalTabs {
		td.tabs.MoveTabTo(from, ts, at)
	}
}

// getTabs returns the [Frame] containing the tabs (the first element within us).
// It configures the [Tabs] if necessary.
func (ts *Tabs) getTabs() *Frame {
	if ts.tabs == nil {
		ts.UpdateWidget()
	}
	return ts.tabs
}

// Frame returns the stacked [Frame] (the second element within us).
// It configures the Tabs if necessary.
func (ts *Tabs) getFrame() *Frame {
	if ts.frame == nil {
		ts.UpdateWidget()
	}
	return ts.frame
}

// unselectOtherTabs turns off all the tabs except given one
func (ts *Tabs) unselectOtherTabs(idx int) {
	sz := ts.NumTabs()
	tbs := ts.getTabs()
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

// Tab is a tab button that contains one or more of a label, an icon,
// and a close icon. Tabs should be made using the [Tabs.NewTab] function.
type Tab struct { //core:no-new
	Frame

	// Type is the styling type of the tab. This property
	// must be set on the parent [Tabs] for it to work correctly.
	Type TabTypes

	// Text is the text for the tab. If it is blank, no text is shown.
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

	// maxChars is the maximum number of characters to include in tab text.
	// It elides text that is longer than that.
	maxChars int
}

func (tb *Tab) Init() {
	tb.Frame.Init()
	tb.maxChars = 16
	tb.CloseIcon = icons.Close
	tb.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Activatable, abilities.Focusable, abilities.Hoverable)

		if !tb.IsReadOnly() {
			s.Cursor = cursors.Pointer
		}

		if tb.Type.effective(tb).isColumn() {
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

		s.Font.Size.Dp(14)
		s.IconSize.Set(units.Em(18.0 / 14))

		if tb.StateIs(states.Selected) {
			s.Color = colors.Scheme.Select.OnContainer
		} else {
			s.Color = colors.Scheme.OnSurfaceVariant
			if tb.Type.effective(tb) == FunctionalTabs {
				s.Background = colors.Scheme.SurfaceContainer
			}
		}
	})

	tb.SendClickOnEnter()

	// FunctionalTabs can be dragged to other places
	tb.FinalStyler(func(s *styles.Style) {
		s.SetAbilities(tb.Type.effective(tb) == FunctionalTabs, abilities.Draggable, abilities.Droppable)
	})
	tb.On(events.DragStart, func(e events.Event) {
		ts := tb.tabs()
		if ts == nil || !tb.AbilityIs(abilities.Draggable) {
			return
		}
		e.SetHandled()
		tb.Scene.Events.DragStart(tb.This.(Widget), &tabDrag{tabs: ts, tab: tb}, e)
	})
	tb.On(events.DragEnter, func(e events.Event) {
		if _, ok := tb.Scene.Events.dragData.(*tabDrag); ok {
			tb.SetState(true, states.DragHovered)
			tb.NeedsRender()
			e.SetHandled()
		}
	})
	tb.On(events.DragLeave, func(e events.Event) {
		tb.SetState(false, states.DragHovered)
		tb.NeedsRender()
		e.SetHandled()
	})
	tb.On(events.Drop, func(e events.Event) {
		td, ok := e.(*events.DragDrop).Data.(*tabDrag)
		ts := tb.tabs()
		if !ok || ts == nil {
			return
		}
		e.SetHandled()
		td.dropAt(ts, tb.IndexInParent())
	})

	tb.Maker(func(p *tree.Plan) {
		if tb.maxChars > 0 { // TODO: find a better time to do this?
			tb.Text = elide.Middle(tb.Text, tb.maxChars)
		}

		if tb.Icon.IsSet() {
			tree.AddAt(p, "icon", func(w *Icon) {
				w.Updater(func() {
					w.SetIcon(tb.Icon)
				})
			})
			if tb.Text != "" {
				tree.AddAt(p, "space", func(w *Space) {})
			}
		}
		if tb.Text != "" {
			tree.AddAt(p, "text", func(w *Text) {
				w.Styler(func(s *styles.Style) {
					s.SetNonSelectable()
					s.SetTextWrap(false)
					s.FillMargin = false
					s.Font.Size = tb.Styles.Font.Size // Directly inherit to override the [Text.Type]-based default
				})
				w.Updater(func() {
					if tb.Type.effective(tb) == FunctionalTabs {
						w.SetType(TextBodyMedium)
					} else {
						w.SetType(TextLabelLarge)
					}
					w.SetText(tb.Text)
				})
			})
		}
		if tb.Type.effective(tb) == FunctionalTabs && tb.CloseIcon.IsSet() {
			tree.AddAt(p, "close-space", func(w *Space) {})
			tree.AddAt(p, "close", func(w *Button) {
				w.SetType(ButtonAction)
				w.Styler(func(s *styles.Style) {
					s.Padding.Zero()
					s.Border.Radius = styles.BorderRadiusFull
				})
				w.OnClick(func(e events.Event) {
					ts := tb.tabs()
					idx := ts.tabIndexByName(tb.Name)
					// if OnlyCloseActiveTab is on, only process delete when already selected
					if AppearanceSettings.OnlyCloseActiveTab && !tb.StateIs(states.Selected) {
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

// dragMovesDefault returns that a drag of a tab moves it (see [dragMover]).
func (tb *Tab) dragMovesDefault() bool { return true }

// dragToWindow moves the tab into a new window when it is dragged out of
// its window, if its tabs have [Tabs.DragToWindow] and another tab (see
// [windowDragger]).
func (tb *Tab) dragToWindow(e events.Event) (*Body, func() image.Point) {
	ts := tb.tabs()
	if ts == nil || ts.DragToWindow == nil || ts.Type.effective(ts) != FunctionalTabs || ts.NumTabs() < 2 {
		return nil, nil
	}
	idx := tb.IndexInParent()
	src := ts.Scene.RenderWindow()
	if idx < 0 || src == nil || src.SystemWindow == nil {
		return nil, nil
	}
	// the place of the mouse in the tab, which stays under the mouse
	in := e.StartPos().Sub(tb.Geom.TotalBBox.Min)
	// (moved into the new tabs before their window runs)
	bd := NewBody(tb.Name)
	nt := ts.DragToWindow(bd)
	if nt == nil {
		return nil, nil
	}
	at := ts.MoveTabTo(idx, nt, nt.NumTabs())
	if at < 0 {
		return nil, nil
	}
	bd.RunWindow()
	if win := bd.Scene.RenderWindow(); win != nil && win.SystemWindow != nil {
		// of the size of the window of the tabs, as in web browsers
		win.SystemWindow.SetSize(src.SystemWindow.Size())
	}
	grab := &grabPoint{}
	grab.set(in)
	// (the new tab is laid out when the window is shown)
	_, ntab := nt.TabAtIndex(at)
	bd.OnShow(func(e events.Event) {
		grab.set(ntab.Geom.TotalBBox.Min.Add(in))
	})
	return bd, grab.get
}

// tabs returns the parent [Tabs] of this [Tab].
func (tb *Tab) tabs() *Tabs {
	if tbr, ok := tb.Parent.AsTree().Parent.(Tabber); ok {
		return tbr.AsCoreTabs()
	}
	return nil
}

func (tb *Tab) Label() string {
	if tb.Text != "" {
		return tb.Text
	}
	return tb.Name
}
