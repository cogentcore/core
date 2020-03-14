// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package token defines a complete set of all lexical tokens for any kind of
// language!  It is based on the alecthomas/chroma / pygments lexical tokens
// plus all the more detailed tokens needed for actually parsing languages
package token

import (
	"fmt"

	"github.com/goki/ki/kit"
)

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

var KiT_Tokens = kit.Enums.AddEnum(TokensN, kit.NotBitFlag, nil)

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

// IsCat returns true if this is a category-level token
func (tk Tokens) IsCat() bool {
	return tk.Cat() == tk
}

// IsSubCat returns true if this is a sub-category-level token
func (tk Tokens) IsSubCat() bool {
	return tk.SubCat() == tk
}

// InCat returns true if this is in same category as given token
func (tk Tokens) InCat(other Tokens) bool {
	return tk.Cat() == other.Cat()
}

// InCat returns true if this is in same sub-category as given token
func (tk Tokens) InSubCat(other Tokens) bool {
	return tk.SubCat() == other.SubCat()
}

// IsCode returns true if this token is in Keyword, Name, Operator, or Punctuation categs.
// these are recognized code (program) elements that can usefully be distinguished from
// other forms of raw text (e.g., for spell checking)
func (tk Tokens) IsCode() bool {
	return tk.InCat(Keyword) || tk.InCat(Name) || tk.InCat(Operator) || tk.InCat(Punctuation)
}

// IsKeyword returns true if this in the Keyword category
func (tk Tokens) IsKeyword() bool {
	return tk.Cat() == Keyword
}

// Parent returns the closest parent-level of this token (subcat or cat)
func (tk Tokens) Parent() Tokens {
	if tk.IsSubCat() {
		return tk.Cat()
	}
	return tk.SubCat()
}

// Match returns true if the two tokens match, in a category / subcategory sensitive manner:
// if receiver token is a category, then it matches other token if it is the same category
// and likewise for subcategory
func (tk Tokens) Match(otk Tokens) bool {
	if tk == otk {
		return true
	}
	if tk.IsCat() && otk.Cat() == tk {
		return true
	}
	if tk.IsSubCat() && otk.SubCat() == tk {
		return true
	}
	return false
}

// IsPunctGpLeft returns true if token is a PunctGpL token -- left paren, brace, bracket
func (tk Tokens) IsPunctGpLeft() bool {
	return (tk == PunctGpLParen || tk == PunctGpLBrack || tk == PunctGpLBrace)
}

// IsPunctGpRight returns true if token is a PunctGpR token -- right paren, brace, bracket
func (tk Tokens) IsPunctGpRight() bool {
	return (tk == PunctGpRParen || tk == PunctGpRBrack || tk == PunctGpRBrace)
}

// PunctGpMatch returns the matching token for given PunctGp token
func (tk Tokens) PunctGpMatch() Tokens {
	switch tk {
	case PunctGpLParen:
		return PunctGpRParen
	case PunctGpRParen:
		return PunctGpLParen
	case PunctGpLBrack:
		return PunctGpRBrack
	case PunctGpRBrack:
		return PunctGpLBrack
	case PunctGpLBrace:
		return PunctGpRBrace
	case PunctGpRBrace:
		return PunctGpLBrace
	}
	return None
}

// IsAmbigUnaryOp returns true if this token is an operator that could either be
// a Unary or Binary operator -- need special matching for this.
// includes * and & which are used for address operations in C-like languages
func (tk Tokens) IsAmbigUnaryOp() bool {
	return (tk == OpMathSub || tk == OpMathMul || tk == OpBitAnd || tk == OpMathAdd || tk == OpBitXor)
}

// IsUnaryOp returns true if this token is an operator that is typically used as
// a Unary operator: - + & * ! ^ ! <-
func (tk Tokens) IsUnaryOp() bool {
	return (tk == OpMathSub || tk == OpMathMul || tk == OpBitAnd || tk == OpMathAdd ||
		tk == OpBitXor || tk == OpLogNot || tk == OpAsgnArrow)
}

// CombineRepeats are token types where repeated tokens of the same type should
// be combined together -- literals, comments, text
func (tk Tokens) CombineRepeats() bool {
	cat := tk.Cat()
	return (cat == Literal || cat == Comment || cat == Text || cat == Name)
}

