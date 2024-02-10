// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textbuf

//go:generate core generate

import (
	"strings"

	"github.com/iancoleman/strcase"
)

// Cases are different string cases: Lower, Upper, Camel, etc
type Cases int32 //enums:enum

const (
	// LowerCase is all lower case
	LowerCase Cases = iota

	// UpperCase is all UPPER CASE
	UpperCase

	// CamelCase is CapitalizedWordsConcatenatedTogether
	CamelCase

	// LowerCamelCase is capitalizedWordsConcatenatedTogether, with the first word lower case
	LowerCamelCase

	// SnakeCase is lower_case_words_with_underscores
	SnakeCase

	// SNAKECase is UPPER_CASE_WORDS_WITH_UNDERSCORES
	SNAKECase

	// KebabCase is lower-case-words-with-dashes
	KebabCase

	// KEBABCase is UPPER-CASE-WORDS-WITH-DASHES
	KEBABCase

	// TitleCase is Captitalized Words With Spaces
	TitleCase

	// SentenceCase is Lower case words with spaces, with the first word capitalized
	SentenceCase
)

// ReCaseString changes the case of the string according to the given case type.
func ReCaseString(str string, c Cases) string {
	switch c {
	case LowerCase:
		return strings.ToLower(str)
	case UpperCase:
		return strings.ToUpper(str)
	case CamelCase:
		return strcase.ToCamel(str)
	case LowerCamelCase:
		return strcase.ToLowerCamel(str)
	case SnakeCase:
		return strcase.ToSnake(str)
	case SNAKECase:
		return strcase.ToScreamingSnake(str)
	case KebabCase:
		return strcase.ToKebab(str)
	}
	return str
}
