// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syms

import (
	"fmt"
	"io"

	"github.com/goki/ki/indent"
	"github.com/goki/ki/ki"
	"github.com/goki/pi/lex"
)

// Type contains all the information about types.  Types can be builtin
// or composed of builtin types.  Each type can have one or more elements,
// e.g., fields for a struct or class, multiple values for a go function,
// or the two types for a map (key, value), etc..
type Type struct {
	Name     string   `desc:"name of the type -- can be the name of a field or the role for a type element"`
	Kind     Kinds    `desc:"kind of type -- overall nature of the type"`
	Desc     string   `desc:"documentation about this type, extracted from code"`
	Els      TypeEls  `desc:"elements of this type -- ordering and meaning varies depending on the Kind of type -- for Primitive types this is the parent type, for Composite types it describes the key elements of the type: Tuple = each element's type; Array = type of elements; Struct = each field, etc (see docs for each in Kinds)"`
	Size     []int    `desc:"for primitive types, this is the number of bytes, for composite types, it is the number of elements, which can be multi-dimensional (e.g., for functions, number of params is [0] and return vals is [1])"`
	Filename string   `desc:"full filename / URI of source where type is defined (may be empty for auto types)"`
	Region   lex.Reg  `desc:"region in source encompassing this type"`
	Scopes   SymNames `desc:"relevant scoping / parent symbols, e.g., namespace, package, module, class, function, etc.."`
	Props    ki.Props `desc:"additional type properties, such as const, virtual, static -- these are just recorded textually and not systematized to keep things open-ended -- many of the most important properties can be inferred from the Kind property"`
	Ast      ki.Ki    `json:"-" xml:"-" desc:"Ast node that corresponds to this type -- only valid during parsing"`
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
	return ty.Name + ": " + ty.Kind.String()
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
	if len(ty.Els) > 0 {
		fmt.Fprint(out, " {\n")
		indp := indent.Tabs(depth + 1)
		for i := range ty.Els {
			fmt.Fprintf(out, "%v%v: %v\n", indp, ty.Els[i].Name, ty.Els[i].Type)
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
	Name string `desc:"element name -- e.g., field name for struct, or functional name for other types"`
	Type string `desc:"type name -- looked up on relevant lists -- includes scoping / package / namespace name as appropriate"`
}

// TypeEls are the type elements for types
type TypeEls []TypeEl

// Add adds a new type element
func (te *TypeEls) Add(nm, typ string) {
	(*te) = append(*te, TypeEl{Name: nm, Type: typ})
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
