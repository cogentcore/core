// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package histyle

import (
	"github.com/alecthomas/chroma"
	"github.com/goki/gi/gi"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

// HiTags are highlighting tags -- one-to-one with chroma.Token but kept in simple
// sequential order for use as an enum -- with enums and the string interface
// the raw numbers don't matter so sequential is best
type HiTags int

//go:generate stringer -type=HiTags

var KiT_HiTags = kit.Enums.AddEnumAltLower(HiTagsN, false, nil, "")

func (ev HiTags) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *HiTags) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// map keys require text marshaling:
func (ev HiTags) MarshalText() ([]byte, error)  { return kit.EnumMarshalText(ev) }
func (ev *HiTags) UnmarshalText(b []byte) error { return kit.EnumUnmarshalText(ev, b) }

// HiTagNames is the complete mapping from a HiTags tag to the css style name
// most are 2 letters -- populated from chroma.StandardTypes plus custom
var HiTagNames map[HiTags]string

// InitHiTagNames initializes the HiTagNames -- called during overall init
func InitHiTagNames() {
	if HiTagNames != nil {
		return
	}
	HiTagNames = make(map[HiTags]string, len(chroma.StandardTypes)+10)
	for ct, nm := range chroma.StandardTypes {
		htv := HiTagFromChroma(ct)
		HiTagNames[htv] = nm
	}
	for ct, nm := range ChromaTagNames {
		htv := HiTagFromChroma(ct)
		HiTagNames[htv] = nm
	}
}

// Chroma tag token types -- our extensions to the chroma token type
const (
	// ChromaTag is our starting tag -- anything above this is custom..
	ChromaTag chroma.TokenType = 10000 + iota

	// ChromaSpellErr tags a spelling error
	ChromaSpellErr
)

// ChromaTagNames are our style names for Chroma tags -- need to ensure CSS exists for these
var ChromaTagNames = map[chroma.TokenType]string{
	ChromaSpellErr: "cse",
}

// HiTagsProps are default properties for custom tags -- if set in style then used there
// but otherwise we use these as a fallback -- typically not overridden
var HiTagsProps = map[HiTags]ki.Props{
	SpellErr: ki.Props{
		"text-decoration": 1 << uint32(gi.DecoDottedUnderline), // bitflag!
	},
}

// FromChroma converts a chroma.TokenType to a HiTags type
func (ht *HiTags) FromChroma(ct chroma.TokenType) {
	if ChromaToHiTagsMap == nil {
		ChromaToHiTagsMap = make(map[chroma.TokenType]HiTags, len(HiTagsToChromaMap))
		for k, v := range HiTagsToChromaMap {
			ChromaToHiTagsMap[v] = k
		}
	}
	*ht = ChromaToHiTagsMap[ct]
}

// HiTagFromChroma returns a HiTags tag from a chroma tag
func HiTagFromChroma(ct chroma.TokenType) HiTags {
	var ht HiTags
	ht.FromChroma(ct)
	return ht
}

// ToChroma converts to a chroma.TokenType
func (ht HiTags) ToChroma() chroma.TokenType {
	return HiTagsToChromaMap[ht]
}

// StyleName returns the abbreviated 2-3 letter style name of the tag
func (ht HiTags) StyleName() string {
	return HiTagNames[ht]
}

// ClassName returns the . prefixed CSS classname of the tag style
// for styling, a CSS property should exist with this name
func (ht HiTags) ClassName() string {
	return "." + HiTagNames[ht]
}

func (ht HiTags) Parent() HiTags {
	ct := ht.ToChroma().Parent()
	return HiTagFromChroma(ct)
}

func (ht HiTags) Category() HiTags {
	ct := ht.ToChroma().Category()
	return HiTagFromChroma(ct)
}

func (ht HiTags) SubCategory() HiTags {
	ct := ht.ToChroma().Category()
	return HiTagFromChroma(ct)
}

func (ht HiTags) InCategory(other HiTags) bool {
	return ht.ToChroma().InCategory(other.ToChroma())
}

