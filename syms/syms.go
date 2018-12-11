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
	Filename  string       `desc:"full filename / URI of source"`
	Region    lex.Reg      `desc:"region in source encompassing this item"`
	SelectReg lex.Reg      `desc:"region that should be selected when activated, etc"`
	Scopes    SymNames     `desc:"relevant scoping / parent symbols, e.g., namespace, package, module, class, function, etc.."`
	Children  SymMap       `desc:"children of this symbol -- this includes e.g., methods and fields of classes / structs / types, and all elements within packages, etc"`
}

// NewSymbol returns a new symbol with the basic info filled in -- SelectReg defaults to Region
func NewSymbol(name string, kind token.Tokens, fname string, reg lex.Reg) *Symbol {
	sy := &Symbol{Name: name, Kind: kind, Filename: fname, Region: reg, SelectReg: reg}
	return sy
}

// AllocScopes allocates scopes map if nil
func (sy *Symbol) AllocScopes() {
	if sy.Scopes == nil {
		sy.Scopes = make(SymNames)
	}
}

// AddScopes adds a given scope element(s) to this Symbol
func (sy *Symbol) AddScopes(sm SymMap) {
	if len(sm) == 0 {
		return
	}
	sy.AllocScopes()
	for _, s := range sm {
		sy.Scopes[s.Kind] = s.Name
	}
}

// SymMap is a map between symbol names and their full information.
// A given project will have a top-level SymMap and perhaps local
// maps for individual files, etc.  Namespaces / packages can be
// created and elements added to them to create appropriate
// scoping structure etc.  Note that we have to use pointers
// for symbols b/c otherwise it is very expensive to re-assign
// values all the time -- https://github.com/golang/go/issues/3117
type SymMap map[string]*Symbol

// Alloc ensures that map is made
func (sm *SymMap) Alloc() {
	if *sm == nil {
		*sm = make(SymMap)
	}
}

// AddNew adds a new symbol to the map with the basic info
func (sm *SymMap) AddNew(name string, kind token.Tokens, fname string, reg lex.Reg) *Symbol {
	sy := NewSymbol(name, kind, fname, reg)
	sm.Alloc()
	(*sm)[name] = sy
	return sy
}

// Reset resets the symbol map
func (sm *SymMap) Reset() {
	*sm = make(SymMap)
}

// CopyFrom copies all the symbols from given source map into this one
func (sm *SymMap) CopyFrom(src SymMap) {
	sm.Alloc()
	for n, s := range src {
		(*sm)[n] = s
	}
}

// todo: probably will need a special json writer to write out the symbols instead of pointers.

// SymNames provides a map-list of symbol names, indexed by their token kinds.
// Used primarily for specifying Scopes
type SymNames map[token.Tokens]string
