// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lsp

// CompletionKind is the Language Server Protocol (LSP) CompletionKind, which
// we map onto the token.Tokens that are used internally.
type CompletionKind int //enums:enum

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
)
