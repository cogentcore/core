// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

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
	"cogentcore.org/core/core"
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
	core.Widget

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
	MakePasteMenu(m *core.Scene, md mimedata.Mimes, fun func())
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
	core.WidgetBase

	// If non-nil, the [tree.Node] that this widget is viewing in the tree (the source)
	SyncNode tree.Node `set:"-" copier:"-" json:"-" xml:"-"`

	// The text to display for the tree item label, which automatically
	// defaults to the [tree.Node.Name] of the tree node. It has no effect
	// if [Tree.SyncNode] is non-nil.
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
	fr := core.NewFrame(parent...).Styler(func(s *styles.Style) {
		s.Overflow.Set(styles.OverflowAuto)
	})
	return NewTree(fr)
}

// AsCoreTree satisfies the [Treer] interface.
func (tv *Tree) AsCoreTree() *Tree {
	return tv
}

func (tv *Tree) BaseType() *types.Type {
	return tv.NodeType()
}

// RootSetViewIndex sets the RootView and ViewIndex for all nodes.
// This must be called from the root node after
// construction or any modification to the tree.
// Returns the total number of leaves in the tree.
func (tv *Tree) RootSetViewIndex() int {
	idx := 0
	tv.WidgetWalkDown(func(wi core.Widget, wb *core.WidgetBase) bool {
		tvn := AsTree(wi)
		if tvn != nil {
			tvn.viewIndex = idx
			tvn.RootView = tv
			// fmt.Println(idx, tvn, "root:", tv, &tv)
			idx++
		}
		return tree.Continue
	})
	return idx
}

