// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"bytes"
	"fmt"
	"image"
	"log/slog"
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
	"cogentcore.org/core/text/text"
	"cogentcore.org/core/tree"
)

// Treer is an interface for [Tree] types
// providing access to the base [Tree] and
// overridable method hooks for actions taken on the [Tree],
// including OnOpen, OnClose, etc.
type Treer interface { //types:add
	Widget

	// AsTree returns the base [Tree] for this node.
	AsCoreTree() *Tree

	// CanOpen returns true if the node is able to open.
	// By default it checks HasChildren(), but could check other properties
	// to perform lazy building of the tree.
	CanOpen() bool

	// OnOpen is called when a node is toggled open.
	// The base version does nothing.
	OnOpen()

	// OnClose is called when a node is toggled closed.
	// The base version does nothing.
	OnClose()

	// The following are all tree editing functions:

	MimeData(md *mimedata.Mimes)
	Cut()
	Copy()
	Paste()
	DeleteSelected()
	DragDrop(e events.Event)
	DropDeleteSource(e events.Event)
}

// AsTree returns the given value as a [Tree] if it has
// an AsCoreTree() method, or nil otherwise.
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
// separately from the rest of the surrounding context, you must
// place it in a [Frame].
//
// If the [Tree.SyncNode] field is non-nil, typically via the
// [Tree.SyncTree] method, then the Tree mirrors another
// tree structure, and tree editing functions apply to
// the source tree first, and then to the Tree by sync.
//
// Otherwise, data can be directly encoded in a Tree
// derived type, to represent any kind of tree structure
// and associated data.
//
// Standard [events.Event]s are sent to any listeners, including
// [events.Select], [events.Change], and [events.DoubleClick].
// The selected nodes are in the root [Tree.SelectedNodes] list;
// select events are sent to both selected nodes and the root node.
// See [Tree.IsRootSelected] to check whether a select event on the root
// node corresponds to the root node or another node.
type Tree struct {
	WidgetBase

	// SyncNode, if non-nil, is the [tree.Node] that this widget is
	// viewing in the tree (the source). It should be set using
	// [Tree.SyncTree].
	SyncNode tree.Node `set:"-" copier:"-" json:"-" xml:"-"`

	// Text is the text to display for the tree item label, which automatically
	// defaults to the [tree.Node.Name] of the tree node. It has no effect
	// if [Tree.SyncNode] is non-nil.
	Text string

	// Icon is an optional icon displayed to the the left of the text label.
	Icon icons.Icon

	// IconOpen is the icon to use for an open (expanded) branch;
	// it defaults to [icons.KeyboardArrowDown].
	IconOpen icons.Icon

	// IconClosed is the icon to use for a closed (collapsed) branch;
	// it defaults to [icons.KeyboardArrowRight].
	IconClosed icons.Icon

	// IconLeaf is the icon to use for a terminal node branch that has no children;
	// it defaults to [icons.Blank].
	IconLeaf icons.Icon

	// TreeInit is a function that can be set on the root node that is called
	// with each child tree node when it is initialized. It is only
	// called with the root node itself in [Tree.SetTreeInit], so you
	// should typically call that instead of setting this directly.
	TreeInit func(tr *Tree) `set:"-" json:"-" xml:"-"`

	// Indent is the amount to indent children relative to this node.
	// It should be set in a Styler like all other style properties.
	Indent units.Value `copier:"-" json:"-" xml:"-"`

	// OpenDepth is the depth for nodes be initialized as open (default 4).
	// Nodes beyond this depth will be initialized as closed.
	OpenDepth int `copier:"-" json:"-" xml:"-"`

	// Closed is whether this tree node is currently toggled closed
	// (children not visible).
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

	// Root is the cached root of the tree. It is automatically set.
	Root Treer `copier:"-" json:"-" xml:"-" edit:"-" set:"-"`

	// SelectedNodes holds the currently selected nodes.
	// It is only set on the root node. See [Tree.GetSelectedNodes]
	// for a version that also works on non-root nodes.
	SelectedNodes []Treer `copier:"-" json:"-" xml:"-" edit:"-" set:"-"`

	// actStateLayer is the actual state layer of the tree, which
	// should be used when rendering it and its parts (but not its children).
	// the reason that it exists is so that the children of the tree
	// (other trees) do not inherit its stateful background color, as
	// that does not look good.
	actStateLayer float32

	// inOpen is set in the Open method to prevent recursive opening for lazy-open nodes.
	inOpen bool

	// Branch is the branch widget that is used to open and close the tree node.
	Branch *Switch `json:"-" xml:"-" copier:"-" set:"-" display:"-"`
}

// AsCoreTree satisfies the [Treer] interface.
func (tr *Tree) AsCoreTree() *Tree {
	return tr
}

// rootSetViewIndex sets the [Tree.root] and [Tree.viewIndex] for all nodes.
// It returns the total number of leaves in the tree.
func (tr *Tree) rootSetViewIndex() int {
	idx := 0
	tr.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
		tvn := AsTree(cw)
		if tvn != nil {
			tvn.viewIndex = idx
			if tvn.Root == nil {
				tvn.Root = tr.This.(Treer)
			}
			idx++
		}
		return tree.Continue
	})
	return idx
}

