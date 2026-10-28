// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// basic is a terminal app that runs the given command, or the user's shell.
package main

import (
	"os"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/text/terminal"
)

func main() {
	b := core.NewBody("Terminal")
	tm := terminal.NewTerminal(b)
	if len(os.Args) > 1 {
		tm.SetCommand(os.Args[1:]...)
	}
	tm.OnTitle(func(title string) {
		b.SetTitle(title)
	})
	tm.OnExit(func(err error) {
		core.MessageSnackbar(tm, "the program has exited: press a key to restart it")
	})
	b.OnShow(func(e events.Event) {
		tm.SetFocus()
	})
	b.RunMainWindow()
}