func (ht HiTags) InSubCategory(other HiTags) bool {
	return ht.ToChroma().InSubCategory(other.ToChroma())
}

// HiTags values -- MUST keep this in correspondence with alecthomas/chroma for interoperability
const (
	// Used as an EOF marker / nil token
	EOFType HiTags = iota
	// Default background style.
	Background
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
	// Keywords.
	Keyword
	KeywordConstant
	KeywordDeclaration
	KeywordNamespace
	KeywordPseudo
	KeywordReserved
	KeywordType
	// Names.
	Name
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
	// Literals.
	Literal
	LiteralDate
	LiteralOther
	// Literal Strings.
	LiteralString
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
	// Literals Numbers.
	LiteralNumber
	LiteralNumberBin
	LiteralNumberFloat
	LiteralNumberHex
	LiteralNumberInteger
	LiteralNumberIntegerLong
	LiteralNumberOct
	// Operators.
	Operator
	OperatorWord
	// Punctuation.
	Punctuation
	// Comments.
	Comment
	CommentHashbang
	CommentMultiline
	CommentSingle
	CommentSpecial
	// Preprocessor "comments".
	CommentPreproc
	CommentPreprocFile
	// Generic tokens.
	Generic
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
	// Text.
	Text
	TextWhitespace
	TextSymbol
	TextPunctuation
	// Our own custom types
	SpellErr

	HiTagsN
)

// Aliases.
const (
	Whitespace = TextWhitespace

	Date = LiteralDate

	String          = LiteralString
	StringAffix     = LiteralStringAffix
	StringBacktick  = LiteralStringBacktick
	StringChar      = LiteralStringChar
	StringDelimiter = LiteralStringDelimiter
	StringDoc       = LiteralStringDoc
	StringDouble    = LiteralStringDouble
	StringEscape    = LiteralStringEscape
	StringHeredoc   = LiteralStringHeredoc
	StringInterpol  = LiteralStringInterpol
	StringOther     = LiteralStringOther
	StringRegex     = LiteralStringRegex
	StringSingle    = LiteralStringSingle
	StringSymbol    = LiteralStringSymbol

	Number            = LiteralNumber
	NumberBin         = LiteralNumberBin
	NumberFloat       = LiteralNumberFloat
	NumberHex         = LiteralNumberHex
	NumberInteger     = LiteralNumberInteger
	NumberIntegerLong = LiteralNumberIntegerLong
	NumberOct         = LiteralNumberOct
)

// ChromaToHiTagsMap maps from chroma.TokenType to HiTags -- built from opposite map
var ChromaToHiTagsMap map[chroma.TokenType]HiTags

