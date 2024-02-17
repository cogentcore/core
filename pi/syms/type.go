// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syms

import (
	"fmt"
	"io"
	"slices"

	"cogentcore.org/core/glop/indent"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/pi/lex"
)

// Type contains all the information about types.  Types can be builtin
// or composed of builtin types.  Each type can have one or more elements,
// e.g., fields for a struct or class, multiple values for a go function,
// or the two types for a map (key, value), etc..
type Type struct {

	// name of the type -- can be the name of a field or the role for a type element
	Name string

	// kind of type -- overall nature of the type
	Kind Kinds

	// documentation about this type, extracted from code
	Desc string

	// set to true after type has been initialized during post-parse processing
	Inited bool `inactive:"-"`

	// elements of this type -- ordering and meaning varies depending on the Kind of type -- for Primitive types this is the parent type, for Composite types it describes the key elements of the type: Tuple = each element's type; Array = type of elements; Struct = each field, etc (see docs for each in Kinds)
	Els TypeEls

	// methods defined for this type
	Meths TypeMap

	// for primitive types, this is the number of bytes, for composite types, it is the number of elements, which can be multi-dimensional (e.g., for functions, number of params is (including receiver param for methods) and return vals is )
	Size []int

	// full filename / URI of source where type is defined (may be empty for auto types)
	Filename string

	// region in source encompassing this type
	Region lex.Reg

	// relevant scoping / parent symbols, e.g., namespace, package, module, class, function, etc..
	Scopes SymNames

	// additional type properties, such as const, virtual, static -- these are just recorded textually and not systematized to keep things open-ended -- many of the most important properties can be inferred from the Kind property
	Props ki.Props

	// Ast node that corresponds to this type -- only valid during parsing
	Ast ki.Ki `json:"-" xml:"-"`
}

// NewType returns a new Type struct initialized with given name and kind
func NewType(name string, kind Kinds) *Type {
	ty := &Type{Name: name, Kind: kind}
	return ty
}

// AllocScopes allocates scopes map if nil
func (ty *Type) AllocScopes() {
	if ty.Scopes == nil {
		ty.Scopes = make(SymNames)
	}
}

// CopyFromSrc copies source-level data from given other type
func (ty *Type) CopyFromSrc(cp *Type) {
	ty.Filename = cp.Filename
	ty.Region = cp.Region
	if cp.Ast != nil {
		ty.Ast = cp.Ast
	}
}

// Clone returns a deep copy of this type, cloning / copying all sub-elements
// except the Ast, and Inited
func (ty *Type) Clone() *Type {
	// note: not copying Inited
	nty := &Type{Name: ty.Name, Kind: ty.Kind, Desc: ty.Desc, Filename: ty.Filename, Region: ty.Region, Ast: ty.Ast}
	nty.Els.CopyFrom(ty.Els)
	nty.Meths = ty.Meths.Clone()
	nty.Size = slices.Clone(ty.Size)
	nty.Scopes = ty.Scopes.Clone()
	ty.Props.IterCb(func(key string, v any) {
		nty.Props.Set(key, v)
	})
	//nty.Props.CopyFrom(ty.Props, true)
	return nty
}

// AddScopesStack adds a given scope element(s) from stack to this Type.
func (ty *Type) AddScopesStack(ss SymStack) {
	sz := len(ss)
	if sz == 0 {
		return
	}
	ty.AllocScopes()
	for i := 0; i < sz; i++ {
		sc := ss[i]
		ty.Scopes[sc.Kind] = sc.Name
	}
}

// String() satisfies the fmt.Stringer interface
func (ty *Type) String() string {
	switch {
	case ty.Kind.Cat() == Function && len(ty.Size) == 2:
		str := "func "
		npars := ty.Size[0]
		if ty.Kind.SubCat() == Method {
			str += "(" + ty.Els.StringRange(0, 1) + ") " + ty.Name + "(" + ty.Els.StringRange(1, npars-1) + ")"
		} else {
			str += ty.Name + "(" + ty.Els.StringRange(0, npars) + ")"
		}
		nrets := ty.Size[1]
		if nrets == 1 {
			tel := ty.Els[npars]
			str += " " + tel.Type
		} else if nrets > 1 {
			str += " (" + ty.Els.StringRange(npars, nrets) + ")"
		}
		return str
	case ty.Kind.SubCat() == Map:
		return "map[" + ty.Els[0].Type + "]" + ty.Els[1].Type
	case ty.Kind == String:
		return "string"
	case ty.Kind.SubCat() == List:
		return "[]" + ty.Els[0].Type
	}
	return ty.Name + ": " + ty.Kind.String()
}

