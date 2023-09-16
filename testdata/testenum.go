package testdata

//go:generate enumgen

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
