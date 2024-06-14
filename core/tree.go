// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"bytes"
	"fmt"
	"image"
	"log/slog"
	"slices"
	"strings"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/base/iox/jsonx"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
)

// Treer is an interface for [Tree] types
// providing access to the base [Tree] and
// overridable method hooks for actions taken on the [Tree],
// including OnOpen, OnClose, etc.
type Treer interface {
	Widget

	// AsTree returns the base [Tree] for this node.
	AsCoreTree() *Tree

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
	MakePasteMenu(m *Scene, md mimedata.Mimes, fun func())
}

// AsTree returns the given value as a value of type Tree if the type
// of the given value embeds Tree, or nil otherwise.
func AsTree(n tree.Node) *Tree {
	if t, ok := n.(Treer); ok {
		return t.AsCoreTree()
	}
	return nil
}

// note: see treesync.go for all the SyncNode mode specific
// functions.

// Tree provides a graphical representation of a tree structure,
// providing full navigation and manipulation abilities.
//
// It does not handle layout by itself, so if you want it to scroll
// separately from the rest of the surrounding context, use [NewTreeFrame].
//
// If the SyncNode field is non-nil, typically via
// SyncRootNode method, then the Tree mirrors another
// tree structure, and tree editing functions apply to
// the source tree first, and then to the Tree by sync.
//
// Otherwise, data can be directly encoded in a Tree
// derived type, to represent any kind of tree structure
// and associated data.
//
// Standard [events.Event]s are sent to any listeners, including
// Select, Change, and DoubleClick. The selected nodes
// are in the root SelectedNodes list.
type Tree struct {
	WidgetBase

	// If non-nil, the [tree.Node] that this widget is viewing in the tree (the source)
	SyncNode tree.Node `set:"-" copier:"-" json:"-" xml:"-"`

	// The text to display for the tree item label, which automatically
	// defaults to the [tree.Node.Name] of the tree node. It has no effect
	// if [Tree.SyncNode] is non-nil.
	Text string

	// optional icon, displayed to the the left of the text label
	Icon icons.Icon

	// icon to use for an open (expanded) branch; defaults to [icons.KeyboardArrowDown]
	IconOpen icons.Icon `display:"show-name"`

	// icon to use for a closed (collapsed) branch; defaults to [icons.KeyboardArrowRight]
	IconClosed icons.Icon `display:"show-name"`

	// icon to use for a terminal node branch that has no children; defaults to [icons.Blank]
	IconLeaf icons.Icon `display:"show-name"`

	// amount to indent children relative to this node
	Indent units.Value `copier:"-" json:"-" xml:"-"`

	// OpenDepth is the depth for nodes be initialized as open (default 4).
	// Nodes beyond this depth will be initialized as closed.
	OpenDepth int `copier:"-" json:"-" xml:"-"`

	// Closed is whether this tree node is currently toggled closed (children not visible).
	Closed bool

	// SelectMode, when set on the root node, determines whether keyboard movements should update selection.
	SelectMode bool

	// Computed fields:

	// linear index of this node within the entire tree.
	// updated on full rebuilds and may sometimes be off,
	// but close enough for expected uses
	viewIndex int

	// size of just this node widget.
	// our alloc includes all of our children, but we only draw us.
	widgetSize math32.Vector2

	// The cached root of the view. It is automatically set and does not need to be
	// set by the end user.
	RootView *Tree `copier:"-" json:"-" xml:"-" edit:"-"`

	// SelectedNodes holds the currently selected nodes, on the
	// RootView node only.
	SelectedNodes []Treer `copier:"-" json:"-" xml:"-" edit:"-"`

	// actStateLayer is the actual state layer of the tree, which
	// should be used when rendering it and its parts (but not its children).
	// the reason that it exists is so that the children of the tree
	// (other trees) do not inherit its stateful background color, as
	// that does not look good.
	actStateLayer float32

	// inOpen is set in the Open method to prevent recursive opening for lazy-open nodes.
	inOpen bool
}

// NewTreeFrame adds a new [Tree] to a new frame with the given
// optional parent that ensures that the tree scrolls
// separately from the surrounding context.
func NewTreeFrame(parent ...tree.Node) *Tree {
	fr := NewFrame(parent...).Styler(func(s *styles.Style) {
		s.Overflow.Set(styles.OverflowAuto)
	})
	return NewTree(fr)
}

// AsCoreTree satisfies the [Treer] interface.
func (tr *Tree) AsCoreTree() *Tree {
	return tr
}

func (tr *Tree) BaseType() *types.Type {
	return tr.NodeType()
}

// RootSetViewIndex sets the RootView and ViewIndex for all nodes.
// This must be called from the root node after
// construction or any modification to the tree.
// Returns the total number of leaves in the tree.
func (tr *Tree) RootSetViewIndex() int {
	idx := 0
	tr.WidgetWalkDown(func(wi Widget, wb *WidgetBase) bool {
		tvn := AsTree(wi)
		if tvn != nil {
			tvn.viewIndex = idx
			tvn.RootView = tr
			// fmt.Println(idx, tvn, "root:", tv, &tv)
			idx++
		}
		return tree.Continue
	})
	return idx
}

