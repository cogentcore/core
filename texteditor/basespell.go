// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

// TODO: consider moving back to core or somewhere else based on the
// result of https://github.com/cogentcore/core/issues/711

import (
	"image"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/spell"
)

// InitSpell ensures that the spell.Spell spell checker is setup
func InitSpell() error {
	if core.TheApp.Platform().IsMobile() { // todo: too slow -- fix with aspell
		return nil
	}
	if spell.Spell != nil {
		return nil
	}
	pdir := core.TheApp.CogentCoreDataDir()
	openpath := filepath.Join(pdir, "user_dict_en_us")
	spell.Spell = spell.NewSpell(openpath)
	return nil
}

// Spell has all the texteditor spell check state
type Spell struct { //types:add -setters
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
	Listeners events.Listeners `set:"-" display:"-"`

	// Stage is the [PopupStage] associated with the [Spell]
	Stage *core.Stage

	ShowMu sync.Mutex `set:"-"`
}

// NewSpell returns a new [Spell]
func NewSpell() *Spell {
	InitSpell()
	return &Spell{}
}

// CheckWord checks the model to determine if the word is known,
// bool is true if known, false otherwise. If not known,
// returns suggestions for close matching words.
func (sp *Spell) CheckWord(word string) ([]string, bool) {
	if spell.Spell == nil {
		return nil, false
	}
	return spell.Spell.CheckWord(word)
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
func (sp *Spell) Show(text string, ctx core.Widget, pos image.Point) {
	if sp.Stage != nil {
		sp.Cancel()
	}
	sp.ShowNow(text, ctx, pos)
}

// ShowNow actually builds the correction popup menu
func (sp *Spell) ShowNow(word string, ctx core.Widget, pos image.Point) {
	if sp.Stage != nil {
		sp.Cancel()
	}
	sp.ShowMu.Lock()
	defer sp.ShowMu.Unlock()

	sc := core.NewScene(ctx.AsTree().Name + "-spell")
	core.MenuSceneConfigStyles(sc)
	sp.Stage = core.NewPopupStage(core.CompleterStage, sc, ctx).SetPos(pos)

	if sp.IsLastLearned(word) {
		core.NewButton(sc).SetText("unlearn").SetTooltip("unlearn the last learned word").
			OnClick(func(e events.Event) {
				sp.Cancel()
				sp.UnLearnLast()
			})
	} else {
		count := len(sp.Suggest)
		if count == 1 && sp.Suggest[0] == word {
			return
		}
		if count == 0 {
			core.NewButton(sc).SetText("no suggestion")
		} else {
			for i := 0; i < count; i++ {
				text := sp.Suggest[i]
				core.NewButton(sc).SetText(text).OnClick(func(e events.Event) {
					sp.Cancel()
					sp.Spell(text)
				})
			}
		}
		core.NewSeparator(sc)
		core.NewButton(sc).SetText("learn").OnClick(func(e events.Event) {
			sp.Cancel()
			sp.LearnWord()
		})
		core.NewButton(sc).SetText("ignore").OnClick(func(e events.Event) {
			sp.Cancel()
			sp.IgnoreWord()
		})
	}
	if sc.NumChildren() > 0 {
		sc.Events.SetStartFocus(sc.Child(0).(core.Widget))
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
	spell.Spell.AddWord(sp.Word)
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
	spell.Spell.DeleteWord(lword)
}

// IgnoreWord adds the word to the ignore list
func (sp *Spell) IgnoreWord() {
	spell.Spell.IgnoreWord(sp.Word)
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
