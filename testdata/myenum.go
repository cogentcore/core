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
