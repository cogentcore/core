// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"bytes"
	"fmt"
	"image"
	"log"
	"log/slog"

	"goki.dev/colors"
	"goki.dev/cursors"
	"goki.dev/enums"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/keyfun"
	"goki.dev/girl/abilities"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/goosi/mimedata"
	"goki.dev/gti"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
	"goki.dev/pi/v2/filecat"
)

// TreeViewer is an interface for TreeView types
// providing access to the base TreeView and
// overridable method hooks for actions taken on the TreeView,
// including OnOpen, OnClose, etc.
type TreeViewer interface {
	// AsTreeView returns the base *TreeView for this node
	AsTreeView() *TreeView

	// OnOpen is called when a node is opened.
	// The base version does nothing.
	OnOpen()

	// OnClose is called when a node is closed
	// The base version does nothing.
	OnClose()

	// UpdateBranchIcons is called during DoLayout to update branch icons
	// when everything should be configured, prior to rendering.
	UpdateBranchIcons()
}

// AsTreeView returns the given value as a value of type TreeView if the type
// of the given value embeds TreeView, or nil otherwise
func AsTreeView(k ki.Ki) *TreeView {
	if k == nil || k.This() == nil {
		return nil
	}
	if t, ok := k.(TreeViewer); ok {
		return t.AsTreeView()
	}
	return nil
}

// note: see treesync.go for all the SyncNode mode specific
// functions.

// TreeView provides a graphical representation of a tree tructure
// providing full navigation and manipulation abilities.
//
// If the SyncNode field is non-nil, typically via
// SyncRootNode method, then the TreeView mirrors another
// Ki tree structure, and tree editing functions apply to
// the source tree first, and then to the TreeView by sync.
//
// Otherwise, data can be directly encoded in a TreeView
// derived type, to represent any kind of tree structure
// and associated data.
//
// Standard events.Event are sent to any listeners, including
// Select, Change, and DoubleClick.  The selected nodes
// are in the root SelectedNodes list.
type TreeView struct {
	gi.WidgetBase

	// If non-nil, the Ki Node that this widget is viewing in the tree (the source)
	SyncNode ki.Ki `set:"-" copy:"-" json:"-" xml:"-"`

	// The text to display for the tree view item label, which automatically
	// defaults to the [ki.Node.Name] of the tree view node. It has no effect
	// if [TreeView.SyncNode] is non-nil.
	Text string

	// optional icon, displayed to the the left of the text label
	Icon icons.Icon

	// amount to indent children relative to this node
	Indent units.Value `copy:"-" json:"-" xml:"-"`

	// depth for nodes be initialized as open (default 4).
	// Nodes beyond this depth will be initialized as closed.
	OpenDepth int `copy:"-" json:"-" xml:"-"`

	/////////////////////////////////////////
	// All fields below are computed

	// linear index of this node within the entire tree.
	// updated on full rebuilds and may sometimes be off,
	// but close enough for expected uses
	ViewIdx int `copy:"-" json:"-" xml:"-" edit:"-"`

	// size of just this node widget.
	// our alloc includes all of our children, but we only draw us.
	WidgetSize mat32.Vec2 `copy:"-" json:"-" xml:"-" edit:"-"`

	// cached root of the view
	RootView *TreeView `copy:"-" json:"-" xml:"-" edit:"-"`

	// SelectedNodes holds the currently-selected nodes, on the
	// RootView node only.
	SelectedNodes []*TreeView `copy:"-" json:"-" xml:"-" edit:"-"`

	// actStateLayer is the actual state layer of the tree view, which
	// should be used when rendering it and its parts (but not its children).
	// the reason that it exists is so that the children of the tree view
	// (other tree views) do not inherit its stateful background color, as
	// that does not look good.
	actStateLayer float32 `set:"-"`
}

func (tv *TreeView) FlagType() enums.BitFlagSetter {
	return (*TreeViewFlags)(&tv.Flags)
}

func (tv *TreeView) CopyFieldsFrom(frm any) {
	fr, ok := frm.(*TreeView)
	if !ok {
		log.Printf("GoGi node of type: %v needs a CopyFieldsFrom method defined -- currently falling back on earlier one\n", tv.KiType().Name)
		return
	}
	tv.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
	// note: can't actually copy anything here
}

// AsTreeView satisfies the [TreeViewEmbedder] interface
func (tv *TreeView) AsTreeView() *TreeView {
	return tv
}

func (tv *TreeView) BaseType() *gti.Type {
	return tv.KiType()
}

// RootSetViewIdx sets the RootView and ViewIdx for all nodes.
// This must be called from the root node after
// construction or any modification to the tree.
// Returns the total number of leaves in the tree.
func (tv *TreeView) RootSetViewIdx() int {
	idx := 0
	tv.WidgetWalkPre(func(wi gi.Widget, wb *gi.WidgetBase) bool {
		tvki := AsTreeView(wi)
		if tvki != nil {
			tvki.ViewIdx = idx
			tvki.RootView = tv
			idx++
		}
		return ki.Continue
	})
	return idx
}

func (tv *TreeView) OnInit() {
	tv.HandleTreeViewEvents()
	tv.TreeViewStyles()
}

func (tv *TreeView) OnAdd() {
	tv.Text = tv.Nm
}

