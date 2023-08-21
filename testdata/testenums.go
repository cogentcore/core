package testdata

// Days is an enum containing the days of the week
type Days int //enums:enum

const (
	// Sunday is the first day of the week
	Sunday Days = iota
	// Monday is the second day of the week
	Monday
	// Tuesday is the third day of the week
	Tuesday
	// Wednesday is the fourth day of the week
	Wednesday
	// Thursday is the fifth day of the week
	Thursday
	// Friday is the sixth day of the week
	Friday
	// Saturday is the seventh day of the week
	Saturday
)

// States is a bitflag enum containing widget states
type States int64 //enums:bitflag

const (
	// Enabled indicates the widget is enabled
	Enabled States = iota
	// Disabled indicates the widget is disabled
	Disabled
	// Focused indicates the widget has keyboard focus
	Focused
	// Hovered indicates the widget is being hovered over
	Hovered
	// Active indicates the widget is being interacted with
	Active
	// Selected indicates the widget is selected
	Selected
)
