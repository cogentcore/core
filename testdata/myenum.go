package testdata

// MyEnum is an enum
type MyEnum int //enums:enum

const (
	Sunday MyEnum = iota
	Monday
	Tuesday
	Wednesday
	Thursday
	Friday
	Saturday
)

// MyBitEnum is a bitflag enum
type MyBitEnum int //enums:bitflag

const (
	Apple MyBitEnum = iota
	Orange
	Peach
	Blueberry
	Grapefruit
	Strawberry
)