func (tv *TreeView) TreeViewStyles() {
	tv.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Activatable, abilities.Focusable, abilities.Selectable, abilities.Hoverable)
		tv.Indent.Em(1)
		tv.OpenDepth = 4
		s.Border.Style.Set(styles.BorderNone)
		// s.Border.Width.Left.SetDp(1)
		// s.Border.Color.Left = colors.Scheme.OutlineVariant
		s.Margin.Zero()
		s.Padding.Set(units.Dp(4))
		s.Text.Align = styles.Start

		// need to copy over to actual and then clear styles one
		tv.actStateLayer = s.StateLayer
		s.StateLayer = 0
		if s.Is(states.Selected) {
			// render handles manually, similar to with actStateLayer
			s.BackgroundColor.SetSolid(colors.Transparent)
		}
	})
	tv.OnWidgetAdded(func(w gi.Widget) {
		// fmt.Println(w.PathFrom(tv))
		switch w.PathFrom(tv) {
		case "parts":
			w.Style(func(s *styles.Style) {
				s.Cursor = cursors.Pointer
				s.SetAbilities(true, abilities.Activatable, abilities.Focusable, abilities.Selectable, abilities.Hoverable, abilities.DoubleClickable)
				s.Gap.X.Ch(0.1)
				s.Padding.Zero()

				// we manually inherit our state layer from the treeview state
				// layer so that the parts get it but not the other tree views
				s.StateLayer = tv.actStateLayer
			})
			// we let the parts handle our state
			// so that we only get it when we are doing
			// something with this treeview specifically,
			// not with any of our children (see HandleTreeViewMouse)
			w.On(events.MouseEnter, func(e events.Event) {
				if tv.This() == nil || tv.Is(ki.Deleted) {
					return
				}
				tv.SetState(true, states.Hovered)
				tv.ApplyStyle(tv.Sc)
				tv.SetNeedsRender()
				e.SetHandled()
			})
			w.On(events.MouseLeave, func(e events.Event) {
				if tv.This() == nil || tv.Is(ki.Deleted) {
					return
				}
				tv.SetState(false, states.Hovered)
				tv.ApplyStyle(tv.Sc)
				tv.SetNeedsRender()
				e.SetHandled()
			})
			w.On(events.MouseDown, func(e events.Event) {
				if tv.This() == nil || tv.Is(ki.Deleted) {
					return
				}
				tv.SetState(true, states.Active)
				tv.ApplyStyle(tv.Sc)
				tv.SetNeedsRender()
				e.SetHandled()
			})
			w.On(events.MouseUp, func(e events.Event) {
				if tv.This() == nil || tv.Is(ki.Deleted) {
					return
				}
				tv.SetState(false, states.Active)
				tv.ApplyStyle(tv.Sc)
				tv.SetNeedsRender()
				e.SetHandled()
			})
			w.OnClick(func(e events.Event) {
				if tv.This() == nil || tv.Is(ki.Deleted) {
					return
				}
				tv.SelectAction(e.SelectMode())
				e.SetHandled()
			})
			w.AsWidget().OnDoubleClick(func(e events.Event) {
				if tv.This() == nil || tv.Is(ki.Deleted) {
					return
				}
				if tv.HasChildren() {
					tv.ToggleClose()
				}
			})
			// the context menu events will get sent to the parts, so it
			// needs to intercept them and send them up
			w.On(events.ContextMenu, tv.ShowContextMenu)
		case "parts/icon":
			w.Style(func(s *styles.Style) {
				s.Min.X.Em(1)
				s.Min.Y.Em(1)
				s.Margin.Zero()
				s.Padding.Zero()
			})
		case "parts/branch":
			sw := w.(*gi.Switch)
			sw.Type = gi.SwitchCheckbox
			sw.IconOn = icons.KeyboardArrowDown   // icons.FolderOpen
			sw.IconOff = icons.KeyboardArrowRight // icons.Folder
			sw.IconDisab = icons.Blank
			sw.Style(func(s *styles.Style) {
				// parent will handle our cursor
				s.Cursor = cursors.None
				s.Color = colors.Scheme.Primary.Base
				s.Margin.Zero()
				s.Padding.Zero()
				s.Min.X.Em(0.8)
				s.Min.Y.Em(0.8)
				s.Align.Self = styles.Center
				// we don't need to visibly tell the user that we are disabled;
				// the lack of an icon accomplishes that; instead, we just inherit
				// our state layer from the actual tree view state layer
				if s.Is(states.Disabled) {
					s.StateLayer = tv.actStateLayer
				}
				// If we are responsible for a positive (non-disabled) state layer
				// (instead of our parent), then we amplify it so that it is clear
				// that we ourself are receiving a state layer amplifying event.
				// Otherwise, we set our state color to that of our parent (OnSurface)
				// so that it does not appear as if we are getting interaction ourself;
				// instead, we are a part of our parent and render a background color no
				// different than them.
				if !s.Is(states.Disabled) && (s.Is(states.Hovered) || s.Is(states.Focused) || s.Is(states.Active)) {
					s.StateLayer *= 3
				} else {
					s.StateColor = colors.Scheme.OnSurface
				}
			})
			sw.OnClick(func(e events.Event) {
				if tv.This() == nil || tv.Is(ki.Deleted) {
					return
				}
				if sw.StateIs(states.Checked) {
					if !tv.IsClosed() {
						tv.Close()
					}
				} else {
					if tv.IsClosed() {
						tv.Open()
					}
				}
			})
		case "parts/branch.parts/stack/icon0":
			w.Style(func(s *styles.Style) {
				s.Min.X.Em(1.0)
				s.Min.Y.Em(1.0)
			})
		case "parts/branch.parts/stack/icon1":
			w.Style(func(s *styles.Style) {
				s.Min.X.Em(1.0)
				s.Min.Y.Em(1.0)
			})
		case "parts/branch.parts/stack/icon2":
			w.Style(func(s *styles.Style) {
				s.Min.X.Em(1.0)
				s.Min.Y.Em(1.0)
			})
		case "parts/space":
			w.Style(func(s *styles.Style) {
				s.Min.X.Em(0.5)
			})
		case "parts/label":
			w.Style(func(s *styles.Style) {
				// todo: (Kai) need to change these for clickable links in glide
				s.SetNonSelectable()
				s.SetTextWrap(false)
				s.Margin.Zero()
				s.Padding.Zero()
				s.Min.X.Ch(16)
				s.Min.Y.Em(1.2)
			})
		case "parts/menu":
			menu := w.(*gi.Button)
			menu.Indicator = icons.None
		}
	})
}

// TreeViewFlags extend WidgetFlags to hold TreeView state
type TreeViewFlags gi.WidgetFlags //enums:bitflag -trim-prefix TreeViewFlag

const (
	// TreeViewFlagClosed means node is toggled closed
	// (children not visible)  Otherwise Open.
	TreeViewFlagClosed TreeViewFlags = TreeViewFlags(gi.WidgetFlagsN) + iota

	// This flag on the Root node determines whether keyboard movements
	// update selection or not.
	TreeViewFlagSelectMode
)

// IsClosed returns whether this node itself closed?
func (tv *TreeView) IsClosed() bool {
	return tv.Is(TreeViewFlagClosed)
}

// SetClosed sets the closed flag for this node.
// Call Close() method to close a node and update view.
func (tv *TreeView) SetClosed(closed bool) {
	tv.SetFlag(closed, TreeViewFlagClosed)
}

// RootIsReadOnly returns the ReadOnly status of the root node,
// which is what controls the functional inactivity of the tree
// if individual nodes are ReadOnly that only affects display typically.
func (tv *TreeView) RootIsReadOnly() bool {
	if tv.RootView == nil {
		return true
	}
	return tv.RootView.IsDisabled()
}

////////////////////////////////////////////////////
// Widget interface

// qt calls the open / close thing a "branch"
// http://doc.qt.io/qt-5/stylesheet-examples.html#customizing-qtreeview

// BranchPart returns the branch in parts, if it exists
func (tv *TreeView) BranchPart() (*gi.Switch, bool) {
	if tv.Parts == nil {
		return nil, false
	}
	if icc := tv.Parts.ChildByName("branch", 0); icc != nil {
		return icc.(*gi.Switch), true
	}
	return nil, false
}

// IconPart returns the icon in parts, if it exists
func (tv *TreeView) IconPart() (*gi.Icon, bool) {
	if tv.Parts == nil {
		return nil, false
	}
	if icc := tv.Parts.ChildByName("icon", 1); icc != nil {
		return icc.(*gi.Icon), true
	}
	return nil, false
}

// LabelPart returns the label in parts, if it exists
func (tv *TreeView) LabelPart() (*gi.Label, bool) {
	if tv.Parts == nil {
		return nil, false
	}
	if lbl := tv.Parts.ChildByName("label", 1); lbl != nil {
		return lbl.(*gi.Label), true
	}
	return nil, false
}

func (tv *TreeView) ConfigParts(sc *gi.Scene) {
	parts := tv.NewParts()
	config := ki.Config{}
	config.Add(gi.SwitchType, "branch")
	if tv.Icon.IsValid() {
		config.Add(gi.IconType, "icon")
	}
	config.Add(gi.LabelType, "label")
	mods, updt := parts.ConfigChildren(config)
	if tv.HasChildren() {
		if wb, ok := tv.BranchPart(); ok {
			tv.SetBranchState()
			wb.Config(sc)
		}
	}
	if tv.Icon.IsValid() {
		if ic, ok := tv.IconPart(); ok {
			ic.SetIcon(tv.Icon)
		}
	}
	if lbl, ok := tv.LabelPart(); ok {
		lbl.SetText(tv.Label())
	}
	if mods {
		parts.UpdateEnd(updt)
		tv.UpdateEndLayout(updt)
	}
}

func (tv *TreeView) ConfigWidget(sc *gi.Scene) {
	tv.ConfigParts(sc)
}

func (tv *TreeView) StyleTreeView(sc *gi.Scene) {
	if !tv.HasChildren() {
		tv.SetClosed(true)
	}
	tv.Indent.ToDots(&tv.Styles.UnContext)
	// tv.Parts.Styles.InheritFields(&tv.Styles)
	tv.ApplyStyleWidget(sc)
	// tv.Styles.StateLayer = 0 // turn off!
	// note: this is essential for reasonable styling behavior
}

func (tv *TreeView) ApplyStyle(sc *gi.Scene) {
	tv.StyMu.Lock() // todo: needed??  maybe not.
	defer tv.StyMu.Unlock()

	tv.StyleTreeView(sc)
}

func (tv *TreeView) UpdateBranchIcons() {
}

func (tv *TreeView) SetBranchState() {
	br, ok := tv.BranchPart()
	if !ok {
		return
	}
	switch {
	case !tv.HasChildren():
		br.SetState(true, states.Disabled)
	case tv.IsClosed():
		br.SetState(false, states.Disabled)
		br.SetState(false, states.Checked)
		br.SetNeedsRender()
	default:
		br.SetState(false, states.Disabled)
		br.SetState(true, states.Checked)
		br.SetNeedsRender()
	}
}

// TreeView is tricky for alloc because it is both a layout
// of its children but has to maintain its own bbox for its own widget.

