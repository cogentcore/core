// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package complete

import (
	"cmp"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"cogentcore.org/core/base/strcase"
	"cogentcore.org/core/icons"
)

// Completion holds one potential completion
type Completion struct {

	// Text is the completion text: what will actually be inserted if selected.
	Text string

	// Label is the label text to show the user. It is only used if
	// non-empty; otherwise Text is used.
	Label string

	// Icon is the icon to render in core for the completion item.
	Icon icons.Icon

	// Desc is extra description information used in tooltips in core.
	Desc string
}

// Completions is a full list (slice) of completion options
type Completions []Completion

// Matches is used for passing completions around.
// contains seed in addition to completions
type Matches struct {

	// the matches based on seed
	Matches Completions

	// seed is the prefix we use to find possible completions
	Seed string
}

// Lookup is used for returning lookup results
type Lookup struct {

	// if non-empty, the result is to view this file (full path)
	Filename string

	// starting line number within file to display
	StLine int

	// ending line number within file
	EdLine int

	// if filename is empty, this is raw text to display for lookup result
	Text []byte
}

// SetFile sets file info
func (lk *Lookup) SetFile(fname string, st, ed int) {
	lk.Filename = fname
	lk.StLine = st
	lk.EdLine = ed
}

// Edit is returned from completion edit function
// to incorporate the selected completion
type Edit struct {

	// completion text after special edits
	NewText string

	// number of runes, past the cursor, to delete, if any
	ForwardDelete int

	// cursor adjustment if cursor should be placed in a location other than at end of newText
	CursorAdjust int
}

// MatchFunc is the function called to get the list of possible completions
// and also determines the correct seed based on the text
// passed as a parameter of CompletionFunc
type MatchFunc func(data any, text string, posLine, posChar int) Matches

// LookupFunc is the function called to get the lookup results for given
// input test and position.
type LookupFunc func(data any, text string, posLine, posChar int) Lookup

// EditFunc is passed the current text and the selected completion for text editing.
// Allows for other editing, e.g. adding "()" or adding "/", etc.
type EditFunc func(data any, text string, cursorPos int, comp Completion, seed string) Edit

// MatchSeedString returns a list of matches given a list of string
// possibilities and a seed. It checks whether different
// transformations of each possible completion contain a lowercase
// version of the seed. It returns nil if there are no matches.
func MatchSeedString(completions []string, seed string) []string {
	if len(seed) == 0 {
		// everything matches
		return completions
	}

	var matches []string
	lseed := strings.ToLower(seed)

	for _, c := range completions {
		if IsSeedMatching(lseed, c) {
			matches = append(matches, c)
		}
	}
	slices.SortStableFunc(matches, func(a, b string) int {
		return cmp.Compare(MatchPrecedence(lseed, a), MatchPrecedence(lseed, b))
	})
	return matches
}

// MatchSeedCompletion returns a list of matches given a list of
// [Completion] possibilities and a seed. It checks whether different
// transformations of each possible completion contain a lowercase
// version of the seed. It returns nil if there are no matches.
func MatchSeedCompletion(completions []Completion, seed string) []Completion {
	if len(seed) == 0 {
		// everything matches
		return completions
	}

	var matches []Completion
	lseed := strings.ToLower(seed)

	for _, c := range completions {
		if IsSeedMatching(lseed, c.Text) {
			matches = append(matches, c)
		}
	}
	slices.SortStableFunc(matches, func(a, b Completion) int {
		return cmp.Compare(MatchPrecedence(lseed, a.Text), MatchPrecedence(lseed, b.Text))
	})
	return matches
}

// IsSeedMatching returns whether the given lowercase seed matches
// the given completion string. It checks whether different
// transformations of the completion contain the lowercase
// version of the seed.
func IsSeedMatching(lseed string, completion string) bool {
	lc := strings.ToLower(completion)
	if strings.Contains(lc, lseed) {
		return true
	}

	// stripped version of completion
	// (space delimeted with no punctuation and symbols)
	cs := strings.Map(func(r rune) rune {
		if unicode.IsPunct(r) || unicode.IsSymbol(r) {
			return -1
		}
		return r
	}, completion)
	cs = strcase.ToWordCase(cs, strcase.WordLowerCase, ' ')
	if strings.Contains(cs, lseed) {
		return true
	}

	// the initials (first letters) of every field
	ci := ""
	csdf := strings.Fields(cs)
	for _, f := range csdf {
		ci += string(f[0])
	}
	return strings.Contains(ci, lseed)
}

// MatchPrecedence returns the sorting precedence of the given
// completion relative to the given lowercase seed. The completion
// is assumed to already match the seed by [IsSeedMatching]. A
// lower return value indicates a higher precedence.
func MatchPrecedence(lseed string, completion string) int {
	lc := strings.ToLower(completion)
	if strings.HasPrefix(lc, lseed) {
		return 0
	}
	if len(lseed) > 0 && strings.HasPrefix(lc, lseed[:1]) {
		return 1
	}
	return 2
}

// SeedSpace returns the text after the last whitespace,
// which is typically used for creating a completion seed string.
func SeedSpace(text string) string {
	return SeedAfter(text, func(r rune) bool {
		return unicode.IsSpace(r)
	})
}

// SeedPath returns the text after the last whitespace and path/filepath
// separator, which is typically used for creating a completion seed string.
func SeedPath(text string) string {
	return SeedAfter(text, func(r rune) bool {
		return unicode.IsSpace(r) || r == '/' || r == filepath.Separator
	})
}

// SeedPath returns the text after the last rune for which the given
// function returns true, which is typically used for creating a completion
// seed string.
func SeedAfter(text string, f func(r rune) bool) string {
	seedStart := 0
	runes := []rune(text)
	for i := len(runes) - 1; i >= 0; i-- {
		r := runes[i]
		if f(r) {
			seedStart = i + 1
			break
		}
	}
	return string(runes[seedStart:])
}

// EditWord replaces the completion seed and any text up to the next whitespace with completion
func EditWord(text string, cursorPos int, completion string, seed string) (ed Edit) {
	s2 := string(text[cursorPos:])

	fd := 0 // number of characters past seed in word to be deleted (forward delete)
	r := rune(0)
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
