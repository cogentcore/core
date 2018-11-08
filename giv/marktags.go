// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// markup tags are based directly on (i.e., copied from)
// https://github.com/alecthomas/chroma

package giv

import "github.com/goki/ki/kit"

// MarkTags are markup tags for syntax highlighting and any other kind of
// text markup.  These tags must remain in correspondence with TokenType
// in https://github.com/alecthomas/chroma as we directly copy from those
type MarkTags int

//go:generate stringer -type=MarkTags

var KiT_MarkTags = kit.Enums.AddEnum(MarkTagsN, false, nil)

func (ev MarkTags) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *MarkTags) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// Meta token types.
const (
	// Default background style.
	Background MarkTags = -1 - iota
	// Line numbers in output.
	LineNumbers
	// Line numbers in output when in table.
	LineNumbersTable
	// Line higlight style.
	LineHighlight
	// Line numbers table wrapper style.
	LineTable
	// Line numbers table TD wrapper style.
	LineTableTD
	// Input that could not be tokenised.
	Error
	// Other is used by the Delegate lexer to indicate which tokens should be handled by the delegate.
	Other
	// No highlighting.
	None
)

// Keywords.
const (
	Keyword MarkTags = 1000 + iota
	KeywordConstant
	KeywordDeclaration
	KeywordNamespace
	KeywordPseudo
	KeywordReserved
	KeywordType
)

// Names.
const (
	Name MarkTags = 2000 + iota
	NameAttribute
	NameBuiltin
	NameBuiltinPseudo
	NameClass
	NameConstant
	NameDecorator
	NameEntity
	NameException
	NameFunction
	NameFunctionMagic
	NameKeyword
	NameLabel
	NameNamespace
	NameOperator
	NameOther
	NamePseudo
	NameProperty
	NameTag
	NameVariable
	NameVariableAnonymous
	NameVariableClass
	NameVariableGlobal
	NameVariableInstance
	NameVariableMagic
)

// Literals.
const (
	Literal MarkTags = 3000 + iota
	LiteralDate
	LiteralOther
)

// Strings.
const (
	LiteralString MarkTags = 3100 + iota
	LiteralStringAffix
	LiteralStringAtom
	LiteralStringBacktick
	LiteralStringBoolean
	LiteralStringChar
	LiteralStringDelimiter
	LiteralStringDoc
	LiteralStringDouble
	LiteralStringEscape
	LiteralStringHeredoc
	LiteralStringInterpol
	LiteralStringName
	LiteralStringOther
	LiteralStringRegex
	LiteralStringSingle
	LiteralStringSymbol
)

// Literals.
const (
	LiteralNumber MarkTags = 3200 + iota
	LiteralNumberBin
	LiteralNumberFloat
	LiteralNumberHex
	LiteralNumberInteger
	LiteralNumberIntegerLong
	LiteralNumberOct
)

// Operators.
const (
	Operator MarkTags = 4000 + iota
	OperatorWord
)

// Punctuation.
const (
	Punctuation MarkTags = 5000 + iota
)

// Comments.
const (
	Comment MarkTags = 6000 + iota
	CommentHashbang
	CommentMultiline
	CommentSingle
	CommentSpecial
)

// Preprocessor "comments".
const (
	CommentPreproc MarkTags = 6100 + iota
	CommentPreprocFile
)

// Generic tokens.
const (
	Generic MarkTags = 7000 + iota
	GenericDeleted
	GenericEmph
	GenericError
	GenericHeading
	GenericInserted
	GenericOutput
	GenericPrompt
	GenericStrong
	GenericSubheading
	GenericTraceback
	GenericUnderline
)

// CustomGeneric tokens -- this is where we put anything not in chroma
const (
	// spelling error
	GenericSpellErr MarkTags = 7900 + iota
)

// Text.
const (
	Text MarkTags = 8000 + iota
	TextWhitespace
	TextSymbol
	TextPunctuation

	MarkTagsN
)
