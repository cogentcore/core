// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package spell

import (
	"bufio"
	"embed"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"cogentcore.org/core/parse/lexer"
)

//go:embed dict_en_us
var embedDict embed.FS

// SaveAfterLearnIntervalSecs is number of seconds since
// dict file has been opened / saved
// above which model is saved after learning.
const SaveAfterLearnIntervalSecs = 20

// global shared spell
var global *SpellData

// Spell returns the global spelling dictionary
func Spell() *SpellData {
	if global != nil {
		return global
	}
	global = NewSpell()
	return global
}

type SpellData struct {
	// UserFile is path to user's dictionary where learned words go
	UserFile string

	model     *Model
	openTime  time.Time    // ModTime() on file
	learnTime time.Time    // last time when a Learn function was called 	-- last mod to model -- zero if not mod
	mu        sync.RWMutex // we need our own mutex in case of loading a new model
}

// NewSpell opens the default dictionary
func NewSpell() *SpellData {

}

// modTime returns the modification time of given file path
func modTime(path string) (time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

func (sp *SpellData) ResetLearnTime() {
	sp.learnTime = time.Time{}
}

// Open opens dictionary of words and creates
// suggestions based on that.
func Open(path string) error {
	spellMu.Lock()
	defer spellMu.Unlock()
	d, err := OpenDict(path)
	if err != nil {
		// slog.Error(err.Error())
		return err
	}
	openTime, err = ModTime(path)
	InitFromDict(d)
	return err
}

// OpenFS opens dictionary of words and creates
// suggestions based on that.
func OpenFS(fsys fs.FS, path string) error {
	spellMu.Lock()
	defer spellMu.Unlock()
	d, err := OpenDictFS(fsys, path)
	if err != nil {
		slog.Error(err.Error())
		return err
	}
	openTime = time.Date(2024, 06, 30, 00, 00, 00, 0, time.UTC)
	InitFromDict(d)
	return err
}

// InitFromDict initializes model from given dict.
// assumed to be under mutex lock
func InitFromDict(base, user Dict) {
	ResetLearnTime()
	model = NewModel()
	model.SetDicts(base, user)
	inited = true
	go model.Train(d.List())
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
		Open(openFPath)
		openTime = tm
		// log.Printf("opened newer spell file: %s\n", openTime.String())
	}
	return err
}

// OpenDefault loads the default spelling file.
// TODO: need different languages obviously!
func OpenDefault() error {
	fn := "dict_en_us"
	return OpenFS(fn)
}

// OpenEmbed loads dict from embedded file
func OpenEmbed(fname string) error {
	spellMu.Lock()
	defer spellMu.Unlock()
	return OpenFS(content, fname)
}

// Save saves the dictionary
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
		openTime, err = ModTime(filename)
	} else {
		log.Printf("spell.Spell: Error saving file %q: %v\n", filename, err)
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
	w := lexer.FirstWordApostrophe(word) // only lookup words
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
		suggests[i] = lexer.MatchCase(orig, s)
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
