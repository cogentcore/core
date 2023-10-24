// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"sync"
	"time"

	"goki.dev/goosi/events"
	"goki.dev/icons"
	"goki.dev/pi/v2/complete"
	"goki.dev/pi/v2/spell"
)

// Completer interface supports the SetCompleter method for setting completer parameters
// This is defined e.g., on TextField and textview.Buf
type Completer interface {
	// SetCompleter sets completion functions so that completions will
	// automatically be offered as the user types.  data provides context where being used.
	SetCompleter(data any, matchFun complete.MatchFunc, editFun complete.EditFunc)
}

////////////////////////////////////////////////////////////////////////////////////////
// Complete

// Complete holds the current completion data and functions to call for building
// the list of possible completions and for editing text after a completion is selected.
// It also holds the [PopupStage] associated with it.
type Complete struct { //gti:add -setters
	// Stage is the [PopupStage] associated with the [Complete]
	Stage *PopupStage

	// function to get the list of possible completions
	MatchFunc complete.MatchFunc

	// function to get the text to show for lookup
	LookupFunc complete.LookupFunc

	// function to edit text using the selected completion
	EditFunc complete.EditFunc

	// the object that implements complete.Func
	Context any

	// line number in source that completion is operating on, if relevant
	SrcLn int

	// character position in source that completion is operating on
	SrcCh int

	// the list of potential completions
	Completions complete.Completions

	// current completion seed
	Seed string

	// the user's completion selection
	Completion string

	DelayTimer *time.Timer `set:"-"`
	DelayMu    sync.Mutex  `set:"-"`
	ShowMu     sync.Mutex  `set:"-"`
}

// CompleteSignals are signals that are sent by Complete
type CompleteSignals int32 //enums:enum -trim-prefix Complete

const (
	// CompleteSelect means the user chose one of the possible completions
	CompleteSelect CompleteSignals = iota

	// CompleteExtend means user has requested that the seed extend if all
	// completions have a common prefix longer than current seed
	CompleteExtend
)

// CompleteWaitMSec is the number of milliseconds to wait before
// showing the completion menu
var CompleteWaitMSec = 0

// CompleteMaxItems is the max number of items to display in completer popup
var CompleteMaxItems = 25

// NewComplete returns a new [Complete] object. It does not show it; see [Complete.Show].
func NewComplete() *Complete {
	return &Complete{}
}

// IsAboutToShow returns true if the DelayTimer is started for
// preparing to show a completion.  note: don't really need to lock
func (c *Complete) IsAboutToShow() bool {
	c.DelayMu.Lock()
	defer c.DelayMu.Unlock()
	return c.DelayTimer != nil
}

// Show is the main call for listing completions.
// Has a builtin delay timer so completions are only shown after
// a delay, which resets every time it is called.
// After delay, Calls ShowNow, which calls MatchFunc
// to get a list of completions and builds the completion popup menu
func (c *Complete) Show(ctx Widget, pos image.Point, text string, force bool) {
	if c.MatchFunc == nil {
		return
	}

	waitMSec := CompleteWaitMSec
	if force {
		waitMSec = 0
	}
	// if PopupIsCompleter(cpop) {
	// 	sc.Win.SetDelPopup(cpop)
	// }
	c.DelayMu.Lock()
	if c.DelayTimer != nil {
		c.DelayTimer.Stop()
	}
	if text == "" {
		c.DelayMu.Unlock()
		return
	}

	c.DelayTimer = time.AfterFunc(time.Duration(waitMSec)*time.Millisecond,
		func() {
			c.DelayMu.Lock()
			c.ShowNow(ctx, pos, text, force)
			c.DelayTimer = nil
			c.DelayMu.Unlock()
		})
	c.DelayMu.Unlock()
}

