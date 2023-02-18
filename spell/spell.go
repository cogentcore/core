// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package spell

import (
	"bufio"
	"bytes"
	"embed"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/goki/pi/lex"
)

//go:embed spell_en_us.json
var content embed.FS

// SaveAfterLearnIntervalSecs is number of seconds since file has been opened / saved
// above which model is saved after learning.
const SaveAfterLearnIntervalSecs = 20

var (
	inited    bool
	spellMu   sync.RWMutex // we need our own mutex in case of loading a new model
	model     *Model
	openTime  time.Time // ModTime() on file
	learnTime time.Time // last time when a Learn function was called -- last mod to model -- zero if not mod
	openFPath string
	Ignore    = map[string]struct{}{}
)

// Initialized returns true if the model has been loaded or created anew
func Initialized() bool {
	return inited
}

// ModTime returns the modification time of given file path
func ModTime(path string) (time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

func ResetLearnTime() {
	learnTime = time.Time{}
}

// Open loads the saved model stored in json format
func Open(path string) error {
	spellMu.Lock()
	defer spellMu.Unlock()

	ResetLearnTime()
	var err error
	openTime, err = ModTime(path)
	if err != nil {
		openFPath = path // save for later, so it will save when learning
		openTime = time.Now()
		return err
	}
	model, err = Load(path)
	if err == nil {
		openFPath = path
		inited = true
	}
	return err
}

// OpenCheck checks if the current file has been modified since last open time
// and re-opens it if so -- call this prior to checking.
func OpenCheck() error {
	if !inited || openFPath == "" {
		return nil
	}
	spellMu.Lock()
	defer spellMu.Unlock()
	tm, err := ModTime(openFPath)
	if err != nil {
		return err
	}
	if tm.After(openTime) {
		model, err = Load(openFPath)
		openTime = tm
		ResetLearnTime()
		// log.Printf("opened newer spell file: %s\n", openTime.String())
	}
	return err
}

// OpenDefault loads the default spelling file.
// TODO: need different languages obviously!
func OpenDefault() error {
	fn := "spell_en_us.json"
	return OpenEmbed(fn)
}

// OpenEmbed loads json-formatted model from embedded data
func OpenEmbed(fname string) error {
	spellMu.Lock()
	defer spellMu.Unlock()

	ResetLearnTime()
	defb, err := content.ReadFile(fname)
	if err != nil {
		return err
	}
	openTime = time.Date(2022, 02, 10, 00, 00, 00, 0, time.UTC)
	inited = true
	model, err = FromReader(bytes.NewBuffer(defb))
	return err
}

// Save saves the spelling model which includes the data and parameters
// note: this will overwrite any existing file -- be sure to have opened
// the current file before making any changes.
func Save(filename string) error {
	spellMu.RLock()
	defer spellMu.RUnlock()

	if model == nil {
		return nil
	}
	ResetLearnTime()
	err := model.Save(filename)
	if err == nil {
		fmt.Printf("spell: %s\n", filename)
		openTime, err = ModTime(filename)
	}
	return err
}

// SaveIfLearn saves the spelling model to file path that was used in last
// Open command, if learning has occurred since last save / open.
// If no changes also checks if file has been modified and opens it if so.
func SaveIfLearn() error {
	if !inited || openFPath == "" {
		return nil
	}
	if learnTime.IsZero() {
		return OpenCheck()
	}
	go Save(openFPath)
	return nil
}

// Train trains the model based on a text file
func Train(file os.File, new bool) (err error) {
	spellMu.Lock()
	defer spellMu.Unlock()

	var out []string

	scanner := bufio.NewScanner(&file)
	// Count the words.
	count := 0
	exp, _ := regexp.Compile("[a-zA-Z]+")
	for scanner.Scan() {
		words := exp.FindAll(scanner.Bytes(), -1)
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
		model = NewModel()
	}
	model.Train(out)
	inited = true
	openTime = time.Now()
	return err
}

// CheckWord checks a single word and returns suggestions if word is unknown
func CheckWord(word string) ([]string, bool) {
	if model == nil {
		log.Println("spell.CheckWord: programmer error -- Spelling not initialized!")
		OpenDefault() // backup
	}
	known := false
	w := lex.FirstWordApostrophe(word) // only lookup words
	orig := w
	w = strings.ToLower(w)

	spellMu.RLock()
	defer spellMu.RUnlock()
	ignore := CheckIgnore(w)
	if ignore {
		return nil, true
	}
	suggests := model.SpellCheckSuggestions(w, 10)
	if suggests == nil { // no sug and not known
		return nil, false
	}
	for i, s := range suggests {
		suggests[i] = lex.MatchCase(orig, s)
	}
	if len(suggests) > 0 && suggests[0] == orig {
		known = true
	}
	return suggests, known
}

// LearnWord adds a single word to the corpus: this is deterministic
// and we set the threshold to 1 to make it learn it immediately.
func LearnWord(word string) {
	if learnTime.IsZero() {
		OpenCheck() // be sure we have latest before learning!
	}

	spellMu.Lock()
	lword := strings.ToLower(word)
	mthr := model.Threshold
	model.Threshold = 1
	model.TrainWord(lword)
	model.Threshold = mthr
	learnTime = time.Now()
	sint := learnTime.Sub(openTime) / time.Second
	spellMu.Unlock()

	if openFPath != "" && sint > SaveAfterLearnIntervalSecs {
		go Save(openFPath)
		// log.Printf("spell.LearnWord: saved updated model after %d seconds\n", sint)
	}
}

// UnLearnWord removes word from dictionary -- in case accidentally added
func UnLearnWord(word string) {
	if learnTime.IsZero() {
		OpenCheck() // be sure we have latest before learning!
	}

	spellMu.Lock()
	lword := strings.ToLower(word)
	model.Delete(lword)
	learnTime = time.Now()
	sint := learnTime.Sub(openTime) / time.Second
	spellMu.Unlock()

	if openFPath != "" && sint > SaveAfterLearnIntervalSecs {
		go Save(openFPath)
	}
	log.Printf("spell.UnLearnLast: unlearned: %s\n", lword)
}

// Complete finds possible completions based on the prefix s
func Complete(s string) []string {
	if model == nil {
		log.Println("spell.Complete: programmer error -- Spelling not initialized!")
		OpenDefault() // backup
	}
	spellMu.RLock()
	defer spellMu.RUnlock()

	result, _ := model.Autocomplete(s)
	return result
}

// IgnoreWord adds the word to the Ignore list
func IgnoreWord(word string) {
	word = strings.ToLower(word)
	Ignore[word] = struct{}{}
}

// CheckIgnore returns true if the word is found in the Ignore list
func CheckIgnore(word string) bool {
	word = strings.ToLower(word)
	_, has := Ignore[word]
	return has
}
