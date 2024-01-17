// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package greasi

import (
	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/glop/sentence"
	"cogentcore.org/core/grease"
	"cogentcore.org/core/grog"
	"github.com/iancoleman/strcase"
)

// GUI starts the GUI for the given Grease app, which must be passed as
// a pointer. It should typically not be called by end-user code; see [Run].
func GUI[T any](opts *grease.Options, cfg T, cmds ...*grease.Cmd[T]) {
	b := gi.NewAppBody(opts.AppName).SetTitle(opts.AppTitle)
	b.App().About = opts.AppAbout

	b.AddAppBar(func(tb *gi.Toolbar) {
		for _, cmd := range cmds {
			cmd := cmd
			if cmd.Name == "gui" { // we are already in GUI so that command is irrelevant
				continue
			}
			// need to go to camel first (it is mostly in kebab)
			gi.NewButton(tb, cmd.Name).SetText(sentence.Case(strcase.ToCamel(cmd.Name))).SetTooltip(cmd.Doc).
				OnClick(func(e events.Event) {
					err := cmd.Func(cfg)
					if err != nil {
						// TODO: snackbar
						grog.PrintlnError(err)
					}
				})
		}
	})

	sv := giv.NewStructView(b)
	sv.SetStruct(cfg)

	b.NewWindow().Run().Wait()
}
