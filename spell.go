// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"go/token"

	"github.com/goki/gi/oswin"
	"github.com/goki/gi/spell"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

////////////////////////////////////////////////////////////////////////////////////////
// Spell

// TextWord represents one word of the input text - used with fuzzy implementation
type TextWord struct {
	Word     string
	Line     int `desc:"the line number"`
	StartPos int `desc:"the starting character position"`
	EndPos   int `desc:"the ending character position"`
}

var input []TextWord
var inputidx int = 0

var notWordChar *regexp.Regexp
var allNum *regexp.Regexp
var wordBounds *regexp.Regexp
var isLetter *regexp.Regexp

// InitSpell tries to load the saved fuzzy.spell model.
// If unsuccessful tries to create a new model from a text file used as input
func InitSpell() error {
	if spell.Initialized() {
		return nil
	}
	err := LoadSpellModel()
	if err != nil {
		// oh well, try creating a new model from corpus
		err = NewSpellModelFromText()
		if err != nil {
			return err
		}
	}

	notWordChar, err = regexp.Compile(`[^0-9A-Za-z]`)
	if err != nil {
		log.Printf("Could not complie regular expression: %v. \n", err)
	}
	allNum, err = regexp.Compile(`^[0-9]*$`)
	if err != nil {
		log.Printf("Could not complie regular expression: %v. \n", err)
	}
	wordBounds, err = regexp.Compile(`\b`)
	if err != nil {
		log.Printf("Could not complie regular expression: %v. \n", err)
	}
	isLetter, err = regexp.Compile(`^[a-zA-Z]+$`)
	if err != nil {
		log.Printf("Could not complie regular expression: %v. \n", err)
	}
	return nil
}

// LoadSpellModel loads a saved spelling model
func LoadSpellModel() error {
	pdir := oswin.TheApp.AppPrefsDir()
	openpath := filepath.Join(pdir, "spell_en_us_plain.json")
	err := spell.Load(openpath)
	return err
}

// NewSpellModelFromText builds a NEW spelling model from text
func NewSpellModelFromText() error {
	bigdatapath, err := kit.GoSrcDir("github.com/goki/gi/spell")
	if err != nil {
		log.Printf("Error getting path to corpus directory: %v.\n", err)
		return err
	}

	bigdatafile := filepath.Join(bigdatapath, "big.txt")
	file, err := os.Open(bigdatafile)
	if err != nil {
		log.Printf("Could not open corpus file: %v. This file is used to create the spelling model.\n", err)
		PromptDialog(nil, DlgOpts{Title: "Corpus File Not Found", Prompt: "You can build a spelling model to check against by clicking the \"Train\" button and selecting text files to train on."}, true, false, nil, nil)
		return err
	}

	err = spell.Train(*file, true) // true - create a NEW spelling model
	if err != nil {
		log.Printf("Failed building model from corpus file: %v.\n", err)
		return err
	}
	return nil
}

// AddToSpellModel trains on additional text - extends model
func AddToSpellModel(filepath string) error {
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
	pdir := oswin.TheApp.AppPrefsDir()
	path := filepath.Join(pdir, "spell_en_us_plain.json")
	err := spell.Save(path)
	if err != nil {
		log.Printf("Could not save spelling model to file: %v.\n", err)
	}
	return err
}

// InitNewSpellCheck builds the input list, i.e. the words to check
func InitNewSpellCheck(text []byte) {
	input = input[:0] // clear past input
	inputidx = 0
	TextToWords(text)
}

// TextToWords generates a slice of words from text
// removes various non-word input, trims symbols, etc
func TextToWords(text []byte) {
	textstr := string(text)

	var words []TextWord
	for l, line := range strings.Split(textstr, "\n") {
		line = notWordChar.ReplaceAllString(line, " ")
		bounds := wordBounds.FindAllStringIndex(line, -1)
		words = words[:0] // reset for new line
		splits := strings.Fields(line)
		for i, w := range splits {
			if allNum.MatchString(w) {
				break
			}
			if len(w) > 1 {
				tw := TextWord{Word: w, Line: l, StartPos: bounds[i*2][0], EndPos: bounds[i*2+1][0]}
				words = append(words, tw)
			}
		}
		input = append(input, words...)
	}
}

// IsWord returns true if the string follows rules to accept as word
func IsWord(word string) bool {
	return isLetter.MatchString(word)
}

// NextUnknownWord returns the next unknown word, i.e. not found in corpus
func NextUnknownWord() (unknown TextWord, suggests []string, err error) {
	var tw TextWord

	for {
		tw = NextWord()
		if tw.Word == "" { // we're done!
			break
		}
		var known = false
		suggests, known, err = spell.CheckWord(tw.Word)
		if !known {
			break
		}
	}
	return tw, suggests, err
}

// NextWord returns the next word of the input words
func NextWord() TextWord {
	tw := TextWord{}
	if inputidx < len(input) {
		tw = input[inputidx]
		inputidx += 1
		return tw
	}
	return tw
}

// LearnWord calls the implementation so the app isn't tied to a particular implementation
func LearnWord(w string) {
	spell.LearnWord(w)
}

// IgnoreWord adds the word to the ignore list
func IgnoreWord(w string) {
	spell.IgnoreWord(w)
}

// DoIgnore returns true if word is on ignore list
func DoIgnore(w string) bool {
	return spell.DoIgnore(w)
}

