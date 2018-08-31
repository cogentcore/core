// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"github.com/goki/gi/complete"
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
	Completions []string           `desc:"possible completions"`
	Seed        string             `desc:"current completion seed"`
	CompleteSig ki.Signal          `json:"-" xml:"-" view:"-" desc:"signal for complete -- see CompleteSignals for the types"`
	Completion  string             `desc:"the user's completion selection'"`
}

var KiT_Complete = kit.Types.AddType(&Complete{}, nil)

// CompleteSignals are signals that are sent by Complete
type CompleteSignals int64

const (
	// CompleteSelect means the user chose one of the possible completions
	CompleteSelect CompleteSignals = iota
)

//go:generate stringer -type=CompleteSignals

// ShowCompletions calls MatchFunc to get a list of completions and builds the completion popup menu
func (c *Complete) ShowCompletions(text string, vp *Viewport2D, x int, y int) {
	if c.MatchFunc == nil {
		return
	}

	c.Completions, c.Seed = c.MatchFunc(text)
	count := len(c.Completions)
	if count > 0 {
		if count == 1 && c.Completions[0] == c.Seed {
			return
		}
		var m Menu
		for i := 0; i < count; i++ {
			s := c.Completions[i]
			m.AddAction(ActOpts{Label: s},
				c, func(recv, send ki.Ki, sig int64, data interface{}) {
					tff := recv.Embed(KiT_Complete).(*Complete)
					tff.Complete(s)
				})
		}
		vp := PopupMenu(m, x, y, vp, "tf-completion-menu")
		bitflag.Set(&vp.Flag, int(VpFlagCompleter))
		vp.KnownChild(0).SetProp("no-focus-name", true) // disable name focusing -- grabs key events in popup instead of in textfield!
	}
}

// Complete emits a signal to let subscribers know that the user has made a selection from the list of possible completions
func (c *Complete) Complete(s string) {
	fmt.Println(s)
	c.CompleteSig.Emit(c.This, int64(CompleteSelect), s)
}