func (tv *TreeView) SizeUp(sc *gi.Scene) {
	tv.WidgetBase.SizeUp(sc)
	tv.WidgetSize = tv.Geom.Size.Actual.Total
	h := tv.WidgetSize.Y
	w := tv.WidgetSize.X

	if !tv.IsClosed() {
		// we layout children under us
		tv.WidgetKidsIter(func(i int, kwi gi.Widget, kwb *gi.WidgetBase) bool {
			kwi.SizeUp(sc)
			h += kwb.Geom.Size.Actual.Total.Y
			w = max(w, tv.Indent.Dots+kwb.Geom.Size.Actual.Total.X)
			// fmt.Println(kwb, w, h)
			return ki.Continue
		})
	}
	sz := &tv.Geom.Size
	sz.Actual.Content = mat32.Vec2{w, h}
	sz.SetTotalFromContent(&sz.Actual)
	sz.Alloc = sz.Actual // need allocation to match!
	tv.WidgetSize.X = w  // stretch
}

func (tv *TreeView) SizeDown(sc *gi.Scene, iter int) bool {
	// note: key to not grab the whole allocation, as widget default does
	redo := tv.SizeDownParts(sc, iter) // give our content to parts
	re := tv.SizeDownChildren(sc, iter)
	return redo || re
}

func (tv *TreeView) Position(sc *gi.Scene) {
	rn := tv.RootView
	if rn == nil {
		slog.Error("giv.TreeView: RootView is nil", "in node:", tv)
		return
	}
	tv.SetBranchState()
	tv.This().(TreeViewer).UpdateBranchIcons()

	tv.Geom.Size.Actual.Total.X = rn.Geom.Size.Actual.Total.X - (tv.Geom.Pos.Total.X - rn.Geom.Pos.Total.X)
	tv.WidgetSize.X = tv.Geom.Size.Actual.Total.X

	tv.WidgetBase.Position(sc)

	if !tv.IsClosed() {
		h := tv.WidgetSize.Y
		tv.WidgetKidsIter(func(i int, kwi gi.Widget, kwb *gi.WidgetBase) bool {
			kwb.Geom.RelPos.Y = h
			kwb.Geom.RelPos.X = tv.Indent.Dots
			h += kwb.Geom.Size.Actual.Total.Y
			kwi.Position(sc)
			return ki.Continue
		})
	}
}

func (tv *TreeView) ScenePos(sc *gi.Scene) {
	sz := &tv.Geom.Size
	if sz.Actual.Total == tv.WidgetSize {
		sz.SetTotalFromContent(&sz.Actual) // restore after scrolling
	}
	tv.WidgetBase.ScenePos(sc)
	tv.ScenePosChildren(sc)
	tv.Geom.Size.Actual.Total = tv.WidgetSize // key: we revert to just ourselves
}

func (tv *TreeView) RenderNode(sc *gi.Scene) {
	rs, pc, st := tv.RenderLock(sc)
	// must use workaround act values
	st.StateLayer = tv.actStateLayer
	if st.Is(states.Selected) {
		st.BackgroundColor.SetSolid(colors.Scheme.Select.Container)
	}
	pbc, psl := tv.ParentBackgroundColor()
	pc.DrawStdBox(rs, st, tv.Geom.Pos.Total, tv.Geom.Size.Actual.Total, &pbc, psl)
	// after we are done rendering, we clear the values so they aren't inherited
	st.StateLayer = 0
	st.BackgroundColor.SetSolid(colors.Transparent)
	tv.RenderUnlock(rs)
	if tv.Parts.HasAnyScroll() {
		fmt.Println(tv, "tv scroll")
	}
}

func (tv *TreeView) Render(sc *gi.Scene) {
	if tv.PushBounds(sc) {
		tv.RenderNode(sc)
		if tv.Parts != nil {
			// we must copy from actual values in parent
			tv.Parts.Styles.StateLayer = tv.actStateLayer
			if tv.StateIs(states.Selected) {
				tv.Parts.Styles.BackgroundColor.SetSolid(colors.Scheme.Select.Container)
			}
			tv.RenderParts(sc)
		}
		tv.PopBounds(sc)
	}
	// we always have to render our kids b/c
	// we could be out of scope but they could be in!
	if !tv.IsClosed() {
		tv.RenderChildren(sc)
	}
}

//////////////////////////////////////////////////////////////////////////////
//    Selection

// SelectMode returns true if keyboard movements
// should automatically select nodes
func (tv *TreeView) SelectMode() bool {
	return tv.RootView.Is(TreeViewFlagSelectMode)
}

// SetSelectMode updates the select mode
func (tv *TreeView) SetSelectMode(selMode bool) {
	tv.RootView.SetFlag(selMode, TreeViewFlagSelectMode)
}

// SelectModeToggle toggles the SelectMode
func (tv *TreeView) SelectModeToggle() {
	tv.SetSelectMode(!tv.SelectMode())
}

// SelectedViews returns a slice of the currently-selected
// TreeViews within the entire tree, using a list maintained
// by the root node
func (tv *TreeView) SelectedViews() []*TreeView {
	if tv.RootView == nil {
		return nil
	}
	return tv.RootView.SelectedNodes
}

// SetSelectedViews updates the selected views to given list
func (tv *TreeView) SetSelectedViews(sl []*TreeView) {
	if tv.RootView != nil {
		tv.RootView.SelectedNodes = sl
	}
}

// HasSelection returns true if there are currently selected items
func (tv *TreeView) HasSelection() bool {
	return len(tv.SelectedViews()) > 0
}

// Select selects this node (if not already selected).
// Must use this method to update global selection list
func (tv *TreeView) Select() {
	if !tv.StateIs(states.Selected) {
		tv.SetSelected(true)
		tv.ApplyStyle(tv.Sc)
		sl := tv.SelectedViews()
		sl = append(sl, tv)
		tv.SetSelectedViews(sl)
		tv.SetNeedsRender()
	}
}

// Unselect unselects this node (if selected).
// Must use this method to update global selection list.
func (tv *TreeView) Unselect() {
	if tv.StateIs(states.Selected) {
		tv.SetSelected(false)
		tv.ApplyStyle(tv.Sc)
		sl := tv.SelectedViews()
		sz := len(sl)
		for i := 0; i < sz; i++ {
			if sl[i] == tv {
				sl = append(sl[:i], sl[i+1:]...)
				break
			}
		}
		tv.SetSelectedViews(sl)
		tv.SetNeedsRender()
	}
}

// UnselectAll unselects all selected items in the view
func (tv *TreeView) UnselectAll() {
	if tv.Sc == nil {
		return
	}
	updt := tv.UpdateStart()
	sl := tv.SelectedViews()
	tv.SetSelectedViews(nil) // clear in advance
	for _, v := range sl {
		v.SetSelected(false)
		v.ApplyStyle(tv.Sc)
		v.SetNeedsRender()
	}
	tv.UpdateEndRender(updt)
}

// SelectAll all items in view
func (tv *TreeView) SelectAll() {
	if tv.Sc == nil {
		return
	}
	updt := tv.UpdateStart()
	tv.UnselectAll()
	nn := tv.RootView
	nn.Select()
	for nn != nil {
		nn = nn.MoveDown(events.SelectQuiet)
	}
	tv.UpdateEndRender(updt)
}

// SelectUpdate updates selection to include this node,
// using selectmode from mouse event (ExtendContinuous, ExtendOne).
// Returns true if this node selected
func (tv *TreeView) SelectUpdate(mode events.SelectModes) bool {
	if mode == events.NoSelect {
		return false
	}
	updt := tv.UpdateStart()
	sel := false
	switch mode {
	case events.SelectOne:
		if tv.StateIs(states.Selected) {
			sl := tv.SelectedViews()
			if len(sl) > 1 {
				tv.UnselectAll()
				tv.Select()
				tv.GrabFocus()
				sel = true
			}
		} else {
			tv.UnselectAll()
			tv.Select()
			tv.GrabFocus()
			sel = true
		}
	case events.ExtendContinuous:
		sl := tv.SelectedViews()
		if len(sl) == 0 {
			tv.Select()
			tv.GrabFocus()
			sel = true
		} else {
			minIdx := -1
			maxIdx := 0
			for _, v := range sl {
				if minIdx < 0 {
					minIdx = v.ViewIdx
				} else {
					minIdx = min(minIdx, v.ViewIdx)
				}
				maxIdx = max(maxIdx, v.ViewIdx)
			}
			cidx := tv.ViewIdx
			nn := tv
			tv.Select()
			if tv.ViewIdx < minIdx {
				for cidx < minIdx {
					nn = nn.MoveDown(events.SelectQuiet) // just select
					cidx = nn.ViewIdx
				}
			} else if tv.ViewIdx > maxIdx {
				for cidx > maxIdx {
					nn = nn.MoveUp(events.SelectQuiet) // just select
					cidx = nn.ViewIdx
				}
			}
		}
	case events.ExtendOne:
		if tv.StateIs(states.Selected) {
			tv.UnselectAction()
		} else {
			tv.Select()
			tv.GrabFocus()
			sel = true
		}
	case events.SelectQuiet:
		tv.Select()
		// not sel -- no signal..
	case events.UnselectQuiet:
		tv.Unselect()
		// not sel -- no signal..
	}
	tv.UpdateEndRender(updt)
	return sel
}