// ShowNow actually calls MatchFunc to get a list of completions and builds the
// completion popup menu.
func (c *Complete) ShowNow(ctx Widget, pos image.Point, text string, force bool) {
	if c.MatchFunc == nil {
		return
	}
	// cpop := sc.Win.CurPopup()
	// if PopupIsCompleter(cpop) && (!keep || sc.Win.CurPopup() == nil) {
	// 	sc.Win.SetDelPopup(cpop)
	// }
	c.ShowMu.Lock()
	defer c.ShowMu.Unlock()
	md := c.MatchFunc(c.Context, text, c.SrcLn, c.SrcCh)
	c.Completions = md.Matches
	c.Seed = md.Seed
	count := len(c.Completions)
	if count == 0 {
		return
	}
	if !force {
		if count > CompleteMaxItems || (count == 1 && c.Completions[0].Text == c.Seed) {
			return
		}
	}

	sc := NewScene(ctx.Name() + "-complete")
	MenuSceneConfigStyles(sc)
	c.Stage = NewPopupStage(CompleterStage, sc, ctx)
	sc.Geom.Pos = ctx.ContextMenuPos(nil)

	for i := 0; i < count; i++ {
		cmp := &c.Completions[i]
		text := cmp.Text
		if cmp.Label != "" {
			text = cmp.Label
		}
		icon := cmp.Icon
		NewButton(sc, text).SetText(text).SetIcon(icons.Icon(icon)).SetTooltip(cmp.Desc).
			OnClick(func(e events.Event) {
				c.Complete(cmp.Text)
			})
	}
	c.Stage.RunPopup()
}

// Cancel cancels any existing *or* pending completion.
// Call when new events nullify prior completions.
// Returns true if canceled.
func (c *Complete) Cancel() bool {
	c.Stage.Close()
	return true
	/*
		did := false
		if c.Sc != nil && c.Sc.Win != nil {
			cpop := c.Sc.Win.CurPopup()
			if PopupIsCompleter(cpop) {
				c.Sc.Win.SetDelPopup(cpop)
				did = true
			}
		}
		ab := c.Abort()
		return did || ab
	*/
}

// Abort aborts *only* pending completions, but does not close existing window.
// Returns true if aborted.
func (c *Complete) Abort() bool {
	c.DelayMu.Lock()
	// c.Sc = nil
	if c.DelayTimer != nil {
		c.DelayTimer.Stop()
		c.DelayTimer = nil
		c.DelayMu.Unlock()
		return true
	}
	c.DelayMu.Unlock()
	return false
}

// Lookup is the main call for doing lookups
func (c *Complete) Lookup(text string, posLn, posCh int, sc *Scene, pt image.Point, force bool) {
	if c.LookupFunc == nil || sc == nil {
		return
	}
	// c.Sc = nil
	c.LookupFunc(c.Context, text, posLn, posCh) // this processes result directly
}

// Complete emits a signal to let subscribers know that the user has made a
// selection from the list of possible completions
func (c *Complete) Complete(s string) {
	c.Cancel()
	c.Completion = s
	// c.CompleteSig.Emit(c.This(), int64(CompleteSelect), s)
}

// KeyInput is the opportunity for completion to act on specific key inputs
func (c *Complete) KeyInput(kf KeyFuns) bool { // true - caller should set key processed
	count := len(c.Completions)
	switch kf {
	case KeyFunFocusNext: // tab will complete if single item or try to extend if multiple items
		if count > 0 {
			if count == 1 { // just complete
				c.Complete(c.Completions[0].Text)
			} else { // try to extend the seed
				// s := complete.ExtendSeed(c.Completions, c.Seed)
				// c.CompleteSig.Emit(c.This(), int64(CompleteExtend), s)
			}
			return true
		}
	case KeyFunMoveDown:
		if count == 1 {
			return true
		}
	case KeyFunMoveUp:
		if count == 1 {
			return true
		}
	}
	return false
}

func (c *Complete) GetCompletion(s string) complete.Completion {
	for _, cc := range c.Completions {
		if s == cc.Text {
			return cc
		}
	}
	return complete.Completion{}
}

// CompleteText is the function for completing text files
func CompleteText(s string) []string {
	return spell.Complete(s)
}

// CompleteEditText is a chance to modify the completion selection before it is inserted
func CompleteEditText(text string, cp int, completion string, seed string) (ed complete.Edit) {
	ed.NewText = completion
	return ed
}