func (tr *Tree) Init() {
	tr.WidgetBase.Init()
	tr.AddContextMenu(tr.ContextMenu)
	tr.IconOpen = icons.KeyboardArrowDown
	tr.IconClosed = icons.KeyboardArrowRight
	tr.IconLeaf = icons.Blank
	tr.OpenDepth = 4
	tr.Styler(func(s *styles.Style) {
		// our parts are draggable and droppable, not us ourself
		s.SetAbilities(true, abilities.Activatable, abilities.Focusable, abilities.Selectable, abilities.Hoverable)
		tr.Indent.Em(1)
		s.Border.Style.Set(styles.BorderNone)
		s.Border.Radius = styles.BorderRadiusFull
		s.MaxBorder = s.Border
		// s.Border.Width.Left.SetDp(1)
		// s.Border.Color.Left = colors.Scheme.OutlineVariant
		s.Margin.Zero()
		s.Padding.Set(units.Dp(4))
		s.Text.Align = styles.Start

		// need to copy over to actual and then clear styles one
		if s.Is(states.Selected) {
			// render handles manually, similar to with actStateLayer
			s.Background = nil
		} else {
			s.Color = colors.C(colors.Scheme.OnSurface)
		}
	})
	tr.FinalStyler(func(s *styles.Style) {
		tr.actStateLayer = s.StateLayer
		s.StateLayer = 0
	})

	// We let the parts handle our state
	// so that we only get it when we are doing
	// something with this tree specifically,
	// not with any of our children (see OnChildAdded).
	// we only need to handle the starting ones here,
	// as the other ones will just set the state to
	// false, which it already is.
	tr.On(events.MouseEnter, func(e events.Event) { e.SetHandled() })
	tr.On(events.MouseLeave, func(e events.Event) { e.SetHandled() })
	tr.On(events.MouseDown, func(e events.Event) { e.SetHandled() })
	tr.OnClick(func(e events.Event) { e.SetHandled() })
	tr.On(events.DragStart, func(e events.Event) { e.SetHandled() })
	tr.On(events.DragEnter, func(e events.Event) { e.SetHandled() })
	tr.On(events.DragLeave, func(e events.Event) { e.SetHandled() })
	tr.On(events.Drop, func(e events.Event) { e.SetHandled() })
	tr.On(events.DropDeleteSource, func(e events.Event) { e.SetHandled() })
	tr.On(events.KeyChord, func(e events.Event) {
		kf := keymap.Of(e.KeyChord())
		selMode := events.SelectModeBits(e.Modifiers())
		if DebugSettings.KeyEventTrace {
			slog.Info("Tree KeyInput", "widget", tr, "keyFunction", kf, "selMode", selMode)
		}

		if selMode == events.SelectOne {
			if tr.SelectMode {
				selMode = events.ExtendContinuous
			}
		}

		tvi := tr.This.(Treer)

		// first all the keys that work for ReadOnly and active
		switch kf {
		case keymap.CancelSelect:
			tr.UnselectAll()
			tr.SetSelectMode(false)
			e.SetHandled()
		case keymap.MoveRight:
			tr.Open()
			e.SetHandled()
		case keymap.MoveLeft:
			tr.Close()
			e.SetHandled()
		case keymap.MoveDown:
			tr.MoveDownAction(selMode)
			e.SetHandled()
		case keymap.MoveUp:
			tr.MoveUpAction(selMode)
			e.SetHandled()
		case keymap.PageUp:
			tr.MovePageUpAction(selMode)
			e.SetHandled()
		case keymap.PageDown:
			tr.MovePageDownAction(selMode)
			e.SetHandled()
		case keymap.Home:
			tr.MoveHomeAction(selMode)
			e.SetHandled()
		case keymap.End:
			tr.MoveEndAction(selMode)
			e.SetHandled()
		case keymap.SelectMode:
			tr.SelectMode = !tr.SelectMode
			e.SetHandled()
		case keymap.SelectAll:
			tr.SelectAll()
			e.SetHandled()
		case keymap.Enter:
			tr.ToggleClose()
			e.SetHandled()
		case keymap.Copy:
			tvi.Copy(true)
			e.SetHandled()
		}
		if !tr.RootIsReadOnly() && !e.IsHandled() {
			switch kf {
			case keymap.Delete:
				tvi.DeleteNode()
				e.SetHandled()
			case keymap.Duplicate:
				tvi.Duplicate()
				e.SetHandled()
			case keymap.Insert:
				tvi.InsertBefore()
				e.SetHandled()
			case keymap.InsertAfter:
				tvi.InsertAfter()
				e.SetHandled()
			case keymap.Cut:
				tvi.Cut()
				e.SetHandled()
			case keymap.Paste:
				tvi.Paste()
				e.SetHandled()
			}
		}
	})

	AddChildAt(tr, "parts", func(w *Frame) {
		InitParts(w)
		tvi := tr.This.(Treer)
		w.Styler(func(s *styles.Style) {
			s.Cursor = cursors.Pointer
			s.SetAbilities(true, abilities.Activatable, abilities.Focusable, abilities.Selectable, abilities.Hoverable, abilities.DoubleClickable)
			s.SetAbilities(!tr.IsReadOnly() && !tr.RootIsReadOnly(), abilities.Draggable, abilities.Droppable)
			s.Gap.X.Em(0.1)
			s.Padding.Zero()

			// we manually inherit our state layer from the tree state
			// layer so that the parts get it but not the other trees
			s.StateLayer = tr.actStateLayer
		})
		w.AsWidget().FinalStyler(func(s *styles.Style) {
			s.Grow.Set(1, 0)
		})
		// we let the parts handle our state
		// so that we only get it when we are doing
		// something with this tree specifically,
		// not with any of our children (see HandleTreeMouse)
		w.On(events.MouseEnter, func(e events.Event) {
			tr.SetState(true, states.Hovered)
			tr.Style()
			tr.NeedsRender()
			e.SetHandled()
		})
		w.On(events.MouseLeave, func(e events.Event) {
			tr.SetState(false, states.Hovered)
			tr.Style()
			tr.NeedsRender()
			e.SetHandled()
		})
		w.On(events.MouseDown, func(e events.Event) {
			tr.SetState(true, states.Active)
			tr.Style()
			tr.NeedsRender()
			e.SetHandled()
		})
		w.On(events.MouseUp, func(e events.Event) {
			tr.SetState(false, states.Active)
			tr.Style()
			tr.NeedsRender()
			e.SetHandled()
		})
		w.OnClick(func(e events.Event) {
			tr.SelectAction(e.SelectMode())
			e.SetHandled()
		})
		w.AsWidget().OnDoubleClick(func(e events.Event) {
			tr.This.(Treer).OnDoubleClick(e)
		})
		w.On(events.DragStart, func(e events.Event) {
			tvi.DragStart(e)
		})
		w.On(events.DragEnter, func(e events.Event) {
			tr.SetState(true, states.DragHovered)
			tr.Style()
			tr.NeedsRender()
			e.SetHandled()
		})
		w.On(events.DragLeave, func(e events.Event) {
			tr.SetState(false, states.DragHovered)
			tr.Style()
			tr.NeedsRender()
			e.SetHandled()
		})
		w.On(events.Drop, func(e events.Event) {
			tvi.DragDrop(e)
		})
		w.On(events.DropDeleteSource, func(e events.Event) {
			tvi.DropDeleteSource(e)
		})
		// the context menu events will get sent to the parts, so it
		// needs to intercept them and send them up
		w.On(events.ContextMenu, func(e events.Event) {
			sels := tr.SelectedViews()
			if len(sels) == 0 {
				tr.SelectAction(e.SelectMode())
			}
			tr.ShowContextMenu(e)
		})
		AddChildAt(w, "branch", func(w *Switch) {
			w.SetType(SwitchCheckbox)
			w.SetIcons(tr.IconOpen, tr.IconClosed, tr.IconLeaf)
			w.Styler(func(s *styles.Style) {
				s.SetAbilities(false, abilities.Focusable)
				// parent will handle our cursor
				s.Cursor = cursors.None
				s.Color = colors.C(colors.Scheme.Primary.Base)
				s.Padding.Zero()
				s.Align.Self = styles.Center
				if !w.StateIs(states.Indeterminate) {
					// we amplify any state layer we receiver so that it is clear
					// we are receiving it, not just our parent
					s.StateLayer *= 3
				} else {
					// no state layer for indeterminate because they are not interactive
					s.StateLayer = 0
				}
			})
			w.OnClick(func(e events.Event) {
				if w.IsChecked() && !w.StateIs(states.Indeterminate) {
					if !tr.Closed {
						tr.Close()
					}
				} else {
					if tr.Closed {
						tr.Open()
					}
				}
			})
			w.Updater(func() {
				if tr.This.(Treer).CanOpen() {
					tr.SetBranchState()
				}
			})
		})
		w.Maker(func(p *Plan) {
			if tr.Icon.IsSet() {
				AddAt(p, "icon", func(w *Icon) {
					w.Styler(func(s *styles.Style) {
						s.Font.Size.Dp(16)
					})
					w.Updater(func() {
						w.SetIcon(tr.Icon)
					})
				})
			}
		})
		AddChildAt(w, "text", func(w *Text) {
			w.Styler(func(s *styles.Style) {
				s.SetNonSelectable()
				s.SetTextWrap(false)
				s.Min.X.Ch(16)
				s.Min.Y.Em(1.2)
			})
			w.Updater(func() {
				w.SetText(tr.Label())
			})
		})
	})
}

