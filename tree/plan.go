// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tree

import (
	"cogentcore.org/core/base/plan"
	"cogentcore.org/core/types"
)

// Plan is a list of [TypeAndName]s used in [BuildSlice].
type Plan []TypeAndName

// TypeAndName holds a type and a name. It is used for specifying [Plan]
// objects in [BuildSlice].
type TypeAndName struct {
	Type *types.Type
	Name string
}

// Add adds a new Plan element with the given type and name.
func (t *Plan) Add(typ *types.Type, name string) {
	*t = append(*t, TypeAndName{typ, name})
}

// BuildSlice ensures that the given Slice contains the elements
// according to the Plan, specified by unique element names,
// with n = total number of items in the target slice.
// The given Node is set as the parent of the created nodes.
// Returns true if any changes were made.
func BuildSlice(sl *Slice, parent Node, p Plan) bool {
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
// according to the Plan, specified by unique element names.
// Returns true if any changes were made.
func Build(n Node, p Plan) bool {
	nb := n.AsTreeNode()
	return BuildSlice(&nb.Kids, n, p)
}
