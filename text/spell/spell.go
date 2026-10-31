// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package spell

import (
	"embed"
	"log"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"cogentcore.org/core/text/parse/lexer"
)

//go:embed dict_en_us
var embedDict embed.FS

// SaveAfterLearnIntervalSecs is number of seconds since
// dict file has been opened / saved
// above which model is saved after learning.
const SaveAfterLearnIntervalSecs = 20

// Spell is the global shared spell
var Spell *SpellData

// SpellData is a spell checker. It uses the operating system spell checker
// when available (see [UseNative]), in which case learned words go to the
// system user dictionary. Otherwise it uses a built-in English dictionary,
// with learned words saved in UserFile.
type SpellData struct {
	// UserFile is path to user's dictionary where learned words go,
	// only for the built-in checker.
	UserFile string

	// native is the operating system spell checker, if available.
	native nativeChecker

	// known caches the words checked with the native checker,
	// with true if spelled correctly.
	known map[string]bool

	// ignore has the lowercase words to ignore for this session,
	// for the native checker.
	ignore Dict

	model     *Model
	openTime  time.Time    // ModTime() on file
	learnTime time.Time    // last time when a Learn function was called 	-- last mod to model -- zero if not mod
	mu        sync.RWMutex // we need our own mutex in case of loading a new model
}

// maxKnownCache is the maximum number of words in the native known cache.
const maxKnownCache = 100_000