func (tr *Tree) OnAdd() {
	tr.WidgetBase.OnAdd()
	tr.Text = tr.Name
	if ptv := AsTree(tr.Parent); ptv != nil {
		tr.RootView = ptv.RootView
		tr.IconOpen = ptv.IconOpen
		tr.IconClosed = ptv.IconClosed
		tr.IconLeaf = ptv.IconLeaf
	} else {
		// fmt.Println("set root to:", tv, &tv)
		tr.RootView = tr
	}
}

// RootIsReadOnly returns the ReadOnly status of the root node,
// which is what controls the functional inactivity of the tree
// if individual nodes are ReadOnly that only affects display typically.
func (tr *Tree) RootIsReadOnly() bool {
	if tr.RootView == nil {
		return true
	}
	return tr.RootView.IsReadOnly()
}

// qt calls the open / close thing a "branch"
// http://doc.qt.io/qt-5/stylesheet-examples.html#customizing-qtree

// Branch returns the branch widget in parts, if it exists
func (tr *Tree) Branch() (*Switch, bool) {
	if tr.Parts == nil {
		return nil, false
	}
	if icc := tr.Parts.ChildByName("branch", 0); icc != nil {
		return icc.(*Switch), true
	}
	return nil, false
}

func (tr *Tree) Style() {
	if !tr.HasChildren() {
		tr.SetClosed(true)
	}
	tr.Indent.ToDots(&tr.Styles.UnitContext)
	tr.WidgetBase.Style()
}

func (tr *Tree) UpdateBranchIcons() {}

func (tr *Tree) SetBranchState() {
	br, ok := tr.Branch()
	if !ok {
		return
	}
	switch {
	case !tr.This.(Treer).CanOpen():
		br.SetState(true, states.Indeterminate)
	case tr.Closed:
		br.SetState(false, states.Indeterminate)
		br.SetState(false, states.Checked)
		br.NeedsRender()
	default:
		br.SetState(false, states.Indeterminate)
		br.SetState(true, states.Checked)
		br.NeedsRender()
	}
}

// Tree is tricky for alloc because it is both a layout
// of its children but has to maintain its own bbox for its own widget.

func (tr *Tree) SizeUp() {
	tr.WidgetBase.SizeUp()
	tr.widgetSize = tr.Geom.Size.Actual.Total
	h := tr.widgetSize.Y
	w := tr.widgetSize.X

	if !tr.Closed {
		// we layout children under us
		tr.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
			kwi.SizeUp()
			h += kwb.Geom.Size.Actual.Total.Y
			kw := kwb.Geom.Size.Actual.Total.X
			if math32.IsNaN(kw) { // somehow getting a nan
				slog.Error("Tree, node width is NaN", "node:", kwb)
			} else {
				w = max(w, tr.Indent.Dots+kw)
			}
			// fmt.Println(kwb, w, h)
			return tree.Continue
		})
	}
	sz := &tr.Geom.Size
	sz.Actual.Content = math32.Vec2(w, h)
	sz.SetTotalFromContent(&sz.Actual)
	sz.Alloc = sz.Actual // need allocation to match!
	tr.widgetSize.X = w  // stretch
}

func (tr *Tree) SizeDown(iter int) bool {
	// note: key to not grab the whole allocation, as widget default does
	redo := tr.SizeDownParts(iter) // give our content to parts
	re := tr.SizeDownChildren(iter)
	return redo || re
}

func (tr *Tree) Position() {
	rn := tr.RootView
	if rn == nil {
		slog.Error("core.Tree: RootView is nil", "in node:", tr)
		return
	}
	tr.SetBranchState()
	tr.This.(Treer).UpdateBranchIcons()

	tr.Geom.Size.Actual.Total.X = rn.Geom.Size.Actual.Total.X - (tr.Geom.Pos.Total.X - rn.Geom.Pos.Total.X)
	tr.widgetSize.X = tr.Geom.Size.Actual.Total.X

	tr.WidgetBase.Position()

	if !tr.Closed {
		h := tr.widgetSize.Y
		tr.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
			kwb.Geom.RelPos.Y = h
			kwb.Geom.RelPos.X = tr.Indent.Dots
			h += kwb.Geom.Size.Actual.Total.Y
			kwi.Position()
			return tree.Continue
		})
	}
}

func (tr *Tree) ScenePos() {
	sz := &tr.Geom.Size
	if sz.Actual.Total == tr.widgetSize {
		sz.SetTotalFromContent(&sz.Actual) // restore after scrolling
	}
	tr.WidgetBase.ScenePos()
	tr.ScenePosChildren()
	tr.Geom.Size.Actual.Total = tr.widgetSize // key: we revert to just ourselves
}

