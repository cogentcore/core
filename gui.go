// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package greasi

import (
	"github.com/iancoleman/strcase"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/gi/v2/giv"
	"goki.dev/glop/sentencecase"
	"goki.dev/goosi/events"
	"goki.dev/grease"
	"goki.dev/grog"
)

// GUI starts the GUI for the given Grease app, which must be passed as
// a pointer. It should typically not be called by end-user code; see [Run].

func GUI[T any](opts *grease.Options, cfg T, cmds ...*grease.Cmd[T]) {
	gimain.Run(func() {
		App(opts, cfg, cmds...)
	})
}

// App does runs the GUI. It should be called on the main thread.
// It should typically not be called by end-user code; see [Run].
func App[T any](opts *grease.Options, cfg T, cmds ...*grease.Cmd[T]) {
	gi.SetAppName(opts.AppName)
	gi.SetAppAbout(opts.AppAbout)

	b := gi.NewBody(opts.AppName).SetTitle(opts.AppTitle)

	gi.DefaultTopAppBar = func(tb *gi.TopAppBar) {
		gi.DefaultTopAppBarStd(tb)

		for _, cmd := range cmds {
			cmd := cmd
			if cmd.Name == "gui" { // we are already in GUI so that command is irrelevant
				continue
			}
			// need to go to camel first (it is mostly in kebab)
			gi.NewButton(tb, cmd.Name).SetText(sentencecase.Of(strcase.ToCamel(cmd.Name))).SetTooltip(cmd.Doc).
				OnClick(func(e events.Event) {
					err := cmd.Func(cfg)
					if err != nil {
						// TODO: snackbar
						grog.PrintlnError(err)
					}
				})
		}
	}

	sv := giv.NewStructView(b)
	sv.SetStruct(cfg)

	b.NewWindow().Run().Wait()
}
