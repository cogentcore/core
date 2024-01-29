// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package complete

import (
	"strings"
	"unicode"

	"cogentcore.org/core/pi/syms"
	"cogentcore.org/core/pi/token"
	"github.com/iancoleman/strcase"
)

// Completion holds one potential completion
type Completion struct {

	// completion text: what will actually be inserted if selected
	Text string

	// label to show the user; only used for menu display if non-empty; otherwise Text is used
	Label string

	// icon name
	Icon string

	// extra information, e.g. detailed description, type, arguments, etc; not currently used in Pi, but is used for tooltips in GUI
	Desc string

	// lang specific or other, e.g. class or type
	Extra map[string]string
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
type MatchFunc func(data any, text string, posLn, posCh int) Matches

// LookupFunc is the function called to get the lookup results for given
// input test and position.
type LookupFunc func(data any, text string, posLn, posCh int) Lookup

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
	return matches
}

// IsSeedMatching returns whether the given lowercase seed matches
// the given completion string.
func IsSeedMatching(lseed string, completion string) bool {
	lc := strings.ToLower(completion)
	if strings.Contains(lc, lseed) {
		return true
	}

	// stripped version of completion
	// (space delimeted with no punctuation)
	cs := strings.Map(func(r rune) rune {
		if unicode.IsPunct(r) {
			return -1
		}
		return r
	}, completion)
	cs = strcase.ToDelimited(cs, ' ')
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

// EditWord replaces the completion seed and any text up to the next whitespace with completion
func EditWord(text string, cp int, completion string, seed string) (ed Edit) {
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

// AddSyms adds given symbols as matches in the given match data
// Scope is e.g., type name (label only)
func AddSyms(sym syms.SymMap, scope string, md *Matches) {
	if len(sym) == 0 {
		return
	}
	sys := sym.Slice(true) // sorted
	for _, sy := range sys {
		if sy.Name[0] == '_' { // internal / import
			continue
		}
		nm := sy.Name
		lbl := sy.Label()
		if sy.Kind.SubCat() == token.NameFunction {
			nm += "()"
		}
		if scope != "" {
			lbl = lbl + " (" + scope + ".)"
		}
		c := Completion{Text: nm, Label: lbl, Icon: sy.Kind.IconName(), Desc: sy.Detail}
		// fmt.Printf("nm: %v  kind: %v  icon: %v\n", nm, sy.Kind, c.Icon)
		md.Matches = append(md.Matches, c)
	}
}

// AddTypeNames adds names from given type as matches in the given match data
// Scope is e.g., type name (label only), and seed is prefix filter for names
func AddTypeNames(typ *syms.Type, scope, seed string, md *Matches) {
	md.Seed = seed
	for _, te := range typ.Els {
		nm := te.Name
		if seed != "" {
			if !strings.HasPrefix(nm, seed) {
				continue
			}
		}
		lbl := nm
		if scope != "" {
			lbl = lbl + " (" + scope + ".)"
		}
		icon := "field" // assume..
		c := Completion{Text: nm, Label: lbl, Icon: icon}
		// fmt.Printf("nm: %v  kind: %v  icon: %v\n", nm, sy.Kind, c.Icon)
		md.Matches = append(md.Matches, c)
	}
	for _, mt := range typ.Meths {
		nm := mt.Name
		if seed != "" {
			if !strings.HasPrefix(nm, seed) {
				continue
			}
		}
		lbl := nm + "(" + mt.ArgString() + ") " + mt.ReturnString()
		if scope != "" {
			lbl = lbl + " (" + scope + ".)"
		}
		nm += "()"
		icon := "method" // assume..
		c := Completion{Text: nm, Label: lbl, Icon: icon}
		// fmt.Printf("nm: %v  kind: %v  icon: %v\n", nm, sy.Kind, c.Icon)
		md.Matches = append(md.Matches, c)
	}
}

// AddSymsPrefix adds subset of symbols that match seed prefix to given match data
func AddSymsPrefix(sym syms.SymMap, scope, seed string, md *Matches) {
	matches := &sym
	if seed != "" {
		matches = &syms.SymMap{}
		md.Seed = seed
		sym.FindNamePrefixRecursive(seed, matches)
	}
	AddSyms(*matches, scope, md)
}