func (tr *Tree) Render() {
	pc := &tr.Scene.PaintContext
	st := &tr.Styles

	pabg := tr.ParentActualBackground()

	// must use workaround act values
	st.StateLayer = tr.actStateLayer
	if st.Is(states.Selected) {
		st.Background = colors.C(colors.Scheme.Select.Container)
	}
	tr.Styles.ComputeActualBackground(pabg)

	pc.DrawStandardBox(st, tr.Geom.Pos.Total, tr.Geom.Size.Actual.Total, pabg)

	// after we are done rendering, we clear the values so they aren't inherited
	st.StateLayer = 0
	st.Background = nil
	tr.Styles.ComputeActualBackground(pabg)
}

func (tr *Tree) RenderWidget() {
	if tr.PushBounds() {
		tr.Render()
		if tr.Parts != nil {
			// we must copy from actual values in parent
			tr.Parts.Styles.StateLayer = tr.actStateLayer
			if tr.StateIs(states.Selected) {
				tr.Parts.Styles.Background = colors.C(colors.Scheme.Select.Container)
			}
			tr.RenderParts()
		}
		tr.PopBounds()
	}
	// we always have to render our kids b/c
	// we could be out of scope but they could be in!
	if !tr.Closed {
		tr.RenderChildren()
	}
}

//////////////////////////////////////////////////////////////////////////////
//    Selection

// SelectedViews returns a slice of the currently selected
// Trees within the entire tree, using a list maintained
// by the root node
func (tr *Tree) SelectedViews() []Treer {
	if tr.RootView == nil {
		return nil
	}
	if len(tr.RootView.SelectedNodes) == 0 {
		return tr.RootView.SelectedNodes
	}
	sels := tr.RootView.SelectedNodes
	return slices.Clone(sels)
}

// SetSelectedViews updates the selected views to given list
func (tr *Tree) SetSelectedViews(sl []Treer) {
	if tr.RootView != nil {
		tr.RootView.SelectedNodes = sl
	}
}

// HasSelection returns true if there are currently selected items
func (tr *Tree) HasSelection() bool {
	return len(tr.SelectedViews()) > 0
}

// Select selects this node (if not already selected).
// Must use this method to update global selection list
func (tr *Tree) Select() {
	if !tr.StateIs(states.Selected) {
		tr.SetSelected(true)
		tr.Style()
		sl := tr.SelectedViews()
		sl = append(sl, tr.This.(Treer))
		tr.SetSelectedViews(sl)
		tr.NeedsRender()
	}
}

// Unselect unselects this node (if selected).
// Must use this method to update global selection list.
func (tr *Tree) Unselect() {
	if tr.StateIs(states.Selected) {
		tr.SetSelected(false)
		tr.Style()
		sl := tr.SelectedViews()
		sz := len(sl)
		for i := 0; i < sz; i++ {
			if sl[i] == tr {
				sl = append(sl[:i], sl[i+1:]...)
				break
			}
		}
		tr.SetSelectedViews(sl)
		tr.NeedsRender()
	}
}

// UnselectAll unselects all selected items in the view
func (tr *Tree) UnselectAll() {
	if tr.Scene == nil {
		return
	}
	sl := tr.SelectedViews()
	tr.SetSelectedViews(nil) // clear in advance
	for _, v := range sl {
		vt := v.AsCoreTree()
		if vt == nil || vt.This == nil {
			continue
		}
		vt.SetSelected(false)
		v.Style()
		vt.NeedsRender()
	}
	tr.NeedsRender()
}

// SelectAll all items in view
func (tr *Tree) SelectAll() {
	if tr.Scene == nil {
		return
	}
	tr.UnselectAll()
	nn := tr.RootView
	nn.Select()
	for nn != nil {
		nn = nn.MoveDown(events.SelectQuiet)
	}
	tr.NeedsRender()
}

// SelectUpdate updates selection to include this node,
// using selectmode from mouse event (ExtendContinuous, ExtendOne).
// Returns true if this node selected
func (tr *Tree) SelectUpdate(mode events.SelectModes) bool {
	if mode == events.NoSelect {
		return false
	}
	sel := false
	switch mode {
	case events.SelectOne:
		if tr.StateIs(states.Selected) {
			sl := tr.SelectedViews()
			if len(sl) > 1 {
				tr.UnselectAll()
				tr.Select()
				tr.SetFocus()
				sel = true
			}
		} else {
			tr.UnselectAll()
			tr.Select()
			tr.SetFocus()
			sel = true
		}
	case events.ExtendContinuous:
		sl := tr.SelectedViews()
		if len(sl) == 0 {
			tr.Select()
			tr.SetFocus()
			sel = true
		} else {
			minIndex := -1
			maxIndex := 0
			for _, v := range sl {
				vn := v.AsCoreTree()
				if minIndex < 0 {
					minIndex = vn.viewIndex
				} else {
					minIndex = min(minIndex, vn.viewIndex)
				}
				maxIndex = max(maxIndex, vn.viewIndex)
			}
			cidx := tr.viewIndex
			nn := tr
			tr.Select()
			if tr.viewIndex < minIndex {
				for cidx < minIndex {
					nn = nn.MoveDown(events.SelectQuiet) // just select
					cidx = nn.viewIndex
				}
			} else if tr.viewIndex > maxIndex {
				for cidx > maxIndex {
					nn = nn.MoveUp(events.SelectQuiet) // just select
					cidx = nn.viewIndex
				}
			}
		}
	case events.ExtendOne:
		if tr.StateIs(states.Selected) {
			tr.UnselectAction()
		} else {
			tr.Select()
			tr.SetFocus()
			sel = true
		}
	case events.SelectQuiet:
		tr.Select()
		// not sel -- no signal..
	case events.UnselectQuiet:
		tr.Unselect()
		// not sel -- no signal..
	}
	tr.NeedsRender()
	return sel
}

// SendSelectEvent sends an [events.Select] event on the RootView node.
func (tr *Tree) SendSelectEvent(original ...events.Event) {
	// fmt.Println("root:", &tv.RootView, tv.RootView.Listeners)
	tr.RootView.Send(events.Select, original...)
}

