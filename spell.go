// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/goki/gi/oswin"
	"github.com/goki/gi/spell"
	"github.com/goki/ki/kit"
)

////////////////////////////////////////////////////////////////////////////////////////
// Spell

// TextWord represents one word of the input text - used with fuzzy implementation
type TextWord struct {
	Word     string
	Line     int `desc:"the line number"`
	StartPos int `desc:"the starting character position"`
	EndPos   int `desc:"the ending character position"`
}

var input []TextWord
var inputidx int = 0

// InitSpell tries to load the saved fuzzy.spell model.
// If unsuccessful tries to create a new model from a text file used as input
func InitSpell() error {
	if spell.Initialized() {
		return nil
	}
	err := LoadSpellModel()
	if err != nil {
		// oh well, try creating a new model from corpus
		err = SpellModelFromCorpus()
		if err != nil {
			return err
		}
	}
	return nil
}

// LoadSpellModel loads a saved spelling model
func LoadSpellModel() error {
	pdir := oswin.TheApp.AppPrefsDir()
	openpath := filepath.Join(pdir, "spell_en_us_plain.json")
	err := spell.Load(openpath)
	return err
}

// SpellModelFromCorpus builds a spelling from text
func SpellModelFromCorpus() error {
	bigdatapath, err := kit.GoSrcDir("github.com/goki/gi/spell")
	if err != nil {
		log.Printf("Error getting path to corpus directory: %v.\n", err)
		return err
	}

	bigdatafile := filepath.Join(bigdatapath, "big.txt")
	file, err := os.Open(bigdatafile)
	if err != nil {
		log.Printf("Could not open corpus file: %v. This file is used to create the spelling model.\n", err)
		return err
	}

	err = spell.ModelFromCorpus(*file)
	if err != nil {
		log.Printf("Failed building model from corpus file: %v.\n", err)
		return err
	}
	return nil
}

// SaveSpellModel saves the spelling model which includes the data and parameters
func SaveSpellModel() error {
	pdir := oswin.TheApp.AppPrefsDir()
	path := filepath.Join(pdir, "spell_en_us_plain.json")
	err := spell.Save(path)
	if err != nil {
		log.Printf("Could not save spelling model to file: %v.\n", err)
	}
	return err
}

// InitNewSpellCheck builds the input list, i.e. the words to check
func InitNewSpellCheck(text []byte) {
	input = input[:0] // clear past input
	inputidx = 0
	TextToWords(text)
}

// TextToWords generates a slice of words from text
// removes various non-word input, trims symbols, etc
func TextToWords(text []byte) {
	notwordchar, err := regexp.Compile(`[^0-9A-Za-z]`)
	if err != nil {
		panic(err)
	}
	allnum, err := regexp.Compile(`^[0-9]*$`)
	if err != nil {
		panic(err)
	}
	wordbounds, err := regexp.Compile(`\b`)
	if err != nil {
		panic(err)
	}

	textstr := string(text)

	var words []TextWord
	for l, line := range strings.Split(textstr, "\n") {
		line = notwordchar.ReplaceAllString(line, " ")
		bounds := wordbounds.FindAllStringIndex(line, -1)
		words = words[:0] // reset for new line
		splits := strings.Fields(line)
		for i, w := range splits {
			if allnum.MatchString(w) {
				break
			}
			if len(w) > 1 {
				tw := TextWord{Word: w, Line: l, StartPos: bounds[i*2][0], EndPos: bounds[i*2+1][0]}
				words = append(words, tw)
			}
		}
		input = append(input, words...)
	}
	//for _, w := range input {
	//	fmt.Println(w)
	//}
}

// NextUnknownWord returns the next unknown word, i.e. not found in corpus
func NextUnknownWord() (unknown TextWord, suggests []string, err error) {
	var tw TextWord

	for {
		tw = NextWord()
		if tw.Word == "" { // we're done!
			break
		}
		var known = false
		suggests, known, err = spell.CheckWord(tw.Word)
		if !known {
			break
		}
	}
	return tw, suggests, err
}

// NextWord returns the next word of the input words
func NextWord() TextWord {
	tw := TextWord{}
	if inputidx < len(input) {
		tw = input[inputidx]
		inputidx += 1
		return tw
	}
	return tw
}

// CheckWord calls the implementation so the app isn't tied to a particular implementation
func CheckWord(w string) (suggests []string, known bool, err error) {
	return spell.CheckWord(w)
}

// LearnWord calls the implementation so the app isn't tied to a particular implementation
func LearnWord(w string) {
	spell.LearnWord(w)
}
