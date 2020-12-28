// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/goki/gi/oswin"
	"github.com/goki/ki/dirs"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/pi/spell"
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
			log.Println(err)
		}
	}
	return nil
}

// OpenSpellModel loads a saved spelling model
func OpenSpellModel() error {
	pdir := oswin.TheApp.GoGiPrefsDir()
	openpath := filepath.Join(pdir, "spell_en_us.json")
	err := spell.Open(openpath)
	return err
}

// NewSpellModelFromText builds a NEW spelling model from text
func NewSpellModelFromText() error {
	bigdatapath, err := dirs.GoSrcDir("github.com/goki/pi/spell")
	if err != nil {
		log.Printf("Error getting path to corpus directory: %v.\n", err)
		return err
	}

	bigdatafile := filepath.Join(bigdatapath, "big.txt")
	file, err := os.Open(bigdatafile)
	if err != nil {
		log.Printf("Could not open corpus file: %v. This file is used to create the spelling model.\n", err)
		PromptDialog(nil, DlgOpts{Title: "Corpus File Not Found", Prompt: "You can build a spelling model to check against by clicking the \"Train\" button and selecting text files to train on."}, AddOk, NoCancel, nil, nil)
		return err
	}

	err = spell.Train(*file, true) // true - create a NEW spelling model
	if err != nil {
		log.Printf("Failed building model from corpus file: %v.\n", err)
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
	pdir := oswin.TheApp.GoGiPrefsDir()
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
type Spell struct {
	ki.Node
	SrcLn       int         `desc:"line number in source that spelling is operating on, if relevant"`
	SrcCh       int         `desc:"character position in source that spelling is operating on (start of word to be corrected)"`
	Suggest     []string    `desc:"list of suggested corrections"`
	Word        string      `desc:"word being checked"`
	LastLearned string      `desc:"last word learned -- can be undone -- stored in lowercase format"`
	SpellSig    ki.Signal   `json:"-" xml:"-" view:"-" desc:"signal for Spell -- see SpellSignals for the types"`
	Correction  string      `desc:"the user's correction selection'"`
	Vp          *Viewport2D `desc:"the viewport where the current popup menu is presented"`
}

var KiT_Spell = kit.Types.AddType(&Spell{}, nil)

func (sc *Spell) Disconnect() {
	sc.Node.Disconnect()
	sc.SpellSig.DisconnectAll()
}

// SpellSignals are signals that are sent by Spell
type SpellSignals int64

const (
	// SpellSelect means the user chose one of the possible corrections
	SpellSelect SpellSignals = iota

	// SpellIgnore signals the user chose ignore so clear the tag
	SpellIgnore
)

//go:generate stringer -type=SpellSignals

// CheckWord checks the model to determine if the word is known.
// automatically checks the Ignore list first.
func (sc *Spell) CheckWord(word string) ([]string, bool) {
	return spell.CheckWord(word)
}

// SetWord sets the word to spell and other associated info
func (sc *Spell) SetWord(word string, sugs []string, srcLn, srcCh int) {
	sc.Word = word
	sc.Suggest = sugs
	sc.SrcLn = srcLn
	sc.SrcCh = srcCh
}

// Show is the main call for listing spelling corrections.
// Calls ShowNow which builds the correction popup menu
// Similar to completion.Show but does not use a timer
// Displays popup immediately for any unknown word
func (sc *Spell) Show(text string, vp *Viewport2D, pt image.Point) {
	if vp == nil || vp.Win == nil {
		return
	}
	cpop := vp.Win.CurPopup()
	if PopupIsCorrector(cpop) {
		vp.Win.SetDelPopup(cpop)
	}
	sc.ShowNow(text, vp, pt)
}

// ShowNow actually builds the correction popup menu
func (sc *Spell) ShowNow(word string, vp *Viewport2D, pt image.Point) {
	if vp == nil || vp.Win == nil {
		return
	}
	cpop := vp.Win.CurPopup()
	if PopupIsCorrector(cpop) {
		vp.Win.SetDelPopup(cpop)
	}

	var m Menu
	var text string
	if sc.IsLastLearned(word) {
		text = "unlearn"
		m.AddAction(ActOpts{Label: text, Data: text},
			sc, func(recv, send ki.Ki, sig int64, data interface{}) {
				scf := recv.Embed(KiT_Spell).(*Spell)
				scf.UnLearnLast()
			})
	} else {
		count := len(sc.Suggest)
		if count == 1 && sc.Suggest[0] == word {
			return
		}
		if count == 0 {
			text = "no suggestion"
			m.AddAction(ActOpts{Label: text, Data: text},
				sc, func(recv, send ki.Ki, sig int64, data interface{}) {
				})
		} else {
			for i := 0; i < count; i++ {
				text = sc.Suggest[i]
				m.AddAction(ActOpts{Label: text, Data: text},
					sc, func(recv, send ki.Ki, sig int64, data interface{}) {
						scf := recv.Embed(KiT_Spell).(*Spell)
						scf.Spell(data.(string))
					})
			}
		}
		m.AddSeparator("")
		text = "learn"
		m.AddAction(ActOpts{Label: text, Data: text},
			sc, func(recv, send ki.Ki, sig int64, data interface{}) {
				scf := recv.Embed(KiT_Spell).(*Spell)
				scf.LearnWord()
			})
		text = "ignore"
		m.AddAction(ActOpts{Label: text, Data: text},
			sc, func(recv, send ki.Ki, sig int64, data interface{}) {
				scf := recv.Embed(KiT_Spell).(*Spell)
				scf.IgnoreWord()
			})
	}
	sc.Vp = vp
	pvp := PopupMenu(m, pt.X, pt.Y, vp, "tf-spellcheck-menu")
	pvp.SetFlag(int(VpFlagCorrector))
	pvp.Child(0).SetProp("no-focus-name", true) // disable name focusing -- grabs key events in popup instead of in textfield!
}

// Spell emits a signal to let subscribers know that the user has made a
// selection from the list of possible corrections
func (sc *Spell) Spell(s string) {
	sc.Cancel()
	sc.Correction = s
	sc.SpellSig.Emit(sc.This(), int64(SpellSelect), s)
}

// KeyInput is the opportunity for the spelling correction popup to act on specific key inputs
func (sc *Spell) KeyInput(kf KeyFuns) bool { // true - caller should set key processed
	switch kf {
	case KeyFunMoveDown:
		return true
	case KeyFunMoveUp:
		return true
	}
	return false
}

// LearnWord gets the misspelled/unknown word and passes to LearnWord
func (sc *Spell) LearnWord() {
	sc.LastLearned = strings.ToLower(sc.Word)
	spell.LearnWord(sc.Word)
	sc.SpellSig.Emit(sc.This(), int64(SpellSelect), sc.Word)
}

// IsLastLearned returns true if given word was the last one learned
func (sc *Spell) IsLastLearned(wrd string) bool {
	lword := strings.ToLower(wrd)
	return lword == sc.LastLearned
}

// UnLearnLast unlearns the last learned word -- in case accidental
func (sc *Spell) UnLearnLast() {
	if sc.LastLearned == "" {
		log.Println("spell.UnLearnLast: no last learned word")
		return
	}
	lword := sc.LastLearned
	sc.LastLearned = ""
	spell.UnLearnWord(lword)
}

// IgnoreWord adds the word to the ignore list
func (sc *Spell) IgnoreWord() {
	spell.IgnoreWord(sc.Word)
	sc.SpellSig.Emit(sc.This(), int64(SpellIgnore), sc.Word)
}

// Cancel cancels any pending spell correction -- call when new events nullify prior correction
// returns true if canceled
func (sc *Spell) Cancel() bool {
	if sc.Vp == nil || sc.Vp.Win == nil {
		return false
	}
	cpop := sc.Vp.Win.CurPopup()
	did := false
	if PopupIsCorrector(cpop) {
		did = true
		sc.Vp.Win.SetDelPopup(cpop)
	}
	sc.Vp = nil
	return did
}
