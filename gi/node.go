// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"log"
	"strings"
	"sync"

	"goki.dev/ki/v2/ki"
	"goki.dev/ki/v2/kit"
	"goki.dev/mat32/v2"
)

// Node is the interface for all GoGi nodes (2D and 3D), for accessing as NodeBase
type Node interface {
	// nodes are Ki elements -- this comes for free by embedding ki.Node in NodeBase
	ki.Ki

	// AsGiNode returns a generic gi.NodeBase for our node -- gives generic
	// access to all the base-level data structures without requiring
	// interface methods.
	AsGiNode() *NodeBase
}

// NodeBase is the base struct type for GoGi graphical interface system,
// containing infrastructure for both 2D and 3D scene graph nodes
type NodeBase struct {
	ki.Node

	// user-defined class name(s) used primarily for attaching CSS styles to different display elements -- multiple class names can be used to combine properties: use spaces to separate per css standard
	Class string `desc:"user-defined class name(s) used primarily for attaching CSS styles to different display elements -- multiple class names can be used to combine properties: use spaces to separate per css standard"`

	// cascading style sheet at this level -- these styles apply here and to everything below, until superceded -- use .class and #name Props elements to apply entire styles to given elements, and type for element type
	CSS ki.Props `xml:"css" desc:"cascading style sheet at this level -- these styles apply here and to everything below, until superceded -- use .class and #name Props elements to apply entire styles to given elements, and type for element type"`

	// [view: no-inline] aggregated css properties from all higher nodes down to me
	CSSAgg ki.Props `copy:"-" json:"-" xml:"-" view:"no-inline" desc:"aggregated css properties from all higher nodes down to me"`

	// raw original 2D bounding box for the object within its parent viewport -- used for computing VpBBox and WinBBox -- this is not updated by Move2D, whereas VpBBox etc are
	BBox image.Rectangle `copy:"-" json:"-" xml:"-" desc:"raw original 2D bounding box for the object within its parent viewport -- used for computing VpBBox and WinBBox -- this is not updated by Move2D, whereas VpBBox etc are"`

	// full object bbox -- this is BBox + Move2D delta, but NOT intersected with parent's parBBox -- used for computing color gradients or other object-specific geometry computations
	ObjBBox image.Rectangle `copy:"-" json:"-" xml:"-" desc:"full object bbox -- this is BBox + Move2D delta, but NOT intersected with parent's parBBox -- used for computing color gradients or other object-specific geometry computations"`

	// 2D bounding box for region occupied within immediate parent Viewport object that we render onto -- these are the pixels we draw into, filtered through parent bounding boxes -- used for render Bounds clipping
	VpBBox image.Rectangle `copy:"-" json:"-" xml:"-" desc:"2D bounding box for region occupied within immediate parent Viewport object that we render onto -- these are the pixels we draw into, filtered through parent bounding boxes -- used for render Bounds clipping"`

	// 2D bounding box for region occupied within parent Window object, projected all the way up to that -- these are the coordinates where we receive events, relative to the window
	WinBBox image.Rectangle `copy:"-" json:"-" xml:"-" desc:"2D bounding box for region occupied within parent Window object, projected all the way up to that -- these are the coordinates where we receive events, relative to the window"`

	// [view: -] mutex protecting access to the WinBBox, which is used for event delegation and could also be updated in another thread
	BBoxMu sync.RWMutex `view:"-" copy:"-" json:"-" xml:"-" desc:"mutex protecting access to the WinBBox, which is used for event delegation and could also be updated in another thread"`
}

var TypeNodeBase = kit.Types.AddType(&NodeBase{}, NodeBaseProps)

var NodeBaseProps = ki.Props{
	"base-type":     true, // excludes type from user selections
	ki.EnumTypeFlag: TypeNodeFlags,
}

func (nb *NodeBase) AsGiNode() *NodeBase {
	return nb
}

func (nb *NodeBase) CopyFieldsFrom(frm any) {
	// note: not copying ki.Node as it doesn't have any copy fields
	fr := frm.(*NodeBase)
	nb.Class = fr.Class
	nb.CSS.CopyFrom(fr.CSS, true)
}

// NodeFlags define gi node bitflags for tracking common high-frequency GUI
// state, mostly having to do with event processing -- use properties map for
// less frequently used information -- uses ki Flags field (64 bit capacity)
type NodeFlags int

