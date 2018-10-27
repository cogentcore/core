// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"go/token"
	"image"
	"time"

	"github.com/goki/gi/complete"
	"github.com/goki/gi/oswin"
	"github.com/goki/ki"
	"github.com/goki/ki/bitflag"
	"github.com/goki/ki/kit"
)

////////////////////////////////////////////////////////////////////////////////////////
// Complete

// Complete holds the current completion data and functions to call for building
// the list of possible completions and for editing text after a completion is selected
type Complete struct {
	ki.Node
	MatchFunc   complete.MatchFunc `desc:"function to get the list of possible completions"`
	EditFunc    complete.EditFunc  `desc:"function to edit text using the selected completion"`
	Context     interface{}        `desc:"the object that implements complete.Func"`
	Completions complete.Completions
	Seed        string    `desc:"current completion seed"`
	CompleteSig ki.Signal `json:"-" xml:"-" view:"-" desc:"signal for complete -- see CompleteSignals for the types"`
	Completion  string    `desc:"the user's completion selection'"`
	DelayTimer  *time.Timer
}

var KiT_Complete = kit.Types.AddType(&Complete{}, nil)

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

// Show is the main call for listing completions.
// Has a builtin delay timer so completions are only shown after
// a delay, which resets every time it is called.
// After delay, Calls ShowNow, which calls MatchFunc
// to get a list of completions and builds the completion popup menu
func (c *Complete) Show(text string, pos token.Position, vp *Viewport2D, pt image.Point) {
	if c.MatchFunc == nil || vp == nil || vp.Win == nil {
		return
	}
	if PopupIsCompleter(vp.Win.Popup) {
		vp.Win.DelPopup = vp.Win.Popup
	}
	if c.DelayTimer != nil {
		c.DelayTimer.Stop()
	}
	c.DelayTimer = time.AfterFunc(time.Duration(CompleteWaitMSec)*time.Millisecond,
		func() {
			c.ShowNow(text, pos, vp, pt)
			c.DelayTimer = nil
		})
}

// ShowNow actually calls MatchFunc to get a list of completions and builds the
// completion popup menu
func (c *Complete) ShowNow(text string, pos token.Position, vp *Viewport2D, pt image.Point) {
	if c.MatchFunc == nil || vp == nil || vp.Win == nil {
		return
	}

	if PopupIsCompleter(vp.Win.Popup) {
		vp.Win.DelPopup = vp.Win.Popup
	}
	c.Completions, c.Seed = c.MatchFunc(c.Context, text, pos)
	count := len(c.Completions)
	if count > 0 {
		if count == 1 && c.Completions[0].Text == c.Seed {
			return
		}
		var m Menu
		for i := 0; i < count; i++ {
			text := c.Completions[i].Text
			icon := c.Completions[i].Icon
			m.AddAction(ActOpts{Icon: icon, Label: text, Data: text},
				c, func(recv, send ki.Ki, sig int64, data interface{}) {
					tff := recv.Embed(KiT_Complete).(*Complete)
					tff.Complete(data.(string))
				})
		}
		pvp := PopupMenu(m, pt.X, pt.Y, vp, "tf-completion-menu")
		bitflag.Set(&pvp.Flag, int(VpFlagCompleter))
		pvp.KnownChild(0).SetProp("no-focus-name", true) // disable name focusing -- grabs key events in popup instead of in textfield!
		oswin.SendCustomEvent(vp.Win.OSWin, nil)         // needs an extra event to show popup
	}
}

// Cancel cancels any pending completion -- call when new events nullify prior completions
// returns true if canceled
func (c *Complete) Cancel(vp *Viewport2D) bool {
	if vp == nil || vp.Win == nil {
		return false
	}
	if PopupIsCompleter(vp.Win.Popup) {
		vp.Win.DelPopup = vp.Win.Popup
	}
	if c.DelayTimer != nil {
		c.DelayTimer.Stop()
		c.DelayTimer = nil
		return true
	}
	return false
}

// Complete emits a signal to let subscribers know that the user has made a
// selection from the list of possible completions
func (c *Complete) Complete(s string) {
	c.Completion = s
	c.CompleteSig.Emit(c.This, int64(CompleteSelect), s)
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
				c.CompleteSig.Emit(c.This, int64(CompleteExtend), s)
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