func (tv *Tree) Init() {
	tv.WidgetBase.Init()
	tv.AddContextMenu(tv.ContextMenu)
	tv.IconOpen = icons.KeyboardArrowDown
	tv.IconClosed = icons.KeyboardArrowRight
	tv.IconLeaf = icons.Blank
	tv.OpenDepth = 4
	tv.Styler(func(s *styles.Style) {
		// our parts are draggable and droppable, not us ourself
		s.SetAbilities(true, abilities.Activatable, abilities.Focusable, abilities.Selectable, abilities.Hoverable)
		tv.Indent.Em(1)
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
	tv.FinalStyler(func(s *styles.Style) {
		tv.actStateLayer = s.StateLayer
		s.StateLayer = 0
	})

	// We let the parts handle our state
	// so that we only get it when we are doing
	// something with this tree specifically,
	// not with any of our children (see OnChildAdded).
	// we only need to handle the starting ones here,
	// as the other ones will just set the state to
	// false, which it already is.
	tv.On(events.MouseEnter, func(e events.Event) { e.SetHandled() })
	tv.On(events.MouseLeave, func(e events.Event) { e.SetHandled() })
	tv.On(events.MouseDown, func(e events.Event) { e.SetHandled() })
	tv.OnClick(func(e events.Event) { e.SetHandled() })
	tv.On(events.DragStart, func(e events.Event) { e.SetHandled() })
	tv.On(events.DragEnter, func(e events.Event) { e.SetHandled() })
	tv.On(events.DragLeave, func(e events.Event) { e.SetHandled() })
	tv.On(events.Drop, func(e events.Event) { e.SetHandled() })
	tv.On(events.DropDeleteSource, func(e events.Event) { e.SetHandled() })
	tv.On(events.KeyChord, func(e events.Event) {
		kf := keymap.Of(e.KeyChord())
		selMode := events.SelectModeBits(e.Modifiers())
		if core.DebugSettings.KeyEventTrace {
			slog.Info("Tree KeyInput", "widget", tv, "keyFunction", kf, "selMode", selMode)
		}

		if selMode == events.SelectOne {
			if tv.SelectMode {
				selMode = events.ExtendContinuous
			}
		}

		tvi := tv.This.(Treer)

		// first all the keys that work for ReadOnly and active
		switch kf {
		case keymap.CancelSelect:
			tv.UnselectAll()
			tv.SetSelectMode(false)
			e.SetHandled()
		case keymap.MoveRight:
			tv.Open()
			e.SetHandled()
		case keymap.MoveLeft:
			tv.Close()
			e.SetHandled()
		case keymap.MoveDown:
			tv.MoveDownAction(selMode)
			e.SetHandled()
		case keymap.MoveUp:
			tv.MoveUpAction(selMode)
			e.SetHandled()
		case keymap.PageUp:
			tv.MovePageUpAction(selMode)
			e.SetHandled()
		case keymap.PageDown:
			tv.MovePageDownAction(selMode)
			e.SetHandled()
		case keymap.Home:
			tv.MoveHomeAction(selMode)
			e.SetHandled()
		case keymap.End:
			tv.MoveEndAction(selMode)
			e.SetHandled()
		case keymap.SelectMode:
			tv.SelectMode = !tv.SelectMode
			e.SetHandled()
		case keymap.SelectAll:
			tv.SelectAll()
			e.SetHandled()
		case keymap.Enter:
			tv.ToggleClose()
			e.SetHandled()
		case keymap.Copy:
			tvi.Copy(true)
			e.SetHandled()
		}
		if !tv.RootIsReadOnly() && !e.IsHandled() {
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

	core.AddChildAt(tv, "parts", func(w *core.Frame) {
		core.InitParts(w)
		tvi := tv.This.(Treer)
		w.Styler(func(s *styles.Style) {
			s.Cursor = cursors.Pointer
			s.SetAbilities(true, abilities.Activatable, abilities.Focusable, abilities.Selectable, abilities.Hoverable, abilities.DoubleClickable)
			s.SetAbilities(!tv.IsReadOnly() && !tv.RootIsReadOnly(), abilities.Draggable, abilities.Droppable)
			s.Gap.X.Em(0.1)
			s.Padding.Zero()

			// we manually inherit our state layer from the tree state
			// layer so that the parts get it but not the other trees
			s.StateLayer = tv.actStateLayer
		})
		w.AsWidget().FinalStyler(func(s *styles.Style) {
			s.Grow.Set(1, 0)
		})
		// we let the parts handle our state
		// so that we only get it when we are doing
		// something with this tree specifically,
		// not with any of our children (see HandleTreeMouse)
		w.On(events.MouseEnter, func(e events.Event) {
			tv.SetState(true, states.Hovered)
			tv.Style()
			tv.NeedsRender()
			e.SetHandled()
		})
		w.On(events.MouseLeave, func(e events.Event) {
			tv.SetState(false, states.Hovered)
			tv.Style()
			tv.NeedsRender()
			e.SetHandled()
		})
		w.On(events.MouseDown, func(e events.Event) {
			tv.SetState(true, states.Active)
			tv.Style()
			tv.NeedsRender()
			e.SetHandled()
		})
		w.On(events.MouseUp, func(e events.Event) {
			tv.SetState(false, states.Active)
			tv.Style()
			tv.NeedsRender()
			e.SetHandled()
		})
		w.OnClick(func(e events.Event) {
			tv.SelectAction(e.SelectMode())
			e.SetHandled()
		})
		w.AsWidget().OnDoubleClick(func(e events.Event) {
			tv.This.(Treer).OnDoubleClick(e)
		})
		w.On(events.DragStart, func(e events.Event) {
			tvi.DragStart(e)
		})
		w.On(events.DragEnter, func(e events.Event) {
			tv.SetState(true, states.DragHovered)
			tv.Style()
			tv.NeedsRender()
			e.SetHandled()
		})
		w.On(events.DragLeave, func(e events.Event) {
			tv.SetState(false, states.DragHovered)
			tv.Style()
			tv.NeedsRender()
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
			sels := tv.SelectedViews()
			if len(sels) == 0 {
				tv.SelectAction(e.SelectMode())
			}
			tv.ShowContextMenu(e)
		})
		core.AddChildAt(w, "branch", func(w *core.Switch) {
			w.SetType(core.SwitchCheckbox)
			w.SetIcons(tv.IconOpen, tv.IconClosed, tv.IconLeaf)
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
					if !tv.Closed {
						tv.Close()
					}
				} else {
					if tv.Closed {
						tv.Open()
					}
				}
			})
			w.Updater(func() {
				if tv.This.(Treer).CanOpen() {
					tv.SetBranchState()
				}
			})
		})
		w.Maker(func(p *core.Plan) {
			if tv.Icon.IsSet() {
				core.AddAt(p, "icon", func(w *core.Icon) {
					w.Styler(func(s *styles.Style) {
						s.Font.Size.Dp(16)
					})
					w.Updater(func() {
						w.SetIcon(tv.Icon)
					})
				})
			}
		})
		core.AddChildAt(w, "text", func(w *core.Text) {
			w.Styler(func(s *styles.Style) {
				s.SetNonSelectable()
				s.SetTextWrap(false)
				s.Min.X.Ch(16)
				s.Min.Y.Em(1.2)
			})
			w.Updater(func() {
				w.SetText(tv.Label())
			})
		})
	})
}

func (tv *Tree) OnAdd() {
	tv.WidgetBase.OnAdd()
	tv.Text = tv.Name
	if ptv := AsTree(tv.Parent); ptv != nil {
		tv.RootView = ptv.RootView
		tv.IconOpen = ptv.IconOpen
		tv.IconClosed = ptv.IconClosed
		tv.IconLeaf = ptv.IconLeaf
	} else {
		// fmt.Println("set root to:", tv, &tv)
		tv.RootView = tv
	}
}

// RootIsReadOnly returns the ReadOnly status of the root node,
// which is what controls the functional inactivity of the tree
// if individual nodes are ReadOnly that only affects display typically.
func (tv *Tree) RootIsReadOnly() bool {
	if tv.RootView == nil {
		return true
	}
	return tv.RootView.IsReadOnly()
}

// qt calls the open / close thing a "branch"
// http://doc.qt.io/qt-5/stylesheet-examples.html#customizing-qtree

// BranchPart returns the branch in parts, if it exists
func (tv *Tree) BranchPart() (*core.Switch, bool) {
	if tv.Parts == nil {
		return nil, false
	}
	if icc := tv.Parts.ChildByName("branch", 0); icc != nil {
		return icc.(*core.Switch), true
	}
	return nil, false
}

// IconPart returns the icon in parts, if it exists
func (tv *Tree) IconPart() (*core.Icon, bool) {
	if tv.Parts == nil {
		return nil, false
	}
	if icc := tv.Parts.ChildByName("icon", 1); icc != nil {
		return icc.(*core.Icon), true
	}
	return nil, false
}

// LabelPart returns the label in parts, if it exists
func (tv *Tree) LabelPart() (*core.Text, bool) {
	if tv.Parts == nil {
		return nil, false
	}
	if lbl := tv.Parts.ChildByName("label", 1); lbl != nil {
		return lbl.(*core.Text), true
	}
	return nil, false
}

func (tv *Tree) Style() {
	if !tv.HasChildren() {
		tv.SetClosed(true)
	}
	tv.Indent.ToDots(&tv.Styles.UnitContext)
	tv.WidgetBase.Style()
}

func (tv *Tree) UpdateBranchIcons() {
}

func (tv *Tree) SetBranchState() {
	br, ok := tv.BranchPart()
	if !ok {
		return
	}
	switch {
	case !tv.This.(Treer).CanOpen():
		br.SetState(true, states.Indeterminate)
	case tv.Closed:
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

func (tv *Tree) SizeUp() {
	tv.WidgetBase.SizeUp()
	tv.widgetSize = tv.Geom.Size.Actual.Total
	h := tv.widgetSize.Y
	w := tv.widgetSize.X

	if !tv.Closed {
		// we layout children under us
		tv.WidgetKidsIter(func(i int, kwi core.Widget, kwb *core.WidgetBase) bool {
			kwi.SizeUp()
			h += kwb.Geom.Size.Actual.Total.Y
			kw := kwb.Geom.Size.Actual.Total.X
			if math32.IsNaN(kw) { // somehow getting a nan
				slog.Error("Tree, node width is NaN", "node:", kwb)
			} else {
				w = max(w, tv.Indent.Dots+kw)
			}
			// fmt.Println(kwb, w, h)
			return tree.Continue
		})
	}
	sz := &tv.Geom.Size
	sz.Actual.Content = math32.Vec2(w, h)
	sz.SetTotalFromContent(&sz.Actual)
	sz.Alloc = sz.Actual // need allocation to match!
	tv.widgetSize.X = w  // stretch
}

func (tv *Tree) SizeDown(iter int) bool {
	// note: key to not grab the whole allocation, as widget default does
	redo := tv.SizeDownParts(iter) // give our content to parts
	re := tv.SizeDownChildren(iter)
	return redo || re
}

func (tv *Tree) Position() {
	rn := tv.RootView
	if rn == nil {
		slog.Error("views.Tree: RootView is nil", "in node:", tv)
		return
	}
	tv.SetBranchState()
	tv.This.(Treer).UpdateBranchIcons()

	tv.Geom.Size.Actual.Total.X = rn.Geom.Size.Actual.Total.X - (tv.Geom.Pos.Total.X - rn.Geom.Pos.Total.X)
	tv.widgetSize.X = tv.Geom.Size.Actual.Total.X

	tv.WidgetBase.Position()

	if !tv.Closed {
		h := tv.widgetSize.Y
		tv.WidgetKidsIter(func(i int, kwi core.Widget, kwb *core.WidgetBase) bool {
			kwb.Geom.RelPos.Y = h
			kwb.Geom.RelPos.X = tv.Indent.Dots
			h += kwb.Geom.Size.Actual.Total.Y
			kwi.Position()
			return tree.Continue
		})
	}
}

func (tv *Tree) ScenePos() {
	sz := &tv.Geom.Size
	if sz.Actual.Total == tv.widgetSize {
		sz.SetTotalFromContent(&sz.Actual) // restore after scrolling
	}
	tv.WidgetBase.ScenePos()
	tv.ScenePosChildren()
	tv.Geom.Size.Actual.Total = tv.widgetSize // key: we revert to just ourselves
}

func (tv *Tree) Render() {
	pc := &tv.Scene.PaintContext
	st := &tv.Styles

	pabg := tv.ParentActualBackground()

	// must use workaround act values
	st.StateLayer = tv.actStateLayer
	if st.Is(states.Selected) {
		st.Background = colors.C(colors.Scheme.Select.Container)
	}
	tv.Styles.ComputeActualBackground(pabg)

	pc.DrawStandardBox(st, tv.Geom.Pos.Total, tv.Geom.Size.Actual.Total, pabg)

	// after we are done rendering, we clear the values so they aren't inherited
	st.StateLayer = 0
	st.Background = nil
	tv.Styles.ComputeActualBackground(pabg)
}

func (tv *Tree) RenderWidget() {
	if tv.PushBounds() {
		tv.Render()
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
	if !tv.Closed {
		tv.RenderChildren()
	}
}

//////////////////////////////////////////////////////////////////////////////
//    Selection

// SelectedViews returns a slice of the currently selected
// Trees within the entire tree, using a list maintained
// by the root node
func (tv *Tree) SelectedViews() []Treer {
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
func (tv *Tree) SetSelectedViews(sl []Treer) {
	if tv.RootView != nil {
		tv.RootView.SelectedNodes = sl
	}
}

// HasSelection returns true if there are currently selected items
func (tv *Tree) HasSelection() bool {
	return len(tv.SelectedViews()) > 0
}

// Select selects this node (if not already selected).
// Must use this method to update global selection list
func (tv *Tree) Select() {
	if !tv.StateIs(states.Selected) {
		tv.SetSelected(true)
		tv.Style()
		sl := tv.SelectedViews()
		sl = append(sl, tv.This.(Treer))
		tv.SetSelectedViews(sl)
		tv.NeedsRender()
	}
}

// Unselect unselects this node (if selected).
// Must use this method to update global selection list.
func (tv *Tree) Unselect() {
	if tv.StateIs(states.Selected) {
		tv.SetSelected(false)
		tv.Style()
		sl := tv.SelectedViews()
		sz := len(sl)
		for i := 0; i < sz; i++ {
			if sl[i] == tv {
				sl = append(sl[:i], sl[i+1:]...)
				break
			}
		}
		tv.SetSelectedViews(sl)
		tv.NeedsRender()
	}
}

// UnselectAll unselects all selected items in the view
func (tv *Tree) UnselectAll() {
	if tv.Scene == nil {
		return
	}
	sl := tv.SelectedViews()
	tv.SetSelectedViews(nil) // clear in advance
	for _, v := range sl {
		vt := v.AsCoreTree()
		if vt == nil || vt.This == nil {
			continue
		}
		vt.SetSelected(false)
		v.Style()
		vt.NeedsRender()
	}
	tv.NeedsRender()
}

// SelectAll all items in view
func (tv *Tree) SelectAll() {
	if tv.Scene == nil {
		return
	}
	tv.UnselectAll()
	nn := tv.RootView
	nn.Select()
	for nn != nil {
		nn = nn.MoveDown(events.SelectQuiet)
	}
	tv.NeedsRender()
}

// SelectUpdate updates selection to include this node,
// using selectmode from mouse event (ExtendContinuous, ExtendOne).
// Returns true if this node selected
func (tv *Tree) SelectUpdate(mode events.SelectModes) bool {
	if mode == events.NoSelect {
		return false
	}
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
			cidx := tv.viewIndex
			nn := tv
			tv.Select()
			if tv.viewIndex < minIndex {
				for cidx < minIndex {
					nn = nn.MoveDown(events.SelectQuiet) // just select
					cidx = nn.viewIndex
				}
			} else if tv.viewIndex > maxIndex {
				for cidx > maxIndex {
					nn = nn.MoveUp(events.SelectQuiet) // just select
					cidx = nn.viewIndex
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
	tv.NeedsRender()
	return sel
}

// SendSelectEvent sends an [events.Select] event on the RootView node.
func (tv *Tree) SendSelectEvent(original ...events.Event) {
	// fmt.Println("root:", &tv.RootView, tv.RootView.Listeners)
	tv.RootView.Send(events.Select, original...)
}

// SendChangeEvent sends an [events.Change] event on the RootView node.
func (tv *Tree) SendChangeEvent(original ...events.Event) {
	tv.RootView.SendChange(original...)
}

// TreeChanged must be called after any structural
// change to the Tree (adding or deleting nodes).
// It calls RootSetViewIndex to update indexes and
// SendChangeEvent to notify of changes.
func (tv *Tree) TreeChanged(original ...events.Event) {
	tv.RootView.RootSetViewIndex()
	tv.SendChangeEvent(original...)
}

// SendChangeEventReSync sends an [events.Change] event on the RootView node.
// If SyncNode != nil, it also does a re-sync from root.
func (tv *Tree) SendChangeEventReSync(original ...events.Event) {
	tv.RootView.SendChange(original...)
	if tv.RootView.SyncNode != nil {
		tv.RootView.ReSync()
	}
}

// SelectAction updates selection to include this node,
// using selectmode from mouse event (ExtendContinuous, ExtendOne),
// and Root sends selection event.  Returns true if signal emitted.
func (tv *Tree) SelectAction(mode events.SelectModes) bool {
	sel := tv.SelectUpdate(mode)
	if sel {
		tv.SendSelectEvent()
	}
	return sel
}

// UnselectAction unselects this node (if selected),
// and Root sends a selection event.
func (tv *Tree) UnselectAction() {
	if tv.StateIs(states.Selected) {
		tv.Unselect()
		tv.SendSelectEvent()
	}
}

//////////////////////////////////////////////////////////////////////////////
//    Moving

// MoveDown moves the selection down to next element in the tree,
// using given select mode (from keyboard modifiers).
// Returns newly selected node.
func (tv *Tree) MoveDown(selMode events.SelectModes) *Tree {
	if tv.Parent == nil {
		return nil
	}
	if tv.Closed || !tv.HasChildren() { // next sibling
		return tv.MoveDownSibling(selMode)
	} else {
		if tv.HasChildren() {
			nn := AsTree(tv.Child(0))
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
func (tv *Tree) MoveDownAction(selMode events.SelectModes) *Tree {
	nn := tv.MoveDown(selMode)
	if nn != nil && nn != tv {
		nn.SetFocus()
		nn.ScrollToMe()
		tv.SendSelectEvent()
	}
	return nn
}

// MoveDownSibling moves down only to siblings, not down into children,
// using given select mode (from keyboard modifiers)
func (tv *Tree) MoveDownSibling(selMode events.SelectModes) *Tree {
	if tv.Parent == nil {
		return nil
	}
	if tv == tv.RootView {
		return nil
	}
	myidx := tv.IndexInParent()
	if myidx < len(tv.Parent.AsTree().Children)-1 {
		nn := AsTree(tv.Parent.AsTree().Child(myidx + 1))
		if nn != nil {
			nn.SelectUpdate(selMode)
			return nn
		}
	} else {
		return AsTree(tv.Parent).MoveDownSibling(selMode) // try up
	}
	return nil
}

// MoveUp moves selection up to previous element in the tree,
// using given select mode (from keyboard modifiers).
// Returns newly selected node
func (tv *Tree) MoveUp(selMode events.SelectModes) *Tree {
	if tv.Parent == nil || tv == tv.RootView {
		return nil
	}
	myidx := tv.IndexInParent()
	if myidx > 0 {
		nn := AsTree(tv.Parent.AsTree().Child(myidx - 1))
		if nn != nil {
			return nn.MoveToLastChild(selMode)
		}
	} else {
		if tv.Parent != nil {
			nn := AsTree(tv.Parent)
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
func (tv *Tree) MoveUpAction(selMode events.SelectModes) *Tree {
	nn := tv.MoveUp(selMode)
	if nn != nil && nn != tv {
		nn.SetFocus()
		nn.ScrollToMe()
		tv.SendSelectEvent()
	}
	return nn
}

// TreePageSteps is the number of steps to take in PageUp / Down events
var TreePageSteps = 10

// MovePageUpAction moves the selection up to previous
// TreePageSteps elements in the tree,
// using given select mode (from keyboard modifiers).
// Sends select event for newly selected item.
func (tv *Tree) MovePageUpAction(selMode events.SelectModes) *Tree {
	mvMode := selMode
	if selMode == events.SelectOne {
		mvMode = events.NoSelect
	} else if selMode == events.ExtendContinuous || selMode == events.ExtendOne {
		mvMode = events.SelectQuiet
	}
	fnn := tv.MoveUp(mvMode)
	if fnn != nil && fnn != tv {
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
		tv.SendSelectEvent()
	}
	tv.NeedsRender()
	return fnn
}

// MovePageDownAction moves the selection up to
// previous TreePageSteps elements in the tree,
// using given select mode (from keyboard modifiers).
// Sends select event for newly selected item.
func (tv *Tree) MovePageDownAction(selMode events.SelectModes) *Tree {
	mvMode := selMode
	if selMode == events.SelectOne {
		mvMode = events.NoSelect
	} else if selMode == events.ExtendContinuous || selMode == events.ExtendOne {
		mvMode = events.SelectQuiet
	}
	fnn := tv.MoveDown(mvMode)
	if fnn != nil && fnn != tv {
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
		tv.SendSelectEvent()
	}
	tv.NeedsRender()
	return fnn
}

// MoveToLastChild moves to the last child under me, using given select mode
// (from keyboard modifiers)
func (tv *Tree) MoveToLastChild(selMode events.SelectModes) *Tree {
	if tv.Parent == nil || tv == tv.RootView {
		return nil
	}
	if !tv.Closed && tv.HasChildren() {
		nn := AsTree(tv.Child(tv.NumChildren() - 1))
		return nn.MoveToLastChild(selMode)
	} else {
		tv.SelectUpdate(selMode)
		return tv
	}
}

// MoveHomeAction moves the selection up to top of the tree,
// using given select mode (from keyboard modifiers)
// and emits select event for newly selected item
func (tv *Tree) MoveHomeAction(selMode events.SelectModes) *Tree {
	tv.RootView.SelectUpdate(selMode)
	tv.RootView.SetFocus()
	tv.RootView.ScrollToMe()
	// tv.RootView.TreeSig.Emit(tv.RootView.This, int64(TreeSelected), tv.RootView.This)
	return tv.RootView
}

// MoveEndAction moves the selection to the very last node in the tree,
// using given select mode (from keyboard modifiers)
// Sends select event for newly selected item.
func (tv *Tree) MoveEndAction(selMode events.SelectModes) *Tree {
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
		tv.SendSelectEvent()
	}
	return fnn
}

func (tv *Tree) SetKidsVisibility(parentClosed bool) {
	for _, k := range tv.Children {
		tvn := AsTree(k)
		if tvn != nil {
			tvn.SetState(parentClosed, states.Invisible)
		}
	}
}

// OnClose is called when a node is closed.
// The base version does nothing.
func (tv *Tree) OnClose() {
}

// Close closes the given node and updates the view accordingly
// (if it is not already closed).
// Calls OnClose in Treer interface for extensible actions.
func (tv *Tree) Close() {
	if tv.Closed {
		return
	}
	tv.SetClosed(true)
	tv.SetBranchState()
	tv.This.(Treer).OnClose()
	tv.SetKidsVisibility(true) // parent closed
	tv.NeedsLayout()
}

// OnOpen is called when a node is opened.
// The base version does nothing.
func (tv *Tree) OnOpen() {
}

func (tv *Tree) OnDoubleClick(e events.Event) {
	if tv.HasChildren() {
		tv.ToggleClose()
	}
}

// CanOpen returns true if the node is able to open.
// By default it checks HasChildren(), but could check other properties
// to perform lazy building of the tree.
func (tv *Tree) CanOpen() bool {
	return tv.HasChildren()
}

// Open opens the given node and updates the view accordingly
// (if it is not already opened).
// Calls OnOpen in Treer interface for extensible actions.
func (tv *Tree) Open() {
	if !tv.Closed || tv.inOpen {
		return
	}
	tv.inOpen = true
	if tv.This.(Treer).CanOpen() {
		tv.SetClosed(false)
		tv.SetBranchState()
		tv.SetKidsVisibility(false)
		tv.This.(Treer).OnOpen()
	}
	tv.inOpen = false
	tv.NeedsLayout()
}

// ToggleClose toggles the close / open status: if closed, opens, and vice-versa
func (tv *Tree) ToggleClose() {
	if tv.Closed {
		tv.Open()
	} else {
		tv.Close()
	}
}

// OpenAll opens the given node and all of its sub-nodes
func (tv *Tree) OpenAll() { //types:add
	tv.WidgetWalkDown(func(wi core.Widget, wb *core.WidgetBase) bool {
		tvn := AsTree(wi)
		if tvn != nil {
			tvn.Open()
			return tree.Continue
		}
		return tree.Break
	})
	tv.NeedsLayout()
}

// CloseAll closes the given node and all of its sub-nodes.
func (tv *Tree) CloseAll() { //types:add
	tv.WidgetWalkDown(func(wi core.Widget, wb *core.WidgetBase) bool {
		tvn := AsTree(wi)
		if tvn != nil {
			tvn.Close()
			return tree.Continue
		}
		return tree.Break
	})
	tv.NeedsLayout()
}

// OpenParents opens all the parents of this node,
// so that it will be visible.
func (tv *Tree) OpenParents() {
	tv.WalkUpParent(func(k tree.Node) bool {
		tvn := AsTree(k)
		if tvn != nil {
			tvn.Open()
			return tree.Continue
		}
		return tree.Break
	})
	tv.NeedsLayout()
}

/////////////////////////////////////////////////////////////
//    Modifying Source Tree

func (tv *Tree) ContextMenuPos(e events.Event) (pos image.Point) {
	if e != nil {
		pos = e.WindowPos()
		return
	}
	pos.X = tv.Geom.TotalBBox.Min.X + int(tv.Indent.Dots)
	pos.Y = (tv.Geom.TotalBBox.Min.Y + tv.Geom.TotalBBox.Max.Y) / 2
	return
}

func (tv *Tree) ContextMenuReadOnly(m *core.Scene) {
	core.NewButton(m).SetText("Copy").SetIcon(icons.ContentCopy).SetKey(keymap.Copy).
		SetEnabled(tv.HasSelection()).
		OnClick(func(e events.Event) {
			tv.Copy(true)
		})
	core.NewButton(m).SetText("View").SetIcon(icons.Visibility).
		SetEnabled(tv.HasSelection()).
		OnClick(func(e events.Event) {
			tv.EditNode()
		})
	core.NewSeparator(m)

	NewFuncButton(m, tv.OpenAll).SetIcon(icons.KeyboardArrowDown).
		SetEnabled(tv.HasSelection())
	NewFuncButton(m, tv.CloseAll).SetIcon(icons.KeyboardArrowRight).
		SetEnabled(tv.HasSelection())
}

func (tv *Tree) ContextMenu(m *core.Scene) {
	if tv.IsReadOnly() || tv.RootIsReadOnly() {
		tv.ContextMenuReadOnly(m)
		return
	}
	tvi := tv.This.(Treer)
	core.NewButton(m).SetText("Add child").SetIcon(icons.Add).
		SetEnabled(tv.HasSelection()).
		OnClick(func(e events.Event) {
			tvi.AddChildNode()
		})
	core.NewButton(m).SetText("Insert before").SetIcon(icons.Add).
		SetEnabled(tv.HasSelection()).
		OnClick(func(e events.Event) {
			tvi.InsertBefore()
		})
	core.NewButton(m).SetText("Insert after").SetIcon(icons.Add).
		SetEnabled(tv.HasSelection()).
		OnClick(func(e events.Event) {
			tvi.InsertAfter()
		})
	core.NewButton(m).SetText("Duplicate").SetIcon(icons.ContentCopy).
		SetEnabled(tv.HasSelection()).
		OnClick(func(e events.Event) {
			tvi.Duplicate()
		})
	core.NewButton(m).SetText("Delete").SetIcon(icons.Delete).
		SetEnabled(tv.HasSelection()).
		OnClick(func(e events.Event) {
			tvi.DeleteNode()
		})
	core.NewSeparator(m)
	core.NewButton(m).SetText("Copy").SetIcon(icons.ContentCopy).SetKey(keymap.Copy).
		SetEnabled(tv.HasSelection()).
		OnClick(func(e events.Event) {
			tvi.Copy(true)
		})
	core.NewButton(m).SetText("Cut").SetIcon(icons.ContentCut).SetKey(keymap.Cut).
		SetEnabled(tv.HasSelection()).
		OnClick(func(e events.Event) {
			tvi.Cut()
		})
	pbt := core.NewButton(m).SetText("Paste").SetIcon(icons.ContentPaste).SetKey(keymap.Paste).
		OnClick(func(e events.Event) {
			tvi.Paste()
		})
	cb := tv.Scene.Events.Clipboard()
	if cb != nil {
		pbt.SetState(cb.IsEmpty(), states.Disabled)
	}
	core.NewSeparator(m)
	NewFuncButton(m, tv.EditNode).SetText("Edit").SetIcon(icons.Edit).
		SetEnabled(tv.HasSelection())
	NewFuncButton(m, tv.InspectNode).SetText("Inspect").SetIcon(icons.EditDocument).
		SetEnabled(tv.HasSelection())
	core.NewSeparator(m)

	NewFuncButton(m, tv.OpenAll).SetIcon(icons.KeyboardArrowDown).
		SetEnabled(tv.HasSelection())
	NewFuncButton(m, tv.CloseAll).SetIcon(icons.KeyboardArrowRight).
		SetEnabled(tv.HasSelection())
}

// IsRoot returns true if given node is the root of the tree.
func (tv *Tree) IsRoot(op string) bool {
	if tv.This == tv.RootView.This {
		if op != "" {
			core.MessageSnackbar(tv, fmt.Sprintf("Cannot %v the root of the tree", op))
		}
		return true
	}
	return false
}

////////////////////////////////////////////////////////////
//    Copy / Cut / Paste

// MimeData adds mimedata for this node: a text/plain of the Path.
func (tv *Tree) MimeData(md *mimedata.Mimes) {
	if tv.SyncNode != nil {
		tv.MimeDataSync(md)
		return
	}
	*md = append(*md, mimedata.NewTextData(tv.PathFrom(tv.RootView)))
	var buf bytes.Buffer
	err := jsonx.Write(tv.This, &buf)
	if err == nil {
		*md = append(*md, &mimedata.Data{Type: fileinfo.DataJson, Data: buf.Bytes()})
	} else {
		core.ErrorSnackbar(tv, err, "Error encoding node")
	}
}

// NodesFromMimeData returns a slice of tree nodes for
// the Tree nodes and paths from mime data.
func (tv *Tree) NodesFromMimeData(md mimedata.Mimes) ([]tree.Node, []string) {
	ni := len(md) / 2
	sl := make([]tree.Node, 0, ni)
	pl := make([]string, 0, ni)
	for _, d := range md {
		if d.Type == fileinfo.DataJson {
			nn, err := tree.UnmarshalRootJSON(d.Data)
			if err == nil {
				sl = append(sl, nn)
			} else {
				core.ErrorSnackbar(tv, err, "Error loading node")
			}
		} else if d.Type == fileinfo.TextPlain { // paths
			pl = append(pl, string(d.Data))
		}
	}
	return sl, pl
}

// Copy copies to system.Clipboard, optionally resetting the selection.
func (tv *Tree) Copy(reset bool) { //types:add
	sels := tv.SelectedViews()
	nitms := max(1, len(sels))
	md := make(mimedata.Mimes, 0, 2*nitms)
	tv.This.(Treer).MimeData(&md) // source is always first..
	if nitms > 1 {
		for _, sn := range sels {
			if sn != tv.This {
				sn.MimeData(&md)
			}
		}
	}
	tv.Clipboard().Write(md)
	if reset {
		tv.UnselectAll()
	}
}

// Cut copies to system.Clipboard and deletes selected items.
func (tv *Tree) Cut() { //types:add
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
	tv.UnselectAll()
	for _, sn := range sels {
		sn.AsTree().Delete()
	}
	root.Update()
	root.TreeChanged()
}

// Paste pastes clipboard at given node.
func (tv *Tree) Paste() { //types:add
	md := tv.Clipboard().Read([]string{fileinfo.DataJson})
	if md != nil {
		tv.PasteMenu(md)
	}
}

// PasteMenu performs a paste from the clipboard using given data,
// by popping up a menu to determine what specifically to do.
func (tv *Tree) PasteMenu(md mimedata.Mimes) {
	tv.UnselectAll()
	mf := func(m *core.Scene) {
		tv.This.(Treer).MakePasteMenu(m, md, nil)
	}
	pos := tv.ContextMenuPos(nil)
	core.NewMenu(mf, tv.This.(core.Widget), pos).Run()
}

// MakePasteMenu makes the menu of options for paste events
// optional function is typically the DropFinalize but could also be other actions
// to take after each optional action.
func (tv *Tree) MakePasteMenu(m *core.Scene, md mimedata.Mimes, fun func()) {
	core.NewButton(m).SetText("Assign To").OnClick(func(e events.Event) {
		tv.PasteAssign(md)
		if fun != nil {
			fun()
		}
	})
	core.NewButton(m).SetText("Add to Children").OnClick(func(e events.Event) {
		tv.PasteChildren(md, events.DropCopy)
		if fun != nil {
			fun()
		}
	})
	if !tv.IsRoot("") && tv.RootView.This != tv.This {
		core.NewButton(m).SetText("Insert Before").OnClick(func(e events.Event) {
			tv.PasteBefore(md, events.DropCopy)
			if fun != nil {
				fun()
			}
		})
		core.NewButton(m).SetText("Insert After").OnClick(func(e events.Event) {
			tv.PasteAfter(md, events.DropCopy)
			if fun != nil {
				fun()
			}
		})
	}
	core.NewButton(m).SetText("Cancel")
}

// PasteAssign assigns mime data (only the first one!) to this node
func (tv *Tree) PasteAssign(md mimedata.Mimes) {
	if tv.SyncNode != nil {
		tv.PasteAssignSync(md)
		return
	}
	sl, _ := tv.NodesFromMimeData(md)
	if len(sl) == 0 {
		return
	}
	tv.CopyFrom(sl[0])    // nodes with data copy here
	tv.SetScene(tv.Scene) // ensure children have scene
	tv.Update()           // could have children
	tv.Open()
	tv.TreeChanged()
}

// PasteBefore inserts object(s) from mime data before this node.
// If another item with the same name already exists, it will
// append _Copy on the name of the inserted objects
func (tv *Tree) PasteBefore(md mimedata.Mimes, mod events.DropMods) {
	tv.PasteAt(md, mod, 0, "Paste Before")
}

// PasteAfter inserts object(s) from mime data after this node.
// If another item with the same name already exists, it will
// append _Copy on the name of the inserted objects
func (tv *Tree) PasteAfter(md mimedata.Mimes, mod events.DropMods) {
	tv.PasteAt(md, mod, 1, "Paste After")
}

// TreeTempMovedTag is a kind of hack to prevent moved items from being deleted, using DND
const TreeTempMovedTag = `_\&MOVED\&`

// todo: these methods require an interface to work for descended
// nodes, based on base code

// PasteAt inserts object(s) from mime data at rel position to this node.
// If another item with the same name already exists, it will
// append _Copy on the name of the inserted objects
func (tv *Tree) PasteAt(md mimedata.Mimes, mod events.DropMods, rel int, actNm string) {
	if tv.Parent == nil {
		return
	}
	parent := AsTree(tv.Parent)
	if parent == nil {
		core.MessageSnackbar(tv, "Error: cannot insert after the root of the tree")
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
		_, nwb := core.AsWidget(ns)
		ntv := AsTree(ns)
		ntv.RootView = tv.RootView
		nwb.SetScene(tv.Scene)
		nwb.Update() // incl children
		npath := ns.AsTree().PathFrom(tv.RootView)
		if mod == events.DropMove && npath == orgpath { // we will be nuked immediately after drag
			ns.AsTree().SetName(ns.AsTree().Name + TreeTempMovedTag) // special keyword :)
		}
		if i == sz-1 {
			selTv = ntv
		}
	}
	tv.TreeChanged()
	parent.NeedsLayout()
	if selTv != nil {
		selTv.SelectAction(events.SelectOne)
	}
}

// PasteChildren inserts object(s) from mime data
// at end of children of this node
func (tv *Tree) PasteChildren(md mimedata.Mimes, mod events.DropMods) {
	if tv.SyncNode != nil {
		tv.PasteChildrenSync(md, mod)
		return
	}
	sl, _ := tv.NodesFromMimeData(md)

	for _, ns := range sl {
		tv.AddChild(ns)
		_, nwb := core.AsWidget(ns)
		ntv := AsTree(ns)
		ntv.RootView = tv.RootView
		nwb.SetScene(tv.Scene)
	}
	tv.Update()
	tv.Open()
	tv.TreeChanged()
}

//////////////////////////////////////////////////////////////////////////////
//    Drag-n-Drop

// DragStart starts a drag-n-drop on this node -- it includes any other
// selected nodes as well, each as additional records in mimedata.
func (tv *Tree) DragStart(e events.Event) {
	sels := tv.SelectedViews()
	nitms := max(1, len(sels))
	md := make(mimedata.Mimes, 0, 2*nitms)
	tv.This.(Treer).MimeData(&md) // source is always first..
	if nitms > 1 {
		for _, sn := range sels {
			if sn != tv.This {
				sn.MimeData(&md)
			}
		}
	}
	tv.Scene.Events.DragStart(tv.This.(core.Widget), md, e)
}

// DropExternal is not handled by base case but could be in derived
func (tv *Tree) DropExternal(md mimedata.Mimes, mod events.DropMods) {
	// todo: not yet implemented
}

// DragClearStates clears the drag-drop related states for this widget
func (tv *Tree) DragClearStates() {
	tv.SetState(false, states.Active, states.Selected, states.Hovered, states.DragHovered)
	tv.Parts.SetState(false, states.Active, states.Selected, states.Hovered, states.DragHovered)
	tv.Style()
	tv.NeedsRender()
}

// DragDrop handles drag drop event
func (tv *Tree) DragDrop(e events.Event) {
	// todo: some kind of validation for source
	tvi := tv.This.(Treer)
	tv.UnselectAll()
	de := e.(*events.DragDrop)
	stv := AsTree(de.Source.(core.Widget))
	if stv != nil {
		stv.DragClearStates()
	}
	md := de.Data.(mimedata.Mimes)
	mf := func(m *core.Scene) {
		tv.Scene.Events.DragMenuAddModText(m, de.DropMod)
		tvi.MakePasteMenu(m, md, func() {
			tvi.DropFinalize(de)
		})
	}
	pos := tv.ContextMenuPos(nil)
	core.NewMenu(mf, tv.This.(core.Widget), pos).Run()
}

// DropFinalize is called to finalize Drop actions on the Source node.
// Only relevant for DropMod == DropMove.
func (tv *Tree) DropFinalize(de *events.DragDrop) {
	tv.UnselectAll()
	tv.DragClearStates()
	tv.Scene.Events.DropFinalize(de) // sends DropDeleteSource to Source
}

// DropDeleteSource handles delete source event for DropMove case
func (tv *Tree) DropDeleteSource(e events.Event) {
	de := e.(*events.DragDrop)
	tv.UnselectAll()
	if tv.SyncNode != nil {
		tv.DropDeleteSourceSync(de)
		return
	}
	md := de.Data.(mimedata.Mimes)
	root := tv.RootView
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
			_, swb := core.AsWidget(sn)
			swb.NeedsRender()
		}
	}
}