// ArgString() returns string of args to function if it is a function type
func (ty *Type) ArgString() string {
	if ty.Kind.Cat() != Function || len(ty.Size) != 2 {
		return ""
	}
	npars := ty.Size[0]
	if ty.Kind.SubCat() == Method {
		return ty.Els.StringRange(1, npars-1)
	} else {
		return ty.Els.StringRange(0, npars)
	}
}

// ReturnString() returns string of return vals of function if it is a function type
func (ty *Type) ReturnString() string {
	if ty.Kind.Cat() != Function || len(ty.Size) != 2 {
		return ""
	}
	npars := ty.Size[0]
	nrets := ty.Size[1]
	if nrets == 1 {
		tel := ty.Els[npars]
		return tel.Type
	} else if nrets > 1 {
		return "(" + ty.Els.StringRange(npars, nrets) + ")"
	}
	return ""
}

// NonPtrType returns the non-pointer name of this type, if it is a pointer type
// otherwise just returns Name
func (ty *Type) NonPtrType() string {
	if !(ty.Kind == Ptr || ty.Kind == Ref || ty.Kind == UnsafePtr) {
		return ty.Name
	}
	if len(ty.Els) == 1 {
		return ty.Els[0].Type
	}
	return ty.Name // shouldn't happen
}

// WriteDoc writes basic doc info
func (ty *Type) WriteDoc(out io.Writer, depth int) {
	ind := indent.Tabs(depth)
	fmt.Fprintf(out, "%v%v: %v", ind, ty.Name, ty.Kind)
	if len(ty.Size) == 1 {
		fmt.Fprintf(out, " Size: %v", ty.Size[0])
	} else if len(ty.Size) > 1 {
		fmt.Fprint(out, " Size: { ")
		for i := range ty.Size {
			fmt.Fprintf(out, "%v, ", ty.Size[i])
		}
		fmt.Fprint(out, " }")
	}
	if ty.Kind.SubCat() == Struct && len(ty.Els) > 0 {
		fmt.Fprint(out, " {\n")
		indp := indent.Tabs(depth + 1)
		for i := range ty.Els {
			fmt.Fprintf(out, "%v%v\n", indp, ty.Els[i].String())
		}
		fmt.Fprintf(out, "%v}\n", ind)
	} else {
		fmt.Fprint(out, "\n")
	}
	if len(ty.Meths) > 0 {
		fmt.Fprint(out, "Methods: {\n")
		indp := indent.Tabs(depth + 1)
		for _, m := range ty.Meths {
			fmt.Fprintf(out, "%v%v\n", indp, m.String())
		}
		fmt.Fprintf(out, "%v}\n", ind)
	} else {
		fmt.Fprint(out, "\n")
	}
}

//////////////////////////////////////////////////////////////////////////////////
// TypeEls

// TypeEl is a type element -- has a name (local to the type, e.g., field name)
// and a type name that can be looked up in a master list of types
type TypeEl struct {

	// element name -- e.g., field name for struct, or functional name for other types
	Name string

	// type name -- looked up on relevant lists -- includes scoping / package / namespace name as appropriate
	Type string
}

// String() satisfies the fmt.Stringer interface
func (tel *TypeEl) String() string {
	if tel.Name != "" {
		return tel.Name + " " + tel.Type
	}
	return tel.Type
}

// Clone() returns a copy of this el
func (tel *TypeEl) Clone() *TypeEl {
	te := &TypeEl{Name: tel.Name, Type: tel.Type}
	return te
}

// TypeEls are the type elements for types
type TypeEls []TypeEl

// Add adds a new type element
func (te *TypeEls) Add(nm, typ string) {
	*te = append(*te, TypeEl{Name: nm, Type: typ})
}

// CopyFrom copies from another list
func (te *TypeEls) CopyFrom(cp TypeEls) {
	for _, t := range cp {
		*te = append(*te, t)
	}
}

// ByName returns type el with given name, nil if not there
func (te *TypeEls) ByName(nm string) *TypeEl {
	for i := range *te {
		el := &(*te)[i]
		if el.Name == nm {
			return el
		}
	}
	return nil
}

// ByType returns type el with given type, nil if not there
func (te *TypeEls) ByType(typ string) *TypeEl {
	for i := range *te {
		el := &(*te)[i]
		if el.Type == typ {
			return el
		}
	}
	return nil
}

// String() satisfies the fmt.Stringer interface
func (te *TypeEls) String() string {
	return te.StringRange(0, len(*te))
}

// StringRange() returns a string rep of range of items
func (te *TypeEls) StringRange(st, n int) string {
	n = min(n, len(*te))
	str := ""
	for i := 0; i < n; i++ {
		tel := (*te)[st+i]
		str += tel.String()
		if i < n-1 {
			str += ", "
		}
	}
	return str
}
