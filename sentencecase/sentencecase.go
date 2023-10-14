// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package sentencecase converts strings to sentence case.
// An example of a string in sentence case is:
//
//	This is a string in sentence case that I wrote
package sentencecase

import (
	"strings"
	"unicode"

	"github.com/fatih/camelcase"
)

// ProperNouns is a set-style map that contains all of the proper
// nouns that will not be lowercased (such as Google, Rob Pike, etc).
// Proper nouns should be added to this map through [AddProperNouns].
// By default, it contains nothing.
var ProperNouns = map[string]struct{}{}

// AddProperNouns adds the given proper nouns (such as Google, Rob Pike, etc)
// to [ProperNouns]. It is the way that end-user code should specify proper nouns.
// Proper nouns will not be lowercased when converting to sentence case.
func AddProperNouns(nouns ...string) {
	for _, noun := range nouns {
		ProperNouns[noun] = struct{}{}
	}
}

// Of returns the sentence case version of the given string.
// It handles proper nouns through [ProperNouns], abbreviations,
// and "I".
//
// An example of a string in sentence case is:
//
//	This is a string in sentence case that I wrote with abbreviations (URL) and proper nouns (Google).
func Of(s string) string {
	words := camelcase.Split(s)
	for i, word := range words {
		// the first word and "I" are always capitalized
		if i == 0 || word == "I" {
			continue
		}
		// proper nouns are always capitalized
		if _, ok := ProperNouns[s]; ok {
			continue
		}
		r := []rune(word)
		// if there are multiple capital letters in a row, we assume
		// that it is an abbreviation
		if len(r) > 1 && unicode.IsUpper(r[1]) {
			continue
		}
		// otherwise, we make it lowercase
		words[i] = strings.ToLower(word)
	}
	s = strings.Join(words, " ")
	return s
}
