// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"sync"
	"time"

	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/text/parse/complete"
)

// Complete holds the current completion data and functions to call for building
// the list of possible completions and for editing text after a completion is selected.
// It also holds the popup [Stage] associated with it.
type Complete struct { //types:add -setters

	// function to get the list of possible completions
	MatchFunc complete.MatchFunc

	// function to get the text to show for lookup
	LookupFunc complete.LookupFunc

	// function to edit text using the selected completion
	EditFunc complete.EditFunc

	// the context object that implements the completion functions
	Context any

	// line number in source that completion is operating on, if relevant
	SrcLn int

	// character position in source that completion is operating on
	SrcCh int

	// the list of potential completions
	completions complete.Completions

	// current completion seed
	Seed string

	// the user's completion selection
	Completion string

	// the event listeners for the completer (it sends [events.Select] events)
	listeners events.Listeners

	// stage is the popup [Stage] associated with the [Complete].
	stage *Stage

	delayTimer *time.Timer
	delayMu    sync.Mutex
	showMu     sync.Mutex
}

// NewComplete returns a new [Complete] object. It does not show it; see [Complete.Show].
func NewComplete() *Complete {
	return &Complete{}
}

// IsAboutToShow returns true if the DelayTimer is started for
// preparing to show a completion.  note: don't really need to lock
func (c *Complete) IsAboutToShow() bool {
	c.delayMu.Lock()
	defer c.delayMu.Unlock()
	return c.delayTimer != nil
}

// Show is the main call for listing completions.
// Has a builtin delay timer so completions are only shown after
// a delay, which resets every time it is called.
// After delay, Calls ShowNow, which calls MatchFunc
// to get a list of completions and builds the completion popup menu
func (c *Complete) Show(ctx Widget, pos image.Point, text string) {
	if c.MatchFunc == nil {
		return
	}
	wait := SystemSettings.CompleteWaitDuration
	if c.stage != nil {
		c.Cancel()
	}
	if wait == 0 {
		c.showNow(ctx, pos, text)
		return
	}
	c.delayMu.Lock()
	if c.delayTimer != nil {
		c.delayTimer.Stop()
	}
	c.delayTimer = time.AfterFunc(wait,
		func() {
			c.showNowAsync(ctx, pos, text)
			c.delayMu.Lock()
			c.delayTimer = nil
			c.delayMu.Unlock()
		})
	c.delayMu.Unlock()
}

// showNow actually calls MatchFunc to get a list of completions and builds the
// completion popup menu.  This is the sync version called from
func (c *Complete) showNow(ctx Widget, pos image.Point, text string) {
	if c.stage != nil {
		c.Cancel()
	}
	c.showMu.Lock()
	defer c.showMu.Unlock()
	if c.showNowImpl(ctx, pos, text) {
		c.stage.runPopup()
	}
}

// showNowAsync actually calls MatchFunc to get a list of completions and builds the
// completion popup menu.  This is the Async version for delayed AfterFunc call.
func (c *Complete) showNowAsync(ctx Widget, pos image.Point, text string) {
	if c.stage != nil {
		c.cancelAsync()
	}
	c.showMu.Lock()
	defer c.showMu.Unlock()
	if c.showNowImpl(ctx, pos, text) {
		c.stage.runPopupAsync()
	}
}

// showNowImpl is the implementation of ShowNow, presenting completions.
// Returns false if nothing to show.
func (c *Complete) showNowImpl(ctx Widget, pos image.Point, text string) bool {
	md := c.MatchFunc(c.Context, text, c.SrcLn, c.SrcCh)
	c.completions = md.Matches
	c.Seed = md.Seed
	if len(c.completions) == 0 {
		return false
	}
	if len(c.completions) > SystemSettings.CompleteMaxItems {
		c.completions = c.completions[0:SystemSettings.CompleteMaxItems]
	}

	sc := NewScene(ctx.AsTree().Name + "-complete")
	StyleMenuScene(sc)
	c.stage = NewPopupStage(CompleterStage, sc, ctx).SetPos(pos)
	// we forward our key events to the context object
	// so that you can keep typing while in a completer
	// sc.OnKeyChord(ctx.HandleEvent)

	for i := 0; i < len(c.completions); i++ {
		cmp := &c.completions[i]
		text := cmp.Text
		if cmp.Label != "" {
			text = cmp.Label
		}
		icon := cmp.Icon
		mi := NewButton(sc).SetText(text).SetIcon(icons.Icon(icon))
		mi.SetTooltip(cmp.Desc)
		mi.OnClick(func(e events.Event) {
			c.complete(cmp.Text)
		})
		mi.OnKeyChord(func(e events.Event) {
			kf := keymap.Of(e.KeyChord())
			if e.KeyRune() == ' ' {
				ctx.AsWidget().HandleEvent(e) // bypass button handler!
			}
			if kf == keymap.Enter {
				e.SetHandled()
				c.complete(cmp.Text)
			}
		})
		if i == 0 {
			sc.Events.SetStartFocus(mi)
		}
	}
	return true
}

// Cancel cancels any existing or pending completion.
// Call when new events nullify prior completions.
// Returns true if canceled.
func (c *Complete) Cancel() bool {
	if c.stage == nil {
		return false
	}
	st := c.stage
	c.stage = nil
	st.ClosePopup()
	return true
}

// cancelAsync cancels any existing *or* pending completion,
// inside a delayed callback function (Async)
// Call when new events nullify prior completions.
// Returns true if canceled.
func (c *Complete) cancelAsync() bool {
	if c.stage == nil {
		return false
	}
	st := c.stage
	c.stage = nil
	st.closePopupAsync()
	return true
}

// Lookup is the main call for doing lookups.
func (c *Complete) Lookup(text string, posLine, posChar int, sc *Scene, pt image.Point) {
	if c.LookupFunc == nil || sc == nil {
		return
	}
	// c.Sc = nil
	c.LookupFunc(c.Context, text, posLine, posChar) // this processes result directly
}

// complete sends Select event to listeners, indicating that the user has made a
// selection from the list of possible completions.
// This is called inside the main event loop.
func (c *Complete) complete(s string) {
	c.Cancel()
	c.Completion = s
	c.listeners.Call(&events.Base{Typ: events.Select})
}

// OnSelect registers given listener function for [events.Select] events on Value.
// This is the primary notification event for all [Complete] elements.
func (c *Complete) OnSelect(fun func(e events.Event)) {
	c.On(events.Select, fun)
}

// On adds an event listener function for the given event type.
func (c *Complete) On(etype events.Types, fun func(e events.Event)) {
	c.listeners.Add(etype, fun)
}

// GetCompletion returns the completion with the given text.
func (c *Complete) GetCompletion(s string) complete.Completion {
	for _, cc := range c.completions {
		if s == cc.Text {
			return cc
		}
	}
	return complete.Completion{}
}

// CompleteEditText is a chance to modify the completion selection before it is inserted.
func CompleteEditText(text string, cp int, completion string, seed string) (ed complete.Edit) {
	ed.NewText = completion
	return ed
}