func (tr *Tree) Init() {
	tr.WidgetBase.Init()
	tr.AddContextMenu(tr.contextMenu)
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
		s.Padding.Left.Dp(ConstantSpacing(4))
		s.Padding.SetVertical(units.Dp(4))
		s.Padding.Right.Zero()
		s.Text.Align = text.Start
		if tr.Root == nil || tr.Root == tr.This {
			s.IconSize.Set(units.Em(1.5))
		} else {
			s.IconSize = tr.Root.AsWidget().Styles.IconSize
		}

		// need to copy over to actual and then clear styles one
		if s.Is(states.Selected) {
			// render handles manually, similar to with actStateLayer
			s.Background = nil
		} else {
			s.Color = colors.Scheme.OnSurface
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
	tr.On(events.DropDeleteSource, func(e events.Event) { tr.This.(Treer).DropDeleteSource(e) })
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

		tri := tr.This.(Treer)

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
			tr.moveDownEvent(selMode)
			e.SetHandled()
		case keymap.MoveUp:
			tr.moveUpEvent(selMode)
			e.SetHandled()
		case keymap.PageUp:
			tr.movePageUpEvent(selMode)
			e.SetHandled()
		case keymap.PageDown:
			tr.movePageDownEvent(selMode)
			e.SetHandled()
		case keymap.Home:
			tr.moveHomeEvent(selMode)
			e.SetHandled()
		case keymap.End:
			tr.moveEndEvent(selMode)
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
			tri.Copy()
			e.SetHandled()
		}
		if !tr.rootIsReadOnly() && !e.IsHandled() {
			switch kf {
			case keymap.Delete:
				tr.DeleteSelected()
				e.SetHandled()
			case keymap.Duplicate:
				tr.Duplicate()
				e.SetHandled()
			case keymap.Insert:
				tr.InsertBefore()
				e.SetHandled()
			case keymap.InsertAfter:
				tr.InsertAfter()
				e.SetHandled()
			case keymap.Cut:
				tri.Cut()
				e.SetHandled()
			case keymap.Paste:
				tri.Paste()
				e.SetHandled()
			}
		}
	})

	parts := tr.newParts()
	tri := tr.This.(Treer)
	parts.Styler(func(s *styles.Style) {
		s.Cursor = cursors.Pointer
		s.SetAbilities(true, abilities.Activatable, abilities.Focusable, abilities.Selectable, abilities.Hoverable, abilities.DoubleClickable)
		s.SetAbilities(!tr.IsReadOnly() && !tr.rootIsReadOnly(), abilities.Draggable, abilities.Droppable)
		s.Gap.X.Em(0.1)
		s.Padding.Zero()
		s.Align.Items = styles.Center

		// we manually inherit our state layer from the tree state
		// layer so that the parts get it but not the other trees
		s.StateLayer = tr.actStateLayer
	})
	parts.AsWidget().FinalStyler(func(s *styles.Style) {
		s.Grow.Set(1, 0)
	})
	// we let the parts handle our state
	// so that we only get it when we are doing
	// something with this tree specifically,
	// not with any of our children (see HandleTreeMouse)
	parts.On(events.MouseEnter, func(e events.Event) {
		tr.SetState(true, states.Hovered)
		tr.Style()
		tr.NeedsRender()
		e.SetHandled()
	})
	parts.On(events.MouseLeave, func(e events.Event) {
		tr.SetState(false, states.Hovered)
		tr.Style()
		tr.NeedsRender()
		e.SetHandled()
	})
	parts.On(events.MouseDown, func(e events.Event) {
		tr.SetState(true, states.Active)
		tr.Style()
		tr.NeedsRender()
		e.SetHandled()
	})
	parts.On(events.MouseUp, func(e events.Event) {
		tr.SetState(false, states.Active)
		tr.Style()
		tr.NeedsRender()
		e.SetHandled()
	})
	parts.OnClick(func(e events.Event) {
		tr.SelectEvent(e.SelectMode())
		e.SetHandled()
	})
	parts.AsWidget().OnDoubleClick(func(e events.Event) {
		if tr.HasChildren() {
			tr.ToggleClose()
		}
	})
	parts.On(events.DragStart, func(e events.Event) {
		tr.dragStart(e)
	})
	parts.On(events.DragEnter, func(e events.Event) {
		tr.SetState(true, states.DragHovered)
		tr.Style()
		tr.NeedsRender()
		e.SetHandled()
	})
	parts.On(events.DragLeave, func(e events.Event) {
		tr.SetState(false, states.DragHovered)
		tr.Style()
		tr.NeedsRender()
		e.SetHandled()
	})
	parts.On(events.Drop, func(e events.Event) {
		tri.DragDrop(e)
	})
	parts.On(events.DropDeleteSource, func(e events.Event) {
		tri.DropDeleteSource(e)
	})
	// the context menu events will get sent to the parts, so it
	// needs to intercept them and send them up
	parts.On(events.ContextMenu, func(e events.Event) {
		sels := tr.GetSelectedNodes()
		if len(sels) == 0 {
			tr.SelectEvent(e.SelectMode())
		}
		tr.ShowContextMenu(e)
	})
	tree.AddChildAt(parts, "branch", func(w *Switch) {
		tr.Branch = w
		w.SetType(SwitchCheckbox)
		w.SetIconOn(tr.IconOpen).SetIconOff(tr.IconClosed).SetIconIndeterminate(tr.IconLeaf)
		w.Styler(func(s *styles.Style) {
			s.SetAbilities(false, abilities.Focusable)
			// parent will handle our cursor
			s.Cursor = cursors.None
			s.Color = colors.Scheme.Primary.Base
			s.Padding.Zero()
			s.IconSize = tr.Styles.IconSize
			s.Align.Self = styles.Center
			if !w.StateIs(states.Indeterminate) {
				// we amplify any state layer we receiver so that it is clear
				// we are receiving it, not just our parent
				s.StateLayer *= 3
			} else {
				// no abilities and state layer for indeterminate because
				// they are not interactive
				s.Abilities = 0
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
				tr.setBranchState()
			}
		})
	})
	parts.Maker(func(p *tree.Plan) {
		if tr.Icon.IsSet() {
			tree.AddAt(p, "icon", func(w *Icon) {
				w.Styler(func(s *styles.Style) {
					s.Color = colors.Scheme.Primary.Base
					s.Align.Self = styles.Center
				})
				w.Updater(func() {
					w.SetIcon(tr.Icon)
				})
			})
		}
	})
	tree.AddChildAt(parts, "text", func(w *Text) {
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
	// note: this causes excessive updates and is not recommended. use Resync() instead.
	// tr.Updater(func() {
	// 	if tr.SyncNode != nil {
	// 		tr.syncToSrc(&tr.viewIndex, false, 0)
	// 	}
	// })
}

func (tr *Tree) OnAdd() {
	tr.WidgetBase.OnAdd()
	tr.Text = tr.Name
	if ptv := AsTree(tr.Parent); ptv != nil {
		tr.Root = ptv.Root
		tr.IconOpen = ptv.IconOpen
		tr.IconClosed = ptv.IconClosed
		tr.IconLeaf = ptv.IconLeaf
	} else {
		if tr.Root == nil {
			tr.Root = tr.This.(Treer)
		}
	}
	troot := tr.Root.AsCoreTree()
	if troot.TreeInit != nil {
		troot.TreeInit(tr)
	}
}

// SetTreeInit sets the [Tree.TreeInit]:
// TreeInit is a function that can be set on the root node that is called
// with each child tree node when it is initialized. It is only
// called with the root node itself in this function, SetTreeInit, so you
// should typically call this instead of setting it directly.
func (tr *Tree) SetTreeInit(v func(tr *Tree)) *Tree {
	tr.TreeInit = v
	v(tr)
	return tr
}

// rootIsReadOnly returns the ReadOnly status of the root node,
// which is what controls the functional inactivity of the tree
// if individual nodes are ReadOnly that only affects display typically.
func (tr *Tree) rootIsReadOnly() bool {
	if tr.Root == nil {
		return true
	}
	return tr.Root.AsCoreTree().IsReadOnly()
}

func (tr *Tree) Style() {
	if !tr.HasChildren() {
		tr.SetClosed(true)
	}
	tr.WidgetBase.Style()
	tr.Indent.ToDots(&tr.Styles.UnitContext)
	tr.Indent.Dots = math32.Ceil(tr.Indent.Dots)
}

func (tr *Tree) setBranchState() {
	br := tr.Branch
	if br == nil {
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
	if tr.IsRoot() { // do it every time on root
		tr.rootSetViewIndex()
	}

	if !tr.Closed {
		// we layout children under us
		tr.ForWidgetChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
			cw.SizeUp()
			h += cwb.Geom.Size.Actual.Total.Y
			kw := cwb.Geom.Size.Actual.Total.X
			if math32.IsNaN(kw) { // somehow getting a nan
				slog.Error("Tree, node width is NaN", "node:", cwb)
			} else {
				w = max(w, tr.Indent.Dots+kw)
			}
			// fmt.Println(kwb, w, h)
			return tree.Continue
		})
	}
	sz := &tr.Geom.Size
	sz.Actual.Content = math32.Vec2(w, h)
	sz.setTotalFromContent(&sz.Actual)
	sz.Alloc = sz.Actual // need allocation to match!
	tr.widgetSize.X = w  // stretch
}

