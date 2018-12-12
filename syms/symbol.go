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

// AddScopesMap adds a given scope element(s) from map to this Symbol, and
// adds this symbol to those scopes if they are not temporary.
func (sy *Symbol) AddScopesMap(sm SymMap) {
	if len(sm) == 0 {
		return
	}
	sy.AllocScopes()
	for _, s := range sm {
		sy.Scopes[s.Kind] = s.Name
		s.AddChild(sy)
	}
}

// AddScopesStack adds a given scope element(s) from stack to this Symbol
// adds this symbol to those scopes if they are not temporary.
func (sy *Symbol) AddScopesStack(ss SymStack) {
	if len(ss) == 0 {
		return
	}
	sy.AllocScopes()
	for _, s := range ss {
		sy.Scopes[s.Kind] = s.Name
		s.AddChild(sy)
	}
}

// SymNames provides a map-list of symbol names, indexed by their token kinds.
// Used primarily for specifying Scopes
type SymNames map[token.Tokens]string
