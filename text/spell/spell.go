// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package spell

import (
	"embed"
	"log/slog"
	"maps"
	"strings"
	"sync"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/sharedfile"
	"cogentcore.org/core/text/parse/lexer"
)

//go:embed dict_en_us
var embedDict embed.FS

// Spell is the global shared spell
var Spell *SpellData

// SpellData is a spell checker. It uses the operating system spell checker
// when available (see [UseNative]), in which case learned words go to the
// system user dictionary. Otherwise it uses a built-in English dictionary,
// with learned words saved in UserFile, which is shared by all apps: it is
// read again when another app has changed it, and the words learned and
// deleted are saved right away, with those of the other apps (see
// [sharedfile]).
type SpellData struct {
	// UserFile is path to user's dictionary where learned words go,
	// only for the built-in checker: a list of words, one per line.
	UserFile string

	// native is the operating system spell checker, if available.
	native nativeChecker

	// known caches the words checked with the native checker,
	// with true if spelled correctly.
	known map[string]bool

	// ignore has the lowercase words to ignore for this session,
	// for the native checker.
	ignore Dict

	model *Model

	// user is the shared file of the user dictionary.
	user sharedfile.File[Dict]

	// added and deleted are the words learned and deleted since the user
	// dictionary was last saved, which are applied to the words of its
	// file when it is saved, so that those of other apps are kept.
	added, deleted Dict

	mu sync.RWMutex // we need our own mutex in case of loading a new model
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
	sp := &SpellData{UserFile: userFile, added: Dict{}, deleted: Dict{}}
	sp.model = NewModel()
	uf := sp.userFile()
	errors.Log(uf.Reload())
	sp.model.SetDicts(d, cloneDict(uf.Data))
	return sp
}

// userFile returns the shared file of the user dictionary, with the path
// of [SpellData.UserFile], which can be changed.
func (sp *SpellData) userFile() *sharedfile.File[Dict] {
	sp.user.Path = sp.UserFile
	sp.user.Encode = func(d Dict) ([]byte, error) {
		return []byte(strings.Join(d.List(), "\n")), nil // as Dict.Save
	}
	sp.user.Decode = func(b []byte) (Dict, error) {
		d := Dict{}
		for _, w := range strings.Split(string(b), "\n") {
			if w = strings.TrimSpace(w); w != "" {
				d.Add(w)
			}
		}
		return d, nil
	}
	return &sp.user
}

// cloneDict returns a copy of the given dictionary, which is empty if nil.
func cloneDict(d Dict) Dict {
	c := make(Dict, len(d))
	maps.Copy(c, d)
	return c
}

// IsNative returns true if the operating system spell checker is being used.
func (sp *SpellData) IsNative() bool {
	return sp != nil && sp.native != nil
}

// OpenUser opens user dictionary of words, for the built-in checker,
// if it has changed since it was last read.
func (sp *SpellData) OpenUser() error {
	if sp.native != nil {
		return nil
	}
	sp.mu.Lock()
	defer sp.mu.Unlock()
	return sp.openUser()
}

// openUser is the implementation of [SpellData.OpenUser], called with
// the mutex locked.
func (sp *SpellData) openUser() error {
	uf := sp.userFile()
	err := uf.Reload()
	sp.setUser()
	return err
}

// setUser sets the user dictionary of the model to that of its file, with
// the words learned and deleted since it was last saved. Called with the
// mutex locked.
func (sp *SpellData) setUser() {
	d := cloneDict(sp.user.Data)
	for w := range sp.added {
		d.Add(w)
	}
	for w := range sp.deleted {
		delete(d, w)
	}
	sp.model.setUserDict(d)
}

// OpenUserCheck opens the user dictionary again if its file has been
// changed (by another app) since it was last read or saved.
func (sp *SpellData) OpenUserCheck() error {
	if sp == nil || sp.native != nil || sp.UserFile == "" {
		return nil
	}
	sp.mu.Lock()
	defer sp.mu.Unlock()
	if !sp.userFile().Changed() {
		return nil
	}
	return sp.openUser()
}

// SaveUser saves the words learned and deleted since the user dictionary
// was last saved to its file, with the words of its file, which another
// app may have changed, which are then those of the user dictionary.
func (sp *SpellData) SaveUser() error {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	if sp.native != nil || sp.model == nil || sp.UserFile == "" {
		return nil
	}
	err := sp.userFile().Update(func(d *Dict) {
		if *d == nil {
			*d = Dict{}
		}
		for w := range sp.added {
			d.Add(w)
		}
		for w := range sp.deleted {
			delete(*d, w)
		}
	})
	if err != nil {
		slog.Error("spell: error saving the user dictionary", "file", sp.UserFile, "err", err)
		return err // (the words are saved next time)
	}
	clear(sp.added)
	clear(sp.deleted)
	sp.setUser()
	return nil
}

// SaveUserIfLearn saves the user dictionary if words have been learned or
// deleted since it was last saved, and otherwise opens it again if its file
// has changed.
func (sp *SpellData) SaveUserIfLearn() error {
	if sp == nil || sp.native != nil || sp.UserFile == "" {
		return nil
	}
	sp.mu.RLock()
	changed := len(sp.added) > 0 || len(sp.deleted) > 0
	sp.mu.RUnlock()
	if !changed {
		return sp.OpenUserCheck()
	}
	return sp.SaveUser()
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
		// If the check fails, nothing is cached, so the words are
		// checked again next time.
		if bad, ok := sp.native.misspelled(check); ok {
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
	sp.mu.Lock()
	lword := strings.ToLower(word)
	if !sp.model.Dict.Exists(lword) { // (words of the dictionary are not learned)
		sp.model.AddWord(lword)
		sp.added.Add(lword)
		delete(sp.deleted, lword)
	}
	sp.mu.Unlock()
	sp.SaveUser()
}

// DeleteWord removes word from the user dictionary,
// in case accidentally added.
func (sp *SpellData) DeleteWord(word string) {
	if sp.native != nil {
		sp.native.unlearn(word)
		sp.clearKnown()
		return
	}
	sp.mu.Lock()
	lword := strings.ToLower(word)
	sp.model.Delete(lword)
	sp.deleted.Add(lword)
	delete(sp.added, lword)
	sp.mu.Unlock()
	sp.SaveUser()
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
