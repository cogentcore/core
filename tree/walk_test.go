// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tree_test

import (
	"testing"

	. "cogentcore.org/core/tree"
	"cogentcore.org/core/tree/testdata"
	"github.com/stretchr/testify/assert"
)

var testTree *NodeBase

func init() {
	testTree = NewNodeBase()
	testTree.SetName("root")
	child0 := NewNodeBase(testTree)
	child0.SetName("child0")
	child1 := NewNodeBase(testTree)
	child1.SetName("child1")
	schild1 := NewNodeBase(child1)
	schild1.SetName("subchild1")
	sschild1 := testdata.NewNodeEmbed(schild1)
	sschild1.SetName("subsubchild1")
	child2 := testdata.NewNodeEmbed(testTree)
	child2.SetName("child2")
	child3 := NewNodeBase(testTree)
	child3.SetName("child3")
}

func TestDown(t *testing.T) {
	var cur Node = testTree
	res := []string{}
	for {
		res = append(res, cur.AsTree().Path())
		curi := Next(cur)
		if curi == nil {
			break
		}
		cur = curi
	}
	assert.Equal(t, []string{"/root", "/root/child0", "/root/child1", "/root/child1/subchild1", "/root/child1/subchild1/subsubchild1", "/root/child2", "/root/child3"}, res)
}

func TestUp(t *testing.T) {
	cur := Last(testTree)
	res := []string{}
	for {
		res = append(res, cur.AsTree().Path())
		curi := Previous(cur)
		if curi == nil {
			break
		}
		cur = curi
	}
	assert.Equal(t, []string{"/root/child3", "/root/child2", "/root/child1/subchild1/subsubchild1", "/root/child1/subchild1", "/root/child1", "/root/child0", "/root"}, res)
}

// benchTree returns a tree of depth levels below the root, each node with
// width children.
func benchTree(depth, width int) *NodeBase {
	root := NewNodeBase()
	level := []*NodeBase{root}
	for range depth {
		var next []*NodeBase
		for _, n := range level {
			for range width {
				next = append(next, NewNodeBase(n))
			}
		}
		level = next
	}
	return root
}

func BenchmarkWalkDown(b *testing.B) {
	root := benchTree(3, 10)
	for range b.N {
		count := 0
		root.WalkDown(func(n Node) bool {
			count++
			return Continue
		})
	}
}

func BenchmarkWalkDownPost(b *testing.B) {
	root := benchTree(3, 10)
	for range b.N {
		count := 0
		root.WalkDownPost(func(n Node) bool { return Continue }, func(n Node) bool {
			count++
			return Continue
		})
	}
}
