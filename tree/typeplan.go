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
// It is used in [Update] and [UpdateSlice] to actually update the items
// according to the plan.
type TypePlan []TypePlanItem

// TypePlanItem holds a type and a name, for specifying the [TypePlan].
type TypePlanItem struct {
	Type *types.Type
	Name string
}

// Add adds a new [TypePlanItem] with the given type and name.
func (t *TypePlan) Add(typ *types.Type, name string) {
	*t = append(*t, TypePlanItem{typ, name})
}

// UpdateSlice ensures that the given slice contains the elements
// according to the [TypePlan], specified by unique element names.
// The given Node is set as the parent of the created nodes.
// It returns whether any changes were made.
func UpdateSlice(slice *[]Node, parent Node, p TypePlan) bool {
	return plan.Update(slice, len(p),
		func(i int) string { return p[i].Name },
		func(name string, i int) Node {
			n := newOfType(p[i].Type)
			n.AsTree().SetName(name)
			InitNode(n)
			return n
		}, func(child Node, i int) {
			if parent != nil {
				SetParent(child, parent)
			}
		},
		func(child Node) {
			child.Destroy()
		},
	)
}

// Update ensures that the children of the given [Node] contain the elements
// according to the [TypePlan], specified by unique element names.
// It returns whether any changes were made.
func Update(n Node, p TypePlan) bool {
	return UpdateSlice(&n.AsTree().Children, n, p)
}