func (tr *Tree) SizeDown(iter int) bool {
	// note: key to not grab the whole allocation, as widget default does
	redo := tr.sizeDownParts(iter) // give our content to parts
	re := tr.sizeDownChildren(iter)
	return redo || re
}

func (tr *Tree) Position() {
	if tr.Root == nil {
		slog.Error("core.Tree: RootView is nil", "in node:", tr)
		return
	}
	rn := tr.Root.AsCoreTree()
	tr.setBranchState()
	sz := &tr.Geom.Size
	sz.Actual.Total.X = rn.Geom.Size.Actual.Total.X - (tr.Geom.Pos.Total.X - rn.Geom.Pos.Total.X)
	sz.Actual.Content.X = sz.Actual.Total.X - sz.Space.X
	tr.widgetSize.X = sz.Actual.Total.X
	sz.Alloc = sz.Actual
	psz := &tr.Parts.Geom.Size
	psz.Alloc.Total = tr.widgetSize
	psz.setContentFromTotal(&psz.Alloc)

	tr.WidgetBase.Position() // just does our parts

	if !tr.Closed {
		h := tr.widgetSize.Y
		tr.ForWidgetChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
			cwb.Geom.RelPos.Y = h
			cwb.Geom.RelPos.X = tr.Indent.Dots
			h += cwb.Geom.Size.Actual.Total.Y
			cw.Position()
			return tree.Continue
		})
	}
}

func (tr *Tree) ApplyScenePos() {
	sz := &tr.Geom.Size
	if sz.Actual.Total == tr.widgetSize {
		sz.setTotalFromContent(&sz.Actual) // restore after scrolling
	}
	tr.WidgetBase.ApplyScenePos()
	tr.applyScenePosChildren()
	sz.Actual.Total = tr.widgetSize // key: we revert to just ourselves
}

func (tr *Tree) Render() {
	pc := &tr.Scene.Painter
	st := &tr.Styles

	pabg := tr.parentActualBackground()

	// must use workaround act values
	st.StateLayer = tr.actStateLayer
	if st.Is(states.Selected) {
		st.Background = colors.Scheme.Select.Container
	}
	tr.Styles.ComputeActualBackground(pabg)

	pc.StandardBox(st, tr.Geom.Pos.Total, tr.Geom.Size.Actual.Total, pabg)

	// after we are done rendering, we clear the values so they aren't inherited
	st.StateLayer = 0
	st.Background = nil
	tr.Styles.ComputeActualBackground(pabg)
}

