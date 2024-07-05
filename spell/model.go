// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// this code is adapted from: https://github.com/sajari/fuzzy
// https://www.sajari.com/
// Most of which seems to have been written by Hamish @sajari
// it does not have a copyright notice in the code itself but does have
// an MIT license file.
//
// key change is to ignore counts and just use a flat Dict dictionary
// list of words.

package spell

import (
	"strings"
	"sync"

	"golang.org/x/exp/maps"
)

// Model is the full data model
type Model struct {
	// list of all words, combining Base and User dictionaries
	Dict Dict

	// user dictionary of additional words
	UserDict Dict

	// words to ignore for this session
	Ignore Dict

	// map of misspelled word to potential correct spellings
	Suggest map[string][]string

	// depth of edits to include in Suggest map (2 is only sensible value)
	Depth int

	sync.RWMutex
}

// Create and initialise a new model
func NewModel() *Model {
	md := new(Model)
	return md.Init()
}

func (md *Model) Init() *Model {
	md.Suggest = make(map[string][]string)
	md.Ignore = make(Dict)
	md.Depth = 2
	return md
}

func (md *Model) SetDicts(base, user Dict) {
	md.Dict = base
	md.UserDict = user
	maps.Copy(md.Dict, md.UserDict)
	go md.addSuggestionsForWords(md.Dict.List())
}

// addSuggestionsForWords
func (md *Model) addSuggestionsForWords(terms []string) {
	md.Lock()
	// st := time.Now()
	for _, term := range terms {
		md.createSuggestKeys(term)
	}
	// fmt.Println("train took:", time.Since(st)) // about 500 msec for 32k words, 5 sec for 235k
	md.Unlock()
}

// AddWord adds a new word to user dictionary,
// and generates new suggestions for it
func (md *Model) AddWord(term string) {
	md.Lock()
	defer md.Unlock()
	if md.Dict.Exists(term) {
		return
	}
	md.UserDict.Add(term)
	md.Dict.Add(term)
	md.createSuggestKeys(term)
}

// Delete removes given word from dictionary -- undoes learning
func (md *Model) Delete(term string) {
	md.Lock()
	edits := md.EditsMulti(term, 1)
	for _, edit := range edits {
		sug := md.Suggest[edit]
		ns := len(sug)
		for i := ns - 1; i >= 0; i-- {
			hit := sug[i]
			if hit == term {
				sug = append(sug[:i], sug[i+1:]...)
			}
		}
		if len(sug) == 0 {
			delete(md.Suggest, edit)
		} else {
			md.Suggest[edit] = sug
		}
	}
	delete(md.Dict, term)
	md.Unlock()
}

// For a given term, create the partially deleted lookup keys
func (md *Model) createSuggestKeys(term string) {
	edits := md.EditsMulti(term, md.Depth)
	for _, edit := range edits {
		skip := false
		for _, hit := range md.Suggest[edit] {
			if hit == term {
				// Already know about this one
				skip = true
				continue
			}
		}
		if !skip && len(edit) > 1 {
			md.Suggest[edit] = append(md.Suggest[edit], term)
		}
	}
}

// Edits at any depth for a given term. The depth of the model is used
func (md *Model) EditsMulti(term string, depth int) []string {
	edits := Edits1(term)
	for {
		depth--
		if depth <= 0 {
			break
		}
		for _, edit := range edits {
			edits2 := Edits1(edit)
			for _, edit2 := range edits2 {
				edits = append(edits, edit2)
			}
		}
	}
	return edits
}

type Pair struct {
	str1 string
	str2 string
}

// Edits1 creates a set of terms that are 1 char delete from the input term
func Edits1(word string) []string {

	splits := []Pair{}
	for i := 0; i <= len(word); i++ {
		splits = append(splits, Pair{word[:i], word[i:]})
	}

	total_set := []string{}
	for _, elem := range splits {

		//deletion
		if len(elem.str2) > 0 {
			total_set = append(total_set, elem.str1+elem.str2[1:])
		} else {
			total_set = append(total_set, elem.str1)
		}

	}

	// Special case ending in "ies" or "ys"
	if strings.HasSuffix(word, "ies") {
		total_set = append(total_set, word[:len(word)-3]+"ys")
	}
	if strings.HasSuffix(word, "ys") {
		total_set = append(total_set, word[:len(word)-2]+"ies")
	}

	return total_set
}

// For a given input term, suggest some alternatives.
// if the input is in the dictionary, it will be the only item
// returned.
func (md *Model) suggestPotential(input string) []string {
	input = strings.ToLower(input)

	// 0 - If this is a dictionary term we're all good, no need to go further
	if md.Dict.Exists(input) {
		return []string{input}
	}

	ss := make(Dict)
	var sord []string

	// 1 - See if the input matches a "suggest" key
	if sugg, ok := md.Suggest[input]; ok {
		for _, pot := range sugg {
			if !ss.Exists(pot) {
				sord = append(sord, pot)
				ss.Add(pot)
			}
		}
	}

	// 2 - See if edit1 matches input
	edits := md.EditsMulti(input, md.Depth)
	got := false
	for _, edit := range edits {
		if len(edit) > 2 && md.Dict.Exists(edit) {
			got = true
			if !ss.Exists(edit) {
				sord = append(sord, edit)
				ss.Add(edit)
			}
		}
	}

	if got {
		return sord
	}

	// 3 - No hits on edit1 distance, look for transposes and replaces
	// Note: these are more complex, we need to check the guesses
	// more thoroughly, e.g. levals=[valves] in a raw sense, which
	// is incorrect
	for _, edit := range edits {
		if sugg, ok := md.Suggest[edit]; ok {
			// Is this a real transpose or replace?
			for _, pot := range sugg {
				lev := Levenshtein(&input, &pot)
				if lev <= md.Depth+1 { // The +1 doesn't seem to impact speed, but has greater coverage when the depth is not sufficient to make suggestions
					if !ss.Exists(pot) {
						sord = append(sord, pot)
						ss.Add(pot)
					}
				}
			}
		}
	}
	return sord
}

// Return the most likely corrections in order from best to worst
func (md *Model) Suggestions(input string, n int) []string {
	md.RLock()
	suggestions := md.suggestPotential(input)
	md.RUnlock()
	return suggestions
}

// Calculate the Levenshtein distance between two strings
func Levenshtein(a, b *string) int {
	la := len(*a)
	lb := len(*b)
	d := make([]int, la+1)
	var lastdiag, olddiag, temp int

	for i := 1; i <= la; i++ {
		d[i] = i
	}
	for i := 1; i <= lb; i++ {
		d[0] = i
		lastdiag = i - 1
		for j := 1; j <= la; j++ {
			olddiag = d[j]
			min := d[j] + 1
			if (d[j-1] + 1) < min {
				min = d[j-1] + 1
			}
			if (*a)[j-1] == (*b)[i-1] {
				temp = 0
			} else {
				temp = 1
			}
			if (lastdiag + temp) < min {
				min = lastdiag + temp
			}
			d[j] = min
			lastdiag = olddiag
		}
	}
	return d[la]
}