// SendSelectEvent sends the events.Select event on the
// RootView node, using context event if avail (else nil).
func (tv *TreeView) SendSelectEvent(ctx events.Event) {
	tv.RootView.Send(events.Select, nil)
}

// SendChangeEvent sends the events.Change event on the
// RootView node, using context event if avail (else nil).
func (tv *TreeView) SendChangeEvent(ctx events.Event) {
	tv.RootView.SendChange(nil)
}

// TreeViewChanged must be called after any structural
// change to the TreeView (adding or deleting nodes).
// It calls: RootSetViewIdx() to update indexes and
// SendChangeEvent to notify of changes.
func (tv *TreeView) TreeViewChanged(ctx events.Event) {
	tv.RootView.RootSetViewIdx()
	tv.SendChangeEvent(ctx)
}

// SendChangeEventReSync sends the events.Change event on the
// RootView node, using context event if avail (else nil).
// If SyncNode != nil, also does a re-sync from root.
func (tv *TreeView) SendChangeEventReSync(ctx events.Event) {
	tv.RootView.SendChange(nil)
	if tv.RootView.SyncNode != nil {
		tv.RootView.ReSync()
	}
}

// SelectAction updates selection to include this node,
// using selectmode from mouse event (ExtendContinuous, ExtendOne),
// and Root sends selection event.  Returns true if signal emitted.
func (tv *TreeView) SelectAction(mode events.SelectModes) bool {
	sel := tv.SelectUpdate(mode)
	if sel {
		tv.SendSelectEvent(nil)
	}
	return sel
}

// UnselectAction unselects this node (if selected),
// and Root sends a selection event.
func (tv *TreeView) UnselectAction() {
	if tv.StateIs(states.Selected) {
		tv.Unselect()
		tv.SendSelectEvent(nil)
	}
}

//////////////////////////////////////////////////////////////////////////////
//    Moving

// MoveDown moves the selection down to next element in the tree,
// using given select mode (from keyboard modifiers).
// Returns newly selected node.
func (tv *TreeView) MoveDown(selMode events.SelectModes) *TreeView {
	if tv.Par == nil {
		return nil
	}
	if tv.IsClosed() || !tv.HasChildren() { // next sibling
		return tv.MoveDownSibling(selMode)
	} else {
		if tv.HasChildren() {
			nn := AsTreeView(tv.Child(0))
			if nn != nil {
				nn.SelectUpdate(selMode)
				return nn
			}
		}
	}
	return nil
}

// MoveDownAction moves the selection down to next element in the tree,
// using given select mode (from keyboard modifiers).
// Sends select event for newly selected item.
func (tv *TreeView) MoveDownAction(selMode events.SelectModes) *TreeView {
	nn := tv.MoveDown(selMode)
	if nn != nil && nn != tv {
		nn.GrabFocus()
		nn.ScrollToMe()
		tv.SendSelectEvent(nil)
	}
	return nn
}

// MoveDownSibling moves down only to siblings, not down into children,
// using given select mode (from keyboard modifiers)
func (tv *TreeView) MoveDownSibling(selMode events.SelectModes) *TreeView {
	if tv.Par == nil {
		return nil
	}
	if tv == tv.RootView {
		return nil
	}
	myidx, ok := tv.IndexInParent()
	if ok && myidx < len(*tv.Par.Children())-1 {
		nn := AsTreeView(tv.Par.Child(myidx + 1))
		if nn != nil {
			nn.SelectUpdate(selMode)
			return nn
		}
	} else {
		return AsTreeView(tv.Par).MoveDownSibling(selMode) // try up
	}
	return nil
}

// MoveUp moves selection up to previous element in the tree,
// using given select mode (from keyboard modifiers).
// Returns newly selected node
func (tv *TreeView) MoveUp(selMode events.SelectModes) *TreeView {
	if tv.Par == nil || tv == tv.RootView {
		return nil
	}
	myidx, ok := tv.IndexInParent()
	if ok && myidx > 0 {
		nn := AsTreeView(tv.Par.Child(myidx - 1))
		if nn != nil {
			return nn.MoveToLastChild(selMode)
		}
	} else {
		if tv.Par != nil {
			nn := AsTreeView(tv.Par)
			if nn != nil {
				nn.SelectUpdate(selMode)
				return nn
			}
		}
	}
	return nil
}

// MoveUpAction moves the selection up to previous element in the tree,
// using given select mode (from keyboard modifiers).
// Sends select event for newly selected item.
func (tv *TreeView) MoveUpAction(selMode events.SelectModes) *TreeView {
	nn := tv.MoveUp(selMode)
	if nn != nil && nn != tv {
		nn.GrabFocus()
		nn.ScrollToMe()
		tv.SendSelectEvent(nil)
	}
	return nn
}

// TreeViewPageSteps is the number of steps to take in PageUp / Down events
var TreeViewPageSteps = 10

// MovePageUpAction moves the selection up to previous
// TreeViewPageSteps elements in the tree,
// using given select mode (from keyboard modifiers).
// Sends select event for newly selected item.
func (tv *TreeView) MovePageUpAction(selMode events.SelectModes) *TreeView {
	updt := tv.UpdateStart()
	mvMode := selMode
	if selMode == events.SelectOne {
		mvMode = events.NoSelect
	} else if selMode == events.ExtendContinuous || selMode == events.ExtendOne {
		mvMode = events.SelectQuiet
	}
	fnn := tv.MoveUp(mvMode)
	if fnn != nil && fnn != tv {
		for i := 1; i < TreeViewPageSteps; i++ {
			nn := fnn.MoveUp(mvMode)
			if nn == nil || nn == fnn {
				break
			}
			fnn = nn
		}
		if selMode == events.SelectOne {
			fnn.SelectUpdate(selMode)
		}
		fnn.GrabFocus()
		fnn.ScrollToMe()
		tv.SendSelectEvent(nil)
	}
	tv.UpdateEndRender(updt)
	return fnn
}

// MovePageDownAction moves the selection up to
// previous TreeViewPageSteps elements in the tree,
// using given select mode (from keyboard modifiers).
// Sends select event for newly selected item.
func (tv *TreeView) MovePageDownAction(selMode events.SelectModes) *TreeView {
	updt := tv.UpdateStart()
	mvMode := selMode
	if selMode == events.SelectOne {
		mvMode = events.NoSelect
	} else if selMode == events.ExtendContinuous || selMode == events.ExtendOne {
		mvMode = events.SelectQuiet
	}
	fnn := tv.MoveDown(mvMode)
	if fnn != nil && fnn != tv {
		for i := 1; i < TreeViewPageSteps; i++ {
			nn := fnn.MoveDown(mvMode)
			if nn == nil || nn == fnn {
				break
			}
			fnn = nn
		}
		if selMode == events.SelectOne {
			fnn.SelectUpdate(selMode)
		}
		fnn.GrabFocus()
		fnn.ScrollToMe()
		tv.SendSelectEvent(nil)
	}
	tv.UpdateEndRender(updt)
	return fnn
}

// MoveToLastChild moves to the last child under me, using given select mode
// (from keyboard modifiers)
func (tv *TreeView) MoveToLastChild(selMode events.SelectModes) *TreeView {
	if tv.Par == nil || tv == tv.RootView {
		return nil
	}
	if !tv.IsClosed() && tv.HasChildren() {
		nnk, err := tv.Children().ElemFromEndTry(0)
		if err == nil {
			nn := AsTreeView(nnk)
			return nn.MoveToLastChild(selMode)
		}
	} else {
		tv.SelectUpdate(selMode)
		return tv
	}
	return nil
}

