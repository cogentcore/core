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

// Completion holds one potential completion
type Completion struct {
	Text  string            `desc:"completion text -- what will actually be inserted if selected"`
	Label string            `desc:"label to show the user -- only used for menu display if non-empty -- otherwise Text is used"`
	Icon  string            `desc:"icon name"`
	Desc  string            `desc:"possible extra information, e.g. type, arguments, etc. - not currently used"	`
	Extra map[string]string `desc:"lang specific or other, e.g. class or type"`
}

// Completions is a full list (slice) of completion options
type Completions []Completion

// MatchData is used for passing completions around -- contains seed in addition to completions
type MatchData struct {
	Matches Completions `desc:"the matches based on seed"`
	Seed    string      `desc:"seed is the prefix we use to find possible completions"`
}

// EditData is returned from completion edit function to incorporate the selected completion
type EditData struct {
	NewText       string `desc:"completion text after special edits"`
	ForwardDelete int    `desc:"number of runes, past the cursor, to delete, if any"`
	CursorAdjust  int    `desc:"cursor adjustment if cursor should be placed in a location other than at end of newText"`
}

// MatchFunc is the function called to get the list of possible completions
// and also determines the correct seed based on the text passed as a parameter of CompletionFunc
type MatchFunc func(data interface{}, text string, pos token.Position) MatchData

// EditFunc is passed the current text and the selected completion for text editing.
// Allows for other editing, e.g. adding "()" or adding "/", etc.
type EditFunc func(data interface{}, text string, cursorPos int, completion Completion, seed string) EditData

// MatchSeed returns a list of matches given a list of string possibilities and a seed.
// The seed is basically a prefix.
func MatchSeedString(completions []string, seed string) (matches []string) {

	matches = completions[0:0]
	start := -1
	end := -1

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
		if end > -1 {
			break
		}
		var noCase = true
		if HasUpperCase(seed) {
			noCase = false
		}
		text := s
		if noCase {
			text = strings.ToLower(s)
		}
		if start == -1 {
			if strings.HasPrefix(text, seed) {
				start = i // first match in sorted list
			}
			continue
		}
		if start > -1 {
			if strings.HasPrefix(text, seed) == false {
				end = i
			}
		}
	}
	if start > -1 && end == -1 { // everything possible was a match!
		end = len(completions)
	}

	//fmt.Printf("match start: %d, end: %d", start, end)
	if start > -1 && end > -1 {
		matches = completions[start:end]
	}
	return matches
}

// MatchSeedCompletion returns a list of matching completion structs given a list of possibilities and a seed.
// The seed is basically a prefix.
func MatchSeedCompletion(completions []Completion, seed string) (matches []Completion) {
	matches = completions[0:0]
	start := -1
	end := -1

	// fast case insensitive sort from Andrew Kutz
	less := func(i, j int) bool {
		return sortfold.CompareFold(completions[i].Text, completions[j].Text) < 0
	}
	sort.Slice(completions, less)

	if len(seed) == 0 {
		matches = completions
		return matches
	}

	var noCase = true
	if HasUpperCase(seed) {
		noCase = false
	}
	for i, c := range completions {
		if end > -1 {
			break
		}
		text := c.Text
		if noCase {
			text = strings.ToLower(text)
		}
		if start == -1 {
			if strings.HasPrefix(text, seed) {
				start = i // first match in sorted list
			}
			continue
		}
		if start > -1 {
			if strings.HasPrefix(text, seed) == false {
				end = i
			}
		}
	}
	if start > -1 && end == -1 { // everything possible was a match!
		end = len(completions)
	}

	//fmt.Printf("match start: %d, end: %d", start, end)
	if start > -1 && end > -1 {
		matches = completions[start:end]
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
		if unicode.IsSpace(r) || r == '.' || r == '(' || r == '[' || r == '{' {
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

// EditWord replaces the completion seed and any text up to the next whitespace with completion
func EditWord(text string, cp int, completion string, seed string) (ed EditData) {
	s2 := string(text[cp:])

	var fd = 0 // number of characters past seed in word to be deleted (forward delete)]
	var r rune
	if len(s2) > 0 {
		for fd, r = range s2 {
			if unicode.IsSpace(r) {
				break
			}
		}
	}
	if fd == len(s2)-1 { // last word case
		fd += 1
	}
	ed.NewText = completion
	ed.ForwardDelete = fd + len(seed)
	ed.CursorAdjust = 0
	return ed
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