// StyleName returns the abbreviated 2-3 letter style name of the tag
func (tk Tokens) StyleName() string {
	return Names[tk]
}

// ClassName returns the . prefixed CSS classname of the tag style
// for styling, a CSS property should exist with this name
func (tk Tokens) ClassName() string {
	return "." + tk.StyleName()
}

/////////////////////////////////////////////////////////////////////////////
//  KeyToken -- keyword + token

// KeyToken combines a token and an optional keyword name for Keyword token types
// if Tok is in Keyword category, then Key string can be used to check for same keyword.
// Also has a Depth for matching against a particular nesting depth
type KeyToken struct {
	Tok   Tokens
	Key   string
	Depth int
}

var KeyTokenZero = KeyToken{}

func (kt KeyToken) String() string {
	ds := ""
	if kt.Depth != 0 {
		ds = fmt.Sprintf("+%d:", kt.Depth)
	}
	if kt.Key != "" {
		return ds + kt.Tok.String() + ": " + kt.Key
	}
	return ds + kt.Tok.String()
}

// Equal compares equality of two tokens including keywords if token is in Keyword category.
// See also Match for version that uses category / subcategory matching
func (kt KeyToken) Equal(okt KeyToken) bool {
	if kt.Tok.IsKeyword() && kt.Key != "" {
		return kt.Tok == okt.Tok && kt.Key == okt.Key
	}
	return kt.Tok == okt.Tok
}

// Match compares equality of two tokens including keywords if token is in Keyword category.
// returns true if the two tokens match, in a category / subcategory sensitive manner:
// if receiver token is a category, then it matches other token if it is the same category
// and likewise for subcategory
func (kt KeyToken) Match(okt KeyToken) bool {
	if kt.Tok.IsKeyword() && kt.Key != "" {
		return kt.Tok.Match(okt.Tok) && kt.Key == okt.Key
	}
	return kt.Tok.Match(okt.Tok)
}

// MatchDepth compares equality of two tokens including depth -- see Match for other matching
// criteria
func (kt KeyToken) MatchDepth(okt KeyToken) bool {
	if kt.Depth != okt.Depth {
		return false
	}
	return kt.Match(okt)
}

// StringKey encodes token into a string for optimized string-based map key lookup
func (kt KeyToken) StringKey() string {
	tstr := string([]byte{byte(kt.Tok)})
	if kt.Tok.IsKeyword() {
		return tstr + kt.Key
	} else {
		return tstr
	}
}

// KeyTokenList is a list (slice) of KeyTokens
type KeyTokenList []KeyToken

// Match returns true if given keytoken matches any of the items on the list
func (kl KeyTokenList) Match(okt KeyToken) bool {
	for _, kt := range kl {
		if kt.Match(okt) {
			return true
		}
	}
	return false
}

// IconName returns the appropriate icon name for the type of lexical item this is
func (tk Tokens) IconName() string {
	switch {
	case tk.SubCat() == NameVar:
		return "var"
	case tk == NameConstant || tk == NameEnum || tk == NameEnumMember:
		return "const"
	case tk == NameField:
		return "field"
	case tk.SubCat() == NameType:
		return "type"
	case tk == NameMethod:
		return "method"
	case tk.SubCat() == NameFunction:
		return "function"
	}
	return ""
}

/////////////////////////////////////////////////////////////////////////////
//  Tokens