// NewSpell returns a new spell checker, using the operating system
// spell checker if available and [UseNative] is true, and otherwise
// the built-in checker with the given user dictionary file.
func NewSpell(userFile string) *SpellData {
	if UseNative && newNative != nil {
		if nc := newNative(); nc != nil {
			return &SpellData{UserFile: userFile, native: nc, known: map[string]bool{}, ignore: Dict{}}
		}
	}
	d, err := OpenDictFS(embedDict, "dict_en_us")
	if err != nil {
		slog.Error(err.Error())
		return nil
	}
	sp := &SpellData{UserFile: userFile}
	sp.ResetLearnTime()
	sp.model = NewModel()
	sp.openTime = time.Date(2024, 06, 30, 00, 00, 00, 0, time.UTC)
	sp.OpenUser()
	sp.model.SetDicts(d, sp.model.UserDict)
	return sp
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

// IsNative returns true if the operating system spell checker is being used.
func (sp *SpellData) IsNative() bool {
	return sp != nil && sp.native != nil
}

// OpenUser opens user dictionary of words, for the built-in checker.
func (sp *SpellData) OpenUser() error {
	if sp.native != nil {
		return nil
	}
	sp.mu.Lock()
	defer sp.mu.Unlock()
	d, err := OpenDict(sp.UserFile)
	if err != nil {
		// slog.Error(err.Error())
		sp.model.UserDict = make(Dict)
		return err
	}
	// note: does not have suggestions for new words
	// future impl will not precompile suggs so it is not worth it
	sp.openTime, err = modTime(sp.UserFile)
	sp.model.UserDict = d
	return err
}

// OpenUserCheck checks if the current user dict file has been modified
// since last open time and re-opens it if so.
func (sp *SpellData) OpenUserCheck() error {
	if sp == nil || sp.native != nil || sp.UserFile == "" {
		return nil
	}
	sp.mu.Lock()
	defer sp.mu.Unlock()
	tm, err := modTime(sp.UserFile)
	if err != nil {
		return err
	}
	if tm.After(sp.openTime) {
		sp.OpenUser()
		sp.openTime = tm
		// log.Printf("opened newer spell file: %s\n", openTime.String())
	}
	return err
}

// SaveUser saves the user dictionary
// note: this will overwrite any existing file; be sure to have opened
// the current file before making any changes.
func (sp *SpellData) SaveUser() error {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	if sp.native != nil || sp.model == nil {
		return nil
	}
	sp.ResetLearnTime()
	err := sp.model.UserDict.Save(sp.UserFile)
	if err == nil {
		sp.openTime, err = modTime(sp.UserFile)
	} else {
		log.Printf("spell.Spell: Error saving file %q: %v\n", sp.UserFile, err)
	}
	return err
}

// SaveUserIfLearn saves the user dictionary
// if learning has occurred since last save / open.
// If no changes also checks if file has been modified and opens it if so.
func (sp *SpellData) SaveUserIfLearn() error {
	if sp == nil || sp.native != nil || sp.UserFile == "" {
		return nil
	}
	if sp.learnTime.IsZero() {
		return sp.OpenUserCheck()
	}
	sp.SaveUser()
	return nil
}

// CheckWord checks a single word and returns suggestions if word is unknown.
// bool is true if word is in the dictionary, false otherwise.
func (sp *SpellData) CheckWord(word string) ([]string, bool) {
	if sp == nil {
		return nil, false
	}
	w := lexer.FirstWordApostrophe(word) // only lookup words
	if sp.native != nil {
		if len(sp.Misspelled([]string{w})) == 0 {
			return nil, true
		}
		return sp.native.suggest(w), false
	}
	if sp.model == nil {
		return nil, false
	}
	orig := w
	w = strings.ToLower(w)

	sp.mu.RLock()
	defer sp.mu.RUnlock()
	if sp.model.Ignore.Exists(w) {
		return nil, true
	}
	suggests := sp.model.Suggestions(w, 10)
	if suggests == nil { // no sug and not known
		return nil, false
	}
	if len(suggests) == 1 && suggests[0] == w {
		return nil, true
	}
	for i, s := range suggests {
		suggests[i] = lexer.MatchCase(orig, s)
	}
	return suggests, false
}

// Misspelled returns the indexes of the misspelled words in the given
// list of words, in increasing order. The words must already be
// isolated, as with [lexer.FirstWordApostrophe].
func (sp *SpellData) Misspelled(words []string) []int {
	if sp == nil {
		return nil
	}
	if sp.native == nil {
		var bad []int
		for i, w := range words {
			if _, known := sp.CheckWord(w); !known {
				bad = append(bad, i)
			}
		}
		return bad
	}
	// Only send the words that are not already cached or ignored
	// to the native checker.
	sp.mu.RLock()
	var check []string
	for _, w := range words {
		if _, ok := sp.known[w]; ok || sp.ignore.Exists(strings.ToLower(w)) {
			continue
		}
		check = append(check, w)
	}
	sp.mu.RUnlock()
	if len(check) > 0 {
		bad := sp.native.misspelled(check)
		sp.mu.Lock()
		if len(sp.known)+len(check) > maxKnownCache {
			clear(sp.known)
		}
		for _, w := range check {
			sp.known[w] = true
		}
		for _, bi := range bad {
			sp.known[check[bi]] = false
		}
		sp.mu.Unlock()
	}
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	var bad []int
	for i, w := range words {
		if sp.ignore.Exists(strings.ToLower(w)) {
			continue
		}
		if known, ok := sp.known[w]; ok && !known {
			bad = append(bad, i)
		}
	}
	return bad
}

// AddWord adds given word to the user dictionary.
// For the native checker, this is the system user dictionary,
// and the word is added with the given case.
func (sp *SpellData) AddWord(word string) {
	if sp.native != nil {
		sp.native.learn(word)
		sp.clearKnown()
		return
	}
	if sp.learnTime.IsZero() {
		sp.OpenUserCheck() // be sure we have latest before learning!
	}
	sp.mu.Lock()
	lword := strings.ToLower(word)
	sp.model.AddWord(lword)
	sp.learnTime = time.Now()
	sint := sp.learnTime.Sub(sp.openTime) / time.Second
	sp.mu.Unlock()

	if sp.UserFile != "" && sint > SaveAfterLearnIntervalSecs {
		sp.SaveUser()
		// log.Printf("spell.LearnWord: saved updated model after %d seconds\n", sint)
	}
}

// DeleteWord removes word from the user dictionary,
// in case accidentally added.
func (sp *SpellData) DeleteWord(word string) {
	if sp.native != nil {
		sp.native.unlearn(word)
		sp.clearKnown()
		return
	}
	if sp.learnTime.IsZero() {
		sp.OpenUserCheck() // be sure we have latest before learning!
	}

	sp.mu.Lock()
	lword := strings.ToLower(word)
	sp.model.Delete(lword)
	sp.learnTime = time.Now()
	sint := sp.learnTime.Sub(sp.openTime) / time.Second
	sp.mu.Unlock()

	if sp.UserFile != "" && sint > SaveAfterLearnIntervalSecs {
		sp.SaveUser()
	}
}

// clearKnown clears the native known word cache.
func (sp *SpellData) clearKnown() {
	sp.mu.Lock()
	clear(sp.known)
	sp.mu.Unlock()
}

/*
// Complete finds possible completions based on the prefix s
func (sp *SpellData) Complete(s string) []string {
	if model == nil {
		log.Println("spell.Complete: programmer error -- Spelling not initialized!")
		OpenDefault() // backup
	}
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	result, _ := model.Autocomplete(s)
	return result
}
*/

// IgnoreWord adds the word to the list of words
// to ignore for this session.
func (sp *SpellData) IgnoreWord(word string) {
	word = strings.ToLower(word)
	sp.mu.Lock()
	defer sp.mu.Unlock()
	if sp.native != nil {
		sp.ignore.Add(word)
		return
	}
	sp.model.Ignore.Add(word)
}
