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

// WordCase is an enumeration of the ways to format a word.
type WordCase int

const (
	// Original - Preserve the original input strcase
	Original WordCase = iota
	// LowerCase - All letters lower cased (example)
	LowerCase
	// UpperCase - All letters upper cased (EXAMPLE)
	UpperCase
	// TitleCase - Only first letter upper cased (Example)
	TitleCase
	// CamelCase - TitleCase except lower case first word (exampleText)
	// Notably, even if the first word is an initialism, it will be lower
	// cased. This is important for code generators where capital letters
	// mean exported functions. i.e. jsonString(), not JSONString()
	CamelCase
)

// convert changes a input string to a certain case with a delimiter,
// respecting arbitrary initialisms and skip characters
//
//nolint:gocyclo
func convert(input string, delimiter rune, wordCase WordCase) string {
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
		if wordCase == TitleCase || (wordCase == CamelCase && !firstWord) {
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

		skipIdx := 0
		for i := start; i < end; i++ {
			if len(skipIndexes) > 0 && skipIdx < len(skipIndexes) && i == skipIndexes[skipIdx] {
				skipIdx++
				continue
			}
			r := runes[i]
			switch wordCase {
			case UpperCase:
				b.WriteRune(toUpper(r))
			case LowerCase:
				b.WriteRune(toLower(r))
			case TitleCase:
				if i == start {
					b.WriteRune(toUpper(r))
				} else {
					b.WriteRune(toLower(r))
				}
			case CamelCase:
				if !firstWord && i == start {
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