func (tr *Tree) RenderWidget() {
	if tr.StartRender() {
		tr.Render()
		if tr.Parts != nil {
			// we must copy from actual values in parent
			tr.Parts.Styles.StateLayer = tr.actStateLayer
			if tr.StateIs(states.Selected) {
				tr.Parts.Styles.Background = colors.Scheme.Select.Container
			}
			tr.renderParts()
		}
		tr.EndRender()
	}
	// We have to render our children outside of `if StartRender`
	// since we could be out of scope but they could still be in!
	if !tr.Closed {
		tr.renderChildren()
	}
}

////////  Selection

// IsRootSelected returns whether the root node is the only node selected.
// This can be used in [events.Select] event handlers to check whether a
// select event on the root node truly corresponds to the root node or whether
// it is for another node, as select events are sent to the root when any node
// is selected.
func (tr *Tree) IsRootSelected() bool {
	return len(tr.SelectedNodes) == 1 && tr.SelectedNodes[0] == tr.Root
}

// GetSelectedNodes returns a slice of the currently selected
// Trees within the entire tree, using a list maintained
// by the root node.
func (tr *Tree) GetSelectedNodes() []Treer {
	if tr.Root == nil {
		return nil
	}
	rn := tr.Root.AsCoreTree()
	if len(rn.SelectedNodes) == 0 {
		return rn.SelectedNodes
	}
	return rn.SelectedNodes
}

// SetSelectedNodes updates the selected nodes on the root node to the given list.
func (tr *Tree) SetSelectedNodes(sl []Treer) {
	if tr.Root != nil {
		tr.Root.AsCoreTree().SelectedNodes = sl
	}
}

// HasSelection returns whether there are currently selected items.
func (tr *Tree) HasSelection() bool {
	return len(tr.GetSelectedNodes()) > 0
}

// Select selects this node (if not already selected).
// You must use this method to update global selection list.
func (tr *Tree) Select() {
	if !tr.StateIs(states.Selected) {
		tr.SetSelected(true)
		tr.Style()
		sl := tr.GetSelectedNodes()
		sl = append(sl, tr.This.(Treer))
		tr.SetSelectedNodes(sl)
		tr.NeedsRender()
	}
}

// Unselect unselects this node (if selected).
// You must use this method to update global selection list.
func (tr *Tree) Unselect() {
	if tr.StateIs(states.Selected) {
		tr.SetSelected(false)
		tr.Style()
		sl := tr.GetSelectedNodes()
		sz := len(sl)
		for i := 0; i < sz; i++ {
			if sl[i] == tr {
				sl = append(sl[:i], sl[i+1:]...)
				break
			}
		}
		tr.SetSelectedNodes(sl)
		tr.NeedsRender()
	}
}

