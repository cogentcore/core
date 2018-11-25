// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package token defines a complete set of all lexical tokens for any kind of
// language!  It is based on the alecthomas/chroma / pygments lexical tokens
// plus all the more detailed tokens needed for actually parsing languages
package token

import "github.com/goki/ki/kit"

// Tokens is a complete set of lexical tokens that encompasses all programming and text
// markup languages.  It includes everything in alecthomas/chroma (pygments) and
// everything needed for Go, C, C++, Python, etc.
//
// There are categories and sub-categories, and methods to get those from a given
// element.  The first category is 'None'.
//
// See http://pygments.org/docs/tokens/ for more docs on the different categories
//
// Anything missing should be added via a pull request etc
//
type Tokens int

//go:generate stringer -type=Tokens

var KiT_Tokens = kit.Enums.AddEnum(TokensN, false, nil)

func (ev Tokens) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *Tokens) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// map keys require text marshaling:
func (ev Tokens) MarshalText() ([]byte, error)  { return kit.EnumMarshalText(ev) }
func (ev *Tokens) UnmarshalText(b []byte) error { return kit.EnumUnmarshalText(ev, b) }

// CatMap is the map into the category level for each token
var CatMap map[Tokens]Tokens

// SubCatMap is the map into the sub-category level for each token
var SubCatMap map[Tokens]Tokens

func init() {
	InitCatMap()
	InitSubCatMap()
}

// Cat returns the category that a given token lives in, using CatMap
func (tk Tokens) Cat() Tokens {
	return CatMap[tk]
}

// SubCat returns the sub-category that a given token lives in, using SubCatMap
func (tk Tokens) SubCat() Tokens {
	return SubCatMap[tk]
}

// CombineRepeats are token types where repeated tokens of the same type should
// be combined together -- literals, comments, text
func (tk Tokens) CombineRepeats() bool {
	cat := tk.Cat()
	return (cat == Literal || cat == Comment || cat == Text)
}

