// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"strings"

	"goki.dev/enums"
	"goki.dev/ki/v2"
)

// WidgetFlags define Widget node bitflags for tracking common high-frequency GUI
// state, mostly having to do with event processing. Extends ki.Flags
type WidgetFlags ki.Flags //enums:bitflag

const (
	// NeedsRender needs to be rendered on next render itration
	NeedsRender WidgetFlags = WidgetFlags(ki.FlagsN) + iota

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
	// interface on OSWin
	HasFocus

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
)

// IsSelected tests if this node is flagged as [Selected]
func (wb *WidgetBase) IsSelected() bool {
	return wb.HasFlag(Selected)
}

// SetSelected sets the selected flag to given value
func (wb *WidgetBase) SetSelected(sel bool) {
	wb.SetFlag(sel, Selected)
}

// CanFocus checks if this node can receive keyboard focus
func (wb *WidgetBase) CanFocus() bool {
	return wb.HasFlag(CanFocus)
}

// SetCanFocusIfActive sets CanFocus flag only if node is active (inactive
// nodes don't need focus typically)
func (wb *WidgetBase) SetCanFocusIfActive() {
	wb.SetFlag(wb.HasFlag(Active), CanFocus)
}

// SetFocusState sets the HasFocus flag
func (wb *WidgetBase) SetFocusState(focus bool) {
	wb.SetFlag(focus, HasFocus)
}

// SetEnabledState sets the Disabled flag
func (wb *WidgetBase) SetEnabledState(enabled bool) {
	wb.SetFlag(!enabled, Disabled)
}

// SetEnabledStateUpdt sets the Disabled flag
func (wb *WidgetBase) SetEnabledStateUpdt(enabled bool) {
	wb.SetFlag(!enabled, Disabled)
	wb.SetStyleUpdate(wb.Sc)
}

// IsDisabled tests if this node is flagged as [Disabled].
// If so, behave and style appropriately.
func (wb *WidgetBase) IsDisabled() bool {
	return wb.HasFlag(Disabled)
}

// HasFlagWithin returns whether the current node or any
// of its children have the given flag.
func (wb *WidgetBase) HasFlagWithin(flag enums.BitFlag) bool {
	got := false
	wb.FuncDownMeFirst(0, nil, func(k ki.Ki, level int, data any) bool {
		_, wb := AsWidget(k)
		if wb == nil || wb.IsDeleted() || wb.IsDestroyed() {
			return ki.Break
		}
		if wb.HasFlag(flag) {
			got = true
			return ki.Break
		}
		return ki.Continue
	})
	return got
}

// AddClass adds a CSS class name -- does proper space separation
func (wb *WidgetBase) AddClass(cls string) {
	if wb.Class == "" {
		wb.Class = cls
	} else {
		wb.Class += " " + cls
	}
}

// HasClass returns whether the node has the given class name
// as one of its classes.
func (wb *WidgetBase) HasClass(cls string) bool {
	fields := strings.Fields(wb.Class)
	for _, field := range fields {
		if field == cls {
			return true
		}
	}
	return false
}
