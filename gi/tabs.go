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
	"goki.dev/glop/elide"
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

	// Maximum number of characters to include in tab label.
	// Elides labels that are longer than that
	MaxChars int

	// show a new tab button at right of list of tabs
	NewTabButton bool

	// if true, tabs are user-deleteable (true by default)
	DeleteTabButtons bool

	// mutex protecting updates to tabs.
	// Tabs can be driven programmatically and via user input so need extra protection
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
		s.Grow.Set(1, 1)
	})
	ts.OnWidgetAdded(func(w Widget) {
		switch w.PathFrom(ts) {
		case "tabs":
			w.Style(func(s *styles.Style) {
				s.Grow.Set(1, 0)
				s.Wrap = true
				s.Overflow.X = styles.OverflowHidden // no scrollbars!
				s.Margin.Zero()
				s.Padding.Zero()
				s.Gap.Zero()
				s.BackgroundColor.SetSolid(colors.Scheme.SurfaceContainer)

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
	defer ts.UpdateEndLayout(updt)

	fr := ts.Frame()
	fr.SetChildAdded()
	nm := ""
	if len(name) > 0 {
		nm = name[0]
	} else {
		nm = strcase.ToKebab(label)
	}
	frame := fr.InsertNewChild(FrameType, idx, nm).(*Frame)
	frame.Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	ts.InsertTabOnlyAt(frame, label, idx, nm)
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
	tab.MaxChars = ts.MaxChars
	tab.SetText(label)
	tab.OnClick(func(e events.Event) {
		ts.SelectTabIndex(idx)
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
// If a name is also passed, the internal name (ID) of the tab will be set
// to that; otherwise, it will default to the kebab-case version of the label.
func (ts *Tabs) InsertTab(frame *Frame, label string, idx int, name ...string) {
	ts.Mu.Lock()
	updt := ts.UpdateStart()
	defer ts.UpdateEndLayout(updt)

	fr := ts.Frame()
	fr.SetChildAdded()
	fr.InsertChild(frame, idx)
	ts.InsertTabOnlyAt(frame, label, idx, name...)
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
	defer ts.UpdateEndLayout(updt)
	ts.UnselectOtherTabs(idx)
	tab.SetSelected(true)
	fr.StackTop = idx
	ts.Mu.Unlock()
	return frame, true
}

// TabByLabel returns tab with given label (nil if not found -- see TabByLabelTry)
func (ts *Tabs) TabByLabel(label string) *Frame {
	t, _ := ts.TabByLabelTry(label)
	return t
}

// TabByLabelTry returns tab with given label, and an error if not found.
func (ts *Tabs) TabByLabelTry(label string) (*Frame, error) {
	idx, err := ts.TabIndexByLabel(label)
	if err != nil {
		return nil, err
	}
	ts.Mu.Lock()
	defer ts.Mu.Unlock()
	fr := ts.Frame()
	frame := fr.Child(idx).(*Frame)
	return frame, nil
}

// TabIndexByLabel returns tab index for given tab label, and an error if not found.
func (ts *Tabs) TabIndexByLabel(label string) (int, error) {
	ts.Mu.Lock()
	defer ts.Mu.Unlock()

	tb := ts.Tabs()
	idx := -1
	n := len(*tb.Children())
	for i := 0; i < n; i++ {
		ti := AsButton(tb.Child(i))
		if ti.Text == label {
			idx = i
			break
		}
	}
	if idx < 0 {
		return -1, fmt.Errorf("gi.Tabs: Tab with label %v not found in %v", label, ts.Path())
	}
	return idx, nil
}

// TabLabel returns tab label at given index
func (ts *Tabs) TabLabel(idx int) string {
	ts.Mu.Lock()
	defer ts.Mu.Unlock()

	tb := ts.Tabs()
	tbut, err := tb.ChildTry(idx)
	if err != nil {
		return ""
	}
	return AsButton(tbut).Text
}

// SelectTabByLabel selects tab by label, returning it.
func (ts *Tabs) SelectTabByLabel(label string) *Frame {
	idx, err := ts.TabIndexByLabel(label)
	if err == nil {
		ts.SelectTabIndex(idx)
		fr := ts.Frame()
		return fr.Child(idx).(*Frame)
	}
	return nil
}

// SelectTabByLabelTry selects tab by label, returning it.  Returns error if not found.
func (ts *Tabs) SelectTabByLabelTry(label string) (*Frame, error) {
	idx, err := ts.TabIndexByLabel(label)
	if err == nil {
		ts.SelectTabIndex(idx)
		fr := ts.Frame()
		return fr.Child(idx).(*Frame), nil
	}
	return nil, err
}

// RecycleTab returns a tab with given label, first by looking for an existing one,
// and if not found, making a new one. If sel, then select it. It returns the
// frame for the tab. If a label is also passed, the internal label (ID) of any new tab
// will be set to that; otherwise, it will default to the kebab-case version of the label.
func (ts *Tabs) RecycleTab(label string, sel bool, name ...string) *Frame {
	frame, err := ts.TabByLabelTry(label)
	if err == nil {
		if sel {
			ts.SelectTabByLabel(label)
		}
		return frame
	}
	frame = ts.NewTab(label, name...)
	if sel {
		ts.SelectTabByLabel(label)
	}
	return frame
}

// RecycleTabWidget returns a tab with given widget type in the tab frame,
// first by looking for an existing one, with given label, and if not found,
// making and configuring a new one.
// If sel, then select it. It returns the Widget item for the tab.
// If a label is also passed, the internal label (ID) of any new tab
// will be set to that; otherwise, it will default to the kebab-case version of the label.
func (ts *Tabs) RecycleTabWidget(label string, sel bool, typ *gti.Type, name ...string) Widget {
	fr := ts.RecycleTab(label, sel, name...)
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
	ts.Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	NewFrame(ts, "tabs")
	NewFrame(ts, "frame")
	ts.ConfigNewTabButton()
}

// Tabs returns the layout containing the tabs (the first element within us).
// It configures the Tabs if necessary.
func (ts *Tabs) Tabs() *Frame {
	ts.ConfigWidget()
	return ts.Child(0).(*Frame)
}

// Frame returns the stacked frame layout (the second element within us).
// It configures the Tabs if necessary.
func (ts *Tabs) Frame() *Frame {
	ts.ConfigWidget()
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
func (ts *Tabs) RenderTabSeps() {
	rs, pc, st := ts.RenderLock()
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

		pos := ni.Geom.Pos.Total
		sz := ni.Geom.Size.Actual.Total.Sub(st.TotalMargin().Size())
		pc.DrawLine(rs, pos.X-bw.Pos().X, pos.Y, pos.X-bw.Pos().X, pos.Y+sz.Y)
	}
	pc.FillStrokeClear(rs)
}

func (ts *Tabs) Render() {
	if ts.PushBounds() {
		ts.RenderScrolls()
		ts.RenderChildren()
		ts.RenderTabSeps()
		ts.PopBounds()
	}
}

////////////////////////////////////////////////////////////////////////////////////////
// Tab

// Tab is a tab button that contains a larger select button
// and a smaller close button. The Indicator icon is used for
// the close icon.
type Tab struct {
	Button

	// Maximum number of characters to include in tab label.
	// Elides labels that are longer than that
	MaxChars int

	// if true, this tab has a delete button (true by default)
	DeleteButton bool
}

func (tb *Tab) OnInit() {
	tb.MaxChars = 16
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
		s.Max.X.Ch(float32(tb.MaxChars))
		// s.Min.Y.Ch(6)

		// s.Border.Style.Right = styles.BorderSolid
		// s.Border.Width.Right.SetDp(1)

		s.Border.Radius.Zero()
		s.Text.Align = styles.Center
		s.Margin.Zero()
		s.Padding.Set(units.Dp(10))

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
				s.Margin.Zero()
				s.Padding.Zero()
			})
		case "parts/label":
			label := w.(*Label)
			label.Type = LabelBodyMedium
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
				// s.Margin.Zero()
				s.Padding.Set(units.Dp(0))
				s.Padding.Left.Dp(16)
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

func (tb *Tab) ConfigParts() {
	if tb.MaxChars > 0 {
		tb.Text = elide.Middle(tb.Text, tb.MaxChars)
	}
	if tb.DeleteButton {
		tb.ConfigPartsDeleteButton()
		return
	}
	tb.Button.ConfigParts() // regular
}

func (tb *Tab) ConfigPartsDeleteButton() {
	parts := tb.NewParts()
	config := ki.Config{}
	icIdx, lbIdx := tb.ConfigPartsIconLabel(&config, tb.Icon, tb.Text)
	// config.Add(StretchType, "close-stretch")
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
			ts := tb.Tabs()
			if ts == nil {
				return
			}
			idx, _ := ts.TabIndexByLabel(tb.Text)
			if !Prefs.Params.OnlyCloseActiveTab || tb.StateIs(states.Selected) { // only process delete when already selected if OnlyCloseActiveTab is on
				ts.DeleteTabIndex(idx, true)
			} else {
				ts.SelectTabIndex(idx) // otherwise select
			}
		})
		tb.UpdateEnd(updt)
	}
}

func (tb *Tab) ConfigWidget() {
	tb.ConfigParts()
}