// UnselectAll unselects all selected items in the tree.
func (tr *Tree) UnselectAll() {
	if tr.Scene == nil {
		return
	}
	sl := tr.GetSelectedNodes()
	tr.SetSelectedNodes(nil) // clear in advance
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

// SelectAll selects all items in the tree.
func (tr *Tree) SelectAll() {
	if tr.Scene == nil {
		return
	}
	tr.UnselectAll()
	nn := tr.Root.AsCoreTree()
	nn.Select()
	for nn != nil {
		nn = nn.moveDown(events.SelectQuiet)
	}
	tr.NeedsRender()
}

// selectUpdate updates selection to include this node,
// using selectmode from mouse event (ExtendContinuous, ExtendOne).
// Returns true if this node selected.
func (tr *Tree) selectUpdate(mode events.SelectModes) bool {
	if mode == events.NoSelect {
		return false
	}
	sel := false
	switch mode {
	case events.SelectOne:
		if tr.StateIs(states.Selected) {
			sl := tr.GetSelectedNodes()
			if len(sl) > 1 {
				tr.UnselectAll()
				tr.Select()
				tr.SetFocusQuiet()
				sel = true
			}
		} else {
			tr.UnselectAll()
			tr.Select()
			tr.SetFocusQuiet()
			sel = true
		}
	case events.ExtendContinuous:
		sl := tr.GetSelectedNodes()
		if len(sl) == 0 {
			tr.Select()
			tr.SetFocusQuiet()
			sel = true
		} else {
			minIndex := -1
			maxIndex := 0
			sel = true
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
					nn = nn.moveDown(events.SelectQuiet) // just select
					cidx = nn.viewIndex
				}
			} else if tr.viewIndex > maxIndex {
				for cidx > maxIndex {
					nn = nn.moveUp(events.SelectQuiet) // just select
					cidx = nn.viewIndex
				}
			}
		}
	case events.ExtendOne:
		if tr.StateIs(states.Selected) {
			tr.UnselectEvent()
		} else {
			tr.Select()
			tr.SetFocusQuiet()
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

// sendSelectEvent sends an [events.Select] event on both this node and the root node.
func (tr *Tree) sendSelectEvent(original ...events.Event) {
	if !tr.IsRoot() {
		tr.Send(events.Select, original...)
	}
	tr.Root.AsCoreTree().Send(events.Select, original...)
}

// sendChangeEvent sends an [events.Change] event on both this node and the root node.
func (tr *Tree) sendChangeEvent(original ...events.Event) {
	if !tr.IsRoot() {
		tr.SendChange(original...)
	}
	tr.Root.AsCoreTree().SendChange(original...)
}

// sendChangeEventReSync sends an [events.Change] event on the RootView node.
// If SyncNode != nil, it also does a re-sync from root.
func (tr *Tree) sendChangeEventReSync(original ...events.Event) {
	tr.sendChangeEvent(original...)
	rn := tr.Root.AsCoreTree()
	if rn.SyncNode != nil {
		rn.Resync()
	}
}

// SelectEvent updates selection to include this node,
// using selectmode from mouse event (ExtendContinuous, ExtendOne),
// and root sends selection event. Returns true if event sent.
func (tr *Tree) SelectEvent(mode events.SelectModes) bool {
	sel := tr.selectUpdate(mode)
	if sel {
		tr.sendSelectEvent()
	}
	return sel
}

// UnselectEvent unselects this node (if selected),
// and root sends a selection event.
func (tr *Tree) UnselectEvent() {
	if tr.StateIs(states.Selected) {
		tr.Unselect()
		tr.sendSelectEvent()
	}
}

////////  Moving

// moveDown moves the selection down to next element in the tree,
// using given select mode (from keyboard modifiers).
// Returns newly selected node.
func (tr *Tree) moveDown(selMode events.SelectModes) *Tree {
	if tr.Parent == nil {
		return nil
	}
	if tr.Closed || !tr.HasChildren() { // next sibling
		return tr.moveDownSibling(selMode)
	}
	if tr.HasChildren() {
		nn := AsTree(tr.Child(0))
		if nn != nil {
			nn.selectUpdate(selMode)
			return nn
		}
	}
	return nil
}

// moveDownEvent moves the selection down to next element in the tree,
// using given select mode (from keyboard modifiers).
// Sends select event for newly selected item.
func (tr *Tree) moveDownEvent(selMode events.SelectModes) *Tree {
	nn := tr.moveDown(selMode)
	if nn != nil && nn != tr {
		nn.SetFocusQuiet()
		nn.ScrollToThis()
		tr.sendSelectEvent()
	}
	return nn
}

// moveDownSibling moves down only to siblings, not down into children,
// using given select mode (from keyboard modifiers)
func (tr *Tree) moveDownSibling(selMode events.SelectModes) *Tree {
	if tr.Parent == nil {
		return nil
	}
	if tr == tr.Root {
		return nil
	}
	myidx := tr.IndexInParent()
	if myidx < len(tr.Parent.AsTree().Children)-1 {
		nn := AsTree(tr.Parent.AsTree().Child(myidx + 1))
		if nn != nil {
			nn.selectUpdate(selMode)
			return nn
		}
	} else {
		return AsTree(tr.Parent).moveDownSibling(selMode) // try up
	}
	return nil
}

// moveUp moves selection up to previous element in the tree,
// using given select mode (from keyboard modifiers).
// Returns newly selected node
func (tr *Tree) moveUp(selMode events.SelectModes) *Tree {
	if tr.Parent == nil || tr == tr.Root {
		return nil
	}
	myidx := tr.IndexInParent()
	if myidx > 0 {
		nn := AsTree(tr.Parent.AsTree().Child(myidx - 1))
		if nn != nil {
			return nn.moveToLastChild(selMode)
		}
	} else {
		if tr.Parent != nil {
			nn := AsTree(tr.Parent)
			if nn != nil {
				nn.selectUpdate(selMode)
				return nn
			}
		}
	}
	return nil
}

// moveUpEvent moves the selection up to previous element in the tree,
// using given select mode (from keyboard modifiers).
// Sends select event for newly selected item.
func (tr *Tree) moveUpEvent(selMode events.SelectModes) *Tree {
	nn := tr.moveUp(selMode)
	if nn != nil && nn != tr {
		nn.SetFocusQuiet()
		nn.ScrollToThis()
		tr.sendSelectEvent()
	}
	return nn
}

// treePageSteps is the number of steps to take in PageUp / Down events
const treePageSteps = 10

// movePageUpEvent moves the selection up to previous
// TreePageSteps elements in the tree,
// using given select mode (from keyboard modifiers).
// Sends select event for newly selected item.
func (tr *Tree) movePageUpEvent(selMode events.SelectModes) *Tree {
	mvMode := selMode
	if selMode == events.SelectOne {
		mvMode = events.NoSelect
	} else if selMode == events.ExtendContinuous || selMode == events.ExtendOne {
		mvMode = events.SelectQuiet
	}
	fnn := tr.moveUp(mvMode)
	if fnn != nil && fnn != tr {
		for i := 1; i < treePageSteps; i++ {
			nn := fnn.moveUp(mvMode)
			if nn == nil || nn == fnn {
				break
			}
			fnn = nn
		}
		if selMode == events.SelectOne {
			fnn.selectUpdate(selMode)
		}
		fnn.SetFocusQuiet()
		fnn.ScrollToThis()
		tr.sendSelectEvent()
	}
	tr.NeedsRender()
	return fnn
}

// movePageDownEvent moves the selection up to
// previous TreePageSteps elements in the tree,
// using given select mode (from keyboard modifiers).
// Sends select event for newly selected item.
func (tr *Tree) movePageDownEvent(selMode events.SelectModes) *Tree {
	mvMode := selMode
	if selMode == events.SelectOne {
		mvMode = events.NoSelect
	} else if selMode == events.ExtendContinuous || selMode == events.ExtendOne {
		mvMode = events.SelectQuiet
	}
	fnn := tr.moveDown(mvMode)
	if fnn != nil && fnn != tr {
		for i := 1; i < treePageSteps; i++ {
			nn := fnn.moveDown(mvMode)
			if nn == nil || nn == fnn {
				break
			}
			fnn = nn
		}
		if selMode == events.SelectOne {
			fnn.selectUpdate(selMode)
		}
		fnn.SetFocusQuiet()
		fnn.ScrollToThis()
		tr.sendSelectEvent()
	}
	tr.NeedsRender()
	return fnn
}

// moveToLastChild moves to the last child under me, using given select mode
// (from keyboard modifiers)
func (tr *Tree) moveToLastChild(selMode events.SelectModes) *Tree {
	if tr.Parent == nil || tr == tr.Root {
		return nil
	}
	if !tr.Closed && tr.HasChildren() {
		nn := AsTree(tr.Child(tr.NumChildren() - 1))
		return nn.moveToLastChild(selMode)
	}
	tr.selectUpdate(selMode)
	return tr
}

// moveHomeEvent moves the selection up to top of the tree,
// using given select mode (from keyboard modifiers)
// and emits select event for newly selected item
func (tr *Tree) moveHomeEvent(selMode events.SelectModes) *Tree {
	rn := tr.Root.AsCoreTree()
	rn.selectUpdate(selMode)
	rn.SetFocusQuiet()
	rn.ScrollToThis()
	rn.sendSelectEvent()
	return rn
}

// moveEndEvent moves the selection to the very last node in the tree,
// using given select mode (from keyboard modifiers)
// Sends select event for newly selected item.
func (tr *Tree) moveEndEvent(selMode events.SelectModes) *Tree {
	mvMode := selMode
	if selMode == events.SelectOne {
		mvMode = events.NoSelect
	} else if selMode == events.ExtendContinuous || selMode == events.ExtendOne {
		mvMode = events.SelectQuiet
	}
	fnn := tr.moveDown(mvMode)
	if fnn != nil && fnn != tr {
		for {
			nn := fnn.moveDown(mvMode)
			if nn == nil || nn == fnn {
				break
			}
			fnn = nn
		}
		if selMode == events.SelectOne {
			fnn.selectUpdate(selMode)
		}
		fnn.SetFocusQuiet()
		fnn.ScrollToThis()
		tr.sendSelectEvent()
	}
	return fnn
}

func (tr *Tree) setChildrenVisibility(parentClosed bool) {
	for _, c := range tr.Children {
		tvn := AsTree(c)
		if tvn != nil {
			tvn.SetState(parentClosed, states.Invisible)
		}
	}
}

// OnClose is called when a node is closed.
// The base version does nothing.
func (tr *Tree) OnClose() {}

// Close closes the given node and updates the tree accordingly
// (if it is not already closed). It calls OnClose in the [Treer]
// interface for extensible actions.
func (tr *Tree) Close() {
	if tr.Closed {
		return
	}
	tr.SetClosed(true)
	tr.setBranchState()
	tr.This.(Treer).OnClose()
	tr.setChildrenVisibility(true) // parent closed
	tr.NeedsLayout()
}

// OnOpen is called when a node is opened.
// The base version does nothing.
func (tr *Tree) OnOpen() {}

// CanOpen returns true if the node is able to open.
// By default it checks HasChildren(), but could check other properties
// to perform lazy building of the tree.
func (tr *Tree) CanOpen() bool {
	return tr.HasChildren()
}

// Open opens the given node and updates the tree accordingly
// (if it is not already opened). It calls OnOpen in the [Treer]
// interface for extensible actions.
func (tr *Tree) Open() {
	if !tr.Closed || tr.inOpen || tr.This == nil {
		return
	}
	tr.inOpen = true
	if tr.This.(Treer).CanOpen() {
		tr.SetClosed(false)
		tr.setBranchState()
		tr.setChildrenVisibility(false)
		tr.This.(Treer).OnOpen()
	}
	tr.inOpen = false
	tr.NeedsLayout()
}

// ToggleClose toggles the close / open status: if closed, opens, and vice-versa.
func (tr *Tree) ToggleClose() {
	if tr.Closed {
		tr.Open()
	} else {
		tr.Close()
	}
}

// OpenAll opens the node and all of its sub-nodes.
func (tr *Tree) OpenAll() { //types:add
	tr.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
		tvn := AsTree(cw)
		if tvn != nil {
			tvn.Open()
			return tree.Continue
		}
		return tree.Break
	})
	tr.NeedsLayout()
}

