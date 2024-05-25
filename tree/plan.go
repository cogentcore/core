// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tree

import (
	"fmt"

	"cogentcore.org/core/base/plan"
	"cogentcore.org/core/types"
)

// TypeAndName holds a type and a name. It is used for specifying [Plan]
// objects in [Node.BuildChildren].
type TypeAndName struct {
	Type *types.Type
	Name string
}

// Plan is a list of [TypeAndName]s used in [Node.BuildChildren].
type Plan []TypeAndName

// Add adds a new Plan element with the given type and name.
func (t *Plan) Add(typ *types.Type, name string) {
	*t = append(*t, TypeAndName{typ, name})
}

func (t Plan) GoString() string {
	var str string
	for i, tn := range t {
		str += fmt.Sprintf("[%02d: %20s\t %20s\n", i, tn.Name, tn.Type.Name)
	}
	return str
}

// BuildSlice ensures that the given Slice contains the elements
// according to the Plan, specified by unique element names,
// with n = total number of items in the target slice.
// The given Node is set as the parent of the created nodes.
// Returns true if any changes were made.
func BuildSlice(sl *Slice, n Node, p Plan) bool {
	mods := false
	*sl, mods = plan.Build(*sl, len(p),
		func(i int) string { return p[i].Name },
		func(name string, i int) Node {
			nk := NewOfType(p[i].Type)
			nk.SetName(name)
			initNode(nk)
			if n != nil {
				SetParent(nk, n)
			}
			return nk
		}, func(k Node) { k.Destroy() })
	return mods
}

// Build ensures that the Children of given Node contains the elements
// according to the Plan, specified by unique element names.
// Returns true if any changes were made.
func Build(n Node, p Plan) bool {
	nb := n.AsTreeNode()
	return BuildSlice(&nb.Kids, n, p)
}

// SetNChildren ensures that there are exactly n children, deleting any
// extra, and creating any new ones, using NewChild with given type and
// naming according to nameStubX where X is the index of the child.
// If nameStub is not specified, it defaults to the ID (kebab-case)
// name of the type. It returns whether any changes were made to the
// children.
//
// Note that this does not ensure existing children are of given type, or
// change their names, or call UniquifyNames; use ConfigChildren for
// those cases; this function is for simpler cases where a parent uses
// this function consistently to manage children all of the same type.
func (n *NodeBase) SetNChildren(trgn int, typ *types.Type, nameStub ...string) bool {
	sz := len(n.Kids)
	if trgn == sz {
		return false
	}
	mods := false
	for sz > trgn {
		mods = true
		sz--
		n.DeleteChildAtIndex(sz)
	}
	ns := typ.IDName
	if len(nameStub) > 0 {
		ns = nameStub[0]
	}
	for sz < trgn {
		mods = true
		nm := fmt.Sprintf("%s%d", ns, sz)
		n.InsertNewChild(typ, sz).SetName(nm)
		sz++
	}
	return mods
}
