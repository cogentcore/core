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

	"github.com/goki/ki/indent"
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
	Kind      token.Tokens `desc:"type of symbol, using token.Tokens list"`
	Index     int          `desc:"index for ordering children within a given scope, e.g., fields in a struct / class"`
	Filename  string       `desc:"full filename / URI of source"`
	Region    lex.Reg      `desc:"region in source encompassing this item -- if = RegZero then this is a temp symbol and children are not added to it"`
	SelectReg lex.Reg      `desc:"region that should be selected when activated, etc"`
	Scopes    SymNames     `desc:"relevant scoping / parent symbols, e.g., namespace, package, module, class, function, etc.."`
	Children  SymMap       `desc:"children of this symbol -- this includes e.g., methods and fields of classes / structs / types, and all elements within packages, etc"`
}

var KiT_Symbol = kit.Types.AddType(&Symbol{}, nil)

// SymNames provides a map-list of symbol names, indexed by their token kinds.
// Used primarily for specifying Scopes
type SymNames map[token.Tokens]string

// NewSymbol returns a new symbol with the basic info filled in -- SelectReg defaults to Region
func NewSymbol(name string, kind token.Tokens, fname string, reg lex.Reg) *Symbol {
	sy := &Symbol{Name: name, Kind: kind, Filename: fname, Region: reg, SelectReg: reg}
	return sy
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
	if sy.Kind.SubCat() == token.NameType {
		child.Index = idx
	}
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
	if sy.HasChildren() {
		fmt.Fprint(out, " {\n")
		sy.Children.WriteDoc(out, depth+1)
		fmt.Fprintf(out, "%v}\n", ind)
	} else {
		fmt.Fprint(out, "\n")
	}
}
