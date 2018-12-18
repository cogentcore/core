// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parse

import (
	"fmt"

	"github.com/goki/ki/kit"
	"github.com/goki/pi/token"
)

// Actions are parsing actions to perform
type Actions int

//go:generate stringer -type=Actions

var KiT_Actions = kit.Enums.AddEnum(ActionsN, false, nil)

func (ev Actions) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *Actions) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// The parsing acts
const (
	// ChgToken changes the token to the Tok specified in the Act action
	ChgToken Actions = iota

	// AddSymbol means add name as a symbol, using current scoping and token type
	// or the token specified in the Act action if != None
	AddSymbol

	// PushScope means look for an existing symbol of given name
	// to push onto current scope -- adding a new one if not found --
	// does not add new item to overall symbol list.  This is useful
	// for e.g., definitions of methods on a type, where this is not
	// the definition of the type itself.
	PushScope

	// PushNewScope means add a new symbol to the list and also push
	// onto scope stack, using given token type or the token specified
	// in the Act action if != None
	PushNewScope

	// PopScope means remove the most recently-added scope item
	PopScope

	// AddDetail adds src at given path as detail info for the last-added symbol
	// if there is already something there, a space is added for this new addition
	AddDetail

	// PushStack adds name to stack -- provides context-sensitivity option for
	// optimizing and ambiguity resolution
	PushStack

	// PopStack pops the stack
	PopStack

	ActionsN
)

// Act is one action to perform, operating on the Ast output
type Act struct {
	RunIdx int          `desc:"at what point during sequence of sub-rules / tokens should this action be run?  -1 = at end, 0 = before first rule, 1 = before second rule, etc -- must be at point when relevant Ast nodes have been added, but for scope setting, must be early enough so that scope is present"`
	Act    Actions      `desc:"what action to perform"`
	Path   string       `width:"50" desc:"Ast path, relative to current node: empty = current node; [idx] specifies a child node by index, and a name specifies it by name -- include name/name for sub-nodes etc -- multiple path options can be specified by | and will be tried in order until one succeeds, in case there are different options; ... means use all nodes with given name (only for change token) -- for PushStack, this is what to push on the stack"`
	Tok    token.Tokens `desc:"for ChgToken, the new token type to assign to token at given path"`
}

// String satisfies fmt.Stringer interface
func (ac Act) String() string {
	return fmt.Sprintf(`%v:%v:"%v":%v`, ac.RunIdx, ac.Act, ac.Path, ac.Tok)
}

// Acts are multiple actions
type Acts []Act

// String satisfies fmt.Stringer interface
func (ac Acts) String() string {
	if len(ac) == 0 {
		return ""
	}
	str := "{ "
	for i := range ac {
		str += ac[i].String() + "; "
	}
	str += "}"
	return str
}

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
