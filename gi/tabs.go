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
//
//goki:embedder
type Tabs struct {
	Layout

	// maximum number of characters to include in tab label -- elides labels that are longer than that
	MaxChars int `desc:"maximum number of characters to include in tab label -- elides labels that are longer than that"`

	// signal for tab widget -- see TabViewSignals for the types
	// TabViewSig ki.Signal `copy:"-" json:"-" xml:"-" desc:"signal for tab widget -- see TabViewSignals for the types"`

	// show a new tab button at right of list of tabs
	NewTabButton bool `desc:"show a new tab button at right of list of tabs"`

	// if true, tabs are not user-deleteable
	NoDeleteTabs bool `desc:"if true, tabs are not user-deleteable"`

	// [view: -] mutex protecting updates to tabs -- tabs can be driven programmatically and via user input so need extra protection
	Mu sync.Mutex `copy:"-" json:"-" xml:"-" view:"-" desc:"mutex protecting updates to tabs -- tabs can be driven programmatically and via user input so need extra protection"`
}

func (ts *Tabs) CopyFieldsFrom(frm any) {
	fr := frm.(*Tabs)
	ts.Layout.CopyFieldsFrom(&fr.Layout)
	ts.MaxChars = fr.MaxChars
	ts.NewTabButton = fr.NewTabButton
}

func (ts *Tabs) OnInit() {
	ts.TabViewHandlers()
	ts.TabViewStyles()
}

func (ts *Tabs) TabViewHandlers() {
	ts.LayoutHandlers()
}

func (ts *Tabs) TabViewStyles() {
	ts.AddStyles(func(s *styles.Style) {
		// need border for separators (see RenderTabSeps)
		// TODO: maybe better solution for tab sep styles?
		s.Border.Style.Set(styles.BorderSolid)
		s.Border.Width.Set(units.Dp(1))
		s.Border.Color.Set(colors.Scheme.OutlineVariant)
		s.BackgroundColor.SetSolid(colors.Scheme.Background)
		s.Color = colors.Scheme.OnBackground
		s.MaxWidth.SetDp(-1)
		s.MaxHeight.SetDp(-1)
	})
}