var TypeNodeFlags = kit.Enums.AddEnumExt(ki.KiT_Flags, NodeFlagsN, kit.BitFlag, nil)

const (
	// NoLayout means that this node does not participate in the layout
	// process (Size, Layout, Move) -- set by e.g., SVG nodes
	NoLayout NodeFlags = NodeFlags(ki.FlagsN) + iota

	// EventsConnected: this node has been connected to receive events from
	// the window -- to optimize event processing, connections are typically
	// only established for visible nodes during render, and disconnected when
	// not visible
	EventsConnected

	// CanFocus: can this node accept focus to receive keyboard input events
	// -- set by default for typical nodes that do so, but can be overridden,
	// including by the style 'can-focus' property
	CanFocus

	// HasFocus: does this node currently have the focus for keyboard input
	// events?  use tab / alt tab and clicking events to update focus -- see
	// interface on Window
	HasFocus

	// FullReRender indicates that a full re-render is required due to nature
	// of update event -- otherwise default is local re-render -- used
	// internally for nodes to determine what to do on the ReRender step
	FullReRender

	// NeedsStyle indicates that a node needs to be styled again before being rendered.
	NeedsStyle

	// ReRenderAnchor: this node has a static size, and repaints its
	// background -- any children under it that need to dynamically resize on
	// a ReRender (Update) can refer the update up to rerendering this node,
	// instead of going further up the tree -- e.g., true of Frame's within a
	// SplitView
	ReRenderAnchor

	// Invisible means that the node has been marked as invisible by a parent
	// that has switch-like powers (e.g., layout stacked / tabview or splitter
	// panel that has been collapsed).  This flag is propagated down to all
	// child nodes, and rendering or other interaction / update routines
	// should not run when this flag is set (PushBounds does this for most
	// cases).  However, it IS a good idea to have styling, layout etc all
	// take place as normal, so that when the flag is cleared, rendering can
	// proceed directly.
	Invisible

	// Disabled disables all interaction with the user or other nodes;
	// nodes should indicate this disabled state in an
	// appropriate way, and each node should interpret events appropriately
	// based on this state
	Disabled

	// Selected indicates that this node has been selected by the user --
	// widely supported across different nodes
	Selected

	// Hovered indicates that the node is being hovered over by a mouse
	// cursor or has been long-pressed on mobile
	Hovered

	// Active indicates that this node is currently being interacted
	// with (typically pressed down) by the user
	Active

	// MouseHasEntered indicates that the MouseFocusEvent Enter was previously
	// registered on this node
	MouseHasEntered

	// DNDHasEntered indicates that the DNDFocusEvent Enter was previously
	// registered on this node
	DNDHasEntered

	// NodeDragging indicates this node is currently dragging -- win.Dragging
	// set to this node
	NodeDragging

	// InstaDrag indicates this node should start dragging immediately when
	// clicked -- otherwise there is a time-and-distance threshold to the
	// start of dragging -- use this for controls that are small and are
	// primarily about dragging (e.g., the Splitter handle)
	InstaDrag

	// can extend node flags from here
	NodeFlagsN
)

// HasNoLayout checks if the current node is flagged as not needing layout
func (nb *NodeBase) HasNoLayout() bool {
	return nb.HasFlag(int(NoLayout))
}

// CanFocus checks if this node can receive keyboard focus
func (nb *NodeBase) CanFocus() bool {
	return nb.HasFlag(int(CanFocus))
}

// HasFocus checks if the current node is flagged as having keyboard focus
func (nb *NodeBase) HasFocus() bool {
	return nb.HasFlag(int(HasFocus))
}

// SetFocusState sets current HasFocus state
func (nb *NodeBase) SetFocusState(focus bool) {
	nb.SetFlagState(focus, int(HasFocus))
}

// CanFocusWithin returns whether the current node or any
// of its children can accept keyboard focus
func (nb *NodeBase) CanFocusWithin() bool {
	return nb.HasFlagWithin(int(CanFocus))
}

// HasFocusWithin returns whether the current node or any
// of its children are flagged as having keyboard focus
func (nb *NodeBase) HasFocusWithin() bool {
	return nb.HasFlagWithin(int(HasFocus))
}