// HiTagsToChromaMap maps from HiTags to chroma.TokenType
var HiTagsToChromaMap = map[HiTags]chroma.TokenType{
	EOFType:                  chroma.EOFType,
	Background:               chroma.Background,
	LineNumbers:              chroma.LineNumbers,
	LineNumbersTable:         chroma.LineNumbersTable,
	LineHighlight:            chroma.LineHighlight,
	LineTable:                chroma.LineTable,
	LineTableTD:              chroma.LineTableTD,
	Error:                    chroma.Error,
	Other:                    chroma.Other,
	None:                     chroma.None,
	Keyword:                  chroma.Keyword,
	KeywordConstant:          chroma.KeywordConstant,
	KeywordDeclaration:       chroma.KeywordDeclaration,
	KeywordNamespace:         chroma.KeywordNamespace,
	KeywordPseudo:            chroma.KeywordPseudo,
	KeywordReserved:          chroma.KeywordReserved,
	KeywordType:              chroma.KeywordType,
	Name:                     chroma.Name,
	NameAttribute:            chroma.NameAttribute,
	NameBuiltin:              chroma.NameBuiltin,
	NameBuiltinPseudo:        chroma.NameBuiltinPseudo,
	NameClass:                chroma.NameClass,
	NameConstant:             chroma.NameConstant,
	NameDecorator:            chroma.NameDecorator,
	NameEntity:               chroma.NameEntity,
	NameException:            chroma.NameException,
	NameFunction:             chroma.NameFunction,
	NameFunctionMagic:        chroma.NameFunctionMagic,
	NameKeyword:              chroma.NameKeyword,
	NameLabel:                chroma.NameLabel,
	NameNamespace:            chroma.NameNamespace,
	NameOperator:             chroma.NameOperator,
	NameOther:                chroma.NameOther,
	NamePseudo:               chroma.NamePseudo,
	NameProperty:             chroma.NameProperty,
	NameTag:                  chroma.NameTag,
	NameVariable:             chroma.NameVariable,
	NameVariableAnonymous:    chroma.NameVariableAnonymous,
	NameVariableClass:        chroma.NameVariableClass,
	NameVariableGlobal:       chroma.NameVariableGlobal,
	NameVariableInstance:     chroma.NameVariableInstance,
	NameVariableMagic:        chroma.NameVariableMagic,
	Literal:                  chroma.Literal,
	LiteralDate:              chroma.LiteralDate,
	LiteralOther:             chroma.LiteralOther,
	LiteralString:            chroma.LiteralString,
	LiteralStringAffix:       chroma.LiteralStringAffix,
	LiteralStringAtom:        chroma.LiteralStringAtom,
	LiteralStringBacktick:    chroma.LiteralStringBacktick,
	LiteralStringBoolean:     chroma.LiteralStringBoolean,
	LiteralStringChar:        chroma.LiteralStringChar,
	LiteralStringDelimiter:   chroma.LiteralStringDelimiter,
	LiteralStringDoc:         chroma.LiteralStringDoc,
	LiteralStringDouble:      chroma.LiteralStringDouble,
	LiteralStringEscape:      chroma.LiteralStringEscape,
	LiteralStringHeredoc:     chroma.LiteralStringHeredoc,
	LiteralStringInterpol:    chroma.LiteralStringInterpol,
	LiteralStringName:        chroma.LiteralStringName,
	LiteralStringOther:       chroma.LiteralStringOther,
	LiteralStringRegex:       chroma.LiteralStringRegex,
	LiteralStringSingle:      chroma.LiteralStringSingle,
	LiteralStringSymbol:      chroma.LiteralStringSymbol,
	LiteralNumber:            chroma.LiteralNumber,
	LiteralNumberBin:         chroma.LiteralNumberBin,
	LiteralNumberFloat:       chroma.LiteralNumberFloat,
	LiteralNumberHex:         chroma.LiteralNumberHex,
	LiteralNumberInteger:     chroma.LiteralNumberInteger,
	LiteralNumberIntegerLong: chroma.LiteralNumberIntegerLong,
	LiteralNumberOct:         chroma.LiteralNumberOct,
	Operator:                 chroma.Operator,
	OperatorWord:             chroma.OperatorWord,
	Punctuation:              chroma.Punctuation,
	Comment:                  chroma.Comment,
	CommentHashbang:          chroma.CommentHashbang,
	CommentMultiline:         chroma.CommentMultiline,
	CommentSingle:            chroma.CommentSingle,
	CommentSpecial:           chroma.CommentSpecial,
	CommentPreproc:           chroma.CommentPreproc,
	CommentPreprocFile:       chroma.CommentPreprocFile,
	Generic:                  chroma.Generic,
	GenericDeleted:           chroma.GenericDeleted,
	GenericEmph:              chroma.GenericEmph,
	GenericError:             chroma.GenericError,
	GenericHeading:           chroma.GenericHeading,
	GenericInserted:          chroma.GenericInserted,
	GenericOutput:            chroma.GenericOutput,
	GenericPrompt:            chroma.GenericPrompt,
	GenericStrong:            chroma.GenericStrong,
	GenericSubheading:        chroma.GenericSubheading,
	GenericTraceback:         chroma.GenericTraceback,
	GenericUnderline:         chroma.GenericUnderline,
	Text:                     chroma.Text,
	TextWhitespace:           chroma.TextWhitespace,
	TextSymbol:               chroma.TextSymbol,
	TextPunctuation:          chroma.TextPunctuation,
	SpellErr:                 ChromaSpellErr,
}
