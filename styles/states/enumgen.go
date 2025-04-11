// Code generated by "core generate"; DO NOT EDIT.

package states

import (
	"cogentcore.org/core/enums"
)

var _StatesValues = []States{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14}

// StatesN is the highest valid value for type States, plus one.
const StatesN States = 15

var _StatesValueMap = map[string]States{`Invisible`: 0, `Disabled`: 1, `ReadOnly`: 2, `Selected`: 3, `Active`: 4, `Dragging`: 5, `Sliding`: 6, `Focused`: 7, `Attended`: 8, `Checked`: 9, `Indeterminate`: 10, `Hovered`: 11, `LongHovered`: 12, `LongPressed`: 13, `DragHovered`: 14}

var _StatesDescMap = map[States]string{0: `Invisible elements are not displayable, and thus do not present a target for GUI events. It is identical to CSS display:none. It is often used for elements such as tabs to hide elements in tabs that are not open. Elements can be made visible by toggling this flag and thus in general should be constructed and styled, but a new layout step must generally be taken after visibility status has changed. See also [cogentcore.org/core/core.WidgetBase.IsDisplayable].`, 1: `Disabled elements cannot be interacted with or selected, but do display.`, 2: `ReadOnly elements cannot be changed, but can be selected. A text input must not be ReadOnly for entering text. A button can be pressed while ReadOnly -- if not ReadOnly then the label on the button can be edited, for example.`, 3: `Selected elements have been marked for clipboard or other such actions.`, 4: `Active elements are currently being interacted with, usually involving a mouse button being pressed in the element. A text field will be active while being clicked on, and this can also result in a [Focused] state. If further movement happens, an element can also end up being Dragged or Sliding.`, 5: `Dragging means this element is currently being dragged by the mouse (i.e., a MouseDown event followed by MouseMove), as part of a drag-n-drop sequence.`, 6: `Sliding means this element is currently being manipulated via mouse to change the slider state, which will continue until the mouse is released, even if it goes off the element. It should also still be [Active].`, 7: `Focused elements receive keyboard input. Only one element can be Focused at a time.`, 8: `Attended elements are the last Activatable elements to be clicked on. Only one element can be Attended at a time. The main effect of Attended is on scrolling events: see [abilities.ScrollableUnattended]`, 9: `Checked is for check boxes or radio buttons or other similar state.`, 10: `Indeterminate indicates that the true state of an item is unknown. For example, [Checked] state items may be in an uncertain state if they represent other checked items, some of which are checked and some of which are not.`, 11: `Hovered indicates that a mouse pointer has entered the space over an element, but it is not [Active] (nor [DragHovered]).`, 12: `LongHovered indicates a Hover event that persists without significant movement for a minimum period of time (e.g., 500 msec), which typically triggers a tooltip popup.`, 13: `LongPressed indicates a MouseDown event that persists without significant movement for a minimum period of time (e.g., 500 msec), which typically triggers a tooltip and/or context menu popup.`, 14: `DragHovered indicates that a mouse pointer has entered the space over an element during a drag-n-drop sequence. This makes it a candidate for a potential drop target.`}

var _StatesMap = map[States]string{0: `Invisible`, 1: `Disabled`, 2: `ReadOnly`, 3: `Selected`, 4: `Active`, 5: `Dragging`, 6: `Sliding`, 7: `Focused`, 8: `Attended`, 9: `Checked`, 10: `Indeterminate`, 11: `Hovered`, 12: `LongHovered`, 13: `LongPressed`, 14: `DragHovered`}

// String returns the string representation of this States value.
func (i States) String() string { return enums.BitFlagString(i, _StatesValues) }

// BitIndexString returns the string representation of this States value
// if it is a bit index value (typically an enum constant), and
// not an actual bit flag value.
func (i States) BitIndexString() string { return enums.String(i, _StatesMap) }

// SetString sets the States value from its string representation,
// and returns an error if the string is invalid.
func (i *States) SetString(s string) error { *i = 0; return i.SetStringOr(s) }

// SetStringOr sets the States value from its string representation
// while preserving any bit flags already set, and returns an
// error if the string is invalid.
func (i *States) SetStringOr(s string) error {
	return enums.SetStringOr(i, s, _StatesValueMap, "States")
}

// Int64 returns the States value as an int64.
func (i States) Int64() int64 { return int64(i) }

// SetInt64 sets the States value from an int64.
func (i *States) SetInt64(in int64) { *i = States(in) }

// Desc returns the description of the States value.
func (i States) Desc() string { return enums.Desc(i, _StatesDescMap) }

// StatesValues returns all possible values for the type States.
func StatesValues() []States { return _StatesValues }

// Values returns all possible values for the type States.
func (i States) Values() []enums.Enum { return enums.Values(_StatesValues) }

// HasFlag returns whether these bit flags have the given bit flag set.
func (i *States) HasFlag(f enums.BitFlag) bool { return enums.HasFlag((*int64)(i), f) }

// SetFlag sets the value of the given flags in these flags to the given value.
func (i *States) SetFlag(on bool, f ...enums.BitFlag) { enums.SetFlag((*int64)(i), on, f...) }

// MarshalText implements the [encoding.TextMarshaler] interface.
func (i States) MarshalText() ([]byte, error) { return []byte(i.String()), nil }

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
func (i *States) UnmarshalText(text []byte) error { return enums.UnmarshalText(i, text, "States") }
