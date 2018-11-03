// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
	Package spell provides functions for spell check and correction
*/

package spell

import (
	"bufio"
	"errors"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/sajari/fuzzy"
)

///////////////////////////////////////////////////////////////////////////////
//  spell check returning suggestions using github.com/sajari/fuzzy

// EditFunc is passed the current text and the selected correction for text editing.
type EditFunc func(data interface{}, text string, cursorPos int, correction string, unknown string) (newText string, delta int)

var inited bool
var model *fuzzy.Model

// Initialized returns true if the model has been loaded or created anew
func Initialized() bool {
	return inited
}

// Load loads the saved model stored in json format
func Load(path string) (err error) {
	model, err = fuzzy.Load(path)
	if err == nil {
		inited = true
	}
	return err
}

// Save saves the spelling model which includes the data and parameters
func Save(filename string) error {
	if model == nil {
		return nil
	}
	return model.Save(filename)
}

func Train(file os.File, new bool) error {
	var out []string
	var err error

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

	if err := scanner.Err(); err != nil {
		log.Println(os.Stderr, "reading input: ", err)
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
	suggests = model.SpellCheckSuggestions(w, 2)
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
	model.TrainWord(word)
}

// Complete finds possible completions based on the prefix s
func Complete(s string) (result []string, err error) {
	if model == nil {
		return result, errors.New("Model is nil")
	}
	result, err = model.Autocomplete(s)
	return result, err
}

// EditTex replaces the old unknown word with the new word chosen from the list of corrections
// delta is the change in cursor position (cp).
func CorrectText(text string, cp int, old string, new string) (newText string, delta int) {
	s1 := string(text[0:cp])
	s2 := string(text[cp:])

	// do a replace from end
	last := strings.LastIndex(text, old)
	s1a := s1[0:last]
	s1c := s1[last+len(old) : len(s1)]
	s1new := s1a + new + s1c
	t := s1new + s2
	delta = len(s1new) - len(s1)
	return t, delta
}