// CloseAll closes the node and all of its sub-nodes.
func (tr *Tree) CloseAll() { //types:add
	tr.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
		tvn := AsTree(cw)
		if tvn != nil {
			tvn.Close()
			return tree.Continue
		}
		return tree.Break
	})
	tr.NeedsLayout()
}

// OpenParents opens all the parents of this node
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

////////  Modifying Source Tree

func (tr *Tree) ContextMenuPos(e events.Event) (pos image.Point) {
	if e != nil {
		pos = e.WindowPos()
		return
	}
	pos.X = tr.Geom.TotalBBox.Min.X + int(tr.Indent.Dots)
	pos.Y = (tr.Geom.TotalBBox.Min.Y + tr.Geom.TotalBBox.Max.Y) / 2
	return
}

func (tr *Tree) contextMenuReadOnly(m *Scene) {
	tri := tr.This.(Treer)

	NewFuncButton(m).SetFunc(tri.Copy).SetKey(keymap.Copy).
		SetEnabled(tr.HasSelection())
	NewFuncButton(m).SetFunc(tr.EditNode).SetText("View").
		SetIcon(icons.Visibility).SetEnabled(tr.HasSelection())
	NewSeparator(m)

	NewFuncButton(m).SetFunc(tr.OpenAll).SetIcon(icons.KeyboardArrowDown).
		SetEnabled(tr.HasSelection())
	NewFuncButton(m).SetFunc(tr.CloseAll).SetIcon(icons.KeyboardArrowRight).
		SetEnabled(tr.HasSelection())
}

