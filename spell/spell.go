// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
	Package spell provides functions for spell check and correction
*/
package spell

import (
	"bufio"
	"bytes"
	"errors"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/sajari/fuzzy"
)

type Edit struct {
	NewText string `desc:"spelling correction text after special edits if needed"`
}

///////////////////////////////////////////////////////////////////////////////
//  spell check returning suggestions using github.com/sajari/fuzzy

// EditFunc is passed the current word and the selected correction for text editing.
type EditFunc func(data interface{}, new string, old string) Edit

var inited bool
var model *fuzzy.Model
var Ignore []string

// Initialized returns true if the model has been loaded or created anew
func Initialized() bool {
	return inited
}

// Load loads the saved model stored in json format
func Load(path string) error {
	var err error
	model, err = fuzzy.Load(path)
	if err == nil {
		inited = true
	}
	return err
}

// LoadDefault loads the default spelling file.
// Todo: need different languages obviously!
func LoadDefault() error {
	defb, err := Asset("spell_en_us_plain.json")
	if err != nil {
		return err
	}
	model, err = fuzzy.FromReader(bytes.NewBuffer(defb))
	return err
}

// Save saves the spelling model which includes the data and parameters
func Save(filename string) error {
	if model == nil {
		return nil
	}
	return model.Save(filename)
}

func Train(file os.File, new bool) (err error) {
	var out []string

	reader := bufio.NewReader(&file)
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)
	// Count the words.
	count := 0
	for scanner.Scan() {
		exp, _ := regexp.Compile("[a-zA-Z]+")
		words := exp.FindAll([]byte(scanner.Text()), -1)
		for _, word := range words {
			if len(word) > 1 {
				out = append(out, strings.ToLower(string(word)))
				count++
			}
		}
	}

	if err = scanner.Err(); err != nil {
		log.Println(err)
		return err
	}
	if new {
		model = fuzzy.NewModel()
	}
	model.Train(out)
	inited = true
	return err
}

// CheckWord checks a single word and returns suggestions if word is unknown
// Programs should call gi.CheckWord - all program calls should be done through that single API
func CheckWord(w string) (suggests []string, known bool, err error) {
	if model == nil {
		err = errors.New("Model not initialized")
		return suggests, false, err
	}

	known = false
	w = strings.Trim(w, "`'*.,?[]():;")
	w = strings.ToLower(w)
	suggests = model.SpellCheckSuggestions(w, 10)
	if suggests == nil {
		return nil, known, err // known is false
	}
	if len(suggests) > 0 && suggests[0] == w {
		known = true
	}
	return suggests, known, err
}

// LearnWord adds a single word to the corpus
func LearnWord(word string) {
	model.TrainWord(strings.ToLower(word))
}

// Complete finds possible completions based on the prefix s
func Complete(s string) (result []string, err error) {
	if model == nil {
		return result, errors.New("Model is nil")
	}
	result, err = model.Autocomplete(s)
	return result, err
}

// CorrectText replaces the old unknown word with the new word chosen from the list of corrections
// delta is the change in cursor position (cp).
func CorrectText(old string, new string) (ed Edit) {
	// do what is possible to keep the casing of old string
	oldlc := strings.ToLower(old)
	min := len(old)
	if len(new) < len(old) {
		min = len(new)
	}
	var new2 []byte
	var i int
	for i = 0; i < min; i++ {
		if oldlc[i] != new[i] {
			break
		}
		new2 = append(new2, byte(old[i]))
	}
	for j := i; j < len(new); j++ {
		new2 = append(new2, byte(new[j]))
	}
	ed.NewText = string(new2)
	return ed
}

// IgnoreWord adds the word to the Ignore list
func IgnoreWord(word string) {
	Ignore = append(Ignore, word)
}

// DoIgnore returns true if the word is found in the Ignore list
func DoIgnore(word string) bool {
	for _, w := range Ignore {
		if w == word {
			return true
		}
	}
	return false
}