// HasFlagWithin returns whether the current node or any
// of its children have the given flag.
func (nb *NodeBase) HasFlagWithin(flag int) bool {
	got := false
	nb.FuncDownMeFirst(0, nil, func(k ki.Ki, level int, data any) bool {
		if cnb, ok := k.Embed(TypeNodeBase).(*NodeBase); ok {
			if cnb.HasFlag(flag) {
				got = true
				return ki.Break
			}
		}
		return ki.Continue
	})
	return got
}

// NeedsStyle returns whether the node needs to be
// styled before being rendered.
func (nb *NodeBase) NeedsStyle() bool {
	return nb.HasFlag(int(NeedsStyle))
}

// SetNeedsStyle sets the node as needing to be
// styled before being rendered.
func (nb *NodeBase) SetNeedsStyle() {
	nb.SetFlag(int(NeedsStyle))
}

// ClearNeedsStyle clears the node as needing to be
// styled before being rendered.
func (nb *NodeBase) ClearNeedsStyle() {
	nb.ClearFlag(int(NeedsStyle))
}

// SetNeedsStyle sets the node as either needing to be styled
// before being rendered or not depending on the given value.
func (nb *NodeBase) SetNeedsStyleState(needs bool) {
	nb.SetFlagState(needs, int(NeedsStyle))
}

// IsDragging tests if the current node is currently flagged as receiving
// dragging events -- flag set by window
func (nb *NodeBase) IsDragging() bool {
	return nb.HasFlag(int(NodeDragging))
}

// IsInstaDrag tests if the current node has InstaDrag property set
func (nb *NodeBase) IsInstaDrag() bool {
	return nb.HasFlag(int(InstaDrag))
}

// IsDisabled tests if this node is flagged as [Disabled].
// If so, behave and style appropriately.
func (nb *NodeBase) IsDisabled() bool {
	return nb.HasFlag(int(Disabled))
}

// IsEnabled tests if this node is NOT flagged as [Disabled].
func (nb *NodeBase) IsEnabled() bool {
	return !nb.IsDisabled()
}

// SetDisabled sets the node as [Disabled].
func (nb *NodeBase) SetDisabled() {
	nb.SetFlag(int(Disabled))
}

// ClearDisabled clears the node as [Disabled].
func (nb *NodeBase) ClearDisabled() {
	nb.ClearFlag(int(Disabled))
}

// SetDisabledState sets flag as [Disabled] or not based on disabled arg
func (nb *NodeBase) SetDisabledState(disabled bool) {
	nb.SetFlagState(disabled, int(Disabled))
}

// SetEnabledState sets flag as enabled or not based on enabled arg -- positive logic
// is easier to understand.
func (nb *NodeBase) SetEnabledState(enabled bool) {
	nb.SetFlagState(!enabled, int(Disabled))
}

// SetDisabledStateUpdt sets flag as [Disabled] or not based on disabled arg, and
// does UpdateSig if state changed.
func (nb *NodeBase) SetDisabledStateUpdt(disabled bool) {
	cur := nb.IsDisabled()
	nb.SetFlagState(disabled, int(Disabled))
	if disabled != cur {
		nb.UpdateSig()
	}
}

// SetEnabledStateUpdt sets flag as enabled or not based on act arg -- positive logic
// is easier to understand -- does UpdateSig if state changed.
func (nb *NodeBase) SetEnabledStateUpdt(enabled bool) {
	cur := nb.IsEnabled()
	nb.SetFlagState(!enabled, int(Disabled))
	if enabled != cur {
		nb.UpdateSig()
	}
}

// IsInvisible tests if this node is flagged as Invisible.  if so, do not
// render, update, interact.
func (nb *NodeBase) IsInvisible() bool {
	return nb.HasFlag(int(Invisible))
}

// SetInvisible sets the node as invisible
func (nb *NodeBase) SetInvisible() {
	nb.SetFlag(int(Invisible))
}

// ClearInvisible clears the node as invisible
func (nb *NodeBase) ClearInvisible() {
	nb.ClearFlag(int(Invisible))
}

// SetInvisibleState sets flag as invisible or not based on invis arg
func (nb *NodeBase) SetInvisibleState(invis bool) {
	nb.SetFlagState(invis, int(Invisible))
}

// SetCanFocusIfActive sets CanFocus flag only if node is active (inactive
// nodes don't need focus typically)
func (nb *NodeBase) SetCanFocusIfActive() {
	nb.SetFlagState(!nb.IsDisabled(), int(CanFocus))
}

