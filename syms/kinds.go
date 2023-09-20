// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syms

//go:generate enumgen

import (
	"reflect"
)

// Kinds is a complete set of basic type categories and sub(sub..) categories -- these
// describe builtin types -- user-defined types must be some combination / version
// of these builtin types.
//
// See: https://en.wikipedia.org/wiki/List_of_data_structures
type Kinds int //enums:enum

// CatMap is the map into the category level for each kind
var CatMap map[Kinds]Kinds

// SubCatMap is the map into the sub-category level for each kind
var SubCatMap map[Kinds]Kinds

// Sub2CatMap is the map into the sub2-category level for each kind
var Sub2CatMap map[Kinds]Kinds

func init() {
	InitCatMap()
	InitSubCatMap()
	InitSub2CatMap()
}

// Cat returns the category that a given kind lives in, using CatMap
func (tk Kinds) Cat() Kinds {
	return CatMap[tk]
}

// SubCat returns the sub-category that a given kind lives in, using SubCatMap
func (tk Kinds) SubCat() Kinds {
	return SubCatMap[tk]
}

// Sub2Cat returns the sub2-category that a given kind lives in, using Sub2CatMap
func (tk Kinds) Sub2Cat() Kinds {
	return Sub2CatMap[tk]
}

// IsCat returns true if this is a category-level kind
func (tk Kinds) IsCat() bool {
	return tk.Cat() == tk
}

// IsSubCat returns true if this is a sub-category-level kind
func (tk Kinds) IsSubCat() bool {
	return tk.SubCat() == tk
}

// IsSub2Cat returns true if this is a sub2-category-level kind
func (tk Kinds) IsSub2Cat() bool {
	return tk.Sub2Cat() == tk
}

func (tk Kinds) InCat(other Kinds) bool {
	return tk.Cat() == other.Cat()
}

func (tk Kinds) InSubCat(other Kinds) bool {
	return tk.SubCat() == other.SubCat()
}

func (tk Kinds) InSub2Cat(other Kinds) bool {
	return tk.Sub2Cat() == other.Sub2Cat()
}

func (tk Kinds) IsPtr() bool {
	return tk >= Ptr && tk <= UnsafePtr
}

func (tk Kinds) IsPrimitiveNonPtr() bool {
	return tk.Cat() == Primitive && !tk.IsPtr()
}

// The list of Kinds
const (
	// Unknown is the nil kind -- kinds should be known in general..
	Unknown Kinds = iota

	// Category: Primitive, in the strict sense of low-level, atomic, small, fixed size
	Primitive

	// SubCat: Numeric
	Numeric

	// Sub2Cat: Integer
	Integer

	// Sub3Cat: Signed -- track this using props in types, not using Sub3 level
	Signed
	Int
	Int8
	Int16
	Int32
	Int64

	// Sub3Cat: Unsigned
	Unsigned
	Uint
	Uint8
	Uint16
	Uint32
	Uint64
	Uintptr // generic raw pointer data value -- see also Ptr, Ref for more semantic cases

	// Sub3Cat: Ptr, Ref etc -- in Numeric, Integer even though in some languages
	// pointer arithmetic might not be allowed, for some cases, etc
	Ptr       // pointer -- element is what we point to (kind of a composite type)
	Ref       // reference -- element is what we refer to
	UnsafePtr // for case where these are distinguished from Ptr (Go) -- similar to Uintptr

	// Sub2Cat: Fixed point -- could be under integer, but..
	Fixed
	Fixed26_6
	Fixed16_6
	Fixed0_32

	// Sub2Cat: Floating point
	Float
	Float16
	Float32
	Float64

	// Sub3Cat: Complex -- under floating point
	Complex
	Complex64
	Complex128

	// SubCat: Bool
	Bool

	// Category: Composite -- types composed of above primitive types
	Composite

	// SubCat: Tuple -- a fixed length 1d collection of elements that can be of any type
	// Type.Els required for each element
	Tuple
	Range // a special kind of tuple for Python ranges

	// SubCat: Array -- a fixed length 1d collection of same-type elements
	// Type.Els has one element for type
	Array

	// SubCat: List -- a variable-length 1d collection of same-type elements
	// This is Slice for Go
	// Type.Els has one element for type
	List
	String // List of some type of char rep -- Type.Els is type, as all Lists

	// SubCat: Matrix -- a twod collection of same-type elements
	// has two Size values, one for each dimension
	Matrix

	// SubCat: Tensor -- an n-dimensional collection of same-type elements
	// first element of Size is number of dimensions, rest are dimensions
	Tensor

	// SubCat: Map -- an associative array / hash map / dictionary
	// Type.Els first el is key, second is type
	Map

	// SubCat: Set -- typically a degenerate form of hash map with no value
	Set
	FrozenSet // python's frozen set of fixed values

	// SubCat: Struct -- like a tuple but with specific semantics in most languages
	// Type.Els are the fields, and if there is an inheritance relationship these
	// are put first with relevant identifiers -- in Go these are unnamed fields
	Struct
	Class
	Object

	// Chan: a channel (Go Specific)
	Chan

	// Category: Function -- types that are functions
	// Type.Els are the params and return values in order, with Size[0] being number
	// of params and Size[1] number of returns
	Function

	// SubCat: Func -- a standalone function
	Func

	// SubCat: Method -- a function with a specific receiver (e.g., on a Class in C++,
	// or on any type in Go).
	// First Type.Els is receiver param -- included in Size[0]
	Method

	// SubCat: Interface -- an abstract definition of a set of methods (in Go)
	// Type.Els are the Methods with the receiver type missing or Unknown
	Interface
)

