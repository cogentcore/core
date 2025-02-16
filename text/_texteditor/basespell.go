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
	"cogentcore.org/core/text/spell"
)

// initSpell ensures that the [spell.Spell] spell checker is set up.
func initSpell() error {
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

// spellCheck has all the texteditor spell check state
type spellCheck struct {
	// line number in source that spelling is operating on, if relevant
	srcLn int

	// character position in source that spelling is operating on (start of word to be corrected)
	srcCh int

	// list of suggested corrections
	suggest []string

	// word being checked
	word string

	// last word learned -- can be undone -- stored in lowercase format
	lastLearned string

	// the user's correction selection
	correction string

	// the event listeners for the spell (it sends Select events)
	listeners events.Listeners

	// stage is the popup [core.Stage] associated with the [spellState]
	stage *core.Stage

	showMu sync.Mutex
}

// newSpell returns a new [spellState]
func newSpell() *spellCheck {
	initSpell()
	return &spellCheck{}
}

// checkWord checks the model to determine if the word is known,
// bool is true if known, false otherwise. If not known,
// returns suggestions for close matching words.
func (sp *spellCheck) checkWord(word string) ([]string, bool) {
	if spell.Spell == nil {
		return nil, false
	}
	return spell.Spell.CheckWord(word)
}

// setWord sets the word to spell and other associated info
func (sp *spellCheck) setWord(word string, sugs []string, srcLn, srcCh int) *spellCheck {
	sp.word = word
	sp.suggest = sugs
	sp.srcLn = srcLn
	sp.srcCh = srcCh
	return sp
}

// show is the main call for listing spelling corrections.
// Calls ShowNow which builds the correction popup menu
// Similar to completion.show but does not use a timer
// Displays popup immediately for any unknown word
func (sp *spellCheck) show(text string, ctx core.Widget, pos image.Point) {
	if sp.stage != nil {
		sp.cancel()
	}
	sp.showNow(text, ctx, pos)
}

// showNow actually builds the correction popup menu
func (sp *spellCheck) showNow(word string, ctx core.Widget, pos image.Point) {
	if sp.stage != nil {
		sp.cancel()
	}
	sp.showMu.Lock()
	defer sp.showMu.Unlock()

	sc := core.NewScene(ctx.AsTree().Name + "-spell")
	core.StyleMenuScene(sc)
	sp.stage = core.NewPopupStage(core.CompleterStage, sc, ctx).SetPos(pos)

	if sp.isLastLearned(word) {
		core.NewButton(sc).SetText("unlearn").SetTooltip("unlearn the last learned word").
			OnClick(func(e events.Event) {
				sp.cancel()
				sp.unLearnLast()
			})
	} else {
		count := len(sp.suggest)
		if count == 1 && sp.suggest[0] == word {
			return
		}
		if count == 0 {
			core.NewButton(sc).SetText("no suggestion")
		} else {
			for i := 0; i < count; i++ {
				text := sp.suggest[i]
				core.NewButton(sc).SetText(text).OnClick(func(e events.Event) {
					sp.cancel()
					sp.spell(text)
				})
			}
		}
		core.NewSeparator(sc)
		core.NewButton(sc).SetText("learn").OnClick(func(e events.Event) {
			sp.cancel()
			sp.learnWord()
		})
		core.NewButton(sc).SetText("ignore").OnClick(func(e events.Event) {
			sp.cancel()
			sp.ignoreWord()
		})
	}
	if sc.NumChildren() > 0 {
		sc.Events.SetStartFocus(sc.Child(0).(core.Widget))
	}
	sp.stage.Run()
}

// spell sends a Select event to Listeners indicating that the user has made a
// selection from the list of possible corrections
func (sp *spellCheck) spell(s string) {
	sp.cancel()
	sp.correction = s
	sp.listeners.Call(&events.Base{Typ: events.Select})
}

// onSelect registers given listener function for Select events on Value.
// This is the primary notification event for all Complete elements.
func (sp *spellCheck) onSelect(fun func(e events.Event)) {
	sp.on(events.Select, fun)
}

// on adds an event listener function for the given event type
func (sp *spellCheck) on(etype events.Types, fun func(e events.Event)) {
	sp.listeners.Add(etype, fun)
}

// learnWord gets the misspelled/unknown word and passes to learnWord
func (sp *spellCheck) learnWord() {
	sp.lastLearned = strings.ToLower(sp.word)
	spell.Spell.AddWord(sp.word)
}

// isLastLearned returns true if given word was the last one learned
func (sp *spellCheck) isLastLearned(wrd string) bool {
	lword := strings.ToLower(wrd)
	return lword == sp.lastLearned
}

// unLearnLast unlearns the last learned word -- in case accidental
func (sp *spellCheck) unLearnLast() {
	if sp.lastLearned == "" {
		slog.Error("spell.UnLearnLast: no last learned word")
		return
	}
	lword := sp.lastLearned
	sp.lastLearned = ""
	spell.Spell.DeleteWord(lword)
}

// ignoreWord adds the word to the ignore list
func (sp *spellCheck) ignoreWord() {
	spell.Spell.IgnoreWord(sp.word)
}

// cancel cancels any pending spell correction.
// call when new events nullify prior correction.
// returns true if canceled
func (sp *spellCheck) cancel() bool {
	if sp.stage == nil {
		return false
	}
	st := sp.stage
	sp.stage = nil
	st.ClosePopup()
	return true
}
