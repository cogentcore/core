// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syms

import (
	"github.com/goki/pi/lex"
	"github.com/goki/pi/token"
)

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