// MoveHomeAction moves the selection up to top of the tree,
// using given select mode (from keyboard modifiers)
// and emits select event for newly selected item
func (tv *TreeView) MoveHomeAction(selMode events.SelectModes) *TreeView {
	tv.RootView.SelectUpdate(selMode)
	tv.RootView.GrabFocus()
	tv.RootView.ScrollToMe()
	// tv.RootView.TreeViewSig.Emit(tv.RootView.This(), int64(TreeViewSelected), tv.RootView.This())
	return tv.RootView
}

// MoveEndAction moves the selection to the very last node in the tree,
// using given select mode (from keyboard modifiers)
// Sends select event for newly selected item.
func (tv *TreeView) MoveEndAction(selMode events.SelectModes) *TreeView {
	updt := tv.UpdateStart()
	mvMode := selMode
	if selMode == events.SelectOne {
		mvMode = events.NoSelect
	} else if selMode == events.ExtendContinuous || selMode == events.ExtendOne {
		mvMode = events.SelectQuiet
	}
	fnn := tv.MoveDown(mvMode)
	if fnn != nil && fnn != tv {
		for {
			nn := fnn.MoveDown(mvMode)
			if nn == nil || nn == fnn {
				break
			}
			fnn = nn
		}
		if selMode == events.SelectOne {
			fnn.SelectUpdate(selMode)
		}
		fnn.GrabFocus()
		fnn.ScrollToMe()
		tv.SendSelectEvent(nil)
	}
	tv.UpdateEnd(updt)
	return fnn
}

func (tv *TreeView) SetKidsVisibility(parentClosed bool) {
	for _, k := range *tv.Children() {
		tvki := AsTreeView(k)
		if tvki != nil {
			tvki.SetState(parentClosed, states.Invisible)
		}
	}
}

// OnClose is called when a node is closed.
// The base version does nothing.
func (tv *TreeView) OnClose() {
}

// Close closes the given node and updates the view accordingly
// (if it is not already closed).
// Calls OnClose in TreeViewer interface for extensible actions.
func (tv *TreeView) Close() {
	if tv.IsClosed() {
		return
	}
	updt := tv.UpdateStart()
	if tv.HasChildren() {
		tv.SetNeedsLayout()
	}
	tv.SetClosed(true)
	tv.SetBranchState()
	tv.This().(TreeViewer).OnClose()
	tv.SetKidsVisibility(true) // parent closed
	tv.UpdateEndRender(updt)
}

// OnOpen is called when a node is opened.
// The base version does nothing.
func (tv *TreeView) OnOpen() {
}

// Open opens the given node and updates the view accordingly
// (if it is not already opened)
// Calls OnOpen in TreeViewer interface for extensible actions.
func (tv *TreeView) Open() {
	if !tv.IsClosed() {
		return
	}
	updt := tv.UpdateStart()
	if tv.HasChildren() {
		tv.SetNeedsLayout()
		tv.SetClosed(false)
		tv.SetBranchState()
		tv.This().(TreeViewer).OnOpen()
		tv.SetKidsVisibility(false)
	}
	tv.UpdateEndRender(updt)
}

// ToggleClose toggles the close / open status: if closed, opens, and vice-versa
func (tv *TreeView) ToggleClose() {
	if tv.IsClosed() {
		tv.Open()
	} else {
		tv.Close()
	}
}

// OpenAll opens the given node and all of its sub-nodes
func (tv *TreeView) OpenAll() { //gti:add
	updt := tv.UpdateStart()
	tv.WidgetWalkPre(func(wi gi.Widget, wb *gi.WidgetBase) bool {
		tvki := AsTreeView(wi)
		if tvki != nil {
			tvki.Open()
			return ki.Continue
		}
		return ki.Break
	})
	tv.UpdateEndLayout(updt)
}

// CloseAll closes the given node and all of its sub-nodes.
func (tv *TreeView) CloseAll() { //gti:add
	updt := tv.UpdateStart()
	tv.WidgetWalkPre(func(wi gi.Widget, wb *gi.WidgetBase) bool {
		tvki := AsTreeView(wi)
		if tvki != nil {
			tvki.Close()
			return ki.Continue
		}
		return ki.Break
	})
	tv.UpdateEndLayout(updt)
}

// OpenParents opens all the parents of this node,
// so that it will be visible.
func (tv *TreeView) OpenParents() {
	updt := tv.UpdateStart()
	tv.WalkUpParent(func(k ki.Ki) bool {
		tvki := AsTreeView(k)
		if tvki != nil {
			tvki.Open()
			return ki.Continue
		}
		return ki.Break
	})
	tv.UpdateEndLayout(updt)
}

/////////////////////////////////////////////////////////////
//    Modifying Source Tree

func (tv *TreeView) ContextMenuPos(e events.Event) (pos image.Point) {
	if e != nil {
		pos = e.Pos()
		return
	}
	pos.X = tv.Geom.TotalBBox.Min.X + int(tv.Indent.Dots)
	pos.Y = (tv.Geom.TotalBBox.Min.Y + tv.Geom.TotalBBox.Max.Y) / 2
	return
}

func (tv *TreeView) TreeViewContextMenuReadOnly(m *gi.Scene) {
	gi.NewButton(m).SetText("Copy").SetIcon(icons.ContentCopy).SetKey(keyfun.Copy).
		SetState(!tv.HasSelection(), states.Disabled).
		OnClick(func(e events.Event) {
			tv.This().(gi.Clipper).Copy(true)
		})
	gi.NewButton(m).SetText("Edit").SetIcon(icons.Edit).
		SetState(!tv.HasSelection(), states.Disabled).
		OnClick(func(e events.Event) {
			tv.EditNode()
		})
	gi.NewButton(m).SetText("Inspector").SetIcon(icons.EditDocument).
		SetState(!tv.HasSelection(), states.Disabled).
		OnClick(func(e events.Event) {
			tv.InspectNode()
		})
}

func (tv *TreeView) TreeViewContextMenu(m *gi.Scene) {
	NewFuncButton(m, tv.AddChildNode).SetText("Add child").SetIcon(icons.Add).
		SetState(!tv.HasSelection(), states.Disabled)
	NewFuncButton(m, tv.InsertBefore).SetIcon(icons.Add).
		SetState(!tv.HasSelection(), states.Disabled)
	NewFuncButton(m, tv.InsertAfter).SetIcon(icons.Add).
		SetState(!tv.HasSelection(), states.Disabled)
	NewFuncButton(m, tv.Duplicate).SetIcon(icons.ContentCopy).
		SetState(!tv.HasSelection(), states.Disabled)
	NewFuncButton(m, tv.DeleteNode).SetText("Delete").SetIcon(icons.Delete).
		SetState(!tv.HasSelection(), states.Disabled)
	gi.NewSeparator(m)
	gi.NewButton(m).SetText("Copy").SetIcon(icons.ContentCopy).SetKey(keyfun.Copy).
		SetState(!tv.HasSelection(), states.Disabled).
		OnClick(func(e events.Event) {
			tv.This().(gi.Clipper).Copy(true)
		})
	gi.NewButton(m).SetText("Cut").SetIcon(icons.ContentCut).SetKey(keyfun.Cut).
		SetState(!tv.HasSelection(), states.Disabled).
		OnClick(func(e events.Event) {
			tv.This().(gi.Clipper).Cut()
		})
	pbt := gi.NewButton(m).SetText("Paste").SetIcon(icons.ContentPaste).SetKey(keyfun.Paste).
		OnClick(func(e events.Event) {
			tv.This().(gi.Clipper).Paste()
		})
	cb := tv.Sc.EventMgr.ClipBoard()
	if cb != nil {
		pbt.SetState(cb.IsEmpty(), states.Disabled)
	}
	gi.NewSeparator(m)
	NewFuncButton(m, tv.EditNode).SetText("Edit").SetIcon(icons.Edit).
		SetState(!tv.HasSelection(), states.Disabled)
	NewFuncButton(m, tv.InspectNode).SetText("Inspector").SetIcon(icons.EditDocument).
		SetState(!tv.HasSelection(), states.Disabled)
	gi.NewSeparator(m)

	// icons.Open, Close
	NewFuncButton(m, tv.OpenAll).SetIcon(icons.KeyboardArrowDown).
		SetState(!tv.HasSelection(), states.Disabled)
	NewFuncButton(m, tv.CloseAll).SetIcon(icons.KeyboardArrowRight).
		SetState(!tv.HasSelection(), states.Disabled)
}

