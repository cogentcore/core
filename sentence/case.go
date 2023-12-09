// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package sentence provides functions for creating formatted sentences.
package sentence

import (
	"strings"
	"unicode"

	"github.com/fatih/camelcase"
	"github.com/iancoleman/strcase"
)

// ProperNouns is a set-style map that contains all of the proper
// nouns that will not be lowercased (such as Google or Thompson). Proper
// nouns should be specified in their CamelCase form (Google, Thompson, etc).
// Proper nouns should be added to this map through [AddProperNouns].
// By default, this map contains nothing.
var ProperNouns = map[string]struct{}{}

// AddProperNouns adds the given proper nouns (such as Google or Thompson)
// to [ProperNouns]. It is the way that end-user code should specify proper nouns.
// Proper nouns will not be lowercased when converting to sentence case. Proper
// nouns should be specified in their CamelCase form (Google, Thompson, etc).
func AddProperNouns(nouns ...string) {
	for _, noun := range nouns {
		ProperNouns[noun] = struct{}{}
	}
}

// Case returns the sentence case version of the given string.
// It handles proper nouns through [ProperNouns], abbreviations,
// and "I".
//
// An example of a string in sentence case is:
//
//	This is a string in sentence case that I wrote in the USA with the help of Google
func Case(s string) string {
	// if we are not already in camel case, we convert to it
	if strings.ContainsAny(s, "-_.\t ") {
		s = strcase.ToCamel(s)
	}
	words := camelcase.Split(s)
	for i, word := range words {
		// "I" is always capitalized
		if word == "I" {
			continue
		}
		// the first letter of proper nouns is always capitalized
		if _, ok := ProperNouns[word]; ok {
			continue
		}
		r := []rune(word)
		// the first letter of the first word is always capitalized
		// (and could be not capitalized in lowerCamelCase, so we need
		// to ensure that it is capitalized)
		if i == 0 {
			if len(r) > 0 {
				r[0] = unicode.ToUpper(r[0])
				words[i] = string(r)
			}
			continue
		}
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