func (tr *Tree) contextMenu(m *Scene) {
	if tr.IsReadOnly() || tr.rootIsReadOnly() {
		tr.contextMenuReadOnly(m)
		return
	}
	tri := tr.This.(Treer)
	NewFuncButton(m).SetFunc(tr.AddChildNode).SetText("Add child").SetIcon(icons.Add).SetEnabled(tr.HasSelection())
	NewFuncButton(m).SetFunc(tr.InsertBefore).SetIcon(icons.Add).
		SetEnabled(tr.HasSelection())
	NewFuncButton(m).SetFunc(tr.InsertAfter).SetIcon(icons.Add).
		SetEnabled(tr.HasSelection())
	NewFuncButton(m).SetFunc(tr.Duplicate).SetIcon(icons.ContentCopy).
		SetEnabled(tr.HasSelection())
	NewFuncButton(m).SetFunc(tr.DeleteSelected).SetText("Delete").SetIcon(icons.Delete).
		SetEnabled(tr.HasSelection())
	NewSeparator(m)
	NewFuncButton(m).SetFunc(tri.Copy).SetIcon(icons.Copy).SetKey(keymap.Copy).
		SetEnabled(tr.HasSelection())
	NewFuncButton(m).SetFunc(tri.Cut).SetIcon(icons.Cut).SetKey(keymap.Cut).
		SetEnabled(tr.HasSelection())
	paste := NewFuncButton(m).SetFunc(tri.Paste).SetIcon(icons.Paste).SetKey(keymap.Paste)
	cb := tr.Scene.Events.Clipboard()
	if cb != nil {
		paste.SetState(cb.IsEmpty(), states.Disabled)
	}
	NewSeparator(m)
	NewFuncButton(m).SetFunc(tr.EditNode).SetText("Edit").SetIcon(icons.Edit).
		SetEnabled(tr.HasSelection())
	NewFuncButton(m).SetFunc(tr.inspectNode).SetText("Inspect").SetIcon(icons.EditDocument).
		SetEnabled(tr.HasSelection())
	NewSeparator(m)

	NewFuncButton(m).SetFunc(tr.OpenAll).SetIcon(icons.KeyboardArrowDown).
		SetEnabled(tr.HasSelection())
	NewFuncButton(m).SetFunc(tr.CloseAll).SetIcon(icons.KeyboardArrowRight).
		SetEnabled(tr.HasSelection())
}

// IsRoot returns true if given node is the root of the tree,
// creating an error snackbar if it is and action is non-empty.
func (tr *Tree) IsRoot(action ...string) bool {
	if tr.This == tr.Root.AsCoreTree().This {
		if len(action) > 0 {
			MessageSnackbar(tr, fmt.Sprintf("Cannot %v the root of the tree", action[0]))
		}
		return true
	}
	return false
}

////////  Copy / Cut / Paste

// DeleteSelected deletes selected items.
// Must be called from first node in selection.
func (tr *Tree) DeleteSelected() { //types:add
	if tr.IsRoot("Delete") {
		return
	}
	if tr.SyncNode != nil {
		tr.deleteSync()
		return
	}
	sels := tr.GetSelectedNodes()
	rn := tr.Root.AsCoreTree()
	tr.UnselectAll()
	for _, sn := range sels {
		sn.AsTree().Delete()
	}
	rn.Update()
	rn.sendChangeEvent()
}

// MimeData adds mimedata for this node: a text/plain of the Path.
func (tr *Tree) MimeData(md *mimedata.Mimes) {
	if tr.SyncNode != nil {
		tr.mimeDataSync(md)
		return
	}
	*md = append(*md, mimedata.NewTextData(tr.PathFrom(tr.Root.AsCoreTree())))
	var buf bytes.Buffer
	err := jsonx.Write(tr.This, &buf)
	if err == nil {
		*md = append(*md, &mimedata.Data{Type: fileinfo.DataJson, Data: buf.Bytes()})
	} else {
		ErrorSnackbar(tr, err, "Error encoding node")
	}
}