// SendChangeEvent sends an [events.Change] event on the RootView node.
func (tr *Tree) SendChangeEvent(original ...events.Event) {
	tr.RootView.SendChange(original...)
}

// TreeChanged must be called after any structural
// change to the Tree (adding or deleting nodes).
// It calls RootSetViewIndex to update indexes and
// SendChangeEvent to notify of changes.
func (tr *Tree) TreeChanged(original ...events.Event) {
	tr.RootView.RootSetViewIndex()
	tr.SendChangeEvent(original...)
}

// SendChangeEventReSync sends an [events.Change] event on the RootView node.
// If SyncNode != nil, it also does a re-sync from root.
func (tr *Tree) SendChangeEventReSync(original ...events.Event) {
	tr.RootView.SendChange(original...)
	if tr.RootView.SyncNode != nil {
		tr.RootView.ReSync()
	}
}

// SelectAction updates selection to include this node,
// using selectmode from mouse event (ExtendContinuous, ExtendOne),
// and Root sends selection event.  Returns true if signal emitted.
func (tr *Tree) SelectAction(mode events.SelectModes) bool {
	sel := tr.SelectUpdate(mode)
	if sel {
		tr.SendSelectEvent()
	}
	return sel
}

// UnselectAction unselects this node (if selected),
// and Root sends a selection event.
func (tr *Tree) UnselectAction() {
	if tr.StateIs(states.Selected) {
		tr.Unselect()
		tr.SendSelectEvent()
	}
}

//////////////////////////////////////////////////////////////////////////////
//    Moving

