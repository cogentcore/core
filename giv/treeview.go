// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"image"
	"log"
	"log/slog"

	"goki.dev/colors"
	"goki.dev/cursors"
	"goki.dev/gi/v2/gi"
	"goki.dev/girl/abilities"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/goosi/mimedata"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
	"goki.dev/pi/v2/filecat"
)

// TreeView provides a graphical representation of a tree tructure
// providing full navigation and manipulation abilities.
// See the TreeSyncView for a version that syncs with another
// Ki tree structure to represent it.
//
// Standard events.Event are sent to any listeners, including
// Select and DoubleClick.
//
// If possible, it is typically easier to directly use
// TreeView nodes to represent data by adding extra fields.
// See FileTreeView for an example.
//
//goki:embedder
type TreeView struct {
	gi.WidgetBase

	// optional icon, displayed to the the left of the text label
	Icon icons.Icon

	// amount to indent children relative to this node
	Indent units.Value

	// depth for nodes be initialized as open (default 4).
	// Nodes beyond this depth will be initialized as closed.
	OpenDepth int

	///////////////////////
	// Computed below

	// linear index of this node within the entire tree.
	// updated on full rebuilds and may sometimes be off,
	// but close enough for expected uses
	ViewIdx int `copy:"-" json:"-" xml:"-" inactive:"+"`

	// size of just this node widget.
	// our alloc includes all of our children, but we only draw us.
	WidgetSize mat32.Vec2 `copy:"-" json:"-" xml:"-" inactive:"+"`

	// cached root of the view
	RootView *TreeView `copy:"-" json:"-" xml:"-" inactive:"+"`

	// SelectedNodes holds the currently-selected nodes, on the
	// RootView node only.
	SelectedNodes []*TreeView `copy:"-" json:"-" xml:"-" inactive:"+"`
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

// SetViewIdx sets the ViewIdx for all nodes.
// This must be called from the root node after
// construction or any modification to the tree.
// Returns the total number of leaves in the tree.
func (tv *TreeView) SetViewIdx() int {
	idx := 0
	tv.WalkPre(func(k ki.Ki) bool {
		tvki := AsTreeView(k)
		if tvki != nil {
			tvki.ViewIdx = idx
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

func (tv *TreeView) TreeViewStyles() {
	tv.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Activatable, abilities.Focusable, abilities.Selectable, abilities.Hoverable)
		tv.Indent.SetEm(1)
		tv.OpenDepth = 4
		s.Cursor = cursors.Pointer
		s.Border.Style.Set(styles.BorderNone)
		// s.Border.Width.Left.SetDp(1)
		// s.Border.Color.Left = colors.Scheme.OutlineVariant
		s.Margin.Set()
		s.Padding.Set(units.Dp(4))
		s.Text.Align = styles.AlignLeft
		s.AlignV = styles.AlignTop
		// s.Color = colors.Scheme.Secondary.OnContainer
		s.BackgroundColor.SetSolid(colors.Scheme.Surface)
		if tv.StateIs(states.Selected) {
			s.BackgroundColor.SetSolid(colors.Scheme.Select.Container)
		}
	})
}

func (tv *TreeView) OnChildAdded(child ki.Ki) {
	w, _ := gi.AsWidget(child)
	switch w.PathFrom(tv) {
	case "parts":
		parts := w.(*gi.Layout)
		parts.Style(func(s *styles.Style) {
			parts.Spacing.SetCh(0.5)
		})
	case "parts/icon":
		w.Style(func(s *styles.Style) {
			s.Color = colors.Scheme.Secondary.OnContainer
			s.Width.SetEm(1)
			s.Height.SetEm(1)
			s.Margin.Set()
			s.Padding.Set()
		})
	case "parts/branch":
		sw := w.(*gi.Switch)
		sw.Type = gi.SwitchCheckbox
		sw.IconOn = icons.KeyboardArrowDown   // icons.FolderOpen
		sw.IconOff = icons.KeyboardArrowRight // icons.Folder
		sw.IconDisab = icons.Blank
		sw.Style(func(s *styles.Style) {
			s.Color = colors.Scheme.Primary.Base
			s.Margin.Set()
			s.Padding.Set()
			s.Width.SetEm(1)
			s.Height.SetEm(1)
			s.AlignV = styles.AlignMiddle
			// we don't need to visibly tell the user that we are disabled;
			// the lack of an icon accomplishes that
			if s.Is(states.Disabled) {
				s.StateLayer = 0
			}
		})
		sw.OnClick(func(e events.Event) {
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
	case "parts/space":
		w.Style(func(s *styles.Style) {
			s.Width.SetEm(0.5)
		})
	case "parts/label":
		w.Style(func(s *styles.Style) {
			s.SetAbilities(false, abilities.Selectable, abilities.DoubleClickable)
			s.Cursor = cursors.None
			s.Margin.Set()
			s.Padding.Set()
			s.MinWidth.SetCh(16)
			s.Text.WhiteSpace = styles.WhiteSpaceNowrap
		})
	case "parts/menu":
		menu := w.(*gi.Button)
		menu.Indicator = icons.None
	}
}

// TreeViewFlags extend WidgetFlags to hold TreeView state
type TreeViewFlags int64 //enums:bitflag

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

// RootIsInactive returns the inactive status of the root node,
// which is what controls the functional inactivity of the tree
// if individual nodes are inactive that only affects display typically.
func (tv *TreeView) RootIsInactive() bool {
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
	if icc := tv.Parts.ChildByName("branch", 0); icc != nil {
		return icc.(*gi.Switch), true
	}
	return nil, false
}

// IconPart returns the icon in parts, if it exists
func (tv *TreeView) IconPart() (*gi.Icon, bool) {
	if icc := tv.Parts.ChildByName("icon", 1); icc != nil {
		return icc.(*gi.Icon), true
	}
	return nil, false
}

// LabelPart returns the label in parts, if it exists
func (tv *TreeView) LabelPart() (*gi.Label, bool) {
	if lbl := tv.Parts.ChildByName("label", 1); lbl != nil {
		return lbl.(*gi.Label), true
	}
	return nil, false
}

func (tv *TreeView) ConfigParts(sc *gi.Scene) {
	parts := tv.NewParts(gi.LayoutHoriz)
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
		lbl.SetText(tv.Name())
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

// TreeView is tricky for alloc because it is both a layout
// of its children but has to maintain its own bbox for its own widget.

func (tv *TreeView) GetSize(sc *gi.Scene, iter int) {
	tv.InitLayout(sc)
	tv.GetSizeParts(sc, iter) // get our size from parts
	tv.WidgetSize = tv.LayState.Alloc.Size
	h := mat32.Ceil(tv.WidgetSize.Y)
	w := tv.WidgetSize.X

	if !tv.IsClosed() {
		// we layout children under us
		for _, kid := range tv.Kids {
			gis := kid.(gi.Widget).AsWidget()
			if gis == nil || gis.This() == nil {
				continue
			}
			h += mat32.Ceil(gis.LayState.Alloc.Size.Y)
			w = mat32.Max(w, tv.Indent.Dots+gis.LayState.Alloc.Size.X)
		}
	}
	tv.LayState.Alloc.Size = mat32.Vec2{w, h}
	tv.WidgetSize.X = w // stretch
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

func (tv *TreeView) DoLayoutParts(sc *gi.Scene, parBBox image.Rectangle, iter int) {
	spc := tv.BoxSpace()
	tv.Parts.LayState.Alloc.Pos = tv.LayState.Alloc.Pos.Add(spc.Pos())
	tv.Parts.LayState.Alloc.Size = tv.WidgetSize.Sub(spc.Size()) // key diff
	tv.Parts.DoLayout(sc, parBBox, iter)
}

func (tv *TreeView) ChildrenBBoxes(sc *gi.Scene) image.Rectangle {
	return tv.EvBBox
}

func (tv *TreeView) DoLayout(sc *gi.Scene, parBBox image.Rectangle, iter int) bool {
	psize := tv.AddParentPos() // have to add our pos first before computing below:

	rn := tv.RootView
	if rn == nil {
		slog.Error("giv.TreeView: RootView is ni", "in node:", tv)
		return false
	}
	tv.SetBranchState()

	wi := tv.This().(gi.Widget)
	// our alloc size is root's size minus our total indentation
	tv.LayState.Alloc.Size.X = rn.LayState.Alloc.Size.X - (tv.LayState.Alloc.Pos.X - rn.LayState.Alloc.Pos.X)
	tv.WidgetSize.X = tv.LayState.Alloc.Size.X

	tv.LayState.Alloc.PosOrig = tv.LayState.Alloc.Pos
	gi.SetUnitContext(&tv.Styles, sc, tv.NodeSize(), psize) // update units with final layout
	tv.BBox = wi.BBoxes()
	wi.ComputeBBoxes(sc, parBBox, image.Point{})

	tv.DoLayoutParts(sc, parBBox, iter) // use OUR version
	h := mat32.Ceil(tv.WidgetSize.Y)
	if !tv.IsClosed() {
		for _, kid := range tv.Kids {
			if kid == nil || kid.This() == nil {
				continue
			}
			ni := kid.(gi.Widget).AsWidget()
			if ni == nil {
				continue
			}
			ni.LayState.Alloc.PosRel.Y = h
			ni.LayState.Alloc.PosRel.X = tv.Indent.Dots
			h += mat32.Ceil(ni.LayState.Alloc.Size.Y)
		}
	}
	redo := tv.DoLayoutChildren(sc, iter)
	// once layout is done, we can get our reg size back
	// but we keep EvBBox as full size including children
	tv.LayState.Alloc.Size = tv.WidgetSize
	tv.ScBBox = tv.BBoxFromAlloc()
	if gi.LayoutTrace {
		// fmt.Printf("Layout: %v reduced X allocsize: %v rn: %v  pos: %v rn pos: %v\n", tv.Path(), tv.WidgetSize.X, rn.LayState.Alloc.Size.X, tv.LayState.Alloc.Pos.X, rn.LayState.Alloc.Pos.X)
		fmt.Printf("Layout: %v alloc pos: %v size: %v bb: %v  scbb: %v winbb: %v\n", tv.Path(), tv.LayState.Alloc.Pos, tv.LayState.Alloc.Size, tv.BBox, tv.ScBBox, tv.ScBBox)
	}
	return redo
}

func (tv *TreeView) RenderNode(sc *gi.Scene) {
	rs, pc, st := tv.RenderLock(sc)
	pc.DrawStdBox(rs, st, tv.LayState.Alloc.Pos, tv.LayState.Alloc.Size, &tv.Styles.BackgroundColor)
	tv.RenderUnlock(rs)
}

func (tv *TreeView) Render(sc *gi.Scene) {
	if tv.PushBounds(sc) {
		tv.RenderNode(sc)
		tv.RenderParts(sc)
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
	tv.RootView.Send(events.Change, nil)
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
	tv.WalkPre(func(k ki.Ki) bool {
		if k.This() == tv.This() {
			return ki.Continue
		}
		tvki := AsTreeView(k)
		if tvki != nil {
			tvki.SetState(parentClosed, states.Invisible)
		}
		return ki.Continue
	})
}

// Close closes the given node and updates the view accordingly
// (if it is not already closed).
// Sends Change event on RootView.
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
	tv.SetKidsVisibility(true) // parent closed
	tv.SendChangeEvent(nil)
	tv.UpdateEndRender(updt)
}

// Open opens the given node and updates the view accordingly
// (if it is not already opened)
// Sends Change event on RootView.
func (tv *TreeView) Open() {
	if !tv.IsClosed() {
		return
	}
	updt := tv.UpdateStart()
	if tv.HasChildren() {
		tv.SetNeedsLayout()
		tv.SetClosed(false)
		tv.SetBranchState()
		tv.SetKidsVisibility(false)
	}
	tv.SendChangeEvent(nil)
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
func (tv *TreeView) OpenAll() {
	updt := tv.UpdateStart()
	tv.WalkPre(func(k ki.Ki) bool {
		tvki := AsTreeView(k)
		if tvki != nil {
			tvki.SetClosed(false)
			tvki.SetState(false, states.Invisible)
		}
		return ki.Continue
	})
	tv.SendChangeEvent(nil)
	tv.UpdateEndLayout(updt)
}

// CloseAll closes the given node and all of its sub-nodes.
func (tv *TreeView) CloseAll() {
	updt := tv.UpdateStart()
	tv.WalkPre(func(k ki.Ki) bool {
		tvki := AsTreeView(k)
		if tvki != nil {
			tvki.SetClosed(true)
			tvki.SetState(true, states.Invisible)
			return ki.Continue
		}
		return ki.Break
	})
	tv.SendChangeEvent(nil)
	tv.UpdateEndLayout(updt)
}

// OpenParents opens all the parents of this node,
// so that it will be visible.
func (tv *TreeView) OpenParents() {
	updt := tv.UpdateStart()
	tv.WalkUpParent(func(k ki.Ki) bool {
		tvki := AsTreeView(k)
		if tvki != nil {
			tvki.SetClosed(false)
			return ki.Continue
		}
		return ki.Break
	})
	tv.SendChangeEvent(nil)
	tv.UpdateEndLayout(updt)
}

/////////////////////////////////////////////////////////////
//    Modifying Source Tree

func (tv *TreeView) ContextMenuPos(e events.Event) (pos image.Point) {
	if e != nil {
		pos = e.Pos()
		return
	}
	pos.X = tv.ScBBox.Min.X + int(tv.Indent.Dots)
	pos.Y = (tv.ScBBox.Min.Y + tv.ScBBox.Max.Y) / 2
	return
}

func (tv *TreeView) MakeTreeViewContextMenu(m *gi.Menu) {
	cpsc := gi.ActiveKeyMap.ChordForFun(gi.KeyFunCopy)
	ac := m.AddButton(gi.ActOpts{Label: "Copy", Shortcut: cpsc}, func(bt *gi.Button) {
		tv.This().(gi.Clipper).Copy(true)
	})
	ac.SetEnabledState(tv.HasSelection())
	if !tv.IsDisabled() {
		ctsc := gi.ActiveKeyMap.ChordForFun(gi.KeyFunCut)
		ptsc := gi.ActiveKeyMap.ChordForFun(gi.KeyFunPaste)
		ac = m.AddButton(gi.ActOpts{Label: "Cut", Shortcut: ctsc}, func(bt *gi.Button) {
			tv.This().(gi.Clipper).Cut()
		})
		ac.SetEnabledState(tv.HasSelection())
		ac = m.AddButton(gi.ActOpts{Label: "Paste", Shortcut: ptsc}, func(bt *gi.Button) {
			tv.This().(gi.Clipper).Paste()
		})
		cb := tv.Sc.EventMgr.ClipBoard()
		if cb != nil {
			ac.SetState(cb.IsEmpty(), states.Disabled)
		}
	}
}

func (tv *TreeView) MakeContextMenu(m *gi.Menu) {
	// derived types put native menu code here
	if tv.CtxtMenuFunc != nil {
		tv.CtxtMenuFunc(tv.This().(gi.Widget), m)
	}
	tv.MakeTreeViewContextMenu(m)
}

// IsRoot returns true if given node is the root of the tree.
func (tv *TreeView) IsRoot(op string) bool {
	if tv.This() == tv.RootView.This() {
		if op != "" {
			gi.PromptDialog(tv, gi.DlgOpts{Title: "TreeView " + op, Prompt: fmt.Sprintf("Cannot %v the root of the tree", op), Ok: true, Cancel: false}, nil)
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
	*md = append(*md, mimedata.NewTextData(tv.PathFrom(tv.RootView)))
}

// NodesFromMimeData returns a slice of paths from mime data.
func (tv *TreeView) NodesFromMimeData(md mimedata.Mimes) []string {
	ni := len(md) / 2
	pl := make([]string, 0, ni)
	for _, d := range md {
		if d.Type == filecat.TextPlain { // paths
			pl = append(pl, string(d.Data))
		}
	}
	return pl
}

// Copy copies to clip.Board, optionally resetting the selection.
// satisfies gi.Clipper interface and can be overridden by subtypes
func (tv *TreeView) Copy(reset bool) {
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
func (tv *TreeView) Cut() {
	if tv.IsRoot("Cut") {
		return
	}
	tv.Copy(false)
	sels := tv.SelectedViews()
	tv.UnselectAll()
	for _, sn := range sels {
		sn.Delete(true)
	}
	// tv.SetChanged()
}

// Paste pastes clipboard at given node.
// satisfies gi.Clipper interface and can be overridden by subtypes
func (tv *TreeView) Paste() {
	md := tv.EventMgr().ClipBoard().Read([]string{filecat.DataJson})
	if md != nil {
		tv.PasteMenu(md)
	}
}

// MakePasteMenu makes the menu of options for paste events
func (tv *TreeView) MakePasteMenu(m *gi.Menu, data any) {
	if len(*m) > 0 {
		return
	}
	m.AddButton(gi.ActOpts{Label: "Assign To", Data: data}, func(act *gi.Button) {
		tv.PasteAssign(data.(mimedata.Mimes))
	})
	m.AddButton(gi.ActOpts{Label: "Add to Children", Data: data}, func(act *gi.Button) {
		tv.PasteChildren(data.(mimedata.Mimes), events.DropCopy)
	})
	if !tv.IsRoot("") && tv.RootView.This() != tv.This() {
		m.AddButton(gi.ActOpts{Label: "Insert Before", Data: data}, func(act *gi.Button) {
			tv.PasteBefore(data.(mimedata.Mimes), events.DropCopy)
		})
		m.AddButton(gi.ActOpts{Label: "Insert After", Data: data}, func(act *gi.Button) {
			tv.PasteAfter(data.(mimedata.Mimes), events.DropCopy)
		})
	}
	m.AddButton(gi.ActOpts{Label: "Cancel", Data: data}, func(act *gi.Button) {
	})
	// todo: compare, etc..
}

// PasteMenu performs a paste from the clipboard using given data -- pops up
// a menu to determine what specifically to do
func (tv *TreeView) PasteMenu(md mimedata.Mimes) {
	tv.UnselectAll()
	var menu gi.Menu
	tv.MakePasteMenu(&menu, md)
	pos := tv.ContextMenuPos(nil)
	gi.NewMenu(menu, tv.This().(gi.Widget), pos).Run()
}

// PasteAssign assigns mime data (only the first one!) to this node
func (tv *TreeView) PasteAssign(md mimedata.Mimes) {
	pl := tv.NodesFromMimeData(md)
	if len(pl) == 0 {
		return
	}
	sk, err := tv.RootView.FindPathTry(pl[0])
	if err != nil {
		slog.Error("TreeView PasteAssign path not found", "path:", pl[0], "target node:", tv)
		return
	}
	tv.CopyFrom(sk) // nodes with data copy here
	// tv.SetChanged()
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
	//	pl := tv.NodesFromMimeData(md)
	//
	//	if tv.Par == nil {
	//		return
	//	}
	//	par := AsTreeView(tv.Par)
	//	if par == nil {
	//		gi.PromptDialog(tv, gi.DlgOpts{Title: actNm, Prompt: "Cannot insert after the root of the tree", Ok: true, Cancel: false}, nil)
	//		return
	//	}
	//	myidx, ok := tv.IndexInParent()
	//	if !ok {
	//		return
	//	}
	//	myidx += rel
	//	updt := par.UpdateStart()
	//
	// sz := len(sl)
	//
	//	for i, orgpath := range pl {
	//		if mod != events.DropMove {
	//
	//	if cn := par.ChildByName(ns.Name(), 0); cn != nil {
	//		ns.SetName(ns.Name() + "_Copy")
	//	}
	//
	//	}
	//
	// todo!
	// par.SetChildAdded()
	// par.InsertChild(ns, myidx+i)
	// npath := ns.PathFrom(sroot)
	// if mod == events.DropMove && npath == orgpath { // we will be nuked immediately after drag
	//
	//		ns.SetName(ns.Name() + TreeViewTempMovedTag) // special keyword :)
	//	}
	//
	//	if i == sz-1 {
	//		ski = ns
	//	}
	//
	// tv.SendChangeEvent(true)
	// }
	// par.UpdateEndLayout(updt)
	// todo:
	//
	//	if ski != nil {
	//		if tvk := tvpar.ChildByName("tv_"+ski.Name(), 0); tvk != nil {
	//			stv := AsTreeView(tvk)
	//			stv.SelectAction(events.SelectOne)
	//		}
	//	}
}

// PasteChildren inserts object(s) from mime data
// at end of children of this node
func (tv *TreeView) PasteChildren(md mimedata.Mimes, mod events.DropMods) {
	pl := tv.NodesFromMimeData(md)
	_ = pl

	updt := tv.UpdateStart()
	tv.SetChildAdded()
	// for _, ns := range sl {
	// 	sk.AddChild(ns)
	// 	// tv.RootView.TreeViewSig.Emit(tv.RootView.This(), int64(TreeViewInserted), ns.This())
	// }
	tv.UpdateEndLayout(updt)
	// tv.SetChanged()
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
	sroot := tv.RootView.SrcNode
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
		m.AddButton(gi.ActOpts{Label: "Assign To", Data: data}, tv.This(), func(recv, send ki.Ki, sig int64, data any) {
			tv := recv.Embed(TreeViewType).(*TreeView)
			tv.DropAssign(data.(mimedata.Mimes))
		})
	}
	m.AddButton(gi.ActOpts{Label: "Add to Children", Data: data}, tv.This(), func(recv, send ki.Ki, sig int64, data any) {
		tv := recv.Embed(TreeViewType).(*TreeView)
		tv.DropChildren(data.(mimedata.Mimes), mod) // captures mod
	})
	if !tv.IsRoot("") && tv.RootView.This() != tv.This() {
		m.AddButton(gi.ActOpts{Label: "Insert Before", Data: data}, tv.This(), func(recv, send ki.Ki, sig int64, data any) {
			tv := recv.Embed(TreeViewType).(*TreeView)
			tv.DropBefore(data.(mimedata.Mimes), mod) // captures mod
		})
		m.AddButton(gi.ActOpts{Label: "Insert After", Data: data}, tv.This(), func(recv, send ki.Ki, sig int64, data any) {
			tv := recv.Embed(TreeViewType).(*TreeView)
			tv.DropAfter(data.(mimedata.Mimes), mod) // captures mod
		})
	}
	m.AddButton(gi.ActOpts{Label: "Cancel", Data: data}, tv.This(), func(recv, send ki.Ki, sig int64, data any) {
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
// 	Widget Infrastructure

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
	kf := gi.KeyFun(kt.KeyChord())
	selMode := events.SelectModeBits(kt.Modifiers())

	if selMode == events.SelectOne {
		if tv.SelectMode() {
			selMode = events.ExtendContinuous
		}
	}

	// first all the keys that work for inactive and active
	switch kf {
	case gi.KeyFunCancelSelect:
		tv.UnselectAll()
		tv.SetSelectMode(false)
		kt.SetHandled()
	case gi.KeyFunMoveRight:
		tv.Open()
		kt.SetHandled()
	case gi.KeyFunMoveLeft:
		tv.Close()
		kt.SetHandled()
	case gi.KeyFunMoveDown:
		tv.MoveDownAction(selMode)
		kt.SetHandled()
	case gi.KeyFunMoveUp:
		tv.MoveUpAction(selMode)
		kt.SetHandled()
	case gi.KeyFunPageUp:
		tv.MovePageUpAction(selMode)
		kt.SetHandled()
	case gi.KeyFunPageDown:
		tv.MovePageDownAction(selMode)
		kt.SetHandled()
	case gi.KeyFunHome:
		tv.MoveHomeAction(selMode)
		kt.SetHandled()
	case gi.KeyFunEnd:
		tv.MoveEndAction(selMode)
		kt.SetHandled()
	case gi.KeyFunSelectMode:
		tv.SelectModeToggle()
		kt.SetHandled()
	case gi.KeyFunSelectAll:
		tv.SelectAll()
		kt.SetHandled()
	case gi.KeyFunEnter:
		tv.ToggleClose()
		kt.SetHandled()
	case gi.KeyFunCopy:
		tv.This().(gi.Clipper).Copy(true)
		kt.SetHandled()
	}
	if !tv.RootIsInactive() && !kt.IsHandled() {
		switch kf {
		// todo:
		// case gi.KeyFunDelete:
		// 	tv.SrcDelete()
		// 	kt.SetHandled()
		// case gi.KeyFunDuplicate:
		// 	tv.SrcDuplicate()
		// 	kt.SetHandled()
		// case gi.KeyFunInsert:
		// 	tv.SrcInsertBefore()
		// 	kt.SetHandled()
		// case gi.KeyFunInsertAfter:
		// 	tv.SrcInsertAfter()
		// 	kt.SetHandled()
		case gi.KeyFunCut:
			tv.This().(gi.Clipper).Cut()
			kt.SetHandled()
		case gi.KeyFunPaste:
			tv.This().(gi.Clipper).Paste()
			kt.SetHandled()
		}
	}
}

func (tv *TreeView) HandleTreeViewMouse() {
	tv.OnClick(func(e events.Event) {
		e.SetHandled()
		tv.SelectAction(e.SelectMode())
	})
	tv.OnDoubleClick(func(e events.Event) {
		e.SetHandled()
		tv.ToggleClose()
	})
	tv.On(events.MouseEnter, func(e events.Event) {
		if tv.PosInScBBox(e.LocalPos()) {
			tv.SetState(true, states.Hovered)
			tv.WalkUpParent(func(k ki.Ki) bool {
				tvki := AsTreeView(k)
				if tvki != nil {
					if tvki.StateIs(states.Hovered) {
						tvki.SetState(false, states.Hovered)
						tvki.ApplyStyle(tvki.Sc)
						tvki.SetNeedsRender()
					}
					return ki.Continue
				}
				return ki.Break
			})
		}
		e.SetHandled()
	})
	tv.On(events.MouseDown, func(e events.Event) {
		if tv.PosInScBBox(e.LocalPos()) {
			tv.SetState(true, states.Active)
		}
		e.SetHandled()
	})
	tv.On(events.MouseUp, func(e events.Event) {
		tv.SetState(false, states.Active)
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
			"shortcut": gi.KeyFunInsert,
			"updtfunc": ActionUpdateFunc(func(tvi any, act *gi.Button) {
				// tv := tvi.(ki.Ki).Embed(TreeViewType).(*TreeView)
				// act.SetState(tv.IsRoot(""), states.Disabled)
			}),
		}},
		{"SrcInsertAfter", ki.Props{
			"label":    "Insert After",
			"shortcut": gi.KeyFunInsertAfter,
			"updtfunc": ActionUpdateFunc(func(tvi any, act *gi.Button) {
				// tv := tvi.(ki.Ki).Embed(TreeViewType).(*TreeView)
				// act.SetState(tv.IsRoot(""), states.Disabled)
			}),
		}},
		{"SrcDuplicate", ki.Props{
			"label":    "Duplicate",
			"shortcut": gi.KeyFunDuplicate,
			"updtfunc": ActionUpdateFunc(func(tvi any, act *gi.Button) {
				// tv := tvi.(ki.Ki).Embed(TreeViewType).(*TreeView)
				// act.SetState(tv.IsRoot(""), states.Disabled)
			}),
		}},
		{"SrcDelete", ki.Props{
			"label":    "Delete",
			"shortcut": gi.KeyFunDelete,
			"updtfunc": ActionUpdateFunc(func(tvi any, act *gi.Button) {
				// tv := tvi.(ki.Ki).Embed(TreeViewType).(*TreeView)
				// act.SetState(tv.IsRoot(""), states.Disabled)
			}),
		}},
		{"sep-edit", ki.BlankProp{}},
		{"Copy", ki.Props{
			"shortcut": gi.KeyFunCopy,
			"Args": ki.PropSlice{
				{"reset", ki.Props{
					"value": true,
				}},
			},
		}},
		{"Cut", ki.Props{
			"shortcut": gi.KeyFunCut,
			"updtfunc": ActionUpdateFunc(func(tvi any, act *gi.Button) {
				// tv := tvi.(ki.Ki).Embed(TreeViewType).(*TreeView)
				// act.SetState(tv.IsRoot(""), states.Disabled)
			}),
		}},
		{"Paste", ki.Props{
			"shortcut": gi.KeyFunPaste,
		}},
		{"sep-win", ki.BlankProp{}},
		{"SrcEdit", ki.Props{
			"label": "Edit",
		}},
		{"SrcGoGiEditor", ki.Props{
			"label": "GoGi Editor",
		}},
		{"sep-open", ki.BlankProp{}},
		{"OpenAll", ki.Props{}},
		{"CloseAll", ki.Props{}},
	},
	"CtxtMenuInactive": ki.PropSlice{
		{"Copy", ki.Props{
			"shortcut": gi.KeyFunCopy,
			"Args": ki.PropSlice{
				{"reset", ki.Props{
					"value": true,
				}},
			},
		}},
		{"SrcEdit", ki.Props{
			"label": "Edit",
		}},
		{"SrcGoGiEditor", ki.Props{
			"label": "GoGi Editor",
		}},
	},
}

//
// func (tv *TreeView) FocusChanged(change gi.FocusChanges) {
// 	switch change {
// 	case gi.FocusLost:
// 		tv.SetNeedsRender()
// 	case gi.FocusGot:
// 		if tv.This() == tv.RootView.This() {
// 			sl := tv.SelectedViews()
// 			if len(sl) > 0 {
// 				fsl := sl[0]
// 				if fsl != tv {
// 					fsl.GrabFocus()
// 					return
// 				}
// 			}
// 		}
// 		tv.ScrollToMe()
// 		tv.EmitFocusedSignal()
// 		tv.SetNeedsRender()
// 	case gi.FocusInactive: // don't care..
// 	case gi.FocusActive:
// 	}
// }
