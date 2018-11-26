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

	// AddAst means create an Ast node for this rule, adding it to the current anchor Ast
	AddAst

	// AnchorAst means create an Ast node and set it as the anchor that subsequent
	// sub-nodes are added into
	AnchorAst

	AstActsN
)
