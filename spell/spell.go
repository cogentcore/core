// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
	Package spell provides functions for spell check and correction
*/

package spell

import (
	"bufio"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/sajari/fuzzy"
)

///////////////////////////////////////////////////////////////////////////////
//  spell check returning suggestions using github.com/sajari/fuzzy

var inited bool
var model *fuzzy.Model

func Initialized() bool {
	return inited
}

// Load Model loads the save model stored in json format
func LoadModel(b []byte) bool {
	model = fuzzy.NewModel()
	err := json.Unmarshal(b, model)
	if err == nil {
		model.SetThreshold(1)
		inited = true
		return true
	} else {
		log.Printf("Failed loading model from saved json: %v.\n", err)
		return false
	}
}

// ModelFromCorpus builds a new fuzzy.model from a text file
func ModelFromCorpus(file os.File) error {
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

	model.Train(out)
	model.SetThreshold(1)
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
	return suggests[1:], known, err
}

// LearnWord adds a single word to the corpus
func LearnWord(word string) {
	model.TrainWord(word)
}

// SaveModel saves the spelling model which includes the data and parameters
func SaveModel(path string) error {
	if model != nil {
		b, err := json.MarshalIndent(model, "", "  ")
		if err != nil {
			log.Println(err) // unlikely
			return err
		}
		err = ioutil.WriteFile(path, b, 0644)
		if err != nil {
			log.Printf("Could not save spelling model to file: %v.\n", err)
		}
		return err
	}
	return nil
}

//func Complete(s string) []string {
//	if model == nil {
//		Init()
//	}
//	results, err := model.Autocomplete(s)
//	if err == nil {
//		fmt.Println(results)
//	}
//	return results
//}
