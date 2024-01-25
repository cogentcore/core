// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"bytes"
	"fmt"
	"image"
	"log/slog"
	"slices"
	"strings"

	"cogentcore.org/core/abilities"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/enums"
	"cogentcore.org/core/events"
	"cogentcore.org/core/fi"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/gti"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keyfun"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/mimedata"
	"cogentcore.org/core/states"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
)

// TreeViewer is an interface for TreeView types
// providing access to the base TreeView and
// overridable method hooks for actions taken on the TreeView,
// including OnOpen, OnClose, etc.
type TreeViewer interface {
	gi.Widget

	// AsTreeView returns the base *TreeView for this node
	AsTreeView() *TreeView

	// CanOpen returns true if the node is able to open.
	// By default it checks HasChildren(), but could check other properties
	// to perform lazy building of the tree.
	CanOpen() bool

	// OnOpen is called when a node is opened.
	// The base version does nothing.
	OnOpen()

	// OnClose is called when a node is closed
	// The base version does nothing.
	OnClose()

	// OnDoubleClick is called when a node is double-clicked
	// The base version does ToggleClose
	OnDoubleClick(e events.Event)

	// UpdateBranchIcons is called during DoLayout to update branch icons
	// when everything should be configured, prior to rendering.
	UpdateBranchIcons()

	// Following are all tree editing functions:
	DeleteNode()
	Duplicate()
	AddChildNode()
	InsertBefore()
	InsertAfter()
	MimeData(md *mimedata.Mimes)
	Cut()
	Copy(reset bool)
	Paste()
	DragStart(e events.Event)
	DragDrop(e events.Event)
	DropFinalize(de *events.DragDrop)
	DropDeleteSource(e events.Event)
	MakePasteMenu(m *gi.Scene, md mimedata.Mimes, fun func())
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
	SyncNode ki.Ki `set:"-" copier:"-" json:"-" xml:"-"`

	// The text to display for the tree view item label, which automatically
	// defaults to the [ki.Node.Name] of the tree view node. It has no effect
	// if [TreeView.SyncNode] is non-nil.
	Text string

	// optional icon, displayed to the the left of the text label
	Icon icons.Icon

	// icon to use for an open (expanded) branch; defaults to [icons.KeyboardArrowDown]
	IconOpen icons.Icon `view:"show-name"`

	// icon to use for a closed (collapsed) branch; defaults to [icons.KeyboardArrowRight]
	IconClosed icons.Icon `view:"show-name"`

	// icon to use for a terminal node branch that has no children; defaults to [icons.Blank]
	IconLeaf icons.Icon `view:"show-name"`

	// amount to indent children relative to this node
	Indent units.Value `copier:"-" json:"-" xml:"-"`

	// depth for nodes be initialized as open (default 4).
	// Nodes beyond this depth will be initialized as closed.
	OpenDepth int `copier:"-" json:"-" xml:"-"`

	/////////////////////////////////////////
	// All fields below are computed

	// linear index of this node within the entire tree.
	// updated on full rebuilds and may sometimes be off,
	// but close enough for expected uses
	ViewIdx int `copier:"-" json:"-" xml:"-" edit:"-"`

	// size of just this node widget.
	// our alloc includes all of our children, but we only draw us.
	WidgetSize mat32.Vec2 `copier:"-" json:"-" xml:"-" edit:"-"`

	// The cached root of the view. It is automatically set and does not need to be
	// set by the end user.
	RootView *TreeView `copier:"-" json:"-" xml:"-" edit:"-"`

	// SelectedNodes holds the currently-selected nodes, on the
	// RootView node only.
	SelectedNodes []TreeViewer `copier:"-" json:"-" xml:"-" edit:"-"`

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
	tv.WidgetBase.OnInit()
	tv.HandleEvents()
	tv.SetStyles()
	tv.AddContextMenu(tv.ContextMenu)
}

func (tv *TreeView) OnAdd() {
	tv.WidgetBase.OnAdd()
	tv.Text = tv.Nm
	if ptv := AsTreeView(tv.Parent()); ptv != nil {
		tv.RootView = ptv.RootView
		tv.IconOpen = ptv.IconOpen
		tv.IconClosed = ptv.IconClosed
		tv.IconLeaf = ptv.IconLeaf
	} else {
		tv.RootView = tv
	}
}