// SetCanFocus sets CanFocus flag to true
func (nb *NodeBase) SetCanFocus() {
	nb.SetFlag(int(CanFocus))
}

// IsSelected tests if this node is flagged as [Selected]
func (nb *NodeBase) IsSelected() bool {
	return nb.HasFlag(int(Selected))
}

// SetSelected sets the node as [Selected]
func (nb *NodeBase) SetSelected() {
	nb.SetFlag(int(Selected))
}

// ClearSelected sets the node as not [Selected]
func (nb *NodeBase) ClearSelected() {
	nb.ClearFlag(int(Selected))
}

// SetSelectedState sets the node as [Selected] or not based on sel arg
func (nb *NodeBase) SetSelectedState(sel bool) {
	nb.SetFlagState(sel, int(Selected))
}

// IsHovered returns whether this node is flagged as [Hovered]
func (nb *NodeBase) IsHovered() bool {
	return nb.HasFlag(int(Hovered))
}

// SetHovered sets the node as [Hovered]
func (nb *NodeBase) SetHovered() {
	nb.SetFlag(int(Hovered))
}

// ClearHovered sets the node as not [Hovered]
func (nb *NodeBase) ClearHovered() {
	nb.ClearFlag(int(Hovered))
}

// SetHoveredState sets the node as [Hovered] or not based on hov arg
func (nb *NodeBase) SetHoveredState(hov bool) {
	nb.SetFlagState(hov, int(Hovered))
}

// IsActive returns whether this node is flagged as [Active]
func (nb *NodeBase) IsActive() bool {
	return nb.HasFlag(int(Active))
}

// SetActive sets the node as [Active]
func (nb *NodeBase) SetActive() {
	nb.SetFlag(int(Active))
}

// ClearActive sets the node as not [Active]
func (nb *NodeBase) ClearActive() {
	nb.ClearFlag(int(Active))
}

// SetActiveState sets the node as [Active] or not based on act arg
func (nb *NodeBase) SetActiveState(act bool) {
	nb.SetFlagState(act, int(Active))
}

// NeedsFullReRender checks if node has said it needs full re-render
func (nb *NodeBase) NeedsFullReRender() bool {
	return nb.HasFlag(int(FullReRender))
}

// SetFullReRender sets node as needing a full ReRender
func (nb *NodeBase) SetFullReRender() {
	nb.SetFlag(int(FullReRender))
}

// ClearFullReRender clears node as needing a full ReRender
func (nb *NodeBase) ClearFullReRender() {
	nb.ClearFlag(int(FullReRender))
}

// IsReRenderAnchor returns whethers the current node is a ReRenderAnchor
func (nb *NodeBase) IsReRenderAnchor() bool {
	return nb.HasFlag(int(ReRenderAnchor))
}

// SetReRenderAnchor sets node as a ReRenderAnchor
func (nb *NodeBase) SetReRenderAnchor() {
	nb.SetFlag(int(ReRenderAnchor))
}

// TODO: add type-based child functions (first child of type, etc)

// IsNthChild returns whether the node is nth-child of its parent
func (nb *NodeBase) IsNthChild(n int) bool {
	idx, ok := nb.IndexInParent()
	return ok && idx == n
}

// IsFirstChild returns whether the node is the first child of its parent
func (nb *NodeBase) IsFirstChild() bool {
	idx, ok := nb.IndexInParent()
	return ok && idx == 0
}

// IsLastChild returns whether the node is the last child of its parent
func (nb *NodeBase) IsLastChild() bool {
	idx, ok := nb.IndexInParent()
	return ok && idx == nb.Par.NumChildren()-1
}

// IsOnlyChild returns whether the node is the only child of its parent
func (nb *NodeBase) IsOnlyChild() bool {
	return nb.Par != nil && nb.Par.NumChildren() == 1
}

// IsRoot returns whether the node is the root element of the GUI
func (nb *NodeBase) IsRoot() bool {
	return nb.Par == nil
}

// PointToRelPos translates a point in global pixel coords
// into relative position within node
func (nb *NodeBase) PointToRelPos(pt image.Point) image.Point {
	nb.BBoxMu.RLock()
	defer nb.BBoxMu.RUnlock()
	return pt.Sub(nb.WinBBox.Min)
}

// PosInWinBBox returns true if given position is within
// this node's win bbox (under read lock)
func (nb *NodeBase) PosInWinBBox(pos image.Point) bool {
	nb.BBoxMu.RLock()
	defer nb.BBoxMu.RUnlock()
	return pos.In(nb.WinBBox)
}

