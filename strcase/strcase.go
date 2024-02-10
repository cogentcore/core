// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/ettle/strcase
// Copyright (c) 2020 Liyan David Chang under the MIT License

// Package strcase provides functions for manipulating the case of strings (CamelCase, kebab-case,
// snake_case, Sentence case, etc). It is based on https://github.com/ettle/strcase, which is Copyright
// (c) 2020 Liyan David Chang under the MIT License. Its principle difference from other strcase packages
// is that it preserves acronyms in input text for CamelCase. Therefore, you must call [strings.ToLower]
// on any SCREAMING_INPUT_STRINGS before passing them to [ToCamel], [ToLowerCamel], [ToTitle], and [ToSentence].
package strcase

//go:generate core generate

// ToSnake returns words in snake_case (lower case words with underscores).
func ToSnake(s string) string {
	return ToWordCase(s, WordLowerCase, '_')
}

// ToSNAKE returns words in SNAKE_CASE (upper case words with underscores).
// Also known as SCREAMING_SNAKE_CASE or UPPER_CASE.
func ToSNAKE(s string) string {
	return ToWordCase(s, WordUpperCase, '_')
}

// ToKebab returns words in kebab-case (lower case words with dashes).
// Also known as dash-case.
func ToKebab(s string) string {
	return ToWordCase(s, WordLowerCase, '-')
}

// ToKEBAB returns words in KEBAB-CASE (upper case words with dashes).
// Also known as SCREAMING-KEBAB-CASE or SCREAMING-DASH-CASE.
func ToKEBAB(s string) string {
	return ToWordCase(s, WordUpperCase, '-')
}

// ToCamel returns words in CamelCase (capitalized words concatenated together).
// Also known as UpperCamelCase.
func ToCamel(s string) string {
	return ToWordCase(s, WordTitleCase, 0)
}

// ToLowerCamel returns words in lowerCamelCase (capitalized words concatenated together,
// with first word lower case). Also known as camelCase or mixedCase.
func ToLowerCamel(s string) string {
	return ToWordCase(s, WordCamelCase, 0)
}

// ToTitle returns words in Title Case (capitalized words with spaces).
func ToTitle(s string) string {
	return ToWordCase(s, WordTitleCase, ' ')
}

// ToSentence returns words in Sentence case (lower case words with spaces, with the first word capitalized).
func ToSentence(s string) string {
	return ToWordCase(s, WordSentenceCase, ' ')
}
