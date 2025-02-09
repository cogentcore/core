// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser

import (
	"fmt"

	"cogentcore.org/core/text/parse/lexer"
	"cogentcore.org/core/text/token"
)

// Actions are parsing actions to perform
type Actions int32 //enums:enum

// The parsing acts
const (
	// ChangeToken changes the token to the Tok specified in the Act action
	ChangeToken Actions = iota

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

	// PopScope means remove the most recently added scope item
	PopScope

	// PopScopeReg means remove the most recently added scope item, and also
	// updates the source region for that item based on final SrcReg from
	// corresponding AST node -- for "definitional" scope
	PopScopeReg

	// AddDetail adds src at given path as detail info for the last-added symbol
	// if there is already something there, a space is added for this new addition
	AddDetail

	// AddType Adds a type with the given name -- sets the AST node for this rule
	// and actual type is resolved later in a second language-specific pass
	AddType

	// PushStack adds name to stack -- provides context-sensitivity option for
	// optimizing and ambiguity resolution
	PushStack

	// PopStack pops the stack
	PopStack
)

// Act is one action to perform, operating on the AST output
type Act struct {

	// at what point during sequence of sub-rules / tokens should this action be run?  -1 = at end, 0 = before first rule, 1 = before second rule, etc -- must be at point when relevant AST nodes have been added, but for scope setting, must be early enough so that scope is present
	RunIndex int

	// what action to perform
	Act Actions

	// AST path, relative to current node: empty = current node; specifies a child node by index, and a name specifies it by name -- include name/name for sub-nodes etc -- multiple path options can be specified by | or & and will be tried in order until one succeeds (for |) or all that succeed will be used for &. ... means use all nodes with given name (only for change token) -- for PushStack, this is what to push on the stack
	Path string `width:"50"`

	// for ChangeToken, the new token type to assign to token at given path
	Token token.Tokens

	// for ChangeToken, only change if token is this to start with (only if != None))
	FromToken token.Tokens
}

// String satisfies fmt.Stringer interface
func (ac Act) String() string {
	if ac.FromToken != token.None {
		return fmt.Sprintf(`%v:%v:"%v":%v<-%v`, ac.RunIndex, ac.Act, ac.Path, ac.Token, ac.FromToken)
	}
	return fmt.Sprintf(`%v:%v:"%v":%v`, ac.RunIndex, ac.Act, ac.Path, ac.Token)
}

// ChangeToken changes the token type, using FromToken logic
func (ac *Act) ChangeToken(lx *lexer.Lex) {
	if ac.FromToken == token.None {
		lx.Token.Token = ac.Token
		return
	}
	if lx.Token.Token != ac.FromToken {
		return
	}
	lx.Token.Token = ac.Token
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

// ASTActs are actions to perform on the [AST] nodes
type ASTActs int32 //enums:enum

// The [AST] actions
const (
	// NoAST means don't create an AST node for this rule
	NoAST ASTActs = iota

	// AddAST means create an AST node for this rule, adding it to the current anchor AST.
	// Any sub-rules within this rule are *not* added as children of this node -- see
	// SubAST and AnchorAST.  This is good for token-only terminal nodes and list elements
	// that should be added to a list.
	AddAST

	// SubAST means create an AST node and add all the elements of *this rule* as
	// children of this new node (including sub-rules), *except* for the very last rule
	// which is assumed to be a recursive rule -- that one goes back up to the parent node.
	// This is good for adding more complex elements with sub-rules to a recursive list,
	// without creating a new hierarchical depth level for every such element.
	SubAST

	// AnchorAST means create an AST node and set it as the anchor that subsequent
	// sub-nodes are added into.  This is for a new hierarchical depth level
	// where everything under this rule gets organized.
	AnchorAST

	// AnchorFirstAST means create an AST node and set it as the anchor that subsequent
	// sub-nodes are added into, *only* if this is the first time that this rule has
	// matched within the current sequence (i.e., if the parent of this rule is the same
	// rule then don't add a new AST node).  This is good for starting a new list
	// of recursively defined elements, without creating increasing depth levels.
	AnchorFirstAST
)
