// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/ettle/strcase
// Copyright (c) 2020 Liyan David Chang under the MIT License

package strcase

import (
	"strings"
	"unicode"
)

// WordCases is an enumeration of the ways to format a word.
type WordCases int32 //enums:enum -trim-prefix Word

const (
	// WordOriginal indicates to preserve the original input case.
	WordOriginal WordCases = iota

	// WordLowerCase indicates to make all letters lower case (example).
	WordLowerCase

	// WordUpperCase indicates to make all letters upper case (EXAMPLE).
	WordUpperCase

	// WordTitleCase indicates to make only the first letter upper case (Example).
	WordTitleCase

	// WordCamelCase indicates to make only the first letter upper case, except
	// in the first word, in which all letters are lower case (exampleText).
	WordCamelCase

	// WordSentenceCase indicates to make only the first letter upper case, and
	// only for the first word (all other words have fully lower case letters).
	WordSentenceCase
)

// ToWordCase converts the given input string to the given word case with the given delimiter.
// Pass 0 for delimeter to use no delimiter.
//
//nolint:gocyclo
func ToWordCase(input string, wordCase WordCases, delimiter rune) string {
	input = strings.TrimSpace(input)
	runes := []rune(input)
	if len(runes) == 0 {
		return ""
	}

	var b strings.Builder
	b.Grow(len(input) + 4) // In case we need to write delimiters where they weren't before

	firstWord := true
	var skipIndexes []int

	addWord := func(start, end int) {
		// If you have nothing good to say, say nothing at all
		if start == end || len(skipIndexes) == end-start {
			skipIndexes = nil
			return
		}

		// If you have something to say, start with a delimiter
		if !firstWord && delimiter != 0 {
			b.WriteRune(delimiter)
		}

		// Check to see if the entire word is an initialism for preserving initialism.
		// Note we don't support preserving initialisms if they are followed
		// by a number and we're not spliting before numbers.
		if wordCase == WordTitleCase || wordCase == WordSentenceCase || (wordCase == WordCamelCase && !firstWord) {
			allCaps := true
			for i := start; i < end; i++ {
				allCaps = allCaps && (isUpper(runes[i]) || !unicode.IsLetter(runes[i]))
			}
			if allCaps {
				b.WriteString(string(runes[start:end]))
				firstWord = false
				return
			}
		}

		skipIndex := 0
		for i := start; i < end; i++ {
			if len(skipIndexes) > 0 && skipIndex < len(skipIndexes) && i == skipIndexes[skipIndex] {
				skipIndex++
				continue
			}
			r := runes[i]
			switch wordCase {
			case WordUpperCase:
				b.WriteRune(toUpper(r))
			case WordLowerCase:
				b.WriteRune(toLower(r))
			case WordTitleCase:
				if i == start {
					b.WriteRune(toUpper(r))
				} else {
					b.WriteRune(toLower(r))
				}
			case WordCamelCase:
				if !firstWord && i == start {
					b.WriteRune(toUpper(r))
				} else {
					b.WriteRune(toLower(r))
				}
			case WordSentenceCase:
				if firstWord && i == start {
					b.WriteRune(toUpper(r))
				} else {
					b.WriteRune(toLower(r))
				}
			default:
				b.WriteRune(r)
			}
		}
		firstWord = false
		skipIndexes = nil
	}

	var prev, curr rune
	next := runes[0] // 0 length will have already returned so safe to index
	wordStart := 0
	for i := 0; i < len(runes); i++ {
		prev = curr
		curr = next
		if i+1 == len(runes) {
			next = 0
		} else {
			next = runes[i+1]
		}

		switch defaultSplitFn(prev, curr, next) {
		case Skip:
			skipIndexes = append(skipIndexes, i)
		case Split:
			addWord(wordStart, i)
			wordStart = i
		case SkipSplit:
			addWord(wordStart, i)
			wordStart = i + 1
		}
	}

	if wordStart != len(runes) {
		addWord(wordStart, len(runes))
	}
	return b.String()
}