func (tv *TreeView) ContextMenu(m *gi.Scene) {
	// derived types put native menu code here
	if tv.CustomContextMenu != nil {
		tv.CustomContextMenu(m)
	}
	// TODO(kai/menu): need a replacement for this:
	// if tv.SyncNode != nil && CtxtMenuView(tv.SyncNode, tv.RootIsReadOnly(), tv.Scene, m) { // our viewed obj's menu
	// 	if tv.ShowViewCtxtMenu {
	// 		m.AddSeparator("sep-tvmenu")
	// 		CtxtMenuView(tv.This(), tv.RootIsReadOnly(), tv.Scene, m)
	// 	}
	// } else {
	if tv.IsReadOnly() {
		tv.TreeViewContextMenuReadOnly(m)
	} else {
		tv.TreeViewContextMenu(m)
	}
}

// IsRoot returns true if given node is the root of the tree.
func (tv *TreeView) IsRoot(op string) bool {
	if tv.This() == tv.RootView.This() {
		if op != "" {
			// TODO(kai/snack)
			// gi.PromptDialog(tv, gi.DlgOpts{Title: "TreeView " + op, Prompt: fmt.Sprintf("Cannot %v the root of the tree", op), Ok: true, Cancel: false}, nil)
		}
		return true
	}
	return false
}

////////////////////////////////////////////////////////////
//    Copy / Cut / Paste

// MimeData adds mimedata for this node: a text/plain of the Path.
// satisfies Clipper.MimeData interface
func (tv *TreeView) MimeData(md *mimedata.Mimes) {
	if tv.SyncNode != nil {
		tv.MimeDataSync(md)
		return
	}
	*md = append(*md, mimedata.NewTextData(tv.PathFrom(tv.RootView)))
	var buf bytes.Buffer
	err := ki.WriteNewJSON(tv.This(), &buf)
	if err == nil {
		*md = append(*md, &mimedata.Data{Type: filecat.DataJson, Data: buf.Bytes()})
	} else {
		slog.Error("giv.TreeView MimeData Write JSON error:", err)
	}
}

// NodesFromMimeData returns a slice of Ki nodes for
// the TreeView nodes and paths from mime data.
func (tv *TreeView) NodesFromMimeData(md mimedata.Mimes) (ki.Slice, []string) {
	ni := len(md) / 2
	sl := make(ki.Slice, 0, ni)
	pl := make([]string, 0, ni)
	for _, d := range md {
		if d.Type == filecat.DataJson {
			nki, err := ki.ReadNewJSON(bytes.NewReader(d.Data))
			if err == nil {
				sl = append(sl, nki)
			} else {
				slog.Error("giv.TreeView NodesFromMimeData: JSON load error:", err)
			}
		} else if d.Type == filecat.TextPlain { // paths
			pl = append(pl, string(d.Data))
		}
	}
	return sl, pl
}

// Copy copies to clip.Board, optionally resetting the selection.
// satisfies gi.Clipper interface and can be overridden by subtypes
func (tv *TreeView) Copy(reset bool) { //gti:add
	sels := tv.SelectedViews()
	nitms := max(1, len(sels))
	md := make(mimedata.Mimes, 0, 2*nitms)
	tv.This().(gi.Clipper).MimeData(&md) // source is always first..
	if nitms > 1 {
		for _, sn := range sels {
			if sn.This() != tv.This() {
				sn.This().(gi.Clipper).MimeData(&md)
			}
		}
	}
	tv.EventMgr().ClipBoard().Write(md)
	if reset {
		tv.UnselectAll()
	}
}

// Cut copies to clip.Board and deletes selected items.
// satisfies gi.Clipper interface and can be overridden by subtypes
func (tv *TreeView) Cut() { //gti:add
	if tv.IsRoot("Cut") {
		return
	}
	if tv.SyncNode != nil {
		tv.CutSync()
		return
	}
	tv.Copy(false)
	sels := tv.SelectedViews()
	root := tv.RootView
	updt := root.UpdateStart()
	tv.UnselectAll()
	for _, sn := range sels {
		sn.Delete(true)
	}
	root.Update()
	root.TreeViewChanged(nil)
	root.UpdateEndLayout(updt)
}

// Paste pastes clipboard at given node.
// satisfies gi.Clipper interface and can be overridden by subtypes
func (tv *TreeView) Paste() { //gti:add
	md := tv.EventMgr().ClipBoard().Read([]string{filecat.DataJson})
	if md != nil {
		tv.PasteMenu(md)
	}
}

// MakePasteMenu makes the menu of options for paste events
func (tv *TreeView) MakePasteMenu(m *gi.Scene, data any) {
	gi.NewButton(m).SetText("Assign To").SetData(data).
		OnClick(func(e events.Event) {
			tv.PasteAssign(data.(mimedata.Mimes))
		})
	gi.NewButton(m).SetText("Add to Children").SetData(data).
		OnClick(func(e events.Event) {
			tv.PasteChildren(data.(mimedata.Mimes), events.DropCopy)
		})
	if !tv.IsRoot("") && tv.RootView.This() != tv.This() {
		gi.NewButton(m).SetText("Insert Before").SetData(data).
			OnClick(func(e events.Event) {
				tv.PasteBefore(data.(mimedata.Mimes), events.DropCopy)
			})
		gi.NewButton(m).SetText("Insert After").SetData(data).
			OnClick(func(e events.Event) {
				tv.PasteAfter(data.(mimedata.Mimes), events.DropCopy)
			})
	}
	gi.NewButton(m).SetText("Cancel").SetData(data)
	// todo: compare, etc..
}

// PasteMenu performs a paste from the clipboard using given data -- pops up
// a menu to determine what specifically to do
func (tv *TreeView) PasteMenu(md mimedata.Mimes) {
	tv.UnselectAll()
	mf := func(m *gi.Scene) {
		tv.MakePasteMenu(m, md)
	}
	pos := tv.ContextMenuPos(nil)
	gi.NewMenu(mf, tv.This().(gi.Widget), pos).Run()
}

// PasteAssign assigns mime data (only the first one!) to this node
func (tv *TreeView) PasteAssign(md mimedata.Mimes) {
	if tv.SyncNode != nil {
		tv.PasteAssignSync(md)
		return
	}
	sl, _ := tv.NodesFromMimeData(md)
	if len(sl) == 0 {
		return
	}
	updt := tv.UpdateStart()
	tv.This().CopyFrom(sl[0]) // nodes with data copy here
	tv.Update()               // could have children
	tv.Open()
	tv.TreeViewChanged(nil)
	tv.UpdateEndLayout(updt)
}

// PasteBefore inserts object(s) from mime data before this node.
// If another item with the same name already exists, it will
// append _Copy on the name of the inserted objects
func (tv *TreeView) PasteBefore(md mimedata.Mimes, mod events.DropMods) {
	tv.PasteAt(md, mod, 0, "Paste Before")
}

// PasteAfter inserts object(s) from mime data after this node.
// If another item with the same name already exists, it will
// append _Copy on the name of the inserted objects
func (tv *TreeView) PasteAfter(md mimedata.Mimes, mod events.DropMods) {
	tv.PasteAt(md, mod, 1, "Paste After")
}

// This is a kind of hack to prevent moved items from being deleted, using DND
const TreeViewTempMovedTag = `_\&MOVED\&`

// todo: these methods require an interface to work for descended
// nodes, based on base code

