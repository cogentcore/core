// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"log"
	"sync"

	"github.com/goki/gi/gist"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
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
	Class   string          `desc:"user-defined class name(s) used primarily for attaching CSS styles to different display elements -- multiple class names can be used to combine properties: use spaces to separate per css standard"`
	CSS     ki.Props        `xml:"css" desc:"cascading style sheet at this level -- these styles apply here and to everything below, until superceded -- use .class and #name Props elements to apply entire styles to given elements, and type for element type"`
	CSSAgg  ki.Props        `copy:"-" json:"-" xml:"-" view:"no-inline" desc:"aggregated css properties from all higher nodes down to me"`
	BBox    image.Rectangle `copy:"-" json:"-" xml:"-" desc:"raw original 2D bounding box for the object within its parent viewport -- used for computing VpBBox and WinBBox -- this is not updated by Move2D, whereas VpBBox etc are"`
	ObjBBox image.Rectangle `copy:"-" json:"-" xml:"-" desc:"full object bbox -- this is BBox + Move2D delta, but NOT intersected with parent's parBBox -- used for computing color gradients or other object-specific geometry computations"`
	VpBBox  image.Rectangle `copy:"-" json:"-" xml:"-" desc:"2D bounding box for region occupied within immediate parent Viewport object that we render onto -- these are the pixels we draw into, filtered through parent bounding boxes -- used for render Bounds clipping"`
	WinBBox image.Rectangle `copy:"-" json:"-" xml:"-" desc:"2D bounding box for region occupied within parent Window object, projected all the way up to that -- these are the coordinates where we receive events, relative to the window"`
	BBoxMu  sync.RWMutex    `view:"-" copy:"-" json:"-" xml:"-" desc:"mutex protecting access to the WinBBox, which is used for event delegation and could also be updated in another thread"`
}

var KiT_NodeBase = kit.Types.AddType(&NodeBase{}, NodeBaseProps)

var NodeBaseProps = ki.Props{
	"base-type":     true, // excludes type from user selections
	"EnumType:Flag": KiT_NodeFlags,
}

func (nb *NodeBase) AsGiNode() *NodeBase {
	return nb
}

func (nb *NodeBase) CopyFieldsFrom(frm interface{}) {
	// note: not copying ki.Node as it doesn't have any copy fields
	fr := frm.(*NodeBase)
	nb.Class = fr.Class
	nb.CSS.CopyFrom(fr.CSS, true)
}

// NodeFlags define gi node bitflags for tracking common high-frequency GUI
// state, mostly having to do with event processing -- use properties map for
// less frequently used information -- uses ki Flags field (64 bit capacity)
type NodeFlags int

//go:generate stringer -type=NodeFlags

var KiT_NodeFlags = kit.Enums.AddEnumExt(ki.KiT_Flags, NodeFlagsN, kit.BitFlag, nil)

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

	// Inactive disables interaction with widgets or other nodes (i.e., they
	// are read-only) -- they should indicate this inactive state in an
	// appropriate way, and each node should interpret events appropriately
	// based on this state (select and context menu events should still be
	// generated)
	Inactive

	// Selected indicates that this node has been selected by the user --
	// widely supported across different nodes
	Selected

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

// IsDragging tests if the current node is currently flagged as receiving
// dragging events -- flag set by window
func (nb *NodeBase) IsDragging() bool {
	return nb.HasFlag(int(NodeDragging))
}

// IsInstaDrag tests if the current node has InstaDrag property set
func (nb *NodeBase) IsInstaDrag() bool {
	return nb.HasFlag(int(InstaDrag))
}

// IsInactive tests if this node is flagged as Inactive.  if so, behave (e.g.,
// ignore events except select, context menu) and style appropriately
func (nb *NodeBase) IsInactive() bool {
	return nb.HasFlag(int(Inactive))
}

// IsActive tests if this node is NOT flagged as Inactive.
func (nb *NodeBase) IsActive() bool {
	return !nb.IsInactive()
}

// SetInactive sets the node as inactive
func (nb *NodeBase) SetInactive() {
	nb.SetFlag(int(Inactive))
}

// ClearInactive clears the node as inactive
func (nb *NodeBase) ClearInactive() {
	nb.ClearFlag(int(Inactive))
}

// SetInactiveState sets flag as inactive or not based on inact arg
func (nb *NodeBase) SetInactiveState(inact bool) {
	nb.SetFlagState(inact, int(Inactive))
}

// SetActiveState sets flag as active or not based on act arg -- positive logic
// is easier to understand.
func (nb *NodeBase) SetActiveState(act bool) {
	nb.SetFlagState(!act, int(Inactive))
}

// SetInactiveStateUpdt sets flag as inactive or not based on inact arg, and
// does UpdateSig if state changed.
func (nb *NodeBase) SetInactiveStateUpdt(inact bool) {
	cur := nb.IsInactive()
	nb.SetFlagState(inact, int(Inactive))
	if inact != cur {
		nb.UpdateSig()
	}
}

// SetActiveStateUpdt sets flag as active or not based on act arg -- positive logic
// is easier to understand -- does UpdateSig if state changed.
func (nb *NodeBase) SetActiveStateUpdt(act bool) {
	cur := nb.IsActive()
	nb.SetFlagState(!act, int(Inactive))
	if act != cur {
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
	nb.SetFlagState(!nb.IsInactive(), int(CanFocus))
}

// SetCanFocus sets CanFocus flag to true
func (nb *NodeBase) SetCanFocus() {
	nb.SetFlag(int(CanFocus))
}

// IsSelected tests if this node is flagged as Selected
func (nb *NodeBase) IsSelected() bool {
	return nb.HasFlag(int(Selected))
}

// SetSelected sets the node as selected
func (nb *NodeBase) SetSelected() {
	nb.SetFlag(int(Selected))
}

// ClearSelected sets the node as not selected
func (nb *NodeBase) ClearSelected() {
	nb.ClearFlag(int(Selected))
}

// SetSelectedState set flag as selected or not based on sel arg
func (nb *NodeBase) SetSelectedState(sel bool) {
	nb.SetFlagState(sel, int(Selected))
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
	pn := nb.Par.Embed(KiT_NodeBase).(*NodeBase)
	return &pn.CSSAgg
}

// SetStdXMLAttr sets standard attributes of node given XML-style name /
// attribute values (e.g., from parsing XML / SVG files) -- returns true if handled
func SetStdXMLAttr(ni Node, name, val string) bool {
	nb := ni.AsGiNode()
	switch name {
	case "id":
		nb.SetName(val)
		return true
	case "class":
		nb.Class = val
		return true
	case "style":
		gist.SetStylePropsXML(val, &nb.Props)
		return true
	}
	return false
}

// FirstContainingPoint finds the first node whose WinBBox contains the given
// point -- nil if none.  If leavesOnly is set then only nodes that have no
// nodes (leaves, terminal nodes) will be considered
func (nb *NodeBase) FirstContainingPoint(pt image.Point, leavesOnly bool) ki.Ki {
	var rval ki.Ki
	nb.FuncDownMeFirst(0, nb.This(), func(k ki.Ki, level int, d interface{}) bool {
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
	nb.FuncDownMeFirst(0, nb.This(), func(k ki.Ki, level int, d interface{}) bool {
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

// standard css properties on nodes apply, including visible, etc.

// see node2d.go for 2d node