// MoveDown moves the selection down to next element in the tree,
// using given select mode (from keyboard modifiers).
// Returns newly selected node.
func (tr *Tree) MoveDown(selMode events.SelectModes) *Tree {
	if tr.Parent == nil {
		return nil
	}
	if tr.Closed || !tr.HasChildren() { // next sibling
		return tr.MoveDownSibling(selMode)
	} else {
		if tr.HasChildren() {
			nn := AsTree(tr.Child(0))
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
func (tr *Tree) MoveDownAction(selMode events.SelectModes) *Tree {
	nn := tr.MoveDown(selMode)
	if nn != nil && nn != tr {
		nn.SetFocus()
		nn.ScrollToMe()
		tr.SendSelectEvent()
	}
	return nn
}

// MoveDownSibling moves down only to siblings, not down into children,
// using given select mode (from keyboard modifiers)
func (tr *Tree) MoveDownSibling(selMode events.SelectModes) *Tree {
	if tr.Parent == nil {
		return nil
	}
	if tr == tr.RootView {
		return nil
	}
	myidx := tr.IndexInParent()
	if myidx < len(tr.Parent.AsTree().Children)-1 {
		nn := AsTree(tr.Parent.AsTree().Child(myidx + 1))
		if nn != nil {
			nn.SelectUpdate(selMode)
			return nn
		}
	} else {
		return AsTree(tr.Parent).MoveDownSibling(selMode) // try up
	}
	return nil
}

// MoveUp moves selection up to previous element in the tree,
// using given select mode (from keyboard modifiers).
// Returns newly selected node
func (tr *Tree) MoveUp(selMode events.SelectModes) *Tree {
	if tr.Parent == nil || tr == tr.RootView {
		return nil
	}
	myidx := tr.IndexInParent()
	if myidx > 0 {
		nn := AsTree(tr.Parent.AsTree().Child(myidx - 1))
		if nn != nil {
			return nn.MoveToLastChild(selMode)
		}
	} else {
		if tr.Parent != nil {
			nn := AsTree(tr.Parent)
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
func (tr *Tree) MoveUpAction(selMode events.SelectModes) *Tree {
	nn := tr.MoveUp(selMode)
	if nn != nil && nn != tr {
		nn.SetFocus()
		nn.ScrollToMe()
		tr.SendSelectEvent()
	}
	return nn
}

// TreePageSteps is the number of steps to take in PageUp / Down events
var TreePageSteps = 10

// MovePageUpAction moves the selection up to previous
// TreePageSteps elements in the tree,
// using given select mode (from keyboard modifiers).
// Sends select event for newly selected item.
func (tr *Tree) MovePageUpAction(selMode events.SelectModes) *Tree {
	mvMode := selMode
	if selMode == events.SelectOne {
		mvMode = events.NoSelect
	} else if selMode == events.ExtendContinuous || selMode == events.ExtendOne {
		mvMode = events.SelectQuiet
	}
	fnn := tr.MoveUp(mvMode)
	if fnn != nil && fnn != tr {
		for i := 1; i < TreePageSteps; i++ {
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
		tr.SendSelectEvent()
	}
	tr.NeedsRender()
	return fnn
}

// MovePageDownAction moves the selection up to
// previous TreePageSteps elements in the tree,
// using given select mode (from keyboard modifiers).
// Sends select event for newly selected item.
func (tr *Tree) MovePageDownAction(selMode events.SelectModes) *Tree {
	mvMode := selMode
	if selMode == events.SelectOne {
		mvMode = events.NoSelect
	} else if selMode == events.ExtendContinuous || selMode == events.ExtendOne {
		mvMode = events.SelectQuiet
	}
	fnn := tr.MoveDown(mvMode)
	if fnn != nil && fnn != tr {
		for i := 1; i < TreePageSteps; i++ {
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
		tr.SendSelectEvent()
	}
	tr.NeedsRender()
	return fnn
}

// MoveToLastChild moves to the last child under me, using given select mode
// (from keyboard modifiers)
func (tr *Tree) MoveToLastChild(selMode events.SelectModes) *Tree {
	if tr.Parent == nil || tr == tr.RootView {
		return nil
	}
	if !tr.Closed && tr.HasChildren() {
		nn := AsTree(tr.Child(tr.NumChildren() - 1))
		return nn.MoveToLastChild(selMode)
	} else {
		tr.SelectUpdate(selMode)
		return tr
	}
}

// MoveHomeAction moves the selection up to top of the tree,
// using given select mode (from keyboard modifiers)
// and emits select event for newly selected item
func (tr *Tree) MoveHomeAction(selMode events.SelectModes) *Tree {
	tr.RootView.SelectUpdate(selMode)
	tr.RootView.SetFocus()
	tr.RootView.ScrollToMe()
	// tv.RootView.TreeSig.Emit(tv.RootView.This, int64(TreeSelected), tv.RootView.This)
	return tr.RootView
}

// MoveEndAction moves the selection to the very last node in the tree,
// using given select mode (from keyboard modifiers)
// Sends select event for newly selected item.
func (tr *Tree) MoveEndAction(selMode events.SelectModes) *Tree {
	mvMode := selMode
	if selMode == events.SelectOne {
		mvMode = events.NoSelect
	} else if selMode == events.ExtendContinuous || selMode == events.ExtendOne {
		mvMode = events.SelectQuiet
	}
	fnn := tr.MoveDown(mvMode)
	if fnn != nil && fnn != tr {
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
		tr.SendSelectEvent()
	}
	return fnn
}

func (tr *Tree) SetKidsVisibility(parentClosed bool) {
	for _, k := range tr.Children {
		tvn := AsTree(k)
		if tvn != nil {
			tvn.SetState(parentClosed, states.Invisible)
		}
	}
}

// OnClose is called when a node is closed.
// The base version does nothing.
func (tr *Tree) OnClose() {}

// Close closes the given node and updates the view accordingly
// (if it is not already closed).
// Calls OnClose in Treer interface for extensible actions.
func (tr *Tree) Close() {
	if tr.Closed {
		return
	}
	tr.SetClosed(true)
	tr.SetBranchState()
	tr.This.(Treer).OnClose()
	tr.SetKidsVisibility(true) // parent closed
	tr.NeedsLayout()
}

// OnOpen is called when a node is opened.
// The base version does nothing.
func (tr *Tree) OnOpen() {}

func (tr *Tree) OnDoubleClick(e events.Event) {
	if tr.HasChildren() {
		tr.ToggleClose()
	}
}

// CanOpen returns true if the node is able to open.
// By default it checks HasChildren(), but could check other properties
// to perform lazy building of the tree.
func (tr *Tree) CanOpen() bool {
	return tr.HasChildren()
}

// Open opens the given node and updates the view accordingly
// (if it is not already opened).
// Calls OnOpen in Treer interface for extensible actions.
func (tr *Tree) Open() {
	if !tr.Closed || tr.inOpen {
		return
	}
	tr.inOpen = true
	if tr.This.(Treer).CanOpen() {
		tr.SetClosed(false)
		tr.SetBranchState()
		tr.SetKidsVisibility(false)
		tr.This.(Treer).OnOpen()
	}
	tr.inOpen = false
	tr.NeedsLayout()
}

// ToggleClose toggles the close / open status: if closed, opens, and vice-versa
func (tr *Tree) ToggleClose() {
	if tr.Closed {
		tr.Open()
	} else {
		tr.Close()
	}
}

// OpenAll opens the given node and all of its sub-nodes
func (tr *Tree) OpenAll() { //types:add
	tr.WidgetWalkDown(func(wi Widget, wb *WidgetBase) bool {
		tvn := AsTree(wi)
		if tvn != nil {
			tvn.Open()
			return tree.Continue
		}
		return tree.Break
	})
	tr.NeedsLayout()
}

// CloseAll closes the given node and all of its sub-nodes.
func (tr *Tree) CloseAll() { //types:add
	tr.WidgetWalkDown(func(wi Widget, wb *WidgetBase) bool {
		tvn := AsTree(wi)
		if tvn != nil {
			tvn.Close()
			return tree.Continue
		}
		return tree.Break
	})
	tr.NeedsLayout()
}

// OpenParents opens all the parents of this node,
// so that it will be visible.
func (tr *Tree) OpenParents() {
	tr.WalkUpParent(func(k tree.Node) bool {
		tvn := AsTree(k)
		if tvn != nil {
			tvn.Open()
			return tree.Continue
		}
		return tree.Break
	})
	tr.NeedsLayout()
}

/////////////////////////////////////////////////////////////
//    Modifying Source Tree

func (tr *Tree) ContextMenuPos(e events.Event) (pos image.Point) {
	if e != nil {
		pos = e.WindowPos()
		return
	}
	pos.X = tr.Geom.TotalBBox.Min.X + int(tr.Indent.Dots)
	pos.Y = (tr.Geom.TotalBBox.Min.Y + tr.Geom.TotalBBox.Max.Y) / 2
	return
}

func (tr *Tree) ContextMenuReadOnly(m *Scene) {
	NewButton(m).SetText("Copy").SetIcon(icons.ContentCopy).SetKey(keymap.Copy).
		SetEnabled(tr.HasSelection()).
		OnClick(func(e events.Event) {
			tr.Copy(true)
		})
	NewButton(m).SetText("View").SetIcon(icons.Visibility).
		SetEnabled(tr.HasSelection()).
		OnClick(func(e events.Event) {
			tr.EditNode()
		})
	NewSeparator(m)

	NewFuncButton(m, tr.OpenAll).SetIcon(icons.KeyboardArrowDown).
		SetEnabled(tr.HasSelection())
	NewFuncButton(m, tr.CloseAll).SetIcon(icons.KeyboardArrowRight).
		SetEnabled(tr.HasSelection())
}

func (tr *Tree) ContextMenu(m *Scene) {
	if tr.IsReadOnly() || tr.RootIsReadOnly() {
		tr.ContextMenuReadOnly(m)
		return
	}
	tvi := tr.This.(Treer)
	NewButton(m).SetText("Add child").SetIcon(icons.Add).
		SetEnabled(tr.HasSelection()).
		OnClick(func(e events.Event) {
			tvi.AddChildNode()
		})
	NewButton(m).SetText("Insert before").SetIcon(icons.Add).
		SetEnabled(tr.HasSelection()).
		OnClick(func(e events.Event) {
			tvi.InsertBefore()
		})
	NewButton(m).SetText("Insert after").SetIcon(icons.Add).
		SetEnabled(tr.HasSelection()).
		OnClick(func(e events.Event) {
			tvi.InsertAfter()
		})
	NewButton(m).SetText("Duplicate").SetIcon(icons.ContentCopy).
		SetEnabled(tr.HasSelection()).
		OnClick(func(e events.Event) {
			tvi.Duplicate()
		})
	NewButton(m).SetText("Delete").SetIcon(icons.Delete).
		SetEnabled(tr.HasSelection()).
		OnClick(func(e events.Event) {
			tvi.DeleteNode()
		})
	NewSeparator(m)
	NewButton(m).SetText("Copy").SetIcon(icons.ContentCopy).SetKey(keymap.Copy).
		SetEnabled(tr.HasSelection()).
		OnClick(func(e events.Event) {
			tvi.Copy(true)
		})
	NewButton(m).SetText("Cut").SetIcon(icons.ContentCut).SetKey(keymap.Cut).
		SetEnabled(tr.HasSelection()).
		OnClick(func(e events.Event) {
			tvi.Cut()
		})
	pbt := NewButton(m).SetText("Paste").SetIcon(icons.ContentPaste).SetKey(keymap.Paste).
		OnClick(func(e events.Event) {
			tvi.Paste()
		})
	cb := tr.Scene.Events.Clipboard()
	if cb != nil {
		pbt.SetState(cb.IsEmpty(), states.Disabled)
	}
	NewSeparator(m)
	NewFuncButton(m, tr.EditNode).SetText("Edit").SetIcon(icons.Edit).
		SetEnabled(tr.HasSelection())
	NewFuncButton(m, tr.InspectNode).SetText("Inspect").SetIcon(icons.EditDocument).
		SetEnabled(tr.HasSelection())
	NewSeparator(m)

	NewFuncButton(m, tr.OpenAll).SetIcon(icons.KeyboardArrowDown).
		SetEnabled(tr.HasSelection())
	NewFuncButton(m, tr.CloseAll).SetIcon(icons.KeyboardArrowRight).
		SetEnabled(tr.HasSelection())
}

// IsRoot returns true if given node is the root of the tree.
func (tr *Tree) IsRoot(op string) bool {
	if tr.This == tr.RootView.This {
		if op != "" {
			MessageSnackbar(tr, fmt.Sprintf("Cannot %v the root of the tree", op))
		}
		return true
	}
	return false
}

////////////////////////////////////////////////////////////
//    Copy / Cut / Paste

// MimeData adds mimedata for this node: a text/plain of the Path.
func (tr *Tree) MimeData(md *mimedata.Mimes) {
	if tr.SyncNode != nil {
		tr.MimeDataSync(md)
		return
	}
	*md = append(*md, mimedata.NewTextData(tr.PathFrom(tr.RootView)))
	var buf bytes.Buffer
	err := jsonx.Write(tr.This, &buf)
	if err == nil {
		*md = append(*md, &mimedata.Data{Type: fileinfo.DataJson, Data: buf.Bytes()})
	} else {
		ErrorSnackbar(tr, err, "Error encoding node")
	}
}

// NodesFromMimeData returns a slice of tree nodes for
// the Tree nodes and paths from mime data.
func (tr *Tree) NodesFromMimeData(md mimedata.Mimes) ([]tree.Node, []string) {
	ni := len(md) / 2
	sl := make([]tree.Node, 0, ni)
	pl := make([]string, 0, ni)
	for _, d := range md {
		if d.Type == fileinfo.DataJson {
			nn, err := tree.UnmarshalRootJSON(d.Data)
			if err == nil {
				sl = append(sl, nn)
			} else {
				ErrorSnackbar(tr, err, "Error loading node")
			}
		} else if d.Type == fileinfo.TextPlain { // paths
			pl = append(pl, string(d.Data))
		}
	}
	return sl, pl
}

// Copy copies to system.Clipboard, optionally resetting the selection.
func (tr *Tree) Copy(reset bool) { //types:add
	sels := tr.SelectedViews()
	nitms := max(1, len(sels))
	md := make(mimedata.Mimes, 0, 2*nitms)
	tr.This.(Treer).MimeData(&md) // source is always first..
	if nitms > 1 {
		for _, sn := range sels {
			if sn != tr.This {
				sn.MimeData(&md)
			}
		}
	}
	tr.Clipboard().Write(md)
	if reset {
		tr.UnselectAll()
	}
}

// Cut copies to system.Clipboard and deletes selected items.
func (tr *Tree) Cut() { //types:add
	if tr.IsRoot("Cut") {
		return
	}
	if tr.SyncNode != nil {
		tr.CutSync()
		return
	}
	tr.Copy(false)
	sels := tr.SelectedViews()
	root := tr.RootView
	tr.UnselectAll()
	for _, sn := range sels {
		sn.AsTree().Delete()
	}
	root.Update()
	root.TreeChanged()
}

// Paste pastes clipboard at given node.
func (tr *Tree) Paste() { //types:add
	md := tr.Clipboard().Read([]string{fileinfo.DataJson})
	if md != nil {
		tr.PasteMenu(md)
	}
}

// PasteMenu performs a paste from the clipboard using given data,
// by popping up a menu to determine what specifically to do.
func (tr *Tree) PasteMenu(md mimedata.Mimes) {
	tr.UnselectAll()
	mf := func(m *Scene) {
		tr.This.(Treer).MakePasteMenu(m, md, nil)
	}
	pos := tr.ContextMenuPos(nil)
	NewMenu(mf, tr.This.(Widget), pos).Run()
}

// MakePasteMenu makes the menu of options for paste events
// optional function is typically the DropFinalize but could also be other actions
// to take after each optional action.
func (tr *Tree) MakePasteMenu(m *Scene, md mimedata.Mimes, fun func()) {
	NewButton(m).SetText("Assign To").OnClick(func(e events.Event) {
		tr.PasteAssign(md)
		if fun != nil {
			fun()
		}
	})
	NewButton(m).SetText("Add to Children").OnClick(func(e events.Event) {
		tr.PasteChildren(md, events.DropCopy)
		if fun != nil {
			fun()
		}
	})
	if !tr.IsRoot("") && tr.RootView.This != tr.This {
		NewButton(m).SetText("Insert Before").OnClick(func(e events.Event) {
			tr.PasteBefore(md, events.DropCopy)
			if fun != nil {
				fun()
			}
		})
		NewButton(m).SetText("Insert After").OnClick(func(e events.Event) {
			tr.PasteAfter(md, events.DropCopy)
			if fun != nil {
				fun()
			}
		})
	}
	NewButton(m).SetText("Cancel")
}

// PasteAssign assigns mime data (only the first one!) to this node
func (tr *Tree) PasteAssign(md mimedata.Mimes) {
	if tr.SyncNode != nil {
		tr.PasteAssignSync(md)
		return
	}
	sl, _ := tr.NodesFromMimeData(md)
	if len(sl) == 0 {
		return
	}
	tr.CopyFrom(sl[0])    // nodes with data copy here
	tr.SetScene(tr.Scene) // ensure children have scene
	tr.Update()           // could have children
	tr.Open()
	tr.TreeChanged()
}

// PasteBefore inserts object(s) from mime data before this node.
// If another item with the same name already exists, it will
// append _Copy on the name of the inserted objects
func (tr *Tree) PasteBefore(md mimedata.Mimes, mod events.DropMods) {
	tr.PasteAt(md, mod, 0, "Paste Before")
}

// PasteAfter inserts object(s) from mime data after this node.
// If another item with the same name already exists, it will
// append _Copy on the name of the inserted objects
func (tr *Tree) PasteAfter(md mimedata.Mimes, mod events.DropMods) {
	tr.PasteAt(md, mod, 1, "Paste After")
}

// TreeTempMovedTag is a kind of hack to prevent moved items from being deleted, using DND
const TreeTempMovedTag = `_\&MOVED\&`

// todo: these methods require an interface to work for descended
// nodes, based on base code

// PasteAt inserts object(s) from mime data at rel position to this node.
// If another item with the same name already exists, it will
// append _Copy on the name of the inserted objects
func (tr *Tree) PasteAt(md mimedata.Mimes, mod events.DropMods, rel int, actNm string) {
	if tr.Parent == nil {
		return
	}
	parent := AsTree(tr.Parent)
	if parent == nil {
		MessageSnackbar(tr, "Error: cannot insert after the root of the tree")
		return
	}
	if tr.SyncNode != nil {
		tr.PasteAtSync(md, mod, rel, actNm)
		return
	}
	sl, pl := tr.NodesFromMimeData(md)

	myidx := tr.IndexInParent()
	if myidx < 0 {
		return
	}
	myidx += rel
	sz := len(sl)
	var selTv *Tree
	for i, ns := range sl {
		orgpath := pl[i]
		if mod != events.DropMove {
			if cn := parent.ChildByName(ns.AsTree().Name, 0); cn != nil {
				ns.AsTree().SetName(ns.AsTree().Name + "_Copy")
			}
		}
		parent.InsertChild(ns, myidx+i)
		_, nwb := AsWidget(ns)
		ntv := AsTree(ns)
		ntv.RootView = tr.RootView
		nwb.SetScene(tr.Scene)
		nwb.Update() // incl children
		npath := ns.AsTree().PathFrom(tr.RootView)
		if mod == events.DropMove && npath == orgpath { // we will be nuked immediately after drag
			ns.AsTree().SetName(ns.AsTree().Name + TreeTempMovedTag) // special keyword :)
		}
		if i == sz-1 {
			selTv = ntv
		}
	}
	tr.TreeChanged()
	parent.NeedsLayout()
	if selTv != nil {
		selTv.SelectAction(events.SelectOne)
	}
}

// PasteChildren inserts object(s) from mime data
// at end of children of this node
func (tr *Tree) PasteChildren(md mimedata.Mimes, mod events.DropMods) {
	if tr.SyncNode != nil {
		tr.PasteChildrenSync(md, mod)
		return
	}
	sl, _ := tr.NodesFromMimeData(md)

	for _, ns := range sl {
		tr.AddChild(ns)
		_, nwb := AsWidget(ns)
		ntv := AsTree(ns)
		ntv.RootView = tr.RootView
		nwb.SetScene(tr.Scene)
	}
	tr.Update()
	tr.Open()
	tr.TreeChanged()
}

//////////////////////////////////////////////////////////////////////////////
//    Drag-n-Drop

// DragStart starts a drag-n-drop on this node -- it includes any other
// selected nodes as well, each as additional records in mimedata.
func (tr *Tree) DragStart(e events.Event) {
	sels := tr.SelectedViews()
	nitms := max(1, len(sels))
	md := make(mimedata.Mimes, 0, 2*nitms)
	tr.This.(Treer).MimeData(&md) // source is always first..
	if nitms > 1 {
		for _, sn := range sels {
			if sn != tr.This {
				sn.MimeData(&md)
			}
		}
	}
	tr.Scene.Events.DragStart(tr.This.(Widget), md, e)
}

// DropExternal is not handled by base case but could be in derived
func (tr *Tree) DropExternal(md mimedata.Mimes, mod events.DropMods) {
	// todo: not yet implemented
}

// DragClearStates clears the drag-drop related states for this widget
func (tr *Tree) DragClearStates() {
	tr.SetState(false, states.Active, states.Selected, states.Hovered, states.DragHovered)
	tr.Parts.SetState(false, states.Active, states.Selected, states.Hovered, states.DragHovered)
	tr.Style()
	tr.NeedsRender()
}

// DragDrop handles drag drop event
func (tr *Tree) DragDrop(e events.Event) {
	// todo: some kind of validation for source
	tvi := tr.This.(Treer)
	tr.UnselectAll()
	de := e.(*events.DragDrop)
	stv := AsTree(de.Source.(Widget))
	if stv != nil {
		stv.DragClearStates()
	}
	md := de.Data.(mimedata.Mimes)
	mf := func(m *Scene) {
		tr.Scene.Events.DragMenuAddModText(m, de.DropMod)
		tvi.MakePasteMenu(m, md, func() {
			tvi.DropFinalize(de)
		})
	}
	pos := tr.ContextMenuPos(nil)
	NewMenu(mf, tr.This.(Widget), pos).Run()
}

// DropFinalize is called to finalize Drop actions on the Source node.
// Only relevant for DropMod == DropMove.
func (tr *Tree) DropFinalize(de *events.DragDrop) {
	tr.UnselectAll()
	tr.DragClearStates()
	tr.Scene.Events.DropFinalize(de) // sends DropDeleteSource to Source
}

// DropDeleteSource handles delete source event for DropMove case
func (tr *Tree) DropDeleteSource(e events.Event) {
	de := e.(*events.DragDrop)
	tr.UnselectAll()
	if tr.SyncNode != nil {
		tr.DropDeleteSourceSync(de)
		return
	}
	md := de.Data.(mimedata.Mimes)
	root := tr.RootView
	for _, d := range md {
		if d.Type != fileinfo.TextPlain { // link
			continue
		}
		path := string(d.Data)
		sn := root.FindPath(path)
		if sn != nil {
			sn.AsTree().Delete()
		}
		sn = root.FindPath(path + TreeTempMovedTag)
		if sn != nil {
			psplt := strings.Split(path, "/")
			orgnm := psplt[len(psplt)-1]
			sn.AsTree().SetName(orgnm)
			_, swb := AsWidget(sn)
			swb.NeedsRender()
		}
	}
}
