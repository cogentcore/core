package testdata

// Fruits is an enum containing fruits
type Fruits uint8 //enums:enum

const (
	Apple Fruits = iota
	Orange
	Peach
	Strawberry
	Blackberry
	Blueberry
	Apricot
	OrangeFruit = Orange
)

// Foods is an enum containing foods
type Foods Fruits //enums:enum

const (
	Bread Foods = Foods(FruitsN) + iota
	Lettuce
	Cheese
	Meat
)

// Days is an enum containing the days of the week
type Days int //enums:enum -transform=snake_upper -addprefix=DAY_ -gql -no-json

const (
	// Sunday is the first day of the week
	Sunday Days = 2*iota + 1
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
type States int64 //enums:bitflag -no-text -line-comment -transform kebab -sql -trim-prefix Ho

const (
	// Enabled indicates the widget is enabled
	Enabled States = 2*iota + 1
	// Disabled indicates the widget is disabled
	Disabled //NotEnabled
	// Focused indicates the widget has keyboard focus
	Focused
	// Hovered indicates the widget is being hovered over
	Hovered
	// Active indicates the widget is being interacted with
	Active //CurrentlyBeingPressedByUser
	// ActivelyFocused indicates the widget has active keyboard focus
	ActivelyFocused
	// Selected indicates the widget is selected
	Selected
)

// Languages is a bitflag enum containing programming languages
type Languages int64 //enums:bitflag

const (
	// Go is the best programming language
	Go Languages = 4*iota + 6
	Python
	// JavaScript is the worst programming language
	JavaScript
	Dart
	Rust
	Ruby
	C
	CPP
	ObjectiveC
	Java
	TypeScript
	Kotlin
	Swift
)

// MoreLanguages contains more programming languages
type MoreLanguages Languages //enums:bitflag

const (
	Perl MoreLanguages = MoreLanguages(LanguagesN) + iota
)
