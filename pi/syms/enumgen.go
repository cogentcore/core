// Code generated by "goki generate"; DO NOT EDIT.

package syms

import (
	"errors"
	"log"
	"strconv"

	"goki.dev/enums"
)

var _KindsValues = []Kinds{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50}

// KindsN is the highest valid value
// for type Kinds, plus one.
const KindsN Kinds = 51

// An "invalid array index" compiler error signifies that the constant values have changed.
// Re-run the enumgen command to generate them again.
func _KindsNoOp() {
	var x [1]struct{}
	_ = x[Unknown-(0)]
	_ = x[Primitive-(1)]
	_ = x[Numeric-(2)]
	_ = x[Integer-(3)]
	_ = x[Signed-(4)]
	_ = x[Int-(5)]
	_ = x[Int8-(6)]
	_ = x[Int16-(7)]
	_ = x[Int32-(8)]
	_ = x[Int64-(9)]
	_ = x[Unsigned-(10)]
	_ = x[Uint-(11)]
	_ = x[Uint8-(12)]
	_ = x[Uint16-(13)]
	_ = x[Uint32-(14)]
	_ = x[Uint64-(15)]
	_ = x[Uintptr-(16)]
	_ = x[Ptr-(17)]
	_ = x[Ref-(18)]
	_ = x[UnsafePtr-(19)]
	_ = x[Fixed-(20)]
	_ = x[Fixed26_6-(21)]
	_ = x[Fixed16_6-(22)]
	_ = x[Fixed0_32-(23)]
	_ = x[Float-(24)]
	_ = x[Float16-(25)]
	_ = x[Float32-(26)]
	_ = x[Float64-(27)]
	_ = x[Complex-(28)]
	_ = x[Complex64-(29)]
	_ = x[Complex128-(30)]
	_ = x[Bool-(31)]
	_ = x[Composite-(32)]
	_ = x[Tuple-(33)]
	_ = x[Range-(34)]
	_ = x[Array-(35)]
	_ = x[List-(36)]
	_ = x[String-(37)]
	_ = x[Matrix-(38)]
	_ = x[Tensor-(39)]
	_ = x[Map-(40)]
	_ = x[Set-(41)]
	_ = x[FrozenSet-(42)]
	_ = x[Struct-(43)]
	_ = x[Class-(44)]
	_ = x[Object-(45)]
	_ = x[Chan-(46)]
	_ = x[Function-(47)]
	_ = x[Func-(48)]
	_ = x[Method-(49)]
	_ = x[Interface-(50)]
}

var _KindsNameToValueMap = map[string]Kinds{
	`Unknown`:    0,
	`Primitive`:  1,
	`Numeric`:    2,
	`Integer`:    3,
	`Signed`:     4,
	`Int`:        5,
	`Int8`:       6,
	`Int16`:      7,
	`Int32`:      8,
	`Int64`:      9,
	`Unsigned`:   10,
	`Uint`:       11,
	`Uint8`:      12,
	`Uint16`:     13,
	`Uint32`:     14,
	`Uint64`:     15,
	`Uintptr`:    16,
	`Ptr`:        17,
	`Ref`:        18,
	`UnsafePtr`:  19,
	`Fixed`:      20,
	`Fixed26_6`:  21,
	`Fixed16_6`:  22,
	`Fixed0_32`:  23,
	`Float`:      24,
	`Float16`:    25,
	`Float32`:    26,
	`Float64`:    27,
	`Complex`:    28,
	`Complex64`:  29,
	`Complex128`: 30,
	`Bool`:       31,
	`Composite`:  32,
	`Tuple`:      33,
	`Range`:      34,
	`Array`:      35,
	`List`:       36,
	`String`:     37,
	`Matrix`:     38,
	`Tensor`:     39,
	`Map`:        40,
	`Set`:        41,
	`FrozenSet`:  42,
	`Struct`:     43,
	`Class`:      44,
	`Object`:     45,
	`Chan`:       46,
	`Function`:   47,
	`Func`:       48,
	`Method`:     49,
	`Interface`:  50,
}

