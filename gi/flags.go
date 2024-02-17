// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"cogentcore.org/core/abilities"
	"cogentcore.org/core/enums"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/states"
)

// WidgetFlags define Widget node bitflags for tracking common high-frequency GUI
// state, mostly having to do with event processing. Extends ki.Flags
type WidgetFlags ki.Flags //enums:bitflag

const (
	// NeedsRender needs to be rendered on next render iteration
	NeedsRender = WidgetFlags(ki.FlagsN) + iota
)

// StateIs returns whether the widget has the given [states.States] flag set
func (wb *WidgetBase) StateIs(state states.States) bool {
	return wb.Styles.State.HasFlag(state)
}

// AbilityIs returns whether the widget has the given [abilities.Abilities] flag set
func (wb *WidgetBase) AbilityIs(able abilities.Abilities) bool {
	return wb.Styles.Abilities.HasFlag(able)
}

// SetState sets the given [states.State] flags to the given value
func (wb *WidgetBase) SetState(on bool, state ...states.States) *WidgetBase {
	bfs := make([]enums.BitFlag, len(state))
	for i, st := range state {
		bfs[i] = st
	}
	wb.Styles.State.SetFlag(on, bfs...)
	return wb
}

// SetAbilities sets the given [abilities.Abilities] flags to the given value
func (wb *WidgetBase) SetAbilities(on bool, able ...abilities.Abilities) *WidgetBase {
	bfs := make([]enums.BitFlag, len(able))
	for i, st := range able {
		bfs[i] = st
	}
	wb.Styles.Abilities.SetFlag(on, bfs...)
	return wb
}

// SetSelected sets the Selected flag to given value for the entire Widget
// and calls ApplyStyleTree to apply any style changes.
func (wb *WidgetBase) SetSelected(sel bool) {
	wb.SetState(sel, states.Selected)
	wb.ApplyStyleTree()
	wb.SetNeedsRender(true)
}

// CanFocus checks if this node can receive keyboard focus
func (wb *WidgetBase) CanFocus() bool {
	return wb.Styles.Abilities.HasFlag(abilities.Focusable)
}

// SetEnabled sets the Disabled flag
func (wb *WidgetBase) SetEnabled(enabled bool) *WidgetBase {
	return wb.SetState(!enabled, states.Disabled)
}

// IsDisabled tests if this node is flagged as [Disabled].
// If so, behave and style appropriately.
func (wb *WidgetBase) IsDisabled() bool {
	return wb.StateIs(states.Disabled)
}

// IsReadOnly tests if this node is flagged as [ReadOnly] or [Disabled].
// If so, behave appropriately.  Styling is based on each state separately,
// but behaviors are often the same for both of these states.
func (wb *WidgetBase) IsReadOnly() bool {
	return wb.StateIs(states.ReadOnly) || wb.StateIs(states.Disabled)
}

// SetReadOnly sets the ReadOnly state flag to given value
func (wb *WidgetBase) SetReadOnly(ro bool) *WidgetBase {
	return wb.SetState(ro, states.ReadOnly)
}

// HasFlagWithin returns whether the current node or any
// of its children have the given flag.
func (wb *WidgetBase) HasFlagWithin(flag enums.BitFlag) bool {
	got := false
	wb.WidgetWalkPre(func(wi Widget, wb *WidgetBase) bool {
		if wb.Is(flag) {
			got = true
			return ki.Break
		}
		return ki.Continue
	})
	return got
}

// HasStateWithin returns whether the current node or any
// of its children have the given state flag.
func (wb *WidgetBase) HasStateWithin(state states.States) bool {
	got := false
	wb.WidgetWalkPre(func(wi Widget, wb *WidgetBase) bool {
		if wb.StateIs(state) {
			got = true
			return ki.Break
		}
		return ki.Continue
	})
	return got
}