// The list of tokens
const (
	// None is the nil token value -- for non-terminal cases or TBD
	None Tokens = iota

	// Error is an input that could not be tokenized due to syntax error etc
	Error

	// EOF is end of file
	EOF

	// EOL is end of line (typically implicit -- used for rule matching)
	EOL

	// EOS is end of statement -- a key meta-token -- in C it is ;, in Go it is either ; or EOL
	EOS

	// Background is for syntax highlight styles based on these tokens
	Background

	// Cat: Keywords (actual keyword is just the string)
	Keyword
	KeywordConstant
	KeywordDeclaration
	KeywordNamespace // incl package, import
	KeywordPseudo
	KeywordReserved
	KeywordType

	// Cat: Names.
	Name
	NameBuiltin       // e.g., true, false -- builtin values..
	NameBuiltinPseudo // e.g., this, self
	NameOther
	NamePseudo

	// SubCat: Type names
	NameType
	NameClass
	NameStruct
	NameField
	NameInterface
	NameConstant
	NameEnum
	NameEnumMember
	NameArray // includes slice etc
	NameMap
	NameObject
	NameTypeParam // for generics, templates

	// SubCat: Function names
	NameFunction
	NameDecorator     // function-like wrappers in python
	NameFunctionMagic // e.g., __init__ in python
	NameMethod
	NameOperator
	NameConstructor // includes destructor..
	NameException
	NameLabel // e.g., goto label
	NameEvent // for LSP -- not really sure what it is..

	// SubCat: Scoping names
	NameScope
	NameNamespace
	NameModule
	NamePackage
	NameLibrary

	// SubCat: NameVar -- variable names
	NameVar
	NameVarAnonymous
	NameVarClass
	NameVarGlobal
	NameVarInstance
	NameVarMagic
	NameVarParam

	// SubCat: Value -- data-like elements
	NameValue
	NameTag // e.g., HTML tag
	NameProperty
	NameAttribute // e.g., HTML attr
	NameEntity    // special entities. (e.g. &nbsp; in HTML).  seems like other..

	// Cat: Literals.
	Literal
	LiteralDate
	LiteralOther
	LiteralBool

	// SubCat: Literal Strings.
	LitStr
	LitStrAffix // unicode specifiers etc
	LitStrAtom
	LitStrBacktick
	LitStrBoolean
	LitStrChar
	LitStrDelimiter
	LitStrDoc // doc-specific strings where syntactically noted
	LitStrDouble
	LitStrEscape   // esc sequences within strings
	LitStrHeredoc  // in ruby, perl
	LitStrInterpol // interpolated parts of strings in #{foo} in Ruby
	LitStrName
	LitStrOther
	LitStrRegex
	LitStrSingle
	LitStrSymbol
	LitStrFile // filename

	// SubCat: Literal Numbers.
	LitNum
	LitNumBin
	LitNumFloat
	LitNumHex
	LitNumInteger
	LitNumIntegerLong
	LitNumOct
	LitNumImag

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
	OpBitNot        // ~
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
	NameType,
	NameFunction,
	NameScope,
	NameVar,
	NameValue,
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

// OpPunctMap provides a lookup of operators and punctuation tokens by their usual
// string representation
var OpPunctMap = map[string]Tokens{
	"+": OpMathAdd,
	"-": OpMathSub,
	"*": OpMathMul,
	"/": OpMathDiv,
	"%": OpMathRem,

	"&":  OpBitAnd,
	"|":  OpBitOr,
	"~":  OpBitNot,
	"^":  OpBitXor,
	"<<": OpBitShiftLeft,
	">>": OpBitShiftRight,
	"&^": OpBitAndNot,

	"=":  OpAsgnAssign,
	"++": OpAsgnInc,
	"--": OpAsgnDec,
	"<-": OpAsgnArrow,
	":=": OpAsgnDefine,

	"+=": OpMathAsgnAdd,
	"-=": OpMathAsgnSub,
	"*=": OpMathAsgnMul,
	"/=": OpMathAsgnDiv,
	"%=": OpMathAsgnRem,

	"&=":  OpBitAsgnAnd,
	"|=":  OpBitAsgnOr,
	"^=":  OpBitAsgnXor,
	"<<=": OpBitAsgnShiftLeft,
	">>=": OpBitAsgnShiftRight,
	"&^=": OpBitAsgnAndNot,

	"&&": OpLogAnd,
	"||": OpLogOr,
	"!":  OpLogNot,

	"==": OpRelEqual,
	"!=": OpRelNotEqual,
	"<":  OpRelLess,
	">":  OpRelGreater,
	"<=": OpRelLtEq,
	">=": OpRelGtEq,

	"...": OpListEllipsis,

	"(": PunctGpLParen,
	")": PunctGpRParen,
	"[": PunctGpLBrack,
	"]": PunctGpRBrack,
	"{": PunctGpLBrace,
	"}": PunctGpRBrace,

	",": PunctSepComma,
	".": PunctSepPeriod,
	";": PunctSepSemicolon,
	":": PunctSepColon,

	"\"": PunctStrDblQuote,
	"'":  PunctStrQuote,
	"`":  PunctStrBacktick,
	"\\": PunctStrEsc,
}

// Names are the short tag names for each token, used e.g., for syntax highlighting
// These are based on alecthomas/chroma / pygments
var Names = map[Tokens]string{
	None:       "",
	Error:      "err",
	EOF:        "EOF",
	EOL:        "EOL",
	EOS:        "EOS",
	Background: "bg",

	Keyword:            "k",
	KeywordConstant:    "kc",
	KeywordDeclaration: "kd",
	KeywordNamespace:   "kn",
	KeywordPseudo:      "kp",
	KeywordReserved:    "kr",
	KeywordType:        "kt",

	Name:              "n",
	NameBuiltin:       "nb",
	NameBuiltinPseudo: "bp",
	NameOther:         "nx",
	NamePseudo:        "pu",

	NameType:       "nt",
	NameClass:      "nc",
	NameStruct:     "ns",
	NameField:      "nfl",
	NameInterface:  "nti",
	NameConstant:   "no",
	NameEnum:       "nen",
	NameEnumMember: "nem",
	NameArray:      "nr",
	NameMap:        "nm",
	NameObject:     "nj",
	NameTypeParam:  "ntp",

	NameFunction:      "nf",
	NameDecorator:     "nd",
	NameFunctionMagic: "fm",
	NameMethod:        "mt",
	NameOperator:      "np",
	NameConstructor:   "cr",
	NameException:     "ne",
	NameLabel:         "nl",
	NameEvent:         "ev",

	NameScope:     "nsc",
	NameNamespace: "nn",
	NameModule:    "md",
	NamePackage:   "pk",
	NameLibrary:   "lb",

	NameVar:          "nv",
	NameVarAnonymous: "ay",
	NameVarClass:     "vc",
	NameVarGlobal:    "vg",
	NameVarInstance:  "vi",
	NameVarMagic:     "vm",
	NameVarParam:     "vp",

	NameValue:     "vl",
	NameTag:       "ng",
	NameProperty:  "py",
	NameAttribute: "na",
	NameEntity:    "ni",

	Literal:      "l",
	LiteralDate:  "ld",
	LiteralOther: "lo",
	LiteralBool:  "bo",

	LitStr:          "s",
	LitStrAffix:     "sa",
	LitStrAtom:      "st",
	LitStrBacktick:  "sb",
	LitStrBoolean:   "so",
	LitStrChar:      "sc",
	LitStrDelimiter: "dl",
	LitStrDoc:       "sd",
	LitStrDouble:    "s2",
	LitStrEscape:    "se",
	LitStrHeredoc:   "sh",
	LitStrInterpol:  "si",
	LitStrName:      "sn",
	LitStrOther:     "sx",
	LitStrRegex:     "sr",
	LitStrSingle:    "s1",
	LitStrSymbol:    "ss",
	LitStrFile:      "fl",

	LitNum:            "m",
	LitNumBin:         "mb",
	LitNumFloat:       "mf",
	LitNumHex:         "mh",
	LitNumInteger:     "mi",
	LitNumIntegerLong: "il",
	LitNumOct:         "mo",
	LitNumImag:        "mj",

	Operator:     "o",
	OperatorWord: "ow",

	// don't really need these -- only have at sub-categ level
	OpMath:     "om",
	OpBit:      "ob",
	OpAsgn:     "oa",
	OpMathAsgn: "pa",
	OpBitAsgn:  "ba",
	OpLog:      "ol",
	OpRel:      "or",
	OpList:     "oi",

	Punctuation: "p",
	PunctGp:     "pg",
	PunctSep:    "ps",
	PunctStr:    "pr",

	Comment:          "c",
	CommentHashbang:  "ch",
	CommentMultiline: "cm",
	CommentSingle:    "c1",
	CommentSpecial:   "cs",

	CommentPreproc:     "cp",
	CommentPreprocFile: "cpf",

	Text:            "",
	TextWhitespace:  "w",
	TextSymbol:      "ts",
	TextPunctuation: "tp",
	TextSpellErr:    "te",

	TextStyle:           "g",
	TextStyleDeleted:    "gd",
	TextStyleEmph:       "ge",
	TextStyleError:      "gr",
	TextStyleHeading:    "gh",
	TextStyleInserted:   "gi",
	TextStyleOutput:     "go",
	TextStylePrompt:     "gp",
	TextStyleStrong:     "gs",
	TextStyleSubheading: "gu",
	TextStyleTraceback:  "gt",
	TextStyleUnderline:  "gl",
	TextStyleLink:       "ga",
}
