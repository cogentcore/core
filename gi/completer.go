// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"sync"
	"time"

	"goki.dev/gi/v2/keyfun"
	"goki.dev/goosi/events"
	"goki.dev/icons"
	"goki.dev/pi/v2/complete"
	"goki.dev/spell"
)

// Completer interface supports the SetCompleter method for setting completer parameters
// This is defined e.g., on TextField and textview.Buf
type Completer interface {
	// SetCompleter sets completion functions so that completions will
	// automatically be offered as the user types.  data provides context where being used.
	SetCompleter(data any, matchFun complete.MatchFunc, editFun complete.EditFunc)
}

//////////////////////////////////////////////////////////////////////////////
// Complete

// NewCompleter returns a new [CompleterStage] with given scene contents,
// in connection with given widget (which provides key context).
// Make further configuration choices using Set* methods, which
// can be chained directly after the New call.
// Use an appropriate Run call at the end to start the Stage running.
func NewCompleter(sc *Scene, ctx Widget) *Stage {
	return NewPopupStage(CompleterStage, sc, ctx)
}

// Complete holds the current completion data and functions to call for building
// the list of possible completions and for editing text after a completion is selected.
// It also holds the [PopupStage] associated with it.
type Complete struct { //gti:add -setters
	// Stage is the [PopupStage] associated with the [Complete]
	Stage *Stage

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

	// the event listeners for the completer (it sends Select events)
	Listeners events.Listeners `set:"-" view:"-"`

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

// CompleteWaitDuration is the amount of time to wait before
// showing the completion menu
var CompleteWaitDuration time.Duration = 0

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

	wait := CompleteWaitDuration
	if force {
		wait = 0
	}
	if c.Stage != nil {
		c.Cancel()
	}
	c.DelayMu.Lock()
	if c.DelayTimer != nil {
		c.DelayTimer.Stop()
	}
	if text == "" {
		c.DelayMu.Unlock()
		return
	}

	c.DelayTimer = time.AfterFunc(wait,
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
	if c.Stage != nil {
		c.Cancel()
	}
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
	c.Stage = NewPopupStage(CompleterStage, sc, ctx).SetPos(pos)
	// we forward our key events to the context object
	// so that you can keep typing while in a completer
	// sc.OnKeyChord(ctx.HandleEvent)

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
			}).
			OnKeyChord(func(e events.Event) {
				kf := keyfun.Of(e.KeyChord())
				if e.KeyRune() == ' ' {
					ctx.HandleEvent(e) // bypass button handler!
				}
				if kf == keyfun.Enter {
					e.SetHandled()
					c.Complete(cmp.Text)
				}
			})
	}
	c.Stage.RunPopup()
}

// Cancel cancels any existing *or* pending completion.
// Call when new events nullify prior completions.
// Returns true if canceled.
func (c *Complete) Cancel() bool {
	if c.Stage == nil {
		return false
	}
	st := c.Stage
	c.Stage = nil
	st.ClosePopup()
	return true
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
	c.Listeners.Call(&events.Base{Typ: events.Select})
}

// OnSelect registers given listener function for Select events on Value.
// This is the primary notification event for all Complete elements.
func (c *Complete) OnSelect(fun func(e events.Event)) {
	c.On(events.Select, fun)
}

// On adds an event listener function for the given event type
func (c *Complete) On(etype events.Types, fun func(e events.Event)) {
	c.Listeners.Add(etype, fun)
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
