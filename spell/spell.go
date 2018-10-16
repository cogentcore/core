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
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/client9/gospell"
	"github.com/client9/gospell/plaintext"
	"github.com/sajari/fuzzy"
)

///////////////////////////////////////////////////////////////////////////////
//  spell check (no suggestions) github.com/client9/gospell

// UnknownWord records one unknown word instance
type UnknownWord struct {
	Filename    string   `desc:"filename only, no path"` // gospell.Diff.Filename
	Path        string   `desc:"path only, no filename"` // gospell.Diff.Path
	Word        string   `desc:"unknown word"`           // gospell.Diff.Original
	Suggestions []string `desc:"a list of suggestions for the unknown word"`
	Text        string   `desc:"full line of text containing unknown word"` // gospell.Diff.Line
	LineNo      int      `desc:"the line number"`                           // gospell.Diff.LineNum
}

// CheckFile makes calls on the gospell package and returns a slice of words not found in dictionary
func CheckFile(fullpath string) ([]UnknownWord, error) {
	var unknowns []UnknownWord

	path, filename := filepath.Split(fullpath)

	// TODO: comment from gospell authoer -  based on OS (Windows vs. Linux)
	//dictPath := flag.String("path", ".:/usr/local/share/hunspell:/usr/share/hunspell", "Search path for dictionaries")
	dictPath := "/usr/local/share/hunspell"

	// TODO : comment from gospell authoer -  based on environment variable settings
	//dicts := flag.String("d", "en_US", "dictionaries to load")
	dicts := "en_US"

	//personalDict := flag.String("p", "", "personal wordlist file")
	personalDict := ""

	affFile := ""
	dicFile := ""
	for _, base := range filepath.SplitList(dictPath) {
		affFile = filepath.Join(base, dicts+".aff")
		dicFile = filepath.Join(base, dicts+".dic")
		//log.Printf("Trying %s", affFile)
		_, err1 := os.Stat(affFile)
		_, err2 := os.Stat(dicFile)
		if err1 == nil && err2 == nil {
			break
		}
		affFile = ""
		dicFile = ""
	}

	if affFile == "" {
		ur := "https://sourceforge.net/projects/hunspell/files/Spelling%20dictionaries/en_US/"
		return unknowns, fmt.Errorf("Unable to load %s. Download en_us.zip from \n\n %v \n\n Unzip into %v", dicts, ur, dictPath)
	}

	log.Printf("Loading %s %s", affFile, dicFile)
	timeStart := time.Now()
	h, err := gospell.NewGoSpell(affFile, dicFile)
	timeEnd := time.Now()

	// note: 10x too slow
	log.Printf("Loaded in %v", timeEnd.Sub(timeStart))
	if err != nil {
		log.Fatalf("%s", err)
	}

	if personalDict != "" {
		raw, err := ioutil.ReadFile(personalDict)
		if err != nil {
			log.Fatalf("Unable to load personal dictionary %s: %s", personalDict, err)
		}
		duplicates, err := h.AddWordList(bytes.NewReader(raw))
		if err != nil {
			log.Fatalf("Unable to process personal dictionary %s: %s", personalDict, err)
		}
		if len(duplicates) > 0 {
			for _, word := range duplicates {
				log.Printf("Word %q in personal dictionary already exists in main dictionary", word)
			}
		}
	}

	// todo: this will read what is on disk - do we want to pass in the bytes rather than the filename?
	raw, err := ioutil.ReadFile(fullpath)
	if err != nil {
		log.Fatalf("Unable to read Stdin: %s", err)
	}
	pt, _ := plaintext.NewIdentity()
	results := gospell.SpellFile(h, pt, raw)
	for _, diff := range results {
		unknown := UnknownWord{
			Filename: filename,
			Path:     path,
			Text:     diff.Line,
			LineNo:   diff.LineNum,
			Word:     diff.Original,
		}
		unknowns = append(unknowns, unknown)
	}
	return unknowns, nil
}

///////////////////////////////////////////////////////////////////////////////
//  spell check returning suggestions using github.com/sajari/fuzzy

var model *fuzzy.Model
var input []string
var inputidx int = 0

func GetCorpus() []string {
	var out []string

	gopath := os.Getenv("GOPATH")
	bigdatapath := gopath + "/src/github.com/goki/gi/spell/big.txt"
	fmt.Println(gopath)

	//file, err := os.Open("/Users/rohrlich/go/src/github.com/goki/gi/spell/big.txt")
	file, err := os.Open(bigdatapath)
	if err != nil {
		fmt.Println(err)
		return out
	}
	reader := bufio.NewReader(file)
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
		fmt.Fprintln(os.Stderr, "reading input:", err)
	}

	return out
}

func InitModel() {
	model = fuzzy.NewModel()
	model.SetThreshold(1)
	words := GetCorpus()
	model.Train(words)
}

// NewSpellCheck builds the input list, i.e. the words to check
func NewSpellCheck(text []byte) {
	input = input[:0] // clear past input
	inputidx = 0
	TextToWords(text)
}

// returns the next word not found in corpus
func NextUnknownWord() (unknown string, suggests []string) {
	var w string
	for {
		w = NextWord()
		if w == "" { // we're done!
			break
		}
		var known = false
		suggests, known = CheckWord(w)
		if suggests != nil && !known {
			for _, s := range suggests {
				fmt.Println(s)
			}
			break
		}
	}
	return w, suggests
}

// returns the next word of the input words
func NextWord() string {
	if inputidx < len(input) {
		w := input[inputidx]
		inputidx += 1
		return w
	}
	return ""
}

// CheckWord checks a single word and returns suggestions if word is unknown
func CheckWord(w string) (suggests []string, known bool) {
	if model == nil {
		InitModel()
	}

	known = false
	w = strings.Trim(w, "`'*.,?[]():;")
	w = strings.ToLower(w)
	suggests = model.SpellCheckSuggestions(w, 2)
	if suggests == nil {
		return nil, known // known is false
	}
	if len(suggests) > 0 && suggests[0] == w {
		known = true
	}
	return suggests[1:], known
}

// LearnWord adds a single word to the corpus
func LearnWord(word string) {
	model.TrainWord(word)
}

// TextToWords generates a slice of words from text
// removes various non-word input, trims symbols, etc
func TextToWords(text []byte) {
	// borrowing code from gospell
	// remove any golang templates
	text = plaintext.StripTemplate(text)

	// extract plain text
	//text = ext.Text(text)

	// do character conversion "smart quotes" to quotes, etc
	// as specified in the Affix file
	//rawstring := gs.InputConversion(raw)

	// zap URLS - takes string
	s := gospell.RemoveURL(string(text))
	// zap file paths
	s = gospell.RemovePath(s)

	// cases like and/or, good-hearted
	s = strings.Replace(s, "/", " ", -1)
	s = strings.Replace(s, "-", " ", -1)
	s = strings.Replace(s, "=", " ", -1)

	var trims []string
	for _, line := range strings.Split(s, "\n") {
		// now get words
		//words := gs.Split(line)
		trims = trims[:0]
		words := strings.Split(line, " ")
		for _, w := range words {
			w = strings.Trim(w, "$`'*.,?[]():;\"")
			if len(w) < 2 {
				break
			}
			trims = append(trims, w)
		}
		input = append(input, trims...)
	}
	//for _, w := range input {
	//	fmt.Println(w)
	//}
}
