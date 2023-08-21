package testdata

// MyEnum is an enum
type MyEnum int //enums:enum

const (
	// Sunday is the first day of the week
	Sunday MyEnum = iota + 1
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

// MyBitEnum is a bitflag enum
type MyBitEnum int //enums:bitflag

const (
	// An Apple is a red fruit
	Apple MyBitEnum = iota
	// An Orange is an orange fruit
	Orange
	// A Peach is a stonefruit
	Peach
	// A Blueberry is a blue berry
	Blueberry
	// A Grapefruit is large fruit
	Grapefruit
	// A Strawberry is a small red fruit
	Strawberry
)
