// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package syms defines the symbols and their properties that
// are accumulated from a parsed file, and are then used for
// e.g., completion lookup, etc.
//
// We looked at several different standards for formats, types, etc:
//
// LSP: https://microsoft.github.io/language-server-protocol/specification
// useful to enable GoPi to act as an LSP server at some point.
// additional symbol kinds:
// https://github.com/Microsoft/language-server-protocol/issues/344
//
// See also: github.com/sourcegraph/sourcegraph
// and specifically: /cmd/frontend/graphqlbackend/search_symbols.go
// it seems to use https://github.com/universal-ctags/ctags
// for the raw data..
//
// Other relevant guidance comes from the go compiler system which is
// used extensively in github.com/mdemsky/gocode for example.
// In particular: go/types/scope.go type.go, and package.go contain the
// relevant data structures for how information is organized for
// compiled go packages, which have all this data cached and avail
// to be imported via the go/importer which returns a go/types/Package
// which in turn contains Scope's which in turn contain Objects that
// define the elements of the compiled language.
package syms

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/goki/ki/indent"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/pi/lex"
	"github.com/goki/pi/token"
)

// Symbol contains the information for everything about a given
// symbol that is created by parsing, and can be looked up.
// It corresponds to the LSP DocumentSymbol structure, and
// the Go Object type.
type Symbol struct {
	Name      string       `desc:"name of the symbol"`
	Detail    string       `desc:"additional detail and specification of the symbol -- e.g. if a function, the signature of the function"`
	Kind      token.Tokens `desc:"lexical kind of symbol, using token.Tokens list"`
	Type      string       `desc:"Type name for this symbol -- if it is a type, this is its corresponding type representation -- if it is a variable then this is its type"`
	Index     int          `desc:"index for ordering children within a given scope, e.g., fields in a struct / class"`
	Filename  string       `desc:"full filename / URI of source"`
	Region    lex.Reg      `desc:"region in source encompassing this item -- if = RegZero then this is a temp symbol and children are not added to it"`
	SelectReg lex.Reg      `desc:"region that should be selected when activated, etc"`
	Scopes    SymNames     `desc:"relevant scoping / parent symbols, e.g., namespace, package, module, class, function, etc.."`
	Children  SymMap       `desc:"children of this symbol -- this includes e.g., methods and fields of classes / structs / types, and all elements within packages, etc"`
	Types     TypeMap      `desc:"types defined within the scope of this symbol"`
	Ast       ki.Ki        `json:"-" xml:"-" desc:"Ast node that created this symbol -- only valid during parsing"`
}

var KiT_Symbol = kit.Types.AddType(&Symbol{}, nil)

// NewSymbol returns a new symbol with the basic info filled in -- SelectReg defaults to Region
func NewSymbol(name string, kind token.Tokens, fname string, reg lex.Reg) *Symbol {
	sy := &Symbol{Name: name, Kind: kind, Filename: fname, Region: reg, SelectReg: reg}
	return sy
}

// CopyFromSrc copies all the source-related fields from other symbol
// (no Type, Types, or Children).  Ast is only copied if non-nil.
func (sy *Symbol) CopyFromSrc(cp *Symbol) {
	sy.Detail = cp.Detail
	sy.Kind = cp.Kind
	sy.Index = cp.Index
	sy.Filename = cp.Filename
	sy.Region = cp.Region
	sy.SelectReg = cp.SelectReg
	if cp.Ast != nil {
		sy.Ast = cp.Ast
	}
}

// IsTemp returns true if this is temporary symbol that is used for scoping but is not
// otherwise permanently added to list of symbols.  Indicated by Zero Region.
func (sy *Symbol) IsTemp() bool {
	return sy.Region == lex.RegZero
}

// HasChildren returns true if this symbol has children
func (sy *Symbol) HasChildren() bool {
	return len(sy.Children) > 0
}

// String satisfied fmt.Stringer interface
func (sy *Symbol) String() string {
	return fmt.Sprintf("%v: %v (%v)", sy.Name, sy.Kind, sy.Region)
}

// Clone returns a clone copy of this symbol.
// Does NOT copy the Children or Types -- caller can decide about that.
func (sy *Symbol) Clone() *Symbol {
	nsy := &Symbol{Name: sy.Name, Detail: sy.Detail, Kind: sy.Kind, Type: sy.Type, Index: sy.Index, Filename: sy.Filename, Region: sy.Region, SelectReg: sy.SelectReg}
	nsy.Scopes = sy.Scopes.Clone()
	nsy.Ast = sy.Ast
	return nsy
}

// AddChild adds a child symbol, if this parent symbol is not temporary
// returns true if item name was added and NOT already on the map,
// and false if it was already or parent is temp.
// Always adds new symbol in any case.
// If parent symbol is of the NameType subcategory, then index of child is set
// to the size of this child map before adding.
func (sy *Symbol) AddChild(child *Symbol) bool {
	if sy.IsTemp() {
		return false
	}
	sy.Children.Alloc()
	_, on := sy.Children[child.Name]
	idx := len(sy.Children)
	child.Index = idx // always record index
	sy.Children[child.Name] = child
	return !on
}

