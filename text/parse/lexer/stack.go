// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lexer

// Stack is the stack for states
type Stack []string

// Top returns the state at the top of the stack
func (ss *Stack) Top() string {
	sz := len(*ss)
	if sz == 0 {
		return ""
	}
	return (*ss)[sz-1]
}

// Push appends state to stack
func (ss *Stack) Push(state string) {
	*ss = append(*ss, state)
}

// Pop takes state off the stack and returns it
func (ss *Stack) Pop() string {
	sz := len(*ss)
	if sz == 0 {
		return ""
	}
	st := (*ss)[sz-1]
	*ss = (*ss)[:sz-1]
	return st
}

// Clone returns a copy of the stack
func (ss *Stack) Clone() Stack {
	sz := len(*ss)
	if sz == 0 {
		return nil
	}
	cl := make(Stack, sz)
	copy(cl, *ss)
	return cl
}

// Reset stack
func (ss *Stack) Reset() {
	*ss = nil
}
