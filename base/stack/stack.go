// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package stack provides a generic stack implementation.
package stack

// Stack provides a generic stack using a slice.
type Stack[T any] []T

// Push pushes item(s) onto the stack.
func (st *Stack[T]) Push(it ...T) {
	*st = append(*st, it...)
}

// Pop pops the top item off the stack.
// Returns nil / zero value if stack is empty.
func (st *Stack[T]) Pop() T {
	n := len(*st)
	if n == 0 {
		var zv T
		return zv
	}
	li := (*st)[n-1]
	*st = (*st)[:n-1]
	return li
}

// Peek returns the last element on the stack.
// Returns nil / zero value if stack is empty.
func (st *Stack[T]) Peek() T {
	n := len(*st)
	if n == 0 {
		var zv T
		return zv
	}
	return (*st)[n-1]
}