func (ts *Tabs) OnChildAdded(child ki.Ki) {
	w, _ := AsWidget(child)
	switch w.Name() {
	case "tabs":
		w.AddStyles(func(s *styles.Style) {
			s.SetStretchMaxWidth()
			s.Height.SetEm(1.8)
			s.Overflow = styles.OverflowHidden // no scrollbars!
			s.Margin.Set()
			s.Padding.Set()
			// tabs.Spacing.SetDp(4 * Prefs.DensityMul())
			s.BackgroundColor.SetSolid(colors.Scheme.SurfaceContainerLow)

			// s.Border.Style.Set(styles.BorderNone)
			// s.Border.Style.Bottom = styles.BorderSolid
			// s.Border.Width.Bottom.SetDp(1)
			// s.Border.Color.Bottom = colors.Scheme.OutlineVariant
		})
	case "frame":
		frame := w.(*Frame)
		frame.StackTopOnly = true // key for allowing each tab to have its own size
		w.AddStyles(func(s *styles.Style) {
			s.SetMinPrefWidth(units.Em(10))
			s.SetMinPrefHeight(units.Em(6))
			s.SetStretchMax()
		})
	}
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
	widg := fr.Child(fr.StackTop).(Widget)
	return widg, fr.StackTop, true
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
	tab.NoDelete = ts.NoDeleteTabs
	tab.SetText(label)
	tab.OnClick(func(e events.Event) {
		ts.SelectTabIndex(idx)
	})
	fr := ts.Frame()
	if len(fr.Kids) == 1 {
		fr.StackTop = 0
		tab.SetSelected(true)
	} else {
		frame.SetFlag(true, Invisible) // new tab is invisible until selected
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
		slog.Error("gi.TabView: index out of range for number of tabs", "index", idx, "numTabs", sz)
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
		return nil, fmt.Errorf("gi.TabView: Tab named %v not found in %v", label, ts.Path())
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
		return -1, fmt.Errorf("gi.TabView: Tab named %v not found in %v", label, ts.Path())
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

// TabViewSignals are signals that the TabView can send
type TabViewSignals int64

const (
	// TabSelected indicates tab was selected -- data is the tab index
	TabSelected TabViewSignals = iota

	// TabAdded indicates tab was added -- data is the tab index
	TabAdded

	// TabDeleted indicates tab was deleted -- data is the tab name
	TabDeleted

	TabViewSignalsN
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
// and a small close button. The Indicator icon is used for
// the close icon.
type Tab struct {
	Button

	// if true, this tab does not have the delete button avail
	NoDelete bool `desc:"if true, this tab does not have the delete button avail"`
}

func (tb *Tab) OnInit() {
	tb.ButtonHandlers()
	tb.TabButtonStyles()
}

func (tb *Tab) TabButtonStyles() {
	tb.AddStyles(func(s *styles.Style) {
		s.Cursor = cursors.Pointer
		s.MinWidth.SetCh(8)
		s.MaxWidth.SetDp(500)
		s.MinHeight.SetEm(1.6)

		// s.Border.Style.Right = styles.BorderSolid
		// s.Border.Width.Right.SetDp(1)

		s.BackgroundColor.SetSolid(colors.Scheme.SurfaceContainerLow)
		s.Color = colors.Scheme.OnSurface

		s.Border.Radius.Set()
		s.Text.Align = styles.AlignCenter
		s.Margin.Set()
		s.Padding.Set(units.Dp(8 * Prefs.DensityMul()))

		// s.Border.Style.Set(styles.BorderNone)
		// if tb.StateIs(states.Selected) {
		// 	s.Border.Style.Bottom = styles.BorderSolid
		// 	s.Border.Width.Bottom.SetDp(2)
		// 	s.Border.Color.Bottom = colors.Scheme.Primary
		// }
	})
}

func (tb *Tab) OnChildAdded(child ki.Ki) {
	w, _ := AsWidget(child)
	switch w.Name() {
	case "Parts":
		w.AddStyles(func(s *styles.Style) {
			s.Overflow = styles.OverflowHidden // no scrollbars!
		})
	case "icon":
		w.AddStyles(func(s *styles.Style) {
			s.Width.SetEm(1)
			s.Height.SetEm(1)
			s.Margin.Set()
			s.Padding.Set()
		})
	case "label":
		label := w.(*Label)
		label.Type = LabelTitleSmall
		w.AddStyles(func(s *styles.Style) {
			s.SetAbilities(false, abilities.Selectable, abilities.DoubleClickable)
			s.Cursor = cursors.None
			s.Margin.Set()
			s.Padding.Set()
		})
	case "close-stretch":
		w.AddStyles(func(s *styles.Style) {
			s.Width.SetCh(1)
		})
	case "close":
		w.AddStyles(func(s *styles.Style) {
			s.Width.SetEx(0.5)
			s.Height.SetEx(0.5)
			s.Margin.Set()
			s.Padding.Set()
			s.AlignV = styles.AlignMiddle
			s.Border.Radius = styles.BorderRadiusFull
			s.BackgroundColor.SetSolid(colors.Transparent)
		})
	case "sc-stretch":
		w.AddStyles(func(s *styles.Style) {
			s.MinWidth.SetCh(2)
		})
	case "shortcut":
		w.AddStyles(func(s *styles.Style) {
			s.Margin.Set()
			s.Padding.Set()
		})
	}
}

func (tb *Tab) Tabs() *Tabs {
	ts := tb.ParentByType(TabsType, ki.Embeds)
	if ts == nil {
		return nil
	}
	return AsTabs(ts)
}

func (tb *Tab) ConfigParts(sc *Scene) {
	if !tb.NoDelete {
		tb.ConfigPartsDeleteButton(sc)
		return
	}
	tb.Button.ConfigParts(sc) // regular
}

func (tb *Tab) ConfigPartsDeleteButton(sc *Scene) {
	config := ki.Config{}
	icIdx, lbIdx := tb.ConfigPartsIconLabel(&config, tb.Icon, tb.Text)
	config.Add(StretchType, "close-stretch")
	clsIdx := len(config)
	config.Add(ButtonType, "close")
	mods, updt := tb.Parts.ConfigChildren(config)
	tb.ConfigPartsSetIconLabel(tb.Icon, tb.Text, icIdx, lbIdx)
	if mods {
		cls := tb.Parts.Child(clsIdx).(*Button)
		if tb.Indicator.IsNil() {
			tb.Indicator = icons.Close
		}

		icnm := tb.Indicator
		cls.SetIcon(icnm)
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