var _KindsDescMap = map[Kinds]string{
	0:  `Unknown is the nil kind -- kinds should be known in general..`,
	1:  `Category: Primitive, in the strict sense of low-level, atomic, small, fixed size`,
	2:  `SubCat: Numeric`,
	3:  `Sub2Cat: Integer`,
	4:  `Sub3Cat: Signed -- track this using props in types, not using Sub3 level`,
	5:  ``,
	6:  ``,
	7:  ``,
	8:  ``,
	9:  ``,
	10: `Sub3Cat: Unsigned`,
	11: ``,
	12: ``,
	13: ``,
	14: ``,
	15: ``,
	16: ``,
	17: `Sub3Cat: Ptr, Ref etc -- in Numeric, Integer even though in some languages pointer arithmetic might not be allowed, for some cases, etc`,
	18: ``,
	19: ``,
	20: `Sub2Cat: Fixed point -- could be under integer, but..`,
	21: ``,
	22: ``,
	23: ``,
	24: `Sub2Cat: Floating point`,
	25: ``,
	26: ``,
	27: ``,
	28: `Sub3Cat: Complex -- under floating point`,
	29: ``,
	30: ``,
	31: `SubCat: Bool`,
	32: `Category: Composite -- types composed of above primitive types`,
	33: `SubCat: Tuple -- a fixed length 1d collection of elements that can be of any type Type.Els required for each element`,
	34: ``,
	35: `SubCat: Array -- a fixed length 1d collection of same-type elements Type.Els has one element for type`,
	36: `SubCat: List -- a variable-length 1d collection of same-type elements This is Slice for Go Type.Els has one element for type`,
	37: ``,
	38: `SubCat: Matrix -- a twod collection of same-type elements has two Size values, one for each dimension`,
	39: `SubCat: Tensor -- an n-dimensional collection of same-type elements first element of Size is number of dimensions, rest are dimensions`,
	40: `SubCat: Map -- an associative array / hash map / dictionary Type.Els first el is key, second is type`,
	41: `SubCat: Set -- typically a degenerate form of hash map with no value`,
	42: ``,
	43: `SubCat: Struct -- like a tuple but with specific semantics in most languages Type.Els are the fields, and if there is an inheritance relationship these are put first with relevant identifiers -- in Go these are unnamed fields`,
	44: ``,
	45: ``,
	46: `Chan: a channel (Go Specific)`,
	47: `Category: Function -- types that are functions Type.Els are the params and return values in order, with Size[0] being number of params and Size[1] number of returns`,
	48: `SubCat: Func -- a standalone function`,
	49: `SubCat: Method -- a function with a specific receiver (e.g., on a Class in C++, or on any type in Go). First Type.Els is receiver param -- included in Size[0]`,
	50: `SubCat: Interface -- an abstract definition of a set of methods (in Go) Type.Els are the Methods with the receiver type missing or Unknown`,
}

var _KindsMap = map[Kinds]string{
	0:  `Unknown`,
	1:  `Primitive`,
	2:  `Numeric`,
	3:  `Integer`,
	4:  `Signed`,
	5:  `Int`,
	6:  `Int8`,
	7:  `Int16`,
	8:  `Int32`,
	9:  `Int64`,
	10: `Unsigned`,
	11: `Uint`,
	12: `Uint8`,
	13: `Uint16`,
	14: `Uint32`,
	15: `Uint64`,
	16: `Uintptr`,
	17: `Ptr`,
	18: `Ref`,
	19: `UnsafePtr`,
	20: `Fixed`,
	21: `Fixed26_6`,
	22: `Fixed16_6`,
	23: `Fixed0_32`,
	24: `Float`,
	25: `Float16`,
	26: `Float32`,
	27: `Float64`,
	28: `Complex`,
	29: `Complex64`,
	30: `Complex128`,
	31: `Bool`,
	32: `Composite`,
	33: `Tuple`,
	34: `Range`,
	35: `Array`,
	36: `List`,
	37: `String`,
	38: `Matrix`,
	39: `Tensor`,
	40: `Map`,
	41: `Set`,
	42: `FrozenSet`,
	43: `Struct`,
	44: `Class`,
	45: `Object`,
	46: `Chan`,
	47: `Function`,
	48: `Func`,
	49: `Method`,
	50: `Interface`,
}

// String returns the string representation
// of this Kinds value.
func (i Kinds) String() string {
	if str, ok := _KindsMap[i]; ok {
		return str
	}
	return strconv.FormatInt(int64(i), 10)
}

// SetString sets the Kinds value from its
// string representation, and returns an
// error if the string is invalid.
func (i *Kinds) SetString(s string) error {
	if val, ok := _KindsNameToValueMap[s]; ok {
		*i = val
		return nil
	}
	return errors.New(s + " is not a valid value for type Kinds")
}

// Int64 returns the Kinds value as an int64.
func (i Kinds) Int64() int64 {
	return int64(i)
}

// SetInt64 sets the Kinds value from an int64.
func (i *Kinds) SetInt64(in int64) {
	*i = Kinds(in)
}

// Desc returns the description of the Kinds value.
func (i Kinds) Desc() string {
	if str, ok := _KindsDescMap[i]; ok {
		return str
	}
	return i.String()
}

// KindsValues returns all possible values
// for the type Kinds.
func KindsValues() []Kinds {
	return _KindsValues
}

// Values returns all possible values
// for the type Kinds.
func (i Kinds) Values() []enums.Enum {
	res := make([]enums.Enum, len(_KindsValues))
	for i, d := range _KindsValues {
		res[i] = d
	}
	return res
}

// IsValid returns whether the value is a
// valid option for type Kinds.
func (i Kinds) IsValid() bool {
	_, ok := _KindsMap[i]
	return ok
}

// MarshalText implements the [encoding.TextMarshaler] interface.
func (i Kinds) MarshalText() ([]byte, error) {
	return []byte(i.String()), nil
}

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
func (i *Kinds) UnmarshalText(text []byte) error {
	if err := i.SetString(string(text)); err != nil {
		log.Println("Kinds.UnmarshalText:", err)
	}
	return nil
}
