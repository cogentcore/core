// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"strings"

	"goki.dev/enums"
	"goki.dev/girl/abilities"
	"goki.dev/girl/states"
	"goki.dev/ki/v2"
)

// WidgetFlags define Widget node bitflags for tracking common high-frequency GUI
// state, mostly having to do with event processing. Extends ki.Flags
type WidgetFlags ki.Flags //enums:bitflag

const (
	// NeedsRender needs to be rendered on next render itration
	NeedsRender WidgetFlags = WidgetFlags(ki.FlagsN) + iota

	// Invisible means that the node has been marked as invisible by a parent
	// that has switch-like powers (e.g., layout stacked / tabview or splitter
	// panel that has been collapsed).  This flag is propagated down to all
	// child nodes, and rendering or other interaction / update routines
	// should not run when this flag is set (PushBounds does this for most
	// cases).  However, it IS a good idea to have styling, layout etc all
	// take place as normal, so that when the flag is cleared, rendering can
	// proceed directly.
	Invisible

	// InstaDrag indicates this node should start dragging immediately when
	// clicked -- otherwise there is a time-and-distance threshold to the
	// start of dragging -- use this for controls that are small and are
	// primarily about dragging (e.g., the Splitter handle)
	InstaDrag
)

// SetSelected sets the selected flag to given value
func (wb *WidgetBase) SetSelected(sel bool) {
	wb.SetState(sel, states.Selected)
}

// CanFocus checks if this node can receive keyboard focus
func (wb *WidgetBase) CanFocus() bool {
	return wb.Style.Abilities.HasFlag(abilities.Focusable)
}

// SetCanFocusIfActive sets CanFocus flag only if node is active (inactive
// nodes don't need focus typically)
func (wb *WidgetBase) SetCanFocusIfActive() {
	wb.SetAbilities(wb.StateIs(states.Active), abilities.Focusable)
}

// SetFocusState sets the HasFocus flag
func (wb *WidgetBase) SetFocusState(focus bool) {
	wb.SetState(focus, states.Focused)
}

// SetEnabledState sets the Disabled flag
func (wb *WidgetBase) SetEnabledState(enabled bool) {
	wb.SetState(!enabled, states.Disabled)
}

// SetEnabledStateUpdt sets the Disabled flag
func (wb *WidgetBase) SetEnabledStateUpdt(enabled bool) {
	wb.SetState(!enabled, states.Disabled)
	wb.ApplyStyleUpdate(wb.Sc)
}

// IsDisabled tests if this node is flagged as [Disabled].
// If so, behave and style appropriately.
func (wb *WidgetBase) IsDisabled() bool {
	return wb.StateIs(states.Disabled)
}

// HasFlagWithin returns whether the current node or any
// of its children have the given flag.
func (wb *WidgetBase) HasFlagWithin(flag enums.BitFlag) bool {
	got := false
	wb.WalkPre(func(k ki.Ki) bool {
		_, wb := AsWidget(k)
		if wb == nil || wb.Is(ki.Deleted) || wb.Is(ki.Destroyed) {
			return ki.Break
		}
		if wb.Is(flag) {
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
