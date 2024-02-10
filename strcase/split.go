// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/ettle/strcase
// Copyright (c) 2020 Liyan David Chang under the MIT License

package strcase

import "unicode"

// SplitAction defines if and how to split a string
type SplitAction int

const (
	// Noop - Continue to next character
	Noop SplitAction = iota
	// Split - Split between words
	// e.g. to split between wordsWithoutDelimiters
	Split
	// SkipSplit - Split the word and drop the character
	// e.g. to split words with delimiters
	SkipSplit
	// Skip - Remove the character completely
	Skip
)

//nolint:gocyclo
func defaultSplitFn(prev, curr, next rune) SplitAction {
	// The most common case will be that it's just a letter so let lowercase letters return early since we know what they should do
	if isLower(curr) {
		return Noop
	}
	// Delimiters are _, -, ., and unicode spaces
	// Handle . lower down as it needs to happen after number exceptions
	if curr == '_' || curr == '-' || isSpace(curr) {
		return SkipSplit
	}

	if isUpper(curr) {
		if isLower(prev) {
			// fooBar
			return Split
		} else if isUpper(prev) && isLower(next) {
			// FOOBar
			return Split
		}
	}

	// Do numeric exceptions last to avoid perf penalty
	if unicode.IsNumber(prev) {
		// v4.3 is not split
		if (curr == '.' || curr == ',') && unicode.IsNumber(next) {
			return Noop
		}
		if !unicode.IsNumber(curr) && curr != '.' {
			return Split
		}
	}
	// While period is a default delimiter, keep it down here to avoid
	// penalty for other delimiters
	if curr == '.' {
		return SkipSplit
	}

	return Noop
}
