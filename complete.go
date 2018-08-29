// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"github.com/goki/gi/complete"
	"github.com/goki/ki"
	"github.com/goki/ki/bitflag"
)

////////////////////////////////////////////////////////////////////////////////////////
// Complete
type Complete struct {
	MatchFunc   complete.MatchFunc `desc:"function to get the list of possible completions"`
	EditFunc    complete.EditFunc  `desc:"function to edit text using the selected completion"`
	Context     interface{}        `desc:"the object that implements complete.Func"`
	Completions []string           `desc:"possible completions"`
	Seed        string             `desc:"current completion seed"`
}

func (c *Complete) ShowCompletions(text string, vp *Viewport2D, x int, y int, f *TextField) {
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
				f, func(recv, send ki.Ki, sig int64, data interface{}) {
					tff := recv.Embed(KiT_TextField).(*TextField)
					tff.Complete(s)
				})
		}
		vp := PopupMenu(m, x, y, vp, "tf-completion-menu")
		bitflag.Set(&vp.Flag, int(VpFlagCompleter))
		vp.KnownChild(0).SetProp("no-focus-name", true) // disable name focusing -- grabs key events in popup instead of in textfield!
	}
}
