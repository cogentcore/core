// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"sync"
	"time"

	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/pi/complete"
	"github.com/goki/pi/spell"
)

// Completer interface supports the SetCompleter method for setting completer parameters
// This is defined e.g., on TextField and TextBuf
type Completer interface {
	// SetCompleter sets completion functions so that completions will
	// automatically be offered as the user types.  data provides context where being used.
	SetCompleter(data interface{}, matchFun complete.MatchFunc, editFun complete.EditFunc)
}

////////////////////////////////////////////////////////////////////////////////////////
// Complete

// Complete holds the current completion data and functions to call for building
// the list of possible completions and for editing text after a completion is selected
type Complete struct {
	ki.Node
	MatchFunc   complete.MatchFunc   `desc:"function to get the list of possible completions"`
	LookupFunc  complete.LookupFunc  `desc:"function to get the text to show for lookup"`
	EditFunc    complete.EditFunc    `desc:"function to edit text using the selected completion"`
	Context     interface{}          `desc:"the object that implements complete.Func"`
	SrcLn       int                  `desc:"line number in source that completion is operating on, if relevant"`
	SrcCh       int                  `desc:"character position in source that completion is operating on"`
	Completions complete.Completions `desc:"the list of potential completions"`
	Seed        string               `desc:"current completion seed"`
	CompleteSig ki.Signal            `json:"-" xml:"-" view:"-" desc:"signal for complete -- see CompleteSignals for the types"`
	Completion  string               `desc:"the user's completion selection'"`
	Vp          *Viewport2D          `desc:"the viewport where the current popup menu is presented"`
	DelayTimer  *time.Timer
	DelayMu     sync.Mutex
	ShowMu      sync.Mutex
}

var KiT_Complete = kit.Types.AddType(&Complete{}, nil)

func (cm *Complete) Disconnect() {
	cm.Node.Disconnect()
	cm.CompleteSig.DisconnectAll()
}

// CompleteSignals are signals that are sent by Complete
type CompleteSignals int64

const (
	// CompleteSelect means the user chose one of the possible completions
	CompleteSelect CompleteSignals = iota

	// CompleteExtend means user has requested that the seed extend if all
	// completions have a common prefix longer than current seed
	CompleteExtend
)

//go:generate stringer -type=CompleteSignals

// CompleteWaitMSec is the number of milliseconds to wait before
// showing the completion menu
var CompleteWaitMSec = 500

// CompleteMaxItems is the max number of items to display in completer popup
var CompleteMaxItems = 25

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
func (c *Complete) Show(text string, posLn, posCh int, vp *Viewport2D, pt image.Point, force bool) {
	if c.MatchFunc == nil || vp == nil || vp.Win == nil {
		return
	}
	cpop := vp.Win.CurPopup()
	if PopupIsCompleter(cpop) {
		vp.Win.SetDelPopup(cpop)
	}
	c.DelayMu.Lock()
	if c.DelayTimer != nil {
		c.DelayTimer.Stop()
	}
	if text == "" {
		c.DelayMu.Unlock()
		return
	}
	waitMSec := CompleteWaitMSec
	if force {
		waitMSec = 0
	}
	c.DelayTimer = time.AfterFunc(time.Duration(waitMSec)*time.Millisecond,
		func() {
			c.ShowNow(text, posLn, posCh, vp, pt, force)
			c.DelayMu.Lock()
			c.DelayTimer = nil
			c.DelayMu.Unlock()
		})
	c.DelayMu.Unlock()
}

// ShowNow actually calls MatchFunc to get a list of completions and builds the
// completion popup menu
func (c *Complete) ShowNow(text string, posLn, posCh int, vp *Viewport2D, pt image.Point, force bool) {
	if c.MatchFunc == nil || vp == nil || vp.Win == nil {
		return
	}
	cpop := vp.Win.CurPopup()
	if PopupIsCompleter(cpop) {
		vp.Win.SetDelPopup(cpop)
	}
	c.ShowMu.Lock()
	defer c.ShowMu.Unlock()
	c.Vp = nil
	md := c.MatchFunc(c.Context, text, posLn, posCh)
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

	var m Menu
	for i := 0; i < count; i++ {
		cmp := &c.Completions[i]
		text := cmp.Text
		if cmp.Label != "" {
			text = cmp.Label
		}
		icon := cmp.Icon
		m.AddAction(ActOpts{Icon: icon, Label: text, Data: cmp.Text},
			c, func(recv, send ki.Ki, sig int64, data interface{}) {
				cc := recv.Embed(KiT_Complete).(*Complete)
				cc.Complete(data.(string))
			})
	}
	c.Vp = vp
	pvp := PopupMenu(m, pt.X, pt.Y, vp, "tf-completion-menu")
	pvp.SetFlag(int(VpFlagCompleter))
	pvp.Child(0).SetProp("no-focus-name", true) // disable name focusing -- grabs key events in popup instead of in textfield!
	vp.Win.OSWin.SendEmptyEvent()               // needs an extra event to show popup
}

// Cancel cancels any existing *or* pending completion.
// Call when new events nullify prior completions.
// Returns true if canceled.
func (c *Complete) Cancel() bool {
	did := false
	if c.Vp != nil && c.Vp.Win != nil {
		cpop := c.Vp.Win.CurPopup()
		if PopupIsCompleter(cpop) {
			c.Vp.Win.SetDelPopup(cpop)
			did = true
		}
	}
	ab := c.Abort()
	return did || ab
}

// Abort aborts *only* pending completions, but does not close existing window.
// Returns true if aborted.
func (c *Complete) Abort() bool {
	c.DelayMu.Lock()
	c.Vp = nil
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
func (c *Complete) Lookup(text string, posLn, posCh int, vp *Viewport2D, pt image.Point, force bool) {
	if c.LookupFunc == nil || vp == nil || vp.Win == nil {
		return
	}
	c.Vp = nil
	c.LookupFunc(c.Context, text, posLn, posCh) // this processes result directly
}

// Complete emits a signal to let subscribers know that the user has made a
// selection from the list of possible completions
func (c *Complete) Complete(s string) {
	c.Cancel()
	c.Completion = s
	c.CompleteSig.Emit(c.This(), int64(CompleteSelect), s)
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
				s := complete.ExtendSeed(c.Completions, c.Seed)
				c.CompleteSig.Emit(c.This(), int64(CompleteExtend), s)
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
