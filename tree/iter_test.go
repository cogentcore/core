// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tree

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAllAncestors tests iterating over a node and its descendants, with
// an early stop, and over the parents of a node.
func TestAllAncestors(t *testing.T) {
	root := NewNodeBase()
	root.SetName("root")
	a := NewNodeBase(root)
	a.SetName("a")
	a1 := NewNodeBase(a)
	a1.SetName("a1")
	b := NewNodeBase(root)
	b.SetName("b")

	var names []string
	for n := range root.All() {
		names = append(names, n.AsTree().Name)
	}
	assert.Equal(t, []string{"root", "a", "a1", "b"}, names)

	names = nil
	for n := range root.All() {
		names = append(names, n.AsTree().Name)
		if n.AsTree().Name == "a1" {
			break
		}
	}
	assert.Equal(t, []string{"root", "a", "a1"}, names, "stops at the break, not just skipping the children")

	names = nil
	for p := range a1.Ancestors() {
		names = append(names, p.AsTree().Name)
	}
	assert.Equal(t, []string{"a", "root"}, names)
	for range root.Ancestors() {
		t.Error("the root has no parents")
	}
	n := 0
	for range a1.Ancestors() {
		n++
		break
	}
	assert.Equal(t, 1, n)
}
