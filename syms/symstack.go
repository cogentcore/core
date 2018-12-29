// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syms

import (
	"github.com/goki/pi/lex"
	"github.com/goki/pi/token"
)

// SymStack is a simple stack (slice) of symbols
type SymStack []*Symbol

// Top returns the state at the top of the stack (could be nil)
func (ss *SymStack) Top() *Symbol {
	sz := len(*ss)
	if sz == 0 {
		return nil
	}
	return (*ss)[sz-1]
}

// Push appends symbol to stack
func (ss *SymStack) Push(sy *Symbol) {
	*ss = append(*ss, sy)
}

// PushNew adds a new symbol to the stack with the basic info
func (ss *SymStack) PushNew(name string, kind token.Tokens, fname string, reg lex.Reg) *Symbol {
	sy := NewSymbol(name, kind, fname, reg)
	ss.Push(sy)
	return sy
}

// Pop takes symbol off the stack and returns it
func (ss *SymStack) Pop() *Symbol {
	sz := len(*ss)
	if sz == 0 {
		return nil
	}
	sy := (*ss)[sz-1]
	*ss = (*ss)[:sz-1]
	return sy
}

// Reset resets the stack
func (ss *SymStack) Reset() {
	*ss = nil
}

// FindNameScoped searches top-down in the stack for something with the given name
// in symbols that are of subcategory token.NameScope (i.e., namespace, module, package, library)
func (ss *SymStack) FindNameScoped(nm string) (*Symbol, bool) {
	sz := len(*ss)
	if sz == 0 {
		return nil, false
	}
	for i := sz - 1; i >= 0; i-- {
		sy := (*ss)[i]
		if sy.Name == nm {
			return sy, true
		}
		ssy, has := sy.Children.FindNameScoped(nm)
		if has {
			return ssy, true
		}
	}
	return nil, false
}