////////////////////////////////////////////////////////////////////////////////////////
// SpellCorrect

// SpellCorrect
type SpellCorrect struct {
	ki.Node
	EditFunc    spell.EditFunc `desc:"function to edit text using the selected spell correction"`
	Context     interface{}    `desc:"the object that implements spell.Func"`
	Suggestions []string
	Word        string    `desc:"word being checked"`
	SpellSig    ki.Signal `json:"-" xml:"-" view:"-" desc:"signal for SpellCorrect -- see SpellSignals for the types"`
	Suggestion  string    `desc:"the user's correction selection'"`
}

var KiT_SpellCorrect = kit.Types.AddType(&SpellCorrect{}, nil)

// SpellSignals are signals that are sent by SpellCorrect
type SpellSignals int64

const (
	// SpellSelect means the user chose one of the possible corrections
	SpellSelect SpellSignals = iota
)

//go:generate stringer -type=SpellSignals

// CheckWordInLine checks the model to determine if the word is known,
// if not known also check the ignore list
func (sc *SpellCorrect) CheckWordInline(word string) (sugs []string, knwn bool, err error) {
	sugs, knwn, err = spell.CheckWord(word)
	if err != nil {
		return sugs, knwn, err
	}
	if !knwn {
		knwn = spell.DoIgnore(word)
	}
	return sugs, knwn, err
}

// Show is the main call for listing spelling corrections.
// Calls ShowNow which builds the correction popup menu
// Similar to completion.Show but does not use a timer
// Displays popup immediately for any unknown word
func (sc *SpellCorrect) Show(text string, pos token.Position, vp *Viewport2D, pt image.Point) {
	if vp == nil || vp.Win == nil {
		return
	}
	cpop := vp.Win.CurPopup()
	if PopupIsCorrector(cpop) {
		vp.Win.SetDelPopup(cpop)
	}
	sc.ShowNow(text, pos, vp, pt)
}

// ShowNow actually builds the correction popup menu
func (sc *SpellCorrect) ShowNow(word string, pos token.Position, vp *Viewport2D, pt image.Point) {
	if vp == nil || vp.Win == nil {
		return
	}
	cpop := vp.Win.CurPopup()
	if PopupIsCorrector(cpop) {
		vp.Win.SetDelPopup(cpop)
	}

	var m Menu
	var text string
	count := len(sc.Suggestions)
	if count == 1 && sc.Suggestions[0] == word {
		return
	}
	if count == 0 {
		text = "no suggestion"
		m.AddAction(ActOpts{Label: text, Data: text},
			sc, func(recv, send ki.Ki, sig int64, data interface{}) {
			})
	} else {
		for i := 0; i < count; i++ {
			text = sc.Suggestions[i]
			m.AddAction(ActOpts{Label: text, Data: text},
				sc, func(recv, send ki.Ki, sig int64, data interface{}) {
					scf := recv.Embed(KiT_SpellCorrect).(*SpellCorrect)
					scf.SpellCorrect(data.(string))
				})
		}
	}
	m.AddSeparator("")
	text = "learn"
	m.AddAction(ActOpts{Label: text, Data: text},
		sc, func(recv, send ki.Ki, sig int64, data interface{}) {
			scf := recv.Embed(KiT_SpellCorrect).(*SpellCorrect)
			scf.LearnWordInline()
		})
	text = "ignore"
	m.AddAction(ActOpts{Label: text, Data: text},
		sc, func(recv, send ki.Ki, sig int64, data interface{}) {
			scf := recv.Embed(KiT_SpellCorrect).(*SpellCorrect)
			scf.IgnoreAllInline()
		})
	pvp := PopupMenu(m, pt.X, pt.Y, vp, "tf-spellcheck-menu")
	pvp.SetFlag(int(VpFlagCorrector))
	pvp.KnownChild(0).SetProp("no-focus-name", true) // disable name focusing -- grabs key events in popup instead of in textfield!
}

// SpellCorrect emits a signal to let subscribers know that the user has made a
// selection from the list of possible corrections
func (sc *SpellCorrect) SpellCorrect(s string) {
	sc.Suggestion = s
	sc.SpellSig.Emit(sc.This(), int64(SpellSelect), s)
}

// KeyInput is the opportunity for the spelling correction popup to act on specific key inputs
func (sc *SpellCorrect) KeyInput(kf KeyFuns) bool { // true - caller should set key processed
	switch kf {
	case KeyFunMoveDown:
		return true
	case KeyFunMoveUp:
		return true
	}
	return false
}

// LearnWordInline gets the misspelled/unknown word and passes to LearnWord
func (sc *SpellCorrect) LearnWordInline() {
	LearnWord(sc.Word)
}

// IgnoreAllInline adds the word to the ignore list
func (sc *SpellCorrect) IgnoreAllInline() {
	IgnoreWord(sc.Word)
}

// Cancel cancels any pending spell correction -- call when new events nullify prior correction
// returns true if canceled
func (c *SpellCorrect) Cancel(vp *Viewport2D) bool {
	if vp == nil || vp.Win == nil {
		return false
	}
	cpop := vp.Win.CurPopup()
	if PopupIsCorrector(cpop) {
		vp.Win.SetDelPopup(cpop)
		return true
	}
	return false
}
