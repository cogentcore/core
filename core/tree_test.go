// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"
)

func TestTree(t *testing.T) {
	b := NewBody()

	fr := NewFrame()
	NewButton(fr)
	NewText(fr)
	NewButton(NewFrame(fr))

	NewTree(b).SyncTree(fr)
	b.AssertRender(t, "tree/basic")
}

func TestTreeReadOnly(t *testing.T) {
	b := NewBody()

	fr := NewFrame()
	NewButton(fr)
	NewText(fr)
	NewButton(NewFrame(fr))

	NewTree(b).SyncTree(fr).SetReadOnly(true)
	b.AssertRender(t, "tree/read-only")
}

func TestTreeDuplicate(t *testing.T) {
	b := NewBody()
	root := NewTree(b).SetText("root")
	a := NewTree(root).SetText("a")
	a1 := NewTree(a).SetText("a1")
	NewTree(a1).SetText("a1x")
	NewTree(a).SetText("a2")
	NewTree(root).SetText("b")

	// The duplicate is checked once its events are handled, before the
	// window is closed and the tree destroyed.
	b.AssertRender(t, "tree/duplicate", func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		a.Duplicate()
	}, func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		if n := root.NumChildren(); n != 3 {
			t.Fatalf("root has %d children after Duplicate, want 3", n)
		}
		dup := AsTree(root.Child(1))
		if dup == a || dup.Name != a.Name+"_Copy" {
			t.Fatalf("duplicate is %q, want a new node named %q", dup.Name, a.Name+"_Copy")
		}
		if dup.Parent != root.This {
			t.Errorf("duplicate parent is %v, want root", dup.Parent)
		}
		if dup.NumChildren() != 2 || AsTree(dup.Child(0)).NumChildren() != 1 {
			t.Fatalf("duplicate structure differs from the original")
		}
		if AsTree(dup.Child(0)).Child(0) == AsTree(a.Child(0)).Child(0) {
			t.Errorf("duplicate shares children with the original")
		}
		dup.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
			if cwb.Scene != b.Scene {
				t.Errorf("node %q is not in the scene of the tree", cwb.Name)
			}
			if ct := AsTree(cw); ct != nil && ct.Root != root.This {
				t.Errorf("node %q has root %v, want the original root", ct.Name, ct.Root)
			}
			return true
		})
	})
}
