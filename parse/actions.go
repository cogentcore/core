// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parse

import (
	"github.com/goki/ki/kit"
)

// Actions are parsing actions to perform
type Actions int

//go:generate stringer -type=Actions

var KiT_Actions = kit.Enums.AddEnum(ActionsN, false, nil)

func (ev Actions) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *Actions) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// The parsing acts
const (
	// AddType means add name as type name
	AddType Actions = iota

	// AddConst means add name as constant
	AddConst

	// AddVar means add name as a variable
	AddVar

	ActionsN
)

// AstActs are actions to perform on the Ast nodes
type AstActs int

//go:generate stringer -type=AstActs

var KiT_AstActs = kit.Enums.AddEnum(AstActsN, false, nil)

func (ev AstActs) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *AstActs) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// The Ast actions
const (
	// NoAst means don't create an Ast node for this rule
	NoAst AstActs = iota

	// AddAst means create an Ast node for this rule, adding it to the current anchor Ast.
	// Any sub-rules within this rule are *not* added as children of this node -- see
	// SubAst and AnchorAst.  This is good for token-only terminal nodes and list elements
	// that should be added to a list.
	AddAst

	// SubAst means create an Ast node and add all the elements of *this rule* as
	// children of this new node (including sub-rules), *except* for the very last rule
	// which is assumed to be a recursive rule -- that one goes back up to the parent node.
	// This is good for adding more complex elements with sub-rules to a recursive list,
	// without creating a new hierarchical depth level for every such element.
	SubAst

	// AnchorAst means create an Ast node and set it as the anchor that subsequent
	// sub-nodes are added into.  This is for a new hierarchical depth level
	// where everything under this rule gets organized.
	AnchorAst

	// AnchorFirstAst means create an Ast node and set it as the anchor that subsequent
	// sub-nodes are added into, *only* if this is the first time that this rule has
	// matched within the current sequence (i.e., if the parent of this rule is the same
	// rule then don't add a new Ast node).  This is good for starting a new list
	// of recursively-defined elements, without creating increasing depth levels.
	AnchorFirstAst

	AstActsN
)
