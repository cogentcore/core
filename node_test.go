// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"testing"
)

func TestNodeAddChild(t *testing.T) {
	parent := NewNode()
	child := NewNode()
	parent.AddChildren(child)
	if len(parent.Children) != 1 {
		t.Errorf("Children length != 1, was %d", len(parent.Children))
	}
}

func TestNodeRemoveChild(t *testing.T) {
	parent := NewNode()
	child := NewNode()
	parent.AddChildren(child)
	parent.RemoveChild(child, true)
	if len(parent.Children) != 0 {
		t.Errorf("Children length != 0, was %d", len(parent.Children))
	}
	if len(parent.deleted) != 1 {
		t.Errorf("deleted length != 1, was %d", len(parent.Children))
	}
}

func TestNodeRemoveChildName(t *testing.T) {
	parent := NewNode()
	child := NewNode()
	child.Name = "test_name"
	parent.AddChildren(child)
	parent.RemoveChildName("test_name", true)
	if len(parent.Children) != 0 {
		t.Errorf("Children length != 0, was %d", len(parent.Children))
	}
	if len(parent.deleted) != 1 {
		t.Errorf("deleted length != 1, was %d", len(parent.Children))
	}
}

func TestNodeFindName(t *testing.T) {
	names := [...]string{"name0", "name1", "name2", "name3", "name4", "name5"}
	parent := NewNode()
	for _, nm := range names {
		child := NewNode()
		child.Name = nm
		parent.AddChildren(child)
	}
	if len(parent.Children) != len(names) {
		t.Errorf("Children length != n, was %d", len(parent.Children))
	}
	for i, nm := range names {
		for st, _ := range names { // test all starting indexes
			idx := parent.FindChildNameIndex(nm, st)
			if idx != i {
				t.Errorf("find index was not correct val of %d, was %d", i, idx)
			}
		}
	}
}
