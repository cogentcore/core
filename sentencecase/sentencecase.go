// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package sentencecase converts CamelCase strings to sentence case.
// An example of a string in sentence case is:
//
//	This is a string in sentence case that I wrote with the help of URLs and Google
package sentencecase

import (
	"strings"
	"unicode"

	"github.com/fatih/camelcase"
)

// ProperNouns is a set-style map that contains all of the proper
// nouns that will not be lowercased (such as Google or Pike). Proper
// nouns should be specified in their CamelCase form (Google, Pike, etc).
// Proper nouns should be added to this map through [AddProperNouns].
// By default, this map contains nothing.
var ProperNouns = map[string]struct{}{}

// AddProperNouns adds the given proper nouns (such as Google or Pike)
// to [ProperNouns]. It is the way that end-user code should specify proper nouns.
// Proper nouns will not be lowercased when converting to sentence case. Proper
// nouns should be specified in their CamelCase form (Google, Pike, etc).
func AddProperNouns(nouns ...string) {
	for _, noun := range nouns {
		ProperNouns[noun] = struct{}{}
	}
}

// Of returns the sentence case version of the given CamelCase string.
// It handles proper nouns through [ProperNouns], abbreviations,
// and "I".
//
// An example of a string in sentence case is:
//
//	This is a string in sentence case that I wrote with the help of URLs and Google
func Of(s string) string {
	words := camelcase.Split(s)
	for i, word := range words {
		// the first word and "I" are always capitalized
		if i == 0 || word == "I" {
			continue
		}
		// proper nouns are always capitalized
		if _, ok := ProperNouns[word]; ok {
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