func (tv *TreeView) SetStyles() {
	tv.IconOpen = icons.KeyboardArrowDown
	tv.IconClosed = icons.KeyboardArrowRight
	tv.IconLeaf = icons.Blank

	tvi := tv.This().(TreeViewer)
	tv.Style(func(s *styles.Style) {
		// our parts are draggable and droppable, not us ourself
		s.SetAbilities(true, abilities.Activatable, abilities.Focusable, abilities.Selectable, abilities.Hoverable)
		tv.Indent.Em(1)
		tv.OpenDepth = 4
		s.Border.Style.Set(styles.BorderNone)
		s.MaxBorder = s.Border
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
			s.Background = nil
		} else {
			s.Color = colors.Scheme.OnSurface
		}
	})
	tv.OnWidgetAdded(func(w gi.Widget) {
		switch w.PathFrom(tv) {
		case "parts":
			w.Style(func(s *styles.Style) {
				s.Cursor = cursors.Pointer
				s.SetAbilities(true, abilities.Activatable, abilities.Focusable, abilities.Selectable, abilities.Hoverable, abilities.DoubleClickable, abilities.Draggable, abilities.Droppable)
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
				tv.ApplyStyle()
				tv.SetNeedsRender(true)
				e.SetHandled()
			})
			w.On(events.MouseLeave, func(e events.Event) {
				if tv.This() == nil || tv.Is(ki.Deleted) {
					return
				}
				tv.SetState(false, states.Hovered)
				tv.ApplyStyle()
				tv.SetNeedsRender(true)
				e.SetHandled()
			})
			w.On(events.MouseDown, func(e events.Event) {
				if tv.This() == nil || tv.Is(ki.Deleted) {
					return
				}
				tv.SetState(true, states.Active)
				tv.ApplyStyle()
				tv.SetNeedsRender(true)
				e.SetHandled()
			})
			w.On(events.MouseUp, func(e events.Event) {
				if tv.This() == nil || tv.Is(ki.Deleted) {
					return
				}
				tv.SetState(false, states.Active)
				tv.ApplyStyle()
				tv.SetNeedsRender(true)
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
				tv.This().(TreeViewer).OnDoubleClick(e)
			})
			w.On(events.DragStart, func(e events.Event) {
				if tv.This() == nil || tv.Is(ki.Deleted) {
					return
				}
				tvi.DragStart(e)
			})
			w.On(events.DragEnter, func(e events.Event) {
				if tv.This() == nil || tv.Is(ki.Deleted) {
					return
				}
				tv.SetState(true, states.DragHovered)
				tv.ApplyStyle()
				tv.SetNeedsRender(true)
				e.SetHandled()
			})
			w.On(events.DragLeave, func(e events.Event) {
				if tv.This() == nil || tv.Is(ki.Deleted) {
					return
				}
				tv.SetState(false, states.DragHovered)
				tv.ApplyStyle()
				tv.SetNeedsRender(true)
				e.SetHandled()
			})
			w.On(events.Drop, func(e events.Event) {
				if tv.This() == nil || tv.Is(ki.Deleted) {
					return
				}
				tvi.DragDrop(e)
			})
			w.On(events.DropDeleteSource, func(e events.Event) {
				if tv.This() == nil || tv.Is(ki.Deleted) {
					return
				}
				tvi.DropDeleteSource(e)
			})
			// the context menu events will get sent to the parts, so it
			// needs to intercept them and send them up
			w.On(events.ContextMenu, func(e events.Event) {
				sels := tv.SelectedViews()
				if len(sels) == 0 {
					tv.SelectAction(e.SelectMode())
				}
				tv.ShowContextMenu(e)
			})
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
			sw.SetIcons(tv.IconOpen, tv.IconClosed, tv.IconLeaf)
			sw.Style(func(s *styles.Style) {
				// parent will handle our cursor
				s.Cursor = cursors.None
				s.Color = colors.Scheme.Primary.Base
				s.Margin.Zero()
				s.Padding.Zero()
				s.Min.X.Em(0.8)
				s.Min.Y.Em(0.8)
				s.Align.Self = styles.Center
				if !sw.StateIs(states.Indeterminate) {
					// we amplify any state layer we receiver so that it is clear
					// we are receiving it, not just our parent
					s.StateLayer *= 3
				} else {
					// no state layer for indeterminate because they are not interactive
					s.StateLayer = 0
				}
			})
			sw.OnClick(func(e events.Event) {
				if tv.This() == nil || tv.Is(ki.Deleted) {
					return
				}
				if sw.IsChecked() && !sw.StateIs(states.Indeterminate) {
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
				// TODO(kai): need to change these for clickable links in glide
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

	// TreeViewFlagSelectMode, when set on the Root node, determines whether keyboard movements
	// update selection or not.
	TreeViewFlagSelectMode

	// TreeViewInOpen is set in the Open method to prevent recursive opening for lazy-open nodes
	TreeViewInOpen
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

func (tv *TreeView) ConfigWidget() {
	parts := tv.NewParts()
	config := ki.Config{}
	config.Add(gi.SwitchType, "branch")
	if tv.Icon.IsSet() {
		config.Add(gi.IconType, "icon")
	}
	config.Add(gi.LabelType, "label")
	mods, updt := parts.ConfigChildren(config)
	if tv.This().(TreeViewer).CanOpen() {
		if wb, ok := tv.BranchPart(); ok {
			tv.SetBranchState()
			wb.Config()
		}
	}
	if tv.Icon.IsSet() {
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

func (tv *TreeView) StyleTreeView() {
	if !tv.HasChildren() {
		tv.SetClosed(true)
	}
	tv.Indent.ToDots(&tv.Styles.UnContext)
	tv.ApplyStyleWidget()
}

func (tv *TreeView) ApplyStyle() {
	tv.StyMu.Lock() // todo: needed??  maybe not.
	defer tv.StyMu.Unlock()

	tv.StyleTreeView()
}

func (tv *TreeView) UpdateBranchIcons() {
}

func (tv *TreeView) SetBranchState() {
	br, ok := tv.BranchPart()
	if !ok {
		return
	}
	switch {
	case !tv.This().(TreeViewer).CanOpen():
		br.SetState(true, states.Indeterminate)
	case tv.IsClosed():
		br.SetState(false, states.Indeterminate)
		br.SetState(false, states.Checked)
		br.SetNeedsRender(true)
	default:
		br.SetState(false, states.Indeterminate)
		br.SetState(true, states.Checked)
		br.SetNeedsRender(true)
	}
}

// TreeView is tricky for alloc because it is both a layout
// of its children but has to maintain its own bbox for its own widget.

func (tv *TreeView) SizeUp() {
	tv.WidgetBase.SizeUp()
	tv.WidgetSize = tv.Geom.Size.Actual.Total
	h := tv.WidgetSize.Y
	w := tv.WidgetSize.X

	if !tv.IsClosed() {
		// we layout children under us
		tv.WidgetKidsIter(func(i int, kwi gi.Widget, kwb *gi.WidgetBase) bool {
			kwi.SizeUp()
			h += kwb.Geom.Size.Actual.Total.Y
			w = max(w, tv.Indent.Dots+kwb.Geom.Size.Actual.Total.X)
			// fmt.Println(kwb, w, h)
			return ki.Continue
		})
	}
	sz := &tv.Geom.Size
	sz.Actual.Content = mat32.V2(w, h)
	sz.SetTotalFromContent(&sz.Actual)
	sz.Alloc = sz.Actual // need allocation to match!
	tv.WidgetSize.X = w  // stretch
}

func (tv *TreeView) SizeDown(iter int) bool {
	// note: key to not grab the whole allocation, as widget default does
	redo := tv.SizeDownParts(iter) // give our content to parts
	re := tv.SizeDownChildren(iter)
	return redo || re
}

func (tv *TreeView) Position() {
	rn := tv.RootView
	if rn == nil {
		slog.Error("giv.TreeView: RootView is nil", "in node:", tv)
		return
	}
	tv.SetBranchState()
	tv.This().(TreeViewer).UpdateBranchIcons()

	tv.Geom.Size.Actual.Total.X = rn.Geom.Size.Actual.Total.X - (tv.Geom.Pos.Total.X - rn.Geom.Pos.Total.X)
	tv.WidgetSize.X = tv.Geom.Size.Actual.Total.X

	tv.WidgetBase.Position()

	if !tv.IsClosed() {
		h := tv.WidgetSize.Y
		tv.WidgetKidsIter(func(i int, kwi gi.Widget, kwb *gi.WidgetBase) bool {
			kwb.Geom.RelPos.Y = h
			kwb.Geom.RelPos.X = tv.Indent.Dots
			h += kwb.Geom.Size.Actual.Total.Y
			kwi.Position()
			return ki.Continue
		})
	}
}

func (tv *TreeView) ScenePos() {
	sz := &tv.Geom.Size
	if sz.Actual.Total == tv.WidgetSize {
		sz.SetTotalFromContent(&sz.Actual) // restore after scrolling
	}
	tv.WidgetBase.ScenePos()
	tv.ScenePosChildren()
	tv.Geom.Size.Actual.Total = tv.WidgetSize // key: we revert to just ourselves
}

func (tv *TreeView) RenderNode() {
	pc, st := tv.RenderLock()
	defer tv.RenderUnlock()

	pabg := tv.ParentActualBackground()

	// must use workaround act values
	st.StateLayer = tv.actStateLayer
	if st.Is(states.Selected) {
		st.Background = colors.C(colors.Scheme.Select.Container)
	}
	tv.Styles.ComputeActualBackground(pabg)

	pc.DrawStdBox(st, tv.Geom.Pos.Total, tv.Geom.Size.Actual.Total, pabg)

	// after we are done rendering, we clear the values so they aren't inherited
	st.StateLayer = 0
	st.Background = nil
	tv.Styles.ComputeActualBackground(pabg)
}

func (tv *TreeView) Render() {
	if tv.PushBounds() {
		tv.RenderNode()
		if tv.Parts != nil {
			// we must copy from actual values in parent
			tv.Parts.Styles.StateLayer = tv.actStateLayer
			if tv.StateIs(states.Selected) {
				tv.Parts.Styles.Background = colors.C(colors.Scheme.Select.Container)
			}
			tv.RenderParts()
		}
		tv.PopBounds()
	}
	// we always have to render our kids b/c
	// we could be out of scope but they could be in!
	if !tv.IsClosed() {
		tv.RenderChildren()
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
func (tv *TreeView) SelectedViews() []TreeViewer {
	if tv.RootView == nil {
		return nil
	}
	if len(tv.RootView.SelectedNodes) == 0 {
		return tv.RootView.SelectedNodes
	}
	sels := tv.RootView.SelectedNodes
	return slices.Clone(sels)
}

// SetSelectedViews updates the selected views to given list
func (tv *TreeView) SetSelectedViews(sl []TreeViewer) {
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
		tv.ApplyStyle()
		sl := tv.SelectedViews()
		sl = append(sl, tv.This().(TreeViewer))
		tv.SetSelectedViews(sl)
		tv.SetNeedsRender(true)
	}
}

// Unselect unselects this node (if selected).
// Must use this method to update global selection list.
func (tv *TreeView) Unselect() {
	if tv.StateIs(states.Selected) {
		tv.SetSelected(false)
		tv.ApplyStyle()
		sl := tv.SelectedViews()
		sz := len(sl)
		for i := 0; i < sz; i++ {
			if sl[i] == tv {
				sl = append(sl[:i], sl[i+1:]...)
				break
			}
		}
		tv.SetSelectedViews(sl)
		tv.SetNeedsRender(true)
	}
}

// UnselectAll unselects all selected items in the view
func (tv *TreeView) UnselectAll() {
	if tv.Scene == nil {
		return
	}
	updt := tv.UpdateStart()
	sl := tv.SelectedViews()
	tv.SetSelectedViews(nil) // clear in advance
	for _, v := range sl {
		vt := v.AsTreeView()
		if vt == nil || vt.This() == nil || vt.Is(ki.Deleted) {
			continue
		}
		vt.SetSelected(false)
		v.ApplyStyle()
		vt.SetNeedsRender(true)
	}
	tv.UpdateEndRender(updt)
}

// SelectAll all items in view
func (tv *TreeView) SelectAll() {
	if tv.Scene == nil {
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
				tv.SetFocus()
				sel = true
			}
		} else {
			tv.UnselectAll()
			tv.Select()
			tv.SetFocus()
			sel = true
		}
	case events.ExtendContinuous:
		sl := tv.SelectedViews()
		if len(sl) == 0 {
			tv.Select()
			tv.SetFocus()
			sel = true
		} else {
			minIdx := -1
			maxIdx := 0
			for _, v := range sl {
				vn := v.AsTreeView()
				if minIdx < 0 {
					minIdx = vn.ViewIdx
				} else {
					minIdx = min(minIdx, vn.ViewIdx)
				}
				maxIdx = max(maxIdx, vn.ViewIdx)
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
			tv.SetFocus()
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
	// tv.SendChangeEvent(ctx)
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
		nn.SetFocus()
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
	myidx := tv.IndexInParent()
	if myidx < len(*tv.Par.Children())-1 {
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
	myidx := tv.IndexInParent()
	if myidx > 0 {
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
		nn.SetFocus()
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
		fnn.SetFocus()
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
		fnn.SetFocus()
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
	tv.RootView.SetFocus()
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
		fnn.SetFocus()
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
		tv.SetNeedsLayout(true)
	}
	tv.SetClosed(true)
	tv.SetBranchState()
	tv.This().(TreeViewer).OnClose()
	tv.SetKidsVisibility(true) // parent closed
	tv.UpdateEndLayout(updt)
}

// OnOpen is called when a node is opened.
// The base version does nothing.
func (tv *TreeView) OnOpen() {
}

func (tv *TreeView) OnDoubleClick(e events.Event) {
	if tv.HasChildren() {
		tv.ToggleClose()
	}
}

// CanOpen returns true if the node is able to open.
// By default it checks HasChildren(), but could check other properties
// to perform lazy building of the tree.
func (tv *TreeView) CanOpen() bool {
	return tv.HasChildren()
}

// Open opens the given node and updates the view accordingly
// (if it is not already opened)
// Calls OnOpen in TreeViewer interface for extensible actions.
func (tv *TreeView) Open() {
	if !tv.IsClosed() {
		return
	}
	if tv.Is(TreeViewInOpen) {
		return
	}
	tv.SetFlag(true, TreeViewInOpen)
	updt := tv.UpdateStart()
	if tv.This().(TreeViewer).CanOpen() {
		tv.SetNeedsLayout(true)
		tv.SetClosed(false)
		tv.SetBranchState()
		tv.SetKidsVisibility(false)
		tv.This().(TreeViewer).OnOpen()
	}
	tv.SetFlag(false, TreeViewInOpen)
	tv.UpdateEndLayout(updt)
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
		pos = e.WindowPos()
		return
	}
	pos.X = tv.Geom.TotalBBox.Min.X + int(tv.Indent.Dots)
	pos.Y = (tv.Geom.TotalBBox.Min.Y + tv.Geom.TotalBBox.Max.Y) / 2
	return
}

func (tv *TreeView) ContextMenuReadOnly(m *gi.Scene) {
	gi.NewButton(m).SetText("Copy").SetIcon(icons.ContentCopy).SetKey(keyfun.Copy).
		SetState(!tv.HasSelection(), states.Disabled).
		OnClick(func(e events.Event) {
			tv.Copy(true)
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

func (tv *TreeView) ContextMenu(m *gi.Scene) {
	if tv.IsReadOnly() {
		tv.ContextMenuReadOnly(m)
		return
	}
	tvi := tv.This().(TreeViewer)
	gi.NewButton(m).SetText("Add child").SetIcon(icons.Add).
		SetState(!tv.HasSelection(), states.Disabled).
		OnClick(func(e events.Event) {
			tvi.AddChildNode()
		})
	gi.NewButton(m).SetText("Insert before").SetIcon(icons.Add).
		SetState(!tv.HasSelection(), states.Disabled).
		OnClick(func(e events.Event) {
			tvi.InsertBefore()
		})
	gi.NewButton(m).SetText("Insert after").SetIcon(icons.Add).
		SetState(!tv.HasSelection(), states.Disabled).
		OnClick(func(e events.Event) {
			tvi.InsertAfter()
		})
	gi.NewButton(m).SetText("Duplicate").SetIcon(icons.ContentCopy).
		SetState(!tv.HasSelection(), states.Disabled).
		OnClick(func(e events.Event) {
			tvi.Duplicate()
		})
	gi.NewButton(m).SetText("Delete").SetIcon(icons.Delete).
		SetState(!tv.HasSelection(), states.Disabled).
		OnClick(func(e events.Event) {
			tvi.DeleteNode()
		})
	gi.NewSeparator(m)
	gi.NewButton(m).SetText("Copy").SetIcon(icons.ContentCopy).SetKey(keyfun.Copy).
		SetState(!tv.HasSelection(), states.Disabled).
		OnClick(func(e events.Event) {
			tvi.Copy(true)
		})
	gi.NewButton(m).SetText("Cut").SetIcon(icons.ContentCut).SetKey(keyfun.Cut).
		SetState(!tv.HasSelection(), states.Disabled).
		OnClick(func(e events.Event) {
			tvi.Cut()
		})
	pbt := gi.NewButton(m).SetText("Paste").SetIcon(icons.ContentPaste).SetKey(keyfun.Paste).
		OnClick(func(e events.Event) {
			tvi.Paste()
		})
	cb := tv.Scene.EventMgr.Clipboard()
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

// IsRoot returns true if given node is the root of the tree.
func (tv *TreeView) IsRoot(op string) bool {
	if tv.This() == tv.RootView.This() {
		if op != "" {
			gi.MessageSnackbar(tv, fmt.Sprintf("Cannot %v the root of the tree", op))
		}
		return true
	}
	return false
}

////////////////////////////////////////////////////////////
//    Copy / Cut / Paste

// MimeData adds mimedata for this node: a text/plain of the Path.
func (tv *TreeView) MimeData(md *mimedata.Mimes) {
	if tv.SyncNode != nil {
		tv.MimeDataSync(md)
		return
	}
	*md = append(*md, mimedata.NewTextData(tv.PathFrom(tv.RootView)))
	var buf bytes.Buffer
	err := ki.WriteNewJSON(tv.This(), &buf)
	if err == nil {
		*md = append(*md, &mimedata.Data{Type: fi.DataJson, Data: buf.Bytes()})
	} else {
		slog.Error("giv.TreeView MimeData Write JSON error", "err", err)
	}
}

// NodesFromMimeData returns a slice of Ki nodes for
// the TreeView nodes and paths from mime data.
func (tv *TreeView) NodesFromMimeData(md mimedata.Mimes) (ki.Slice, []string) {
	ni := len(md) / 2
	sl := make(ki.Slice, 0, ni)
	pl := make([]string, 0, ni)
	for _, d := range md {
		if d.Type == fi.DataJson {
			nki, err := ki.ReadNewJSON(bytes.NewReader(d.Data))
			if err == nil {
				sl = append(sl, nki)
			} else {
				slog.Error("giv.TreeView NodesFromMimeData: JSON load error:", err)
			}
		} else if d.Type == fi.TextPlain { // paths
			pl = append(pl, string(d.Data))
		}
	}
	return sl, pl
}

// Copy copies to goosi.Clipboard, optionally resetting the selection.
func (tv *TreeView) Copy(reset bool) { //gti:add
	sels := tv.SelectedViews()
	nitms := max(1, len(sels))
	md := make(mimedata.Mimes, 0, 2*nitms)
	tv.This().(TreeViewer).MimeData(&md) // source is always first..
	if nitms > 1 {
		for _, sn := range sels {
			if sn.This() != tv.This() {
				sn.MimeData(&md)
			}
		}
	}
	tv.Clipboard().Write(md)
	if reset {
		tv.UnselectAll()
	}
}

// Cut copies to goosi.Clipboard and deletes selected items.
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
func (tv *TreeView) Paste() { //gti:add
	md := tv.Clipboard().Read([]string{fi.DataJson})
	if md != nil {
		tv.PasteMenu(md)
	}
}

// PasteMenu performs a paste from the clipboard using given data,
// by popping up a menu to determine what specifically to do.
func (tv *TreeView) PasteMenu(md mimedata.Mimes) {
	tv.UnselectAll()
	mf := func(m *gi.Scene) {
		tv.This().(TreeViewer).MakePasteMenu(m, md, nil)
	}
	pos := tv.ContextMenuPos(nil)
	gi.NewMenu(mf, tv.This().(gi.Widget), pos).Run()
}

// MakePasteMenu makes the menu of options for paste events
// optional function is typically the DropFinalize but could also be other actions
// to take after each optional action.
func (tv *TreeView) MakePasteMenu(m *gi.Scene, md mimedata.Mimes, fun func()) {
	gi.NewButton(m).SetText("Assign To").OnClick(func(e events.Event) {
		tv.PasteAssign(md)
		if fun != nil {
			fun()
		}
	})
	gi.NewButton(m).SetText("Add to Children").OnClick(func(e events.Event) {
		tv.PasteChildren(md, events.DropCopy)
		if fun != nil {
			fun()
		}
	})
	if !tv.IsRoot("") && tv.RootView.This() != tv.This() {
		gi.NewButton(m).SetText("Insert Before").OnClick(func(e events.Event) {
			tv.PasteBefore(md, events.DropCopy)
			if fun != nil {
				fun()
			}
		})
		gi.NewButton(m).SetText("Insert After").OnClick(func(e events.Event) {
			tv.PasteAfter(md, events.DropCopy)
			if fun != nil {
				fun()
			}
		})
	}
	gi.NewButton(m).SetText("Cancel")
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
	tv.SetScene(tv.Scene)     // ensure children have scene
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

// TreeViewTempMovedTag is a kind of hack to prevent moved items from being deleted, using DND
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

	myidx := tv.IndexInParent()
	if myidx < 0 {
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
		_, nwb := gi.AsWidget(ns.This())
		ntv := AsTreeView(ns.This())
		ntv.RootView = tv.RootView
		nwb.SetScene(tv.Scene)
		nwb.Update() // incl children
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
		_, nwb := gi.AsWidget(ns.This())
		ntv := AsTreeView(ns.This())
		ntv.RootView = tv.RootView
		nwb.SetScene(tv.Scene)
	}
	tv.Update()
	tv.Open()
	tv.TreeViewChanged(nil)
	tv.UpdateEndLayout(updt)
}

//////////////////////////////////////////////////////////////////////////////
//    Drag-n-Drop

// DragStart starts a drag-n-drop on this node -- it includes any other
// selected nodes as well, each as additional records in mimedata.
func (tv *TreeView) DragStart(e events.Event) {
	sels := tv.SelectedViews()
	nitms := max(1, len(sels))
	md := make(mimedata.Mimes, 0, 2*nitms)
	tv.This().(TreeViewer).MimeData(&md) // source is always first..
	if nitms > 1 {
		for _, sn := range sels {
			if sn.This() != tv.This() {
				sn.MimeData(&md)
			}
		}
	}
	tv.Scene.EventMgr.DragStart(tv.This().(gi.Widget), md, e)
}

// DropExternal is not handled by base case but could be in derived
func (tv *TreeView) DropExternal(md mimedata.Mimes, mod events.DropMods) {
	// todo: not yet implemented
}

// DragDrop handles drag drop event
func (tv *TreeView) DragDrop(e events.Event) {
	// todo: some kind of validation for source
	tvi := tv.This().(TreeViewer)
	tv.UnselectAll()
	de := e.(*events.DragDrop)
	stv := AsTreeView(de.Source.(gi.Widget).Parent()) // note: source is actually .parts!
	if stv != nil {
		stv.SetState(false, states.Active, states.Selected, states.Hovered, states.DragHovered)
		stv.Parts.SetState(false, states.Active, states.Selected, states.Hovered, states.DragHovered)
		stv.ApplyStyleTree()
		stv.SetNeedsRender(true)
	}
	md := de.Data.(mimedata.Mimes)
	mf := func(m *gi.Scene) {
		tv.Scene.EventMgr.DragMenuAddModLabel(m, de.DropMod)
		tvi.MakePasteMenu(m, md, func() {
			tvi.DropFinalize(de)
		})
	}
	pos := tv.ContextMenuPos(nil)
	gi.NewMenu(mf, tv.This().(gi.Widget), pos).Run()
}

// DropFinalize is called to finalize Drop actions on the Source node.
// Only relevant for DropMod == DropMove.
func (tv *TreeView) DropFinalize(de *events.DragDrop) {
	tv.UnselectAll()
	tv.Scene.EventMgr.DropFinalize(de) // sends DropDeleteSource to Source
}

// DropDeleteSource handles delete source event for DropMove case
func (tv *TreeView) DropDeleteSource(e events.Event) {
	de := e.(*events.DragDrop)
	tv.UnselectAll()
	if tv.SyncNode != nil {
		tv.DropDeleteSourceSync(de)
		return
	}
	md := de.Data.(mimedata.Mimes)
	root := tv.RootView
	for _, d := range md {
		if d.Type != fi.TextPlain { // link
			continue
		}
		path := string(d.Data)
		sn := root.FindPath(path)
		if sn != nil {
			sn.Delete(true)
		}
		sn = root.FindPath(path + TreeViewTempMovedTag)
		if sn != nil {
			psplt := strings.Split(path, "/")
			orgnm := psplt[len(psplt)-1]
			sn.SetName(orgnm)
			_, swb := gi.AsWidget(sn)
			swb.SetNeedsRender(true)
		}
	}
}

////////////////////////////////////////////////////
// 	Event Handlers

func (tv *TreeView) TreeViewParent() *TreeView {
	if tv.Par == nil {
		return nil
	}
	return AsTreeView(tv.Par)
}

func (tv *TreeView) HandleEvents() {
	tv.HandleMouse()
	tv.HandleKeys()
}

func (tv *TreeView) HandleKeys() {
	tv.On(events.KeyChord, func(e events.Event) {
		if gi.DebugSettings.KeyEventTrace {
			fmt.Printf("TreeView KeyInput: %v\n", tv.Path())
		}
		kf := keyfun.Of(e.KeyChord())
		selMode := events.SelectModeBits(e.Modifiers())

		if selMode == events.SelectOne {
			if tv.SelectMode() {
				selMode = events.ExtendContinuous
			}
		}

		tvi := tv.This().(TreeViewer)

		// first all the keys that work for ReadOnly and active
		switch kf {
		case keyfun.CancelSelect:
			tv.UnselectAll()
			tv.SetSelectMode(false)
			e.SetHandled()
		case keyfun.MoveRight:
			tv.Open()
			e.SetHandled()
		case keyfun.MoveLeft:
			tv.Close()
			e.SetHandled()
		case keyfun.MoveDown:
			tv.MoveDownAction(selMode)
			e.SetHandled()
		case keyfun.MoveUp:
			tv.MoveUpAction(selMode)
			e.SetHandled()
		case keyfun.PageUp:
			tv.MovePageUpAction(selMode)
			e.SetHandled()
		case keyfun.PageDown:
			tv.MovePageDownAction(selMode)
			e.SetHandled()
		case keyfun.Home:
			tv.MoveHomeAction(selMode)
			e.SetHandled()
		case keyfun.End:
			tv.MoveEndAction(selMode)
			e.SetHandled()
		case keyfun.SelectMode:
			tv.SelectModeToggle()
			e.SetHandled()
		case keyfun.SelectAll:
			tv.SelectAll()
			e.SetHandled()
		case keyfun.Enter:
			tv.ToggleClose()
			e.SetHandled()
		case keyfun.Copy:
			tvi.Copy(true)
			e.SetHandled()
		}
		if !tv.RootIsReadOnly() && !e.IsHandled() {
			switch kf {
			case keyfun.Delete:
				tvi.DeleteNode()
				e.SetHandled()
			case keyfun.Duplicate:
				tvi.Duplicate()
				e.SetHandled()
			case keyfun.Insert:
				tvi.InsertBefore()
				e.SetHandled()
			case keyfun.InsertAfter:
				tvi.InsertAfter()
				e.SetHandled()
			case keyfun.Cut:
				tvi.Cut()
				e.SetHandled()
			case keyfun.Paste:
				tvi.Paste()
				e.SetHandled()
			}
		}
	})
}

func (tv *TreeView) HandleMouse() {
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
	tv.On(events.MouseLeave, func(e events.Event) {
		e.SetHandled()
	})
	tv.On(events.MouseDown, func(e events.Event) {
		e.SetHandled()
	})
	tv.OnClick(func(e events.Event) {
		e.SetHandled()
	})
	tv.On(events.DragStart, func(e events.Event) {
		e.SetHandled()
	})
	tv.On(events.DragEnter, func(e events.Event) {
		e.SetHandled()
	})
	tv.On(events.DragLeave, func(e events.Event) {
		e.SetHandled()
	})
	tv.On(events.Drop, func(e events.Event) {
		e.SetHandled()
	})
	tv.On(events.DropDeleteSource, func(e events.Event) {
		e.SetHandled()
	})
}
