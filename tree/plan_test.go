// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tree_test

import (
	"runtime"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	. "cogentcore.org/core/tree"
)

// line returns the line of its caller, also in a JavaScript build.
//
//goscript:callsite
func line() int {
	_, _, l, _ := runtime.Caller(1)
	return l
}

// TestAutoPlanName checks that the names of [Add] and [AddChild] come from the
// file and line of their calls, including in a JavaScript build.
func TestAutoPlanName(t *testing.T) {
	p := &Plan{}
	l := line()
	Add(p, func(w *NodeBase) {})
	Add(p, func(w *NodeBase) {
	})
	Add(
		p,
		func(w *NodeBase) {},
	)
	if assert.Len(t, p.Children, 3) {
		assert.Equal(t, "tree-plan_test-go-"+strconv.Itoa(l+1), p.Children[0].Name)
		assert.Equal(t, "tree-plan_test-go-"+strconv.Itoa(l+2), p.Children[1].Name)
		assert.Equal(t, "tree-plan_test-go-"+strconv.Itoa(l+4), p.Children[2].Name)
	}

	parent := NewNodeBase()
	l = line()
	AddChild(parent, func(w *NodeBase) {})
	parent.UpdateFromMake()
	if assert.Len(t, parent.Children, 1) {
		assert.Equal(t, "tree-plan_test-go-"+strconv.Itoa(l+1), parent.Children[0].AsTree().Name)
	}
}
