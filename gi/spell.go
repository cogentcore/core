// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"goki.dev/glop/dirs"
	"goki.dev/goosi/events"
	"goki.dev/spell"
)

// InitSpell tries to load the saved fuzzy.spell model.
// If unsuccessful tries to create a new model from a text file used as input
func InitSpell() error {
	if spell.Initialized() {
		return nil
	}
	err := OpenSpellModel()
	if err != nil {
		err = spell.OpenDefault()
		if err != nil {
			slog.Error(err.Error())
		}
	}
	return nil
}

// OpenSpellModel loads a saved spelling model
func OpenSpellModel() error {
	pdir := GoGiDataDir()
	openpath := filepath.Join(pdir, "spell_en_us.json")
	err := spell.Open(openpath)
	if err != nil {
		slog.Error("opening spelling dictionary", "path", openpath, "err", err)
	}
	return err
}

// NewSpellModelFromText builds a NEW spelling model from text
func NewSpellModelFromText() error {
	bigdatapath, err := dirs.GoSrcDir("goki.dev/pi/v2/spell")
	if err != nil {
		slog.Error("getting path to corpus directory", "err", err)
		return err
	}

	bigdatafile := filepath.Join(bigdatapath, "big.txt")
	file, err := os.Open(bigdatafile)
	if err != nil {
		// TODO(kai/snack)
		slog.Error("Could not open corpus file. This file is used to create the spelling model", "file", bigdatafile, "err", err)
		ErrorDialog(nil, err)
		// nil).AddTitle("Corpus File Not Found").AddText("You can build a spelling model to check against by clicking the \"Train\" button and selecting text files to train on.").Ok().Run()
		return err
	}

	err = spell.Train(*file, true) // true - create a NEW spelling model
	if err != nil {
		slog.Error("Failed building model from corpus file", "err", err)
		return err
	}

	SaveSpellModel()

	return nil
}

// AddToSpellModel trains on additional text - extends model
func AddToSpellModel(filepath string) error {
	InitSpell() // make sure model is initialized
	file, err := os.Open(filepath)
	if err != nil {
		log.Printf("Could not open text file selected for training: %v.\n", err)
		return err
	}

	err = spell.Train(*file, false) // false - append rather than create new
	if err != nil {
		log.Printf("Failed appending to spell model: %v.\n", err)
		return err
	}
	return nil
}

// SaveSpellModel saves the spelling model which includes the data and parameters
func SaveSpellModel() error {
	pdir := GoGiDataDir()
	path := filepath.Join(pdir, "spell_en_us.json")
	err := spell.Save(path)
	if err != nil {
		log.Printf("Could not save spelling model to file: %v.\n", err)
	}
	return err
}

////////////////////////////////////////////////////////////////////////////////////////
// Spell

// Spell
type Spell struct { //gti:add -setters
	// line number in source that spelling is operating on, if relevant
	SrcLn int

	// character position in source that spelling is operating on (start of word to be corrected)
	SrcCh int

	// list of suggested corrections
	Suggest []string

	// word being checked
	Word string `set:"-"`

	// last word learned -- can be undone -- stored in lowercase format
	LastLearned string `set:"-"`

	// the user's correction selection
	Correction string `set:"-"`

	// the event listeners for the spell (it sends Select events)
	Listeners events.Listeners `set:"-" view:"-"`

	// Stage is the [PopupStage] associated with the [Spell]
	Stage *Stage

	ShowMu sync.Mutex `set:"-"`

	// the scene where the current popup menu is presented
	// Sc *Scene ` set:"-"`
}

// SpellSignals are signals that are sent by Spell
type SpellSignals int32 //enums:enum -trim-prefix Spell

const (
	// SpellSelect means the user chose one of the possible corrections
	SpellSelect SpellSignals = iota

	// SpellIgnore signals the user chose ignore so clear the tag
	SpellIgnore
)

// NewSpell returns a new [Spell]
func NewSpell() *Spell {
	return &Spell{}
}