// WinBBoxInBBox returns true if our BBox is contained within
// given BBox (under read lock)
func (nb *NodeBase) WinBBoxInBBox(bbox image.Rectangle) bool {
	nb.BBoxMu.RLock()
	defer nb.BBoxMu.RUnlock()
	return mat32.RectInNotEmpty(nb.WinBBox, bbox)
}

// AddClass adds a CSS class name -- does proper space separation
func (nb *NodeBase) AddClass(cls string) {
	if nb.Class == "" {
		nb.Class = cls
	} else {
		nb.Class += " " + cls
	}
}

// HasClass returns whether the node has the given class name
// as one of its classes.
func (nb *NodeBase) HasClass(cls string) bool {
	fields := strings.Fields(nb.Class)
	for _, field := range fields {
		if field == cls {
			return true
		}
	}
	return false
}

// StyleProps returns a property that contains another map of properties for a
// given styling selector, such as :normal :active :hover etc -- the
// convention is to prefix this selector with a : and use lower-case names, so
// we follow that.
func (nb *NodeBase) StyleProps(selector string) ki.Props {
	sp, ok := nb.PropInherit(selector, ki.NoInherit, ki.TypeProps) // yeah, use type's
	if !ok {
		return nil
	}
	spm, ok := sp.(ki.Props)
	if ok {
		return spm
	}
	log.Printf("gist.StyleProps: looking for a ki.Props for style selector: %v, instead got type: %T, for node: %v\n", selector, spm, nb.Path())
	return nil
}

// AggCSS aggregates css properties
func AggCSS(agg *ki.Props, css ki.Props) {
	if *agg == nil {
		*agg = make(ki.Props, len(css))
	}
	for key, val := range css {
		(*agg)[key] = val
	}
}

// ParentCSSAgg returns parent's CSSAgg styles or nil if not avail
func (nb *NodeBase) ParentCSSAgg() *ki.Props {
	if nb.Par == nil {
		return nil
	}
	pn := nb.Par.Embed(TypeNodeBase)
	if pn == nil {
		return nil
	}
	return &pn.(*NodeBase).CSSAgg
}

// FirstContainingPoint finds the first node whose WinBBox contains the given
// point -- nil if none.  If leavesOnly is set then only nodes that have no
// nodes (leaves, terminal nodes) will be considered
func (nb *NodeBase) FirstContainingPoint(pt image.Point, leavesOnly bool) ki.Ki {
	var rval ki.Ki
	nb.FuncDownMeFirst(0, nb.This(), func(k ki.Ki, level int, d any) bool {
		if k == nb.This() {
			return ki.Continue
		}
		if leavesOnly && k.HasChildren() {
			return ki.Continue
		}
		_, ni := KiToNode2D(k)
		if ni == nil || ni.IsDeleted() || ni.IsDestroyed() {
			// 3D?
			return ki.Break
		}
		if ni.PosInWinBBox(pt) {
			rval = ni.This()
			return ki.Break
		}
		return ki.Continue
	})
	return rval
}

// AllWithinBBox returns a list of all nodes whose WinBBox is fully contained
// within the given BBox. If leavesOnly is set then only nodes that have no
// nodes (leaves, terminal nodes) will be considered.
func (nb *NodeBase) AllWithinBBox(bbox image.Rectangle, leavesOnly bool) ki.Slice {
	var rval ki.Slice
	nb.FuncDownMeFirst(0, nb.This(), func(k ki.Ki, level int, d any) bool {
		if k == nb.This() {
			return ki.Continue
		}
		if leavesOnly && k.HasChildren() {
			return ki.Continue
		}
		_, ni := KiToNode2D(k)
		if ni == nil || ni.IsDeleted() || ni.IsDestroyed() {
			// 3D?
			return ki.Break
		}
		if ni.WinBBoxInBBox(bbox) {
			rval = append(rval, ni.This())
		}
		return ki.Continue
	})
	return rval
}

// ElementAndParentSize returns the size
// of this node as a [mat32.Vec2] object.
func (nb *NodeBase) NodeSize() mat32.Vec2 {
	return mat32.NewVec2FmPoint(nb.BBox.Size())
}

// standard css properties on nodes apply, including visible, etc.

// see node2d.go for 2d node