// AllocScopes allocates scopes map if nil
func (sy *Symbol) AllocScopes() {
	if sy.Scopes == nil {
		sy.Scopes = make(SymNames)
	}
}

// AddScopesMap adds a given scope element(s) from map to this Symbol.
// if add is true, add this symbol to those scopes if they are not temporary.
func (sy *Symbol) AddScopesMap(sm SymMap, add bool) {
	if len(sm) == 0 {
		return
	}
	sy.AllocScopes()
	for _, s := range sm {
		sy.Scopes[s.Kind] = s.Name
		if add {
			s.AddChild(sy)
		}
	}
}

// AddScopesStack adds a given scope element(s) from stack to this Symbol.
// Adds this symbol as a child to the top of the scopes if it is not temporary --
// returns true if so added.
func (sy *Symbol) AddScopesStack(ss SymStack) bool {
	sz := len(ss)
	if sz == 0 {
		return false
	}
	sy.AllocScopes()
	added := false
	for i := 0; i < sz; i++ {
		sc := ss[i]
		sy.Scopes[sc.Kind] = sc.Name
		if i == sz-1 {
			added = sc.AddChild(sy)
		}
	}
	return added
}

// FindAnyChildren finds children of this symbol using either
// direct children if those are present, or the type name if
// present -- used for completion routines.  Adds to kids map.
// scope1, scope2 are used for looking up type name.
// If seed is non-empty it is used as a prefix for filtering children names.
// Returns false if no children were found.
func (sy *Symbol) FindAnyChildren(seed string, scope1, scope2 SymMap, kids *SymMap) bool {
	sym := sy
	if len(sym.Children) == 0 {
		if sym.Type != "" {
			tynm := sym.NonPtrTypeName()
			if typ, got := scope1.FindNameScoped(tynm); got {
				sym = typ
			} else if typ, got := scope2.FindNameScoped(tynm); got {
				sym = typ
			} else {
				return false
			}
		}
	}
	if seed != "" {
		sym.Children.FindNamePrefixRecursive(seed, kids)
	} else {
		kids.CopyFrom(sym.Children, true) // srcIsNewer
	}
	return len(*kids) > 0

}

// NonPtrTypeName returns the name of the type without any leading * or &
func (sy *Symbol) NonPtrTypeName() string {
	return strings.TrimPrefix(strings.TrimPrefix(sy.Type, "&"), "*")
}

// CopyFromScope copies the Children and Types from given other symbol
// for scopes (e.g., Go package), to merge with existing.
func (sy *Symbol) CopyFromScope(src *Symbol) {
	sy.Children.CopyFrom(src.Children, false) // src is NOT newer
	sy.Types.CopyFrom(src.Types, false)       // src is NOT newer
}

// OpenJSON opens from a JSON-formatted file.
func (sy *Symbol) OpenJSON(filename string) error {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, sy)
}

// SaveJSON saves to a JSON-formatted file.
func (sy *Symbol) SaveJSON(filename string) error {
	b, err := json.MarshalIndent(sy, "", "  ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filename, b, 0644)
	return err
}

// WriteDoc writes basic doc info
func (sy *Symbol) WriteDoc(out io.Writer, depth int) {
	ind := indent.Tabs(depth)
	fmt.Fprintf(out, "%v%v: %v", ind, sy.Name, sy.Kind)
	if sy.Type != "" {
		fmt.Fprintf(out, " (%v)", sy.Type)
	}
	if len(sy.Types) > 0 {
		fmt.Fprint(out, " Types: {\n")
		sy.Types.WriteDoc(out, depth+1)
		fmt.Fprintf(out, "%v}\n", ind)
	}
	if sy.HasChildren() {
		fmt.Fprint(out, " {\n")
		sy.Children.WriteDoc(out, depth+1)
		fmt.Fprintf(out, "%v}\n", ind)
	} else {
		fmt.Fprint(out, "\n")
	}
}

//////////////////////////////////////////////////////////////////
// SymNames

// SymNames provides a map-list of symbol names, indexed by their token kinds.
// Used primarily for specifying Scopes
type SymNames map[token.Tokens]string

// SubCat returns a scope with the given SubCat type, or false if not found
func (sn *SymNames) SubCat(sc token.Tokens) (string, bool) {
	for tk, nm := range *sn {
		if tk.SubCat() == sc {
			return nm, true
		}
	}
	return "", false
}

// Clone returns a clone copy of this map (nil if empty)
func (sn *SymNames) Clone() SymNames {
	sz := len(*sn)
	if sz == 0 {
		return nil
	}
	nsn := make(SymNames, sz)
	for tk, nm := range *sn {
		nsn[tk] = nm
	}
	return nsn
}
