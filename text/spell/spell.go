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

type SpellData struct {
	// UserFile is path to user's dictionary where learned words go
	UserFile string

	model     *Model
	openTime  time.Time    // ModTime() on file
	learnTime time.Time    // last time when a Learn function was called 	-- last mod to model -- zero if not mod
	mu        sync.RWMutex // we need our own mutex in case of loading a new model
}

// NewSpell opens spell data with given user dictionary file
func NewSpell(userFile string) *SpellData {
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

// OpenUser opens user dictionary of words
func (sp *SpellData) OpenUser() error {
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
	if sp.UserFile == "" {
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

	if sp.model == nil {
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
	if sp == nil {
		return nil
	}
	if sp.UserFile == "" {
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
	if sp.model == nil {
		log.Println("spell.CheckWord: programmer error -- Spelling not initialized!")
		return nil, false
	}
	w := lexer.FirstWordApostrophe(word) // only lookup words
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

// AddWord adds given word to the User dictionary
func (sp *SpellData) AddWord(word string) {
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

// DeleteWord removes word from dictionary, in case accidentally added
func (sp *SpellData) DeleteWord(word string) {
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
	log.Printf("spell.DeleteWord: %s\n", lword)
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

// IgnoreWord adds the word to the Ignore list
func (sp *SpellData) IgnoreWord(word string) {
	word = strings.ToLower(word)
	sp.model.Ignore.Add(word)
}
