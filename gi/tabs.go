// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/iancoleman/strcase"
	"goki.dev/colors"
	"goki.dev/cursors"
	"goki.dev/girl/abilities"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/gti"
	"goki.dev/icons"
	"goki.dev/ki/v2"
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

	// maximum number of characters to include in tab label -- elides labels that are longer than that
	MaxChars int

	// show a new tab button at right of list of tabs
	NewTabButton bool

	// if true, tabs are user-deleteable (true by default)
	DeleteTabButtons bool

	// mutex protecting updates to tabs -- tabs can be driven programmatically and via user input so need extra protection
	Mu sync.Mutex `copy:"-" json:"-" xml:"-" view:"-" set:"-"`
}

func (ts *Tabs) CopyFieldsFrom(frm any) {
	fr := frm.(*Tabs)
	ts.Layout.CopyFieldsFrom(&fr.Layout)
	ts.MaxChars = fr.MaxChars
	ts.NewTabButton = fr.NewTabButton
}

func (ts *Tabs) OnInit() {
	ts.DeleteTabButtons = true
	ts.HandleTabsEvents()
	ts.TabsStyles()
}

func (ts *Tabs) HandleTabsEvents() {
	ts.HandleLayoutEvents()
}

func (ts *Tabs) TabsStyles() {
	ts.Style(func(s *styles.Style) {
		// need border for separators (see RenderTabSeps)
		// TODO: maybe better solution for tab sep styles?
		s.Border.Style.Set(styles.BorderSolid)
		s.Border.Width.Set(units.Dp(1))
		s.Border.Color.Set(colors.Scheme.OutlineVariant)
		s.Color = colors.Scheme.OnBackground
		s.SetStretchMax()
	})
	ts.OnWidgetAdded(func(w Widget) {
		switch w.PathFrom(ts) {
		case "tabs":
			w.Style(func(s *styles.Style) {
				s.SetStretchMaxWidth()
				s.MaxHeight.Zero()
				s.Height.Em(1.8)
				s.Overflow = styles.OverflowHidden // no scrollbars!
				s.Margin.Set()
				s.Padding.Set()
				s.Spacing.Zero()
				s.BackgroundColor.SetSolid(colors.Scheme.SurfaceContainer)

				// s.Border.Style.Set(styles.BorderNone)
				// s.Border.Style.Bottom = styles.BorderSolid
				// s.Border.Width.Bottom.SetDp(1)
				// s.Border.Color.Bottom = colors.Scheme.OutlineVariant
			})
		case "frame":
			frame := w.(*Frame)
			frame.SetFlag(true, LayoutStackTopOnly) // key for allowing each tab to have its own size
			w.Style(func(s *styles.Style) {
				s.SetMinPrefWidth(units.Em(10))
				s.SetMinPrefHeight(units.Em(6))
				s.SetStretchMax()
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

// TODO(kai): once subscenes are working, we should make tabs be subscenes

// NewTab adds a new tab with the given label and returns the resulting tab frame.
// It is the main end-user API for creating new tabs. If a name is also passed,
// the internal name (ID) of the tab will be set to that; otherwise, it will default
// to the kebab-case version of the label.
func (ts *Tabs) NewTab(label string, name ...string) *Frame {
	fr := ts.Frame()
	idx := len(*fr.Children())
	frame := ts.InsertNewTab(label, idx, name...)
	return frame
}

// InsertNewTab inserts a new tab with the given label at the given index position
// within the list of tabs and returns the resulting tab frame. If a name is also
// passed, the internal name (ID) of the tab will be set to that; otherwise, it will default
// to the kebab-case version of the label.
func (ts *Tabs) InsertNewTab(label string, idx int, name ...string) *Frame {
	updt := ts.UpdateStart()
	fr := ts.Frame()
	fr.SetChildAdded()
	nm := ""
	if len(name) > 0 {
		nm = name[0]
	} else {
		nm = strcase.ToKebab(label)
	}
	frame := fr.InsertNewChild(FrameType, idx, nm).(*Frame)
	frame.SetLayout(LayoutVert)
	ts.InsertTabOnlyAt(frame, label, idx, nm)
	ts.UpdateEndLayout(updt)
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

// InsertTabOnlyAt inserts just the tab at given index, after the panel has
// already been added to the frame; assumed to be wrapped in update. Generally
// for internal use only. If a name is also passed, the internal name (ID) of the tab
// will be set to that; otherwise, it will default to the kebab-case version of the label.
func (ts *Tabs) InsertTabOnlyAt(frame *Frame, label string, idx int, name ...string) {
	tb := ts.Tabs()
	tb.SetChildAdded()
	nm := ""
	if len(name) > 0 {
		nm = name[0]
	} else {
		nm = strcase.ToKebab(label)
	}
	tab := tb.InsertNewChild(TabType, idx, nm).(*Tab)
	tab.Data = idx
	tab.Tooltip = label
	tab.DeleteButton = ts.DeleteTabButtons
	tab.SetText(label)
	tab.OnClick(func(e events.Event) {
		ts.SelectTabIndex(idx)
	})
	fr := ts.Frame()
	if len(fr.Kids) == 1 {
		fr.StackTop = 0
		tab.SetSelected(true)
	} else {
		frame.SetState(true, states.Invisible) // new tab is invisible until selected
	}
}

// InsertTab inserts a frame into given index position within list of tabs.
// If a name is also passed, the internal name (ID) of the tab will be set
// to that; otherwise, it will default to the kebab-case version of the label.
func (ts *Tabs) InsertTab(frame *Frame, label string, idx int, name ...string) {
	ts.Mu.Lock()
	fr := ts.Frame()
	updt := ts.UpdateStart()
	fr.SetChildAdded()
	fr.InsertChild(frame, idx)
	ts.InsertTabOnlyAt(frame, label, idx, name...)
	ts.Mu.Unlock()
	ts.UpdateEndLayout(updt)
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

// SelectTabIndex selects tab at given index, returning it -- returns false if
// index is invalid
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
	ts.UnselectOtherTabs(idx)
	tab.SetSelected(true)
	fr.StackTop = idx
	ts.Mu.Unlock()
	ts.UpdateEndLayout(updt)
	return frame, true
}

// TabByName returns tab with given name (nil if not found -- see TabByNameTry)
func (ts *Tabs) TabByName(label string) *Frame {
	t, _ := ts.TabByNameTry(label)
	return t
}

// TabByNameTry returns tab with given name, and an error if not found.
func (ts *Tabs) TabByNameTry(label string) (*Frame, error) {
	ts.Mu.Lock()
	defer ts.Mu.Unlock()

	tb := ts.Tabs()
	idx, ok := tb.Children().IndexByName(label, 0)
	if !ok {
		return nil, fmt.Errorf("gi.Tabs: Tab named %v not found in %v", label, ts.Path())
	}
	fr := ts.Frame()
	frame := fr.Child(idx).(*Frame)
	return frame, nil
}

// TabIndexByName returns tab index for given tab name, and an error if not found.
func (ts *Tabs) TabIndexByName(label string) (int, error) {
	ts.Mu.Lock()
	defer ts.Mu.Unlock()

	tb := ts.Tabs()
	idx, ok := tb.Children().IndexByName(label, 0)
	if !ok {
		return -1, fmt.Errorf("gi.Tabs: Tab named %v not found in %v", label, ts.Path())
	}
	return idx, nil
}

// TabName returns tab name at given index
func (ts *Tabs) TabName(idx int) string {
	ts.Mu.Lock()
	defer ts.Mu.Unlock()

	tb := ts.Tabs()
	tbut, err := tb.ChildTry(idx)
	if err != nil {
		return ""
	}
	return tbut.Name()
}

// SelectTabByName selects tab by name, returning it.
func (ts *Tabs) SelectTabByName(label string) *Frame {
	idx, err := ts.TabIndexByName(label)
	if err == nil {
		ts.SelectTabIndex(idx)
		fr := ts.Frame()
		return fr.Child(idx).(*Frame)
	}
	return nil
}

// SelectTabByNameTry selects tab by name, returning it.  Returns error if not found.
func (ts *Tabs) SelectTabByNameTry(label string) (*Frame, error) {
	idx, err := ts.TabIndexByName(label)
	if err == nil {
		ts.SelectTabIndex(idx)
		fr := ts.Frame()
		return fr.Child(idx).(*Frame), nil
	}
	return nil, err
}

// RecycleTab returns a tab with given name, first by looking for an existing one,
// and if not found, making a new one. If sel, then select it. It returns the
// frame for the tab. If a name is also passed, the internal name (ID) of any new tab
// will be set to that; otherwise, it will default to the kebab-case version of the label.
func (ts *Tabs) RecycleTab(label string, sel bool, name ...string) *Frame {
	frame, err := ts.TabByNameTry(label)
	if err == nil {
		if sel {
			ts.SelectTabByName(label)
		}
		return frame
	}
	frame = ts.NewTab(label, name...)
	if sel {
		ts.SelectTabByName(label)
	}
	return frame
}

// RecycleTabWidget returns a tab with given widget type in the tab frame,
// first by looking for an existing one, with given name, and if not found,
// making and configuring a new one.
// If sel, then select it. It returns the Widget item for the tab.
// If a name is also passed, the internal name (ID) of any new tab
// will be set to that; otherwise, it will default to the kebab-case version of the label.
func (ts *Tabs) RecycleTabWidget(label string, sel bool, typ *gti.Type, name ...string) Widget {
	fr := ts.RecycleTab(label, sel, name...)
	if fr.HasChildren() {
		wi, _ := AsWidget(fr.Child(0))
		return wi
	}
	wi, _ := AsWidget(fr.NewChild(typ, fr.Nm))
	wi.Config(ts.Sc)
	return wi
}

// DeleteTabIndex deletes tab at given index, optionally calling destroy on
// tab contents -- returns frame if destroy == false, tab name, and bool success
func (ts *Tabs) DeleteTabIndex(idx int, destroy bool) (*Frame, string, bool) {
	frame, _, ok := ts.TabAtIndex(idx)
	if !ok {
		return nil, "", false
	}

	tnm := ts.TabName(idx)
	ts.Mu.Lock()
	fr := ts.Frame()
	sz := len(*fr.Children())
	tb := ts.Tabs()
	updt := ts.UpdateStart()
	nxtidx := -1
	if fr.StackTop == idx {
		if idx > 0 {
			nxtidx = idx - 1
		} else if idx < sz-1 {
			nxtidx = idx
		}
	}
	fr.DeleteChildAtIndex(idx, destroy)
	tb.DeleteChildAtIndex(idx, ki.DestroyKids) // always destroy -- we manage
	ts.RenumberTabs()
	ts.Mu.Unlock()
	if nxtidx >= 0 {
		ts.SelectTabIndex(nxtidx)
	}
	ts.UpdateEndLayout(updt)
	if destroy {
		return nil, tnm, true
	} else {
		return frame, tnm, true
	}
}

// ConfigNewTabButton configures the new tab + button at end of list of tabs
func (ts *Tabs) ConfigNewTabButton(sc *Scene) bool {
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

// TabsSignals are signals that the Tabs can send
type TabsSignals int64

const (
	// TabSelected indicates tab was selected -- data is the tab index
	TabSelected TabsSignals = iota

	// TabAdded indicates tab was added -- data is the tab index
	TabAdded

	// TabDeleted indicates tab was deleted -- data is the tab name
	TabDeleted

	TabsSignalsN
)

// ConfigWidget initializes the tab widget children if it hasn't been done yet.
// only the 2 primary children (Frames) need to be configured.
// no re-config needed when adding / deleting tabs -- just new layout.
func (ts *Tabs) ConfigWidget(sc *Scene) {
	if len(ts.Kids) != 0 {
		return
	}
	ts.Lay = LayoutVert

	frame := NewFrame(ts, "tabs")
	frame.Lay = LayoutHorizFlow

	frame = NewFrame(ts, "frame")
	frame.Lay = LayoutStacked

	ts.ConfigNewTabButton(sc)
}

// Tabs returns the layout containing the tabs (the first element within us).
// It configures the Tabs if necessary.
func (ts *Tabs) Tabs() *Frame {
	ts.ConfigWidget(ts.Sc)
	return ts.Child(0).(*Frame)
}

// Frame returns the stacked frame layout (the second element within us).
// It configures the Tabs if necessary.
func (ts *Tabs) Frame() *Frame {
	ts.ConfigWidget(ts.Sc)
	return ts.Child(1).(*Frame)
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

// RenumberTabs assigns proper index numbers to each tab
func (ts *Tabs) RenumberTabs() {
	sz := ts.NTabs()
	tbs := ts.Tabs()
	for i := 0; i < sz; i++ {
		tb := tbs.Child(i).(*Tab)
		tb.Data = i
	}
}

// RenderTabSeps renders the separators between tabs
func (ts *Tabs) RenderTabSeps(sc *Scene) {
	rs, pc, st := ts.RenderLock(sc)
	defer ts.RenderUnlock(rs)

	// just like with standard separator, use top width like CSS
	// (see https://www.w3schools.com/howto/howto_css_dividers.asp)
	pc.StrokeStyle.Width = st.Border.Width.Top
	pc.StrokeStyle.SetColor(&st.Border.Color.Top)
	bw := st.Border.Width.Dots()

	tbs := ts.Tabs()
	sz := len(tbs.Kids)
	for i := 1; i < sz; i++ {
		tb := tbs.Child(i).(Widget)
		ni := tb.AsWidget()

		pos := ni.LayState.Alloc.Pos
		sz := ni.LayState.Alloc.Size.Sub(st.TotalMargin().Size())
		pc.DrawLine(rs, pos.X-bw.Pos().X, pos.Y, pos.X-bw.Pos().X, pos.Y+sz.Y)
	}
	pc.FillStrokeClear(rs)
}

func (ts *Tabs) Render(sc *Scene) {
	if ts.PushBounds(sc) {
		ts.RenderScrolls(sc)
		ts.RenderChildren(sc)
		ts.RenderTabSeps(sc)
		ts.PopBounds(sc)
	}
}

////////////////////////////////////////////////////////////////////////////////////////
// Tab

// Tab is a tab button that contains a larger select button
// and a smaller close button. The Indicator icon is used for
// the close icon.
type Tab struct {
	Button

	// if true, this tab has a delete button (true by default)
	DeleteButton bool
}

func (tb *Tab) OnInit() {
	tb.DeleteButton = true
	tb.HandleButtonEvents()
	tb.TabStyles()
}

func (tb *Tab) TabStyles() {
	tb.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Activatable, abilities.Focusable, abilities.Hoverable)
		s.SetAbilities(tb.ShortcutTooltip() != "", abilities.LongHoverable)

		if !tb.IsReadOnly() {
			s.Cursor = cursors.Pointer
		}
		s.MinWidth.Ch(8)
		s.MaxWidth.Dp(500)
		s.MinHeight.Em(1.6)

		// s.Border.Style.Right = styles.BorderSolid
		// s.Border.Width.Right.SetDp(1)

		s.Border.Radius.Set()
		s.Text.Align = styles.AlignCenter
		s.Margin.Set()
		s.Padding.Set(units.Dp(8))

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
				s.Spacing.Zero()
				s.Overflow = styles.OverflowHidden // no scrollbars!
			})
		case "parts/icon":
			w.Style(func(s *styles.Style) {
				s.Width.Em(1)
				s.Height.Em(1)
				s.Margin.Set()
				s.Padding.Set()
			})
		case "parts/label":
			label := w.(*Label)
			label.Type = LabelBodyMedium
			w.Style(func(s *styles.Style) {
				s.SetAbilities(false, abilities.Selectable, abilities.DoubleClickable)
				s.Cursor = cursors.None
				s.Margin.Set()
				s.Padding.Set()
			})
		case "parts/close-stretch":
			w.Style(func(s *styles.Style) {
				s.Width.Ch(1)
			})
		case "parts/close":
			w.Style(func(s *styles.Style) {
				s.Width.Ex(0.5)
				s.Height.Ex(0.5)
				s.Margin.Set()
				s.Padding.Set()
				s.AlignV = styles.AlignMiddle
				s.Border.Radius = styles.BorderRadiusFull
				s.BackgroundColor.SetSolid(colors.Transparent)
				// if we have some state, we amplify it so we
				// are clearly distinguishable from our parent button
				// TODO: get this working
				// if s.StateLayer > 0 {
				// 	s.StateLayer += 0.12
				// }
			})
		case "parts/sc-stretch":
			w.Style(func(s *styles.Style) {
				s.MinWidth.Ch(2)
			})
		case "parts/shortcut":
			w.Style(func(s *styles.Style) {
				s.Margin.Set()
				s.Padding.Set()
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

func (tb *Tab) ConfigParts(sc *Scene) {
	if tb.DeleteButton {
		tb.ConfigPartsDeleteButton(sc)
		return
	}
	tb.Button.ConfigParts(sc) // regular
}

func (tb *Tab) ConfigPartsDeleteButton(sc *Scene) {
	parts := tb.NewParts(LayoutHoriz)
	config := ki.Config{}
	icIdx, lbIdx := tb.ConfigPartsIconLabel(&config, tb.Icon, tb.Text)
	config.Add(StretchType, "close-stretch")
	clsIdx := len(config)
	config.Add(ButtonType, "close")
	mods, updt := parts.ConfigChildren(config)
	tb.ConfigPartsSetIconLabel(tb.Icon, tb.Text, icIdx, lbIdx)
	if mods {
		cls := parts.Child(clsIdx).(*Button)
		if tb.Indicator.IsNil() {
			tb.Indicator = icons.Close
		}

		cls.SetType(ButtonAction)
		icnm := tb.Indicator
		cls.SetIcon(icnm)
		cls.Update()
		cls.SetProp("no-focus", true)
		cls.OnClick(func(e events.Event) {
			tabIdx := tb.Data.(int)
			ts := tb.Tabs()
			if ts != nil {
				if !Prefs.Params.OnlyCloseActiveTab || tb.StateIs(states.Selected) { // only process delete when already selected if OnlyCloseActiveTab is on
					ts.DeleteTabIndex(tabIdx, true)
				} else {
					ts.SelectTabIndex(tabIdx) // otherwise select
				}
			}
		})
		tb.UpdateEnd(updt)
	}
}

func (tb *Tab) ConfigWidget(sc *Scene) {
	tb.ConfigParts(sc)
}
