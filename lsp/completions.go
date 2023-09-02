// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lsp

import (
	"goki.dev/ki/v2/kit"
)

// CompletionKind is the Language Server Protocol (LSP) CompletionKind, which
// we map onto the token.Tokens that are used internally.
type CompletionKind int

//go:generate stringer -type=CompletionKind

var KiT_CompletionKind = kit.Enums.AddEnumAltLower(CompletionKindN, kit.NotBitFlag, nil, "Ck")

func (ev CompletionKind) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *CompletionKind) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// map keys require text marshaling:
func (ev CompletionKind) MarshalText() ([]byte, error)  { return kit.EnumMarshalText(ev) }
func (ev *CompletionKind) UnmarshalText(b []byte) error { return kit.EnumUnmarshalText(ev, b) }

// CompletionKinds -- note these largely overlap with CompletionKinds and are
// thus kinda partially redundant..
const (
	CkNone CompletionKind = iota
	CkText
	CkMethod
	CkFunction
	CkConstructor
	CkField
	CkVariable
	CkClass
	CkInterface
	CkModule
	CkProperty
	CkUnit
	CkValue
	CkEnum
	CkKeyword
	CkSnippet
	Color
	CkFile
	CkReference
	CkFolder
	CkEnumMember
	CkConstant
	CkStruct
	CkEvent
	CkOperator
	CkTypeParameter

	CompletionKindN
)