// CheckWord checks the model to determine if the word is known.
// automatically checks the Ignore list first.
func (sp *Spell) CheckWord(word string) ([]string, bool) {
	return spell.CheckWord(word)
}

// SetWord sets the word to spell and other associated info
func (sp *Spell) SetWord(word string, sugs []string, srcLn, srcCh int) *Spell {
	sp.Word = word
	sp.Suggest = sugs
	sp.SrcLn = srcLn
	sp.SrcCh = srcCh
	return sp
}

// Show is the main call for listing spelling corrections.
// Calls ShowNow which builds the correction popup menu
// Similar to completion.Show but does not use a timer
// Displays popup immediately for any unknown word
func (sp *Spell) Show(text string, ctx Widget, pos image.Point) {
	if sp.Stage != nil {
		sp.Cancel()
	}
	sp.ShowNow(text, ctx, pos)
}

// ShowNow actually builds the correction popup menu
func (sp *Spell) ShowNow(word string, ctx Widget, pos image.Point) {
	if sp.Stage != nil {
		sp.Cancel()
	}
	sp.ShowMu.Lock()
	defer sp.ShowMu.Unlock()

	sc := NewScene(ctx.Name() + "-spell")
	MenuSceneConfigStyles(sc)
	sp.Stage = NewPopupStage(CompleterStage, sc, ctx).SetPos(pos)

	if sp.IsLastLearned(word) {
		NewButton(sc).SetText("unlearn").SetTooltip("unlearn the last learned word").
			OnClick(func(e events.Event) {
				sp.UnLearnLast()
			})
	} else {
		count := len(sp.Suggest)
		if count == 1 && sp.Suggest[0] == word {
			return
		}
		if count == 0 {
			NewButton(sc).SetText("no suggestion")
		} else {
			for i := 0; i < count; i++ {
				text := sp.Suggest[i]
				NewButton(sc).SetText(text).OnClick(func(e events.Event) {
					sp.Spell(text)
				})
			}
		}
		NewSeparator(sc)
		NewButton(sc).SetText("learn").OnClick(func(e events.Event) {
			sp.LearnWord()
		})
		NewButton(sc).SetText("ignore").OnClick(func(e events.Event) {
			sp.IgnoreWord()
		})
	}
	sp.Stage.RunPopup()
}

// Spell sends a Select event to Listeners indicating that the user has made a
// selection from the list of possible corrections
func (sp *Spell) Spell(s string) {
	sp.Cancel()
	sp.Correction = s
	sp.Listeners.Call(&events.Base{Typ: events.Select})
}

// OnSelect registers given listener function for Select events on Value.
// This is the primary notification event for all Complete elements.
func (sp *Spell) OnSelect(fun func(e events.Event)) {
	sp.On(events.Select, fun)
}

// On adds an event listener function for the given event type
func (sp *Spell) On(etype events.Types, fun func(e events.Event)) {
	sp.Listeners.Add(etype, fun)
}

// LearnWord gets the misspelled/unknown word and passes to LearnWord
func (sp *Spell) LearnWord() {
	sp.LastLearned = strings.ToLower(sp.Word)
	spell.LearnWord(sp.Word)
}

// IsLastLearned returns true if given word was the last one learned
func (sp *Spell) IsLastLearned(wrd string) bool {
	lword := strings.ToLower(wrd)
	return lword == sp.LastLearned
}

// UnLearnLast unlearns the last learned word -- in case accidental
func (sp *Spell) UnLearnLast() {
	if sp.LastLearned == "" {
		slog.Error("spell.UnLearnLast: no last learned word")
		return
	}
	lword := sp.LastLearned
	sp.LastLearned = ""
	spell.UnLearnWord(lword)
}

// IgnoreWord adds the word to the ignore list
func (sp *Spell) IgnoreWord() {
	spell.IgnoreWord(sp.Word)
}

// Cancel cancels any pending spell correction.
// call when new events nullify prior correction.
// returns true if canceled
func (sp *Spell) Cancel() bool {
	if sp.Stage == nil {
		return false
	}
	st := sp.Stage
	sp.Stage = nil
	st.ClosePopup()
	return true
}
