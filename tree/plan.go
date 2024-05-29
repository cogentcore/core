// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tree

import (
	"cogentcore.org/core/base/plan"
	"cogentcore.org/core/types"
)

// TypePlan is a plan for the organization of a list of tree nodes,
// specified by the Type of element at a given index, with a given name.
// Used in Build and BuildSlice to actually build the items according
// to the plan.
type TypePlan []TypePlanItem

// TypePlanItem holds a type and a name, for specifying the [TypePlan].
type TypePlanItem struct {
	Type *types.Type
	Name string
}

// Add adds a new TypePlan element with the given type and name.
func (t *TypePlan) Add(typ *types.Type, name string) {
	*t = append(*t, TypePlanItem{typ, name})
}

// BuildSlice ensures that the given Slice contains the elements
// according to the TypePlan, specified by unique element names,
// with n = total number of items in the target slice.
// The given Node is set as the parent of the created nodes.
// Returns true if any changes were made.
func BuildSlice(sl *Slice, parent Node, p TypePlan) bool {
	mods := false
	*sl, mods = plan.Build(*sl, len(p),
		func(i int) string { return p[i].Name },
		func(name string, i int) Node {
			n := NewOfType(p[i].Type)
			n.SetName(name)
			initNode(n)
			if parent != nil {
				SetParent(n, parent)
			}
			return n
		}, func(k Node) { k.Destroy() })
	return mods
}

// Build ensures that the Children of given Node contains the elements
// according to the TypePlan, specified by unique element names.
// Returns true if any changes were made.
func Build(n Node, p TypePlan) bool {
	nb := n.AsTreeNode()
	return BuildSlice(&nb.Kids, n, p)
}
