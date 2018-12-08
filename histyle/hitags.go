// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package histyle

import (
	"github.com/alecthomas/chroma"
	"github.com/goki/pi/token"
)

// FromChroma converts a chroma.TokenType to a pi token.Tokens
func TokenFromChroma(ct chroma.TokenType) token.Tokens {
	if ChromaToTokensMap == nil {
		ChromaToTokensMap = make(map[chroma.TokenType]token.Tokens, len(TokensToChromaMap))
		for k, v := range TokensToChromaMap {
			ChromaToTokensMap[v] = k
		}
	}
	tok := ChromaToTokensMap[ct]
	return tok
}

// TokenToChroma converts to a chroma.TokenType
func TokenToChroma(tok token.Tokens) chroma.TokenType {
	return TokensToChromaMap[tok]
}

// ChromaToTokensMap maps from chroma.TokenType to Tokens -- built from opposite map
var ChromaToTokensMap map[chroma.TokenType]token.Tokens

// TokensToChromaMap maps from Tokens to chroma.TokenType
var TokensToChromaMap = map[token.Tokens]chroma.TokenType{
	token.EOF:                 chroma.EOFType,
	token.Background:          chroma.Background,
	token.Error:               chroma.Error,
	token.None:                chroma.None,
	token.Keyword:             chroma.Keyword,
	token.KeywordConstant:     chroma.KeywordConstant,
	token.KeywordDeclaration:  chroma.KeywordDeclaration,
	token.KeywordNamespace:    chroma.KeywordNamespace,
	token.KeywordPseudo:       chroma.KeywordPseudo,
	token.KeywordReserved:     chroma.KeywordReserved,
	token.KeywordType:         chroma.KeywordType,
	token.Name:                chroma.Name,
	token.NameAttribute:       chroma.NameAttribute,
	token.NameBuiltin:         chroma.NameBuiltin,
	token.NameBuiltinPseudo:   chroma.NameBuiltinPseudo,
	token.NameClass:           chroma.NameClass,
	token.NameConstant:        chroma.NameConstant,
	token.NameDecorator:       chroma.NameDecorator,
	token.NameEntity:          chroma.NameEntity,
	token.NameException:       chroma.NameException,
	token.NameFunction:        chroma.NameFunction,
	token.NameFunctionMagic:   chroma.NameFunctionMagic,
	token.NameLabel:           chroma.NameLabel,
	token.NameNamespace:       chroma.NameNamespace,
	token.NameOperator:        chroma.NameOperator,
	token.NameOther:           chroma.NameOther,
	token.NamePseudo:          chroma.NamePseudo,
	token.NameProperty:        chroma.NameProperty,
	token.NameTag:             chroma.NameTag,
	token.NameVar:             chroma.NameVariable,
	token.NameVarAnonymous:    chroma.NameVariableAnonymous,
	token.NameVarClass:        chroma.NameVariableClass,
	token.NameVarGlobal:       chroma.NameVariableGlobal,
	token.NameVarInstance:     chroma.NameVariableInstance,
	token.NameVarMagic:        chroma.NameVariableMagic,
	token.Literal:             chroma.Literal,
	token.LiteralDate:         chroma.LiteralDate,
	token.LiteralOther:        chroma.LiteralOther,
	token.LitStr:              chroma.LiteralString,
	token.LitStrAffix:         chroma.LiteralStringAffix,
	token.LitStrAtom:          chroma.LiteralStringAtom,
	token.LitStrBacktick:      chroma.LiteralStringBacktick,
	token.LitStrBoolean:       chroma.LiteralStringBoolean,
	token.LitStrChar:          chroma.LiteralStringChar,
	token.LitStrDelimiter:     chroma.LiteralStringDelimiter,
	token.LitStrDoc:           chroma.LiteralStringDoc,
	token.LitStrDouble:        chroma.LiteralStringDouble,
	token.LitStrEscape:        chroma.LiteralStringEscape,
	token.LitStrHeredoc:       chroma.LiteralStringHeredoc,
	token.LitStrInterpol:      chroma.LiteralStringInterpol,
	token.LitStrName:          chroma.LiteralStringName,
	token.LitStrOther:         chroma.LiteralStringOther,
	token.LitStrRegex:         chroma.LiteralStringRegex,
	token.LitStrSingle:        chroma.LiteralStringSingle,
	token.LitStrSymbol:        chroma.LiteralStringSymbol,
	token.LitNum:              chroma.LiteralNumber,
	token.LitNumBin:           chroma.LiteralNumberBin,
	token.LitNumFloat:         chroma.LiteralNumberFloat,
	token.LitNumHex:           chroma.LiteralNumberHex,
	token.LitNumInteger:       chroma.LiteralNumberInteger,
	token.LitNumIntegerLong:   chroma.LiteralNumberIntegerLong,
	token.LitNumOct:           chroma.LiteralNumberOct,
	token.Operator:            chroma.Operator,
	token.OperatorWord:        chroma.OperatorWord,
	token.Punctuation:         chroma.Punctuation,
	token.Comment:             chroma.Comment,
	token.CommentHashbang:     chroma.CommentHashbang,
	token.CommentMultiline:    chroma.CommentMultiline,
	token.CommentSingle:       chroma.CommentSingle,
	token.CommentSpecial:      chroma.CommentSpecial,
	token.CommentPreproc:      chroma.CommentPreproc,
	token.CommentPreprocFile:  chroma.CommentPreprocFile,
	token.Text:                chroma.Text,
	token.TextWhitespace:      chroma.TextWhitespace,
	token.TextSymbol:          chroma.TextSymbol,
	token.TextPunctuation:     chroma.TextPunctuation,
	token.TextStyle:           chroma.Generic,
	token.TextStyleDeleted:    chroma.GenericDeleted,
	token.TextStyleEmph:       chroma.GenericEmph,
	token.TextStyleError:      chroma.GenericError,
	token.TextStyleHeading:    chroma.GenericHeading,
	token.TextStyleInserted:   chroma.GenericInserted,
	token.TextStyleOutput:     chroma.GenericOutput,
	token.TextStylePrompt:     chroma.GenericPrompt,
	token.TextStyleStrong:     chroma.GenericStrong,
	token.TextStyleSubheading: chroma.GenericSubheading,
	token.TextStyleTraceback:  chroma.GenericTraceback,
	token.TextStyleUnderline:  chroma.GenericUnderline,
}