// PasteAt inserts object(s) from mime data at rel position to this node.
// If another item with the same name already exists, it will
// append _Copy on the name of the inserted objects
func (tv *TreeView) PasteAt(md mimedata.Mimes, mod events.DropMods, rel int, actNm string) {
	if tv.Par == nil {
		return
	}
	par := AsTreeView(tv.Par)
	if par == nil {
		// TODO(kai/snack)
		// gi.PromptDialog(tv, gi.DlgOpts{Title: actNm, Prompt: "Cannot insert after the root of the tree", Ok: true, Cancel: false}, nil)
		return
	}
	if tv.SyncNode != nil {
		tv.PasteAtSync(md, mod, rel, actNm)
		return
	}
	sl, pl := tv.NodesFromMimeData(md)

	myidx, ok := tv.IndexInParent()
	if !ok {
		return
	}
	myidx += rel
	updt := par.UpdateStart()
	sz := len(sl)
	var selTv *TreeView
	for i, ns := range sl {
		orgpath := pl[i]
		if mod != events.DropMove {
			if cn := par.ChildByName(ns.Name(), 0); cn != nil {
				ns.SetName(ns.Name() + "_Copy")
			}
		}
		par.SetChildAdded()
		par.InsertChild(ns, myidx+i)
		nwi := ns.This().(gi.Widget)
		ntv := AsTreeView(ns)
		ntv.RootView = tv.RootView
		ntv.Sc = tv.Sc
		nwi.AsWidget().Update() // incl children
		npath := ns.PathFrom(tv.RootView)
		if mod == events.DropMove && npath == orgpath { // we will be nuked immediately after drag

			ns.SetName(ns.Name() + TreeViewTempMovedTag) // special keyword :)
		}
		if i == sz-1 {
			selTv = ntv
		}
	}
	tv.TreeViewChanged(nil)
	par.UpdateEndLayout(updt)
	if selTv != nil {
		selTv.SelectAction(events.SelectOne)
	}
}

// PasteChildren inserts object(s) from mime data
// at end of children of this node
func (tv *TreeView) PasteChildren(md mimedata.Mimes, mod events.DropMods) {
	if tv.SyncNode != nil {
		tv.PasteChildrenSync(md, mod)
		return
	}
	sl, _ := tv.NodesFromMimeData(md)

	updt := tv.UpdateStart()
	tv.SetChildAdded()
	for _, ns := range sl {
		tv.AddChild(ns)
	}
	tv.Update()
	tv.Open()
	tv.TreeViewChanged(nil)
	tv.UpdateEndLayout(updt)
}

//////////////////////////////////////////////////////////////////////////////
//    Drag-n-Drop

// DragNDropStart starts a drag-n-drop on this node -- it includes any other
// selected nodes as well, each as additional records in mimedata
func (tv *TreeView) DragNDropStart() {
	sels := tv.SelectedViews()
	nitms := max(1, len(sels))
	md := make(mimedata.Mimes, 0, 2*nitms)
	tv.This().(gi.Clipper).MimeData(&md) // source is always first..
	if nitms > 1 {
		for _, sn := range sels {
			if sn.This() != tv.This() {
				sn.This().(gi.Clipper).MimeData(&md)
			}
		}
	}
	sp := &gi.Sprite{}
	sp.GrabRenderFrom(tv) // todo: show number of items?
	gi.ImageClearer(sp.Pixels, 50.0)
	// tv.ParentRenderWin().StartDragNDrop(tv.This(), md, sp)
}

/*
// DragNDropTarget handles a drag-n-drop onto this node
func (tv *TreeView) DragNDropTarget(de events.Event) {
	de.Target = tv.This()
	if de.Mod == events.DropLink {
		de.Mod = events.DropCopy // link not supported -- revert to copy
	}
	de.SetHandled()
	tv.This().(gi.DragNDropper).Drop(de.Data, de.Mod)
}

// DragNDropExternal handles a drag-n-drop external drop onto this node
func (tv *TreeView) DragNDropExternal(de events.Event) {
	de.Target = tv.This()
	if de.Mod == events.DropLink {
		de.Mod = events.DropCopy // link not supported -- revert to copy
	}
	de.SetHandled()
	tv.This().(gi.DragNDropper).DropExternal(de.Data, de.Mod)
}

// DragNDropFinalize is called to finalize actions on the Source node prior to
// performing target actions -- mod must indicate actual action taken by the
// target, including ignore
func (tv *TreeView) DragNDropFinalize(mod events.DropMods) {
	if tv.Scene == nil {
		return
	}
	tv.UnselectAll()
	tv.ParentRenderWin().FinalizeDragNDrop(mod)
}

// DragNDropFinalizeDefMod is called to finalize actions on the Source node prior to
// performing target actions -- uses default drop mod in place when event was dropped.
func (tv *TreeView) DragNDropFinalizeDefMod() {
	win := tv.ParentRenderWin()
	if win == nil {
		return
	}
	tv.UnselectAll()
	win.FinalizeDragNDrop(win.EventMgr.DNDDropMod)
}

// Dragged is called after target accepts the drop -- we just remove
// elements that were moved
// satisfies gi.DragNDropper interface and can be overridden by subtypes
func (tv *TreeView) Dragged(de events.Event) {
	if de.Mod != events.DropMove {
		return
	}
	sroot := tv.RootView.SyncNode
	md := de.Data
	for _, d := range md {
		if d.Type == filecat.TextPlain { // link
			path := string(d.Data)
			sn := sroot.FindPath(path)
			if sn != nil {
				sn.Delete(true)
				// tv.RootView.TreeViewSig.Emit(tv.RootView.This(), int64(TreeViewDeleted), sn.This())
			}
			sn = sroot.FindPath(path + TreeViewTempMovedTag)
			if sn != nil {
				psplt := strings.Split(path, "/")
				orgnm := psplt[len(psplt)-1]
				sn.SetName(orgnm)
				sn.SetNeedsRender()
			}
		}
	}
}

// MakeDropMenu makes the menu of options for dropping on a target
func (tv *TreeView) MakeDropMenu(m *gi.Menu, data any, mod events.DropMods) {
	if len(*m) > 0 {
		return
	}
	switch mod {
	case events.DropCopy:
		m.AddLabel("Copy (Use Shift to Move):")
	case events.DropMove:
		m.AddLabel("Move:")
	}
	if mod == events.DropCopy {
		gi.NewButton(m).SetText("Assign To", Data: data}, tv.This(), func(recv, send ki.Ki, sig int64, data any) {
			tv := recv.Embed(TreeViewType).(*TreeView)
			tv.DropAssign(data.(mimedata.Mimes))
		})
	}
	gi.NewButton(m).SetText("Add to Children", Data: data}, tv.This(), func(recv, send ki.Ki, sig int64, data any) {
		tv := recv.Embed(TreeViewType).(*TreeView)
		tv.DropChildren(data.(mimedata.Mimes), mod) // captures mod
	})
	if !tv.IsRoot("") && tv.RootView.This() != tv.This() {
		gi.NewButton(m).SetText("Insert Before", Data: data}, tv.This(), func(recv, send ki.Ki, sig int64, data any) {
			tv := recv.Embed(TreeViewType).(*TreeView)
			tv.DropBefore(data.(mimedata.Mimes), mod) // captures mod
		})
		gi.NewButton(m).SetText("Insert After", Data: data}, tv.This(), func(recv, send ki.Ki, sig int64, data any) {
			tv := recv.Embed(TreeViewType).(*TreeView)
			tv.DropAfter(data.(mimedata.Mimes), mod) // captures mod
		})
	}
	gi.NewButton(m).SetText("Cancel", Data: data}, tv.This(), func(recv, send ki.Ki, sig int64, data any) {
		tv := recv.Embed(TreeViewType).(*TreeView)
		tv.DropCancel()
	})
	// todo: compare, etc..
}

// Drop pops up a menu to determine what specifically to do with dropped items
// satisfies gi.DragNDropper interface and can be overridden by subtypes
func (tv *TreeView) Drop(md mimedata.Mimes, mod events.DropMods) {
	var menu gi.Menu
	tv.MakeDropMenu(&menu, md, mod)
	pos := tv.ContextMenuPos()
	gi.NewMenu(menu, tv.This().(gi.Widget), pos).Run()
}

// DropExternal is not handled by base case but could be in derived
func (tv *TreeView) DropExternal(md mimedata.Mimes, mod events.DropMods) {
	tv.DropCancel()
}

// DropAssign assigns mime data (only the first one!) to this node
func (tv *TreeView) DropAssign(md mimedata.Mimes) {
	tv.PasteAssign(md)
	tv.DragNDropFinalize(events.DropCopy)
}

// DropBefore inserts object(s) from mime data before this node
func (tv *TreeView) DropBefore(md mimedata.Mimes, mod events.DropMods) {
	tv.PasteBefore(md, mod)
	tv.DragNDropFinalize(mod)
}

// DropAfter inserts object(s) from mime data after this node
func (tv *TreeView) DropAfter(md mimedata.Mimes, mod events.DropMods) {
	tv.PasteAfter(md, mod)
	tv.DragNDropFinalize(mod)
}

// DropChildren inserts object(s) from mime data at end of children of this node
func (tv *TreeView) DropChildren(md mimedata.Mimes, mod events.DropMods) {
	tv.PasteChildren(md, mod)
	tv.DragNDropFinalize(mod)
}

// DropCancel cancels the drop action e.g., preventing deleting of source
// items in a Move case
func (tv *TreeView) DropCancel() {
	tv.DragNDropFinalize(events.DropIgnore)
}

*/