// The list of tokens
const (
	// None is the nil token value -- for non-terminal cases or TBD
	None Tokens = iota
	// Error is an input that could not be tokenized due to syntax error etc
	Error
	// end of file
	EOF
	// end of line (typically implicit -- used for rule matching)
	EOL
	// end of statement is a key meta-token -- in C it is ;, in Go it is either ; or EOL
	EOS

	// Cat: Keywords (actual keyword is just the string)
	Keyword
	KeywordConstant
	KeywordDeclaration
	KeywordNamespace
	KeywordPseudo
	KeywordReserved
	KeywordType

	// Cat: Names.
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

	// SubCat: NameVar -- variable names
	NameVar
	NameVarAnonymous
	NameVarClass
	NameVarGlobal
	NameVarInstance
	NameVarMagic

	// Cat: Literals.
	Literal
	LiteralDate
	LiteralOther

	// SubCat: Literal Strings.
	LitStr
	LitStrAffix
	LitStrAtom
	LitStrBacktick
	LitStrBoolean
	LitStrChar
	LitStrDelimiter
	LitStrDoc
	LitStrDouble
	LitStrEscape
	LitStrHeredoc
	LitStrInterpol
	LitStrName
	LitStrOther
	LitStrRegex
	LitStrSingle
	LitStrSymbol

	// SubCat: Literal Numbers.
	LitNum
	LitNumBin
	LitNumFloat
	LitNumHex
	LitNumInteger
	LitNumIntegerLong
	LitNumOct
	LitNumImag
	LitNumComplex

	// Cat: Operators.
	Operator
	OperatorWord

	// SubCat: Math operators
	OpMath
	OpMathAdd // +
	OpMathSub // -
	OpMathMul // *
	OpMathDiv // /
	OpMathRem // %

	// SubCat: Bitwise operators
	OpBit
	OpBitAnd        // &
	OpBitOr         // |
	OpBitXor        // ^
	OpBitShiftLeft  // <<
	OpBitShiftRight // >>
	OpBitAndNot     // &^

	// SubCat: Assign operators
	OpAsgn
	OpAsgnAssign // =
	OpAsgnInc    // ++
	OpAsgnDec    // --
	OpAsgnArrow  // <-
	OpAsgnDefine // :=

	// SubCat: Math Assign operators
	OpMathAsgn
	OpMathAsgnAdd // +=
	OpMathAsgnSub // -=
	OpMathAsgnMul // *=
	OpMathAsgnDiv // /=
	OpMathAsgnRem // %=

	// SubCat: Bitwise Assign operators
	OpBitAsgn
	OpBitAsgnAnd        // &=
	OpBitAsgnOr         // |=
	OpBitAsgnXor        // ^=
	OpBitAsgnShiftLeft  // <<=
	OpBitAsgnShiftRight // >>=
	OpBitAsgnAndNot     // &^=

	// SubCat: Logical operators
	OpLog
	OpLogAnd // &&
	OpLogOr  // ||
	OpLogNot // !

	// SubCat: Relational operators
	OpRel
	OpRelEqual    // ==
	OpRelNotEqual // !=
	OpRelLess     // <
	OpRelGreater  // >
	OpRelLtEq     // <=
	OpRelGtEq     // >=

	// SubCat: List operators
	OpList
	OpListEllipsis // ...

	// Cat: Punctuation.
	Punctuation

	// SubCat: Grouping punctuation
	PunctGp
	PunctGpLParen // (
	PunctGpRParen // )
	PunctGpLBrack // [
	PunctGpRBrack // ]
	PunctGpLBrace // {
	PunctGpRBrace // }

	// SubCat: Separator punctuation
	PunctSep
	PunctSepComma     // ,
	PunctSepPeriod    // .
	PunctSepSemicolon // ;
	PunctSepColon     // :

	// SubCat: String punctuation
	PunctStr
	PunctStrDblQuote // "
	PunctStrQuote    // '
	PunctStrBacktick // `
	PunctStrEsc      // \

	// Cat: Comments.
	Comment
	CommentHashbang
	CommentMultiline
	CommentSingle
	CommentSpecial

	// SubCat: Preprocessor "comments".
	CommentPreproc
	CommentPreprocFile

	// Cat: Text.
	Text
	TextWhitespace
	TextSymbol
	TextPunctuation
	TextSpellErr

	// SubCat: TextStyle (corresponds to Generic in chroma / pygments) todo: look in font deco for more
	TextStyle
	TextStyleDeleted // strike-through
	TextStyleEmph    // italics
	TextStyleError
	TextStyleHeading
	TextStyleInserted
	TextStyleOutput
	TextStylePrompt
	TextStyleStrong // bold
	TextStyleSubheading
	TextStyleTraceback
	TextStyleUnderline
	TextStyleLink

	TokensN
)

// Categories
var Cats = []Tokens{
	None,
	Keyword,
	Name,
	Literal,
	Operator,
	Punctuation,
	Comment,
	Text,
	TokensN,
}

// Sub-Categories
var SubCats = []Tokens{
	None,
	Keyword,
	Name,
	NameVar,
	Literal,
	LitStr,
	LitNum,
	Operator,
	OpMath,
	OpBit,
	OpAsgn,
	OpMathAsgn,
	OpBitAsgn,
	OpLog,
	OpRel,
	OpList,
	Punctuation,
	PunctGp,
	PunctSep,
	PunctStr,
	Comment,
	CommentPreproc,
	Text,
	TextStyle,
	TokensN,
}

// InitCatMap initializes the CatMap
func InitCatMap() {
	if CatMap != nil {
		return
	}
	CatMap = make(map[Tokens]Tokens, TokensN)
	for tk := None; tk < TokensN; tk++ {
		for c := 1; c < len(Cats); c++ {
			if tk < Cats[c] {
				CatMap[tk] = Cats[c-1]
				break
			}
		}
	}
}

// InitSubCatMap initializes the SubCatMap
func InitSubCatMap() {
	if SubCatMap != nil {
		return
	}
	SubCatMap = make(map[Tokens]Tokens, TokensN)
	for tk := None; tk < TokensN; tk++ {
		for c := 1; c < len(SubCats); c++ {
			if tk < SubCats[c] {
				SubCatMap[tk] = SubCats[c-1]
				break
			}
		}
	}
}
