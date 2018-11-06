// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
	Package Complete provides functions for text completion
*/
package complete

import (
	"go/token"
	"sort"
	"strings"
	"unicode"

	"github.com/akutz/sortfold"
)

type Completion struct {
	Text string // completion text
	Icon string // icon name
	Desc string // possible extra information, e.g. type, arguments, etc. - not currently used
}

type Completions []Completion

// MatchFunc is the function called to get the list of possible completions
// and also determines the correct seed based on the text passed as a parameter of CompletionFunc
type MatchFunc func(data interface{}, text string, pos token.Position) (matches Completions, seed string)

// EditFunc is passed the current text and the selected completion for text editing.
// Allows for other editing, e.g. adding "()" or adding "/", etc.
type EditFunc func(data interface{}, text string, cursorPos int, completion string, seed string) (newText string, delta int)

// MatchSeed returns a list of matches given a list of string possibilities and a seed.
// The seed is basically a prefix.
func MatchSeedString(completions []string, seed string) (matches []string) {
	matches = completions[0:0]
	match_start := -1
	match_end := -1

	// fast case insensitive sort from Andrew Kutz
	less := func(i, j int) bool {
		return sortfold.CompareFold(completions[i], completions[j]) < 0
	}
	sort.Slice(completions, less)

	if len(seed) == 0 {
		matches = completions
		return matches
	}

	for i, s := range completions {
		if match_end > -1 {
			break
		}

		var case_insensitive = true
		if HasUpperCase(seed) {
			case_insensitive = false
		}
		text := s
		if case_insensitive {
			text = strings.ToLower(s)
		}

		if match_start == -1 {
			if strings.HasPrefix(text, seed) {
				match_start = i // first match in sorted list
			}
			continue
		}
		if match_start > -1 {
			if strings.HasPrefix(text, seed) == false {
				match_end = i
			}
		}
	}
	if match_start > -1 && match_end == -1 { // everything possible was a match!
		match_end = len(completions)
	}

	//fmt.Printf("match start: %d, match_end: %d", match_start, match_end)
	if match_start > -1 && match_end > -1 {
		matches = completions[match_start:match_end]
	}
	return matches
}

// MatchSeedCompletion returns a list of matching completion structs given a list of possibilities and a seed.
// The seed is basically a prefix.
func MatchSeedCompletion(completions []Completion, seed string) (matches []Completion) {
	matches = completions[0:0]
	match_start := -1
	match_end := -1

	// fast case insensitive sort from Andrew Kutz
	less := func(i, j int) bool {
		return sortfold.CompareFold(completions[i].Text, completions[j].Text) < 0
	}
	sort.Slice(completions, less)

	if len(seed) == 0 {
		matches = completions
		return matches
	}

	var case_insensitive = true
	if HasUpperCase(seed) {
		case_insensitive = false
	}
	for i, c := range completions {
		if match_end > -1 {
			break
		}
		text := c.Text
		if case_insensitive {
			text = strings.ToLower(text)
		}
		if match_start == -1 {
			if strings.HasPrefix(text, seed) {
				match_start = i // first match in sorted list
			}
			continue
		}
		if match_start > -1 {
			if strings.HasPrefix(text, seed) == false {
				match_end = i
			}
		}
	}
	if match_start > -1 && match_end == -1 { // everything possible was a match!
		match_end = len(completions)
	}

	//fmt.Printf("match start: %d, match_end: %d", match_start, match_end)
	if match_start > -1 && match_end > -1 {
		matches = completions[match_start:match_end]
	}
	return matches
}

// ExtendSeed tries to extend the current seed checking possible completions for a longer common seed
// e.g. if the current seed is "ab" and the completions are "abcde" and "abcdf" then Extend returns "cd"
// but if the possible completions are "abcde" and "abz" then Extend returns ""
func ExtendSeed(matches Completions, seed string) string {
	keep_looking := true
	new_seed := seed
	potential_seed := new_seed
	first_match := matches[0]
	for keep_looking {
		if len(first_match.Text) <= len(new_seed) {
			keep_looking = false // ran out of chars
			break
		}

		potential_seed = first_match.Text[0 : len(new_seed)+1]
		for _, s := range matches {
			if !strings.HasPrefix(s.Text, potential_seed) {
				keep_looking = false
				break
			}
		}
		if keep_looking {
			new_seed = potential_seed
		}
	}
	return new_seed
}

// SeedWhiteSpace returns the text after the last whitespace
func SeedWhiteSpace(text string) string {
	seedStart := 0
	for i := len(text) - 1; i >= 0; i-- {
		r := rune(text[i])
		if unicode.IsSpace(r) {
			seedStart = i + 1
			break
		}
	}
	return text[seedStart:]
}

// SeedGolang returns the seed that makes sense for the go language
// e.g. strip [] or . prefix before matching text
func SeedGolang(text string) string {
	seedStart := 0
	for i := len(text) - 1; i >= 0; i-- {
		r := rune(text[i])
		if unicode.IsSpace(r) || r == '.' {
			seedStart = i + 1
			break
		}
	}
	seed := text[seedStart:]
	if strings.HasPrefix(seed, "[]") {
		seed = strings.TrimPrefix(seed, "[]")
		return seed
	}

	return seed
}

// EditBasic replaces the completion seed with the completion
// delta is the change in cursor position (cp)
func EditBasic(text string, cp int, completion string, seed string) (newText string, delta int) {
	s1 := string(text[0:cp])
	s2 := string(text[cp:])
	s1 = strings.TrimSuffix(s1, seed)
	s1 += completion
	t := s1 + s2
	delta = len(completion) - len(seed)
	return t, delta
}

// EditWord replaces the completion seed and any text up to the next whitespace with completion
// delta is the change in cursor position (cp)
func EditWord(text string, cp int, completion string, seed string) (newText string, delta int) {
	s1 := string(text[0:cp])
	s2 := string(text[cp:])

	if len(s2) > 0 {
		r := rune(s2[0])
		// find the next whitespace or end of text
		if !(unicode.IsSpace(r)) {
			count := len(s2)
			for i, c := range s2 {
				r = rune(c)
				if unicode.IsSpace(r) {
					s2 = s2[i:]
					break
				}
				// might be last word
				if i == count-1 {
					s2 = ""
				}
			}
		}
	}

	s1 = strings.TrimSuffix(s1, seed)
	s1 += completion
	t := s1 + s2
	delta = len(completion) - len(seed)
	return t, delta
}

// HasUpperCase returns true if string has an upper-case letter
func HasUpperCase(str string) bool {
	for _, r := range str {
		if unicode.IsUpper(r) {
			return true
		}
	}
	return false
}