////////////////////////////////////////////////////
// 	Event Handlers

func (tv *TreeView) TreeViewParent() *TreeView {
	if tv.Par == nil {
		return nil
	}
	return AsTreeView(tv.Par)
}

func (tv *TreeView) HandleTreeViewEvents() {
	tv.HandleWidgetEvents()
	tv.On(events.KeyChord, func(e events.Event) {
		tv.HandleTreeViewKeyChord(e)
	})
	tv.HandleTreeViewMouse()
	tv.HandleTreeViewDrag()
}

func (tv *TreeView) HandleTreeViewKeyChord(kt events.Event) {
	if gi.KeyEventTrace {
		fmt.Printf("TreeView KeyInput: %v\n", tv.Path())
	}
	kf := keyfun.Of(kt.KeyChord())
	selMode := events.SelectModeBits(kt.Modifiers())

	if selMode == events.SelectOne {
		if tv.SelectMode() {
			selMode = events.ExtendContinuous
		}
	}

	// first all the keys that work for ReadOnly and active
	switch kf {
	case keyfun.CancelSelect:
		tv.UnselectAll()
		tv.SetSelectMode(false)
		kt.SetHandled()
	case keyfun.MoveRight:
		tv.Open()
		kt.SetHandled()
	case keyfun.MoveLeft:
		tv.Close()
		kt.SetHandled()
	case keyfun.MoveDown:
		tv.MoveDownAction(selMode)
		kt.SetHandled()
	case keyfun.MoveUp:
		tv.MoveUpAction(selMode)
		kt.SetHandled()
	case keyfun.PageUp:
		tv.MovePageUpAction(selMode)
		kt.SetHandled()
	case keyfun.PageDown:
		tv.MovePageDownAction(selMode)
		kt.SetHandled()
	case keyfun.Home:
		tv.MoveHomeAction(selMode)
		kt.SetHandled()
	case keyfun.End:
		tv.MoveEndAction(selMode)
		kt.SetHandled()
	case keyfun.SelectMode:
		tv.SelectModeToggle()
		kt.SetHandled()
	case keyfun.SelectAll:
		tv.SelectAll()
		kt.SetHandled()
	case keyfun.Enter:
		tv.ToggleClose()
		kt.SetHandled()
	case keyfun.Copy:
		tv.This().(gi.Clipper).Copy(true)
		kt.SetHandled()
	}
	if !tv.RootIsReadOnly() && !kt.IsHandled() {
		switch kf {
		case keyfun.Delete:
			tv.DeleteNode()
			kt.SetHandled()
		case keyfun.Duplicate:
			tv.Duplicate()
			kt.SetHandled()
		case keyfun.Insert:
			tv.InsertBefore()
			kt.SetHandled()
		case keyfun.InsertAfter:
			tv.InsertAfter()
			kt.SetHandled()
		case keyfun.Cut:
			tv.This().(gi.Clipper).Cut()
			kt.SetHandled()
		case keyfun.Paste:
			tv.This().(gi.Clipper).Paste()
			kt.SetHandled()
		}
	}
}

func (tv *TreeView) HandleTreeViewMouse() {
	// we let the parts handle our state
	// so that we only get it when we are doing
	// something with this treeview specifically,
	// not with any of our children (see OnChildAdded).
	// we only need to handle the starting ones here,
	// as the other ones will just set the state to
	// false, which it already is.
	tv.On(events.MouseEnter, func(e events.Event) {
		e.SetHandled()
	})
	tv.On(events.MouseDown, func(e events.Event) {
		e.SetHandled()
	})
	tv.OnClick(func(e events.Event) {
		e.SetHandled()
	})
}

func (tv *TreeView) HandleTreeViewDrag() {
	/*
		tvwe.AddFunc(goosi.DNDEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d any) {
			if recv == nil {
				return
			}
			de := d.(events.Event)
			tv := recv.Embed(TreeViewType).(*TreeView)
			switch de.Action {
			case events.Start:
				tv.DragNDropStart()
			case events.DropOnTarget:
				tv.DragNDropTarget(de)
			case events.DropFmSource:
				tv.This().(gi.DragNDropper).Dragged(de)
			case events.External:
				tv.DragNDropExternal(de)
			}
		})
		tvwe.AddFunc(goosi.DNDFocusEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d any) {
			if recv == nil {
				return
			}
			de := d.(*events.FocusEvent)
			tv := recv.Embed(TreeViewType).(*TreeView)
			switch de.Action {
			case events.Enter:
				tv.ParentRenderWin().DNDSetCursor(de.Mod)
			case events.Exit:
				tv.ParentRenderWin().DNDNotCursor()
			case events.Hover:
				tv.Open()
			}
		})
	*/
}

var TreeViewProps = ki.Props{
	"CtxtMenuActive": ki.PropSlice{
		{"SrcAddChild", ki.Props{
			"label": "Add Child",
		}},
		{"SrcInsertBefore", ki.Props{
			"label":    "Insert Before",
			"shortcut": keyfun.Insert,
			"updtfunc": ActionUpdateFunc(func(tvi any, act *gi.Button) {
				// tv := tvi.(ki.Ki).Embed(TreeViewType).(*TreeView)
				// act.SetState(tv.IsRoot(""), states.Disabled)
			}),
		}},
		{"SrcInsertAfter", ki.Props{
			"label":    "Insert After",
			"shortcut": keyfun.InsertAfter,
			"updtfunc": ActionUpdateFunc(func(tvi any, act *gi.Button) {
				// tv := tvi.(ki.Ki).Embed(TreeViewType).(*TreeView)
				// act.SetState(tv.IsRoot(""), states.Disabled)
			}),
		}},
		{"SrcDuplicate", ki.Props{
			"label":    "Duplicate",
			"shortcut": keyfun.Duplicate,
			"updtfunc": ActionUpdateFunc(func(tvi any, act *gi.Button) {
				// tv := tvi.(ki.Ki).Embed(TreeViewType).(*TreeView)
				// act.SetState(tv.IsRoot(""), states.Disabled)
			}),
		}},
		{"SrcDelete", ki.Props{
			"label":    "Delete",
			"shortcut": keyfun.Delete,
			"updtfunc": ActionUpdateFunc(func(tvi any, act *gi.Button) {
				// tv := tvi.(ki.Ki).Embed(TreeViewType).(*TreeView)
				// act.SetState(tv.IsRoot(""), states.Disabled)
			}),
		}},
		{"sep-edit", ki.BlankProp{}},
		{"Copy", ki.Props{
			"shortcut": keyfun.Copy,
			"Args": ki.PropSlice{
				{"reset", ki.Props{
					"value": true,
				}},
			},
		}},
		{"Cut", ki.Props{
			"shortcut": keyfun.Cut,
			"updtfunc": ActionUpdateFunc(func(tvi any, act *gi.Button) {
				// tv := tvi.(ki.Ki).Embed(TreeViewType).(*TreeView)
				// act.SetState(tv.IsRoot(""), states.Disabled)
			}),
		}},
		{"Paste", ki.Props{
			"shortcut": keyfun.Paste,
		}},
		{"sep-win", ki.BlankProp{}},
		{"SrcEdit", ki.Props{
			"label": "Edit",
		}},
		{"SrcInspector", ki.Props{
			"label": "Inspector",
		}},
		{"sep-open", ki.BlankProp{}},
		{"OpenAll", ki.Props{}},
		{"CloseAll", ki.Props{}},
	},
	"CtxtMenuReadOnly": ki.PropSlice{
		{"Copy", ki.Props{
			"shortcut": keyfun.Copy,
			"Args": ki.PropSlice{
				{"reset", ki.Props{
					"value": true,
				}},
			},
		}},
		{"SrcEdit", ki.Props{
			"label": "Edit",
		}},
		{"SrcInspector", ki.Props{
			"label": "Inspector",
		}},
	},
}