// Categories
var Cats = []Kinds{
	Unknown,
	Primitive,
	Composite,
	Function,
	KindsN,
}

// Sub-Categories
var SubCats = []Kinds{
	Unknown,
	Primitive,
	Numeric,
	Bool,
	Composite,
	Tuple,
	Array,
	List,
	Matrix,
	Tensor,
	Map,
	Set,
	Struct,
	Chan,
	Function,
	Func,
	Method,
	Interface,
	KindsN,
}

// Sub2-Categories
var Sub2Cats = []Kinds{
	Unknown,
	Primitive,
	Numeric,
	Integer,
	Fixed,
	Float,
	Bool,
	Composite,
	Tuple,
	Array,
	List,
	Matrix,
	Tensor,
	Map,
	Set,
	Struct,
	Chan,
	Function,
	Func,
	Method,
	Interface,
	KindsN,
}

// InitCatMap initializes the CatMap
func InitCatMap() {
	if CatMap != nil {
		return
	}
	CatMap = make(map[Kinds]Kinds, KindsN)
	for tk := Unknown; tk < KindsN; tk++ {
		for c := 1; c < len(Cats); c++ {
			if tk < Cats[c] {
				CatMap[tk] = Cats[c-1]
				break
			}
		}
	}
}

// InitSubCatMap initializes the SubCatMap
func InitSubCatMap() {
	if SubCatMap != nil {
		return
	}
	SubCatMap = make(map[Kinds]Kinds, KindsN)
	for tk := Unknown; tk < KindsN; tk++ {
		for c := 1; c < len(SubCats); c++ {
			if tk < SubCats[c] {
				SubCatMap[tk] = SubCats[c-1]
				break
			}
		}
	}
}

// InitSub2CatMap initializes the SubCatMap
func InitSub2CatMap() {
	if Sub2CatMap != nil {
		return
	}
	Sub2CatMap = make(map[Kinds]Kinds, KindsN)
	for tk := Unknown; tk < KindsN; tk++ {
		for c := 1; c < len(SubCats); c++ {
			if tk < Sub2Cats[c] {
				Sub2CatMap[tk] = Sub2Cats[c-1]
				break
			}
		}
	}
}

///////////////////////////////////////////////////////////////////////
//   Reflect map

// ReflectKindMap maps reflect kinds to syms.Kinds

var ReflectKindMap = map[reflect.Kind]Kinds{
	reflect.Invalid:       Unknown,
	reflect.Int:           Int,
	reflect.Int8:          Int8,
	reflect.Int16:         Int16,
	reflect.Int32:         Int32,
	reflect.Int64:         Int64,
	reflect.Uint:          Uint,
	reflect.Uint8:         Uint8,
	reflect.Uint16:        Uint16,
	reflect.Uint32:        Uint32,
	reflect.Uint64:        Uint64,
	reflect.Uintptr:       Uintptr,
	reflect.Float32:       Float32,
	reflect.Float64:       Float64,
	reflect.Complex64:     Complex64,
	reflect.Complex128:    Complex128,
	reflect.Bool:          Bool,
	reflect.Array:         Array,
	reflect.Chan:          Chan,
	reflect.Func:          Func,
	reflect.Interface:     Interface,
	reflect.Map:           Map,
	reflect.Ptr:           Ptr,
	reflect.Slice:         List,
	reflect.String:        String,
	reflect.Struct:        Struct,
	reflect.UnsafePointer: Ptr,
}