// nodesFromMimeData returns a slice of tree nodes for
// the Tree nodes and paths from mime data.
func (tr *Tree) nodesFromMimeData(md mimedata.Mimes) ([]tree.Node, []string) {
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

// Copy copies the selected items to the clipboard.
// This must be called on the first item in the selected list.
func (tr *Tree) Copy() { //types:add
	sels := tr.GetSelectedNodes()
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
}

// Cut copies to [system.Clipboard] and deletes selected items.
// This must be called on the first item in the selected list.
func (tr *Tree) Cut() { //types:add
	if tr.IsRoot("Cut") {
		return
	}
	if tr.SyncNode != nil {
		tr.cutSync()
		return
	}
	tr.Copy()
	sels := tr.GetSelectedNodes()
	rn := tr.Root.AsCoreTree()
	tr.UnselectAll()
	for _, sn := range sels {
		sn.AsTree().Delete()
	}
	rn.Update()
	rn.sendChangeEvent()
}

// Paste pastes clipboard at given node.
func (tr *Tree) Paste() { //types:add
	md := tr.Clipboard().Read([]string{fileinfo.DataJson})
	if md != nil {
		tr.pasteMenu(md)
	}
}

// pasteMenu performs a paste from the clipboard using given data,
// by popping up a menu to determine what specifically to do.
func (tr *Tree) pasteMenu(md mimedata.Mimes) {
	tr.UnselectAll()
	mf := func(m *Scene) {
		tr.makePasteMenu(m, md, nil)
	}
	pos := tr.ContextMenuPos(nil)
	NewMenu(mf, tr.This.(Widget), pos).Run()
}

// makePasteMenu makes the menu of options for paste events
// Optional function is typically the DropFinalize but could also be other actions
// to take after each optional action.
func (tr *Tree) makePasteMenu(m *Scene, md mimedata.Mimes, fun func()) {
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
	if !tr.IsRoot() {
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
		tr.pasteAssignSync(md)
		return
	}
	sl, _ := tr.nodesFromMimeData(md)
	if len(sl) == 0 {
		return
	}
	tr.CopyFrom(sl[0])    // nodes with data copy here
	tr.setScene(tr.Scene) // ensure children have scene
	tr.Update()           // could have children
	tr.Open()
	tr.sendChangeEvent()
}

// PasteBefore inserts object(s) from mime data before this node.
// If another item with the same name already exists, it will
// append _Copy on the name of the inserted objects
func (tr *Tree) PasteBefore(md mimedata.Mimes, mod events.DropMods) {
	tr.pasteAt(md, mod, 0, "Paste before")
}

// PasteAfter inserts object(s) from mime data after this node.
// If another item with the same name already exists, it will
// append _Copy on the name of the inserted objects
func (tr *Tree) PasteAfter(md mimedata.Mimes, mod events.DropMods) {
	tr.pasteAt(md, mod, 1, "Paste after")
}

// treeTempMovedTag is a kind of hack to prevent moved items from being deleted, using DND
const treeTempMovedTag = `_\&MOVED\&`

// todo: these methods require an interface to work for descended
// nodes, based on base code

// pasteAt inserts object(s) from mime data at rel position to this node.
// If another item with the same name already exists, it will
// append _Copy on the name of the inserted objects
func (tr *Tree) pasteAt(md mimedata.Mimes, mod events.DropMods, rel int, actNm string) {
	if tr.Parent == nil {
		return
	}
	parent := AsTree(tr.Parent)
	if parent == nil {
		MessageSnackbar(tr, "Error: cannot insert after the root of the tree")
		return
	}
	if tr.SyncNode != nil {
		tr.pasteAtSync(md, mod, rel, actNm)
		return
	}
	sl, pl := tr.nodesFromMimeData(md)

	myidx := tr.IndexInParent()
	if myidx < 0 {
		return
	}
	myidx += rel
	sz := len(sl)
	var selTv *Tree
	for i, ns := range sl {
		nst := ns.AsTree()
		orgpath := pl[i]
		tree.SetUniqueNameIfDuplicate(parent, ns)
		parent.InsertChild(ns, myidx+i)
		nwb := AsWidget(ns)
		AsTree(ns).Root = tr.Root
		nwb.setScene(tr.Scene)
		nwb.Update() // incl children
		npath := nst.PathFrom(tr.Root)
		if mod == events.DropMove && npath == orgpath { // we will be nuked immediately after drag
			nst.SetName(nst.Name + treeTempMovedTag) // special keyword :)
		}
		if i == sz-1 {
			selTv = AsTree(ns)
		}
	}
	tr.sendChangeEvent()
	parent.NeedsLayout()
	if selTv != nil {
		selTv.SelectEvent(events.SelectOne)
	}
}

// PasteChildren inserts object(s) from mime data
// at end of children of this node.
func (tr *Tree) PasteChildren(md mimedata.Mimes, mod events.DropMods) {
	if tr.SyncNode != nil {
		tr.pasteChildrenSync(md, mod)
		return
	}
	sl, _ := tr.nodesFromMimeData(md)

	for _, ns := range sl {
		tree.SetUniqueNameIfDuplicate(tr.This, ns)
		tr.AddChild(ns)
		AsTree(ns).Root = tr.Root
		AsWidget(ns).setScene(tr.Scene)
	}
	tr.Update()
	tr.Open()
	tr.sendChangeEvent()
}

////////  Drag-n-Drop

// dragStart starts a drag-n-drop on this node -- it includes any other
// selected nodes as well, each as additional records in mimedata.
func (tr *Tree) dragStart(e events.Event) {
	sels := tr.GetSelectedNodes()
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

// dropExternal is not handled by base case but could be in derived
func (tr *Tree) dropExternal(md mimedata.Mimes, mod events.DropMods) {
	// todo: not yet implemented
}

// dragClearStates clears the drag-drop related states for this widget
func (tr *Tree) dragClearStates() {
	tr.dragStateReset()
	tr.Parts.dragStateReset()
	tr.Style()
	tr.NeedsRender()
}

// DragDrop handles drag drop event
func (tr *Tree) DragDrop(e events.Event) {
	// todo: some kind of validation for source
	tr.UnselectAll()
	de := e.(*events.DragDrop)
	stv := AsTree(de.Source.(Widget))
	if stv != nil {
		stv.dragClearStates()
	}
	md := de.Data.(mimedata.Mimes)
	mf := func(m *Scene) {
		tr.Scene.Events.DragMenuAddModText(m, de.DropMod)
		tr.makePasteMenu(m, md, func() {
			tr.DropFinalize(de)
		})
	}
	pos := tr.ContextMenuPos(nil)
	NewMenu(mf, tr.This.(Widget), pos).Run()
}

// DropFinalize is called to finalize Drop actions on the Source node.
// Only relevant for DropMod == DropMove.
func (tr *Tree) DropFinalize(de *events.DragDrop) {
	tr.UnselectAll()
	tr.dragClearStates()
	tr.Scene.Events.DropFinalize(de) // sends DropDeleteSource to Source
}

// DropDeleteSource handles delete source event for DropMove case
func (tr *Tree) DropDeleteSource(e events.Event) {
	de := e.(*events.DragDrop)
	tr.UnselectAll()
	if tr.SyncNode != nil {
		tr.dropDeleteSourceSync(de)
		return
	}
	md := de.Data.(mimedata.Mimes)
	rn := tr.Root.AsCoreTree()
	for _, d := range md {
		if d.Type != fileinfo.TextPlain { // link
			continue
		}
		path := string(d.Data)
		sn := rn.FindPath(path)
		if sn != nil {
			sn.AsTree().Delete()
		}
		sn = rn.FindPath(path + treeTempMovedTag)
		if sn != nil {
			psplt := strings.Split(path, "/")
			orgnm := psplt[len(psplt)-1]
			sn.AsTree().SetName(orgnm)
			AsWidget(sn).NeedsRender()
		}
	}
}
