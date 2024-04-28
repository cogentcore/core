// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/ettle/strcase
// Copyright (c) 2020 Liyan David Chang under the MIT License

package strcase

import (
	"strings"
)

// Cases is an enum with all of the different string cases.
type Cases int32 //enums:enum

const (
	// LowerCase is all lower case
	LowerCase Cases = iota

	// UpperCase is all UPPER CASE
	UpperCase

	// SnakeCase is lower_case_words_with_underscores
	SnakeCase

	// SNAKECase is UPPER_CASE_WORDS_WITH_UNDERSCORES
	SNAKECase

	// KebabCase is lower-case-words-with-dashes
	KebabCase

	// KEBABCase is UPPER-CASE-WORDS-WITH-DASHES
	KEBABCase

	// CamelCase is CapitalizedWordsConcatenatedTogether
	CamelCase

	// LowerCamelCase is capitalizedWordsConcatenatedTogether, with the first word lower case
	LowerCamelCase

	// TitleCase is Captitalized Words With Spaces
	TitleCase

	// SentenceCase is Lower case words with spaces, with the first word capitalized
	SentenceCase
)

// To converts the given string to the given case.
func To(s string, c Cases) string {
	switch c {
	case LowerCase:
		return strings.ToLower(s)
	case UpperCase:
		return strings.ToUpper(s)
	case SnakeCase:
		return ToSnake(s)
	case SNAKECase:
		return ToSNAKE(s)
	case KebabCase:
		return ToKebab(s)
	case KEBABCase:
		return ToKEBAB(s)
	case CamelCase:
		return ToCamel(s)
	case LowerCamelCase:
		return ToLowerCamel(s)
	case TitleCase:
		return ToTitle(s)
	case SentenceCase:
		return ToSentence(s)
	}
	return s
}
