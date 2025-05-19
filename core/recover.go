// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"path/filepath"
	"runtime/debug"
	"strings"

	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/system"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
)

// timesCrashed is the number of times that the program has
// crashed. It is used to prevent an infinite crash loop
// when rendering the crash window.
var timesCrashed int

// webCrashDialog is the function used to display the crash dialog on web.
// It cannot be displayed normally due to threading and single-window issues.
var webCrashDialog func(title, txt, body string)

// handleRecover is the core value of [system.HandleRecover]. If r is not nil,
// it makes a window displaying information about the panic. [system.HandleRecover]
// is initialized to this in init.
func handleRecover(r any) {
	if r == nil {
		return
	}
	timesCrashed++
	system.HandleRecoverBase(r)
	if timesCrashed > 1 {
		return
	}

	stack := string(debug.Stack())

	// we have to handle the quit button indirectly so that it has the
	// right stack for debugging when panicking
	quit := make(chan struct{})

	title := TheApp.Name() + " stopped unexpectedly"
	txt := "There was an unexpected error and " + TheApp.Name() + " stopped running."

	clpath := filepath.Join(TheApp.AppDataDir(), "crash-logs")
	clpath = strings.ReplaceAll(clpath, " ", `\ `) // escape spaces
	body := fmt.Sprintf("Crash log saved in %s\n\n%s", clpath, system.CrashLogText(r, stack))

	if webCrashDialog != nil {
		webCrashDialog(title, txt, body)
		return
	}

	b := NewBody(title)
	NewText(b).SetText(title).SetType(TextHeadlineSmall)
	NewText(b).SetType(TextSupporting).SetText(txt)
	b.AddBottomBar(func(bar *Frame) {
		NewButton(bar).SetText("Details").SetType(ButtonOutlined).OnClick(func(e events.Event) {
			d := NewBody("Crash details")
			NewText(d).SetText(body).Styler(func(s *styles.Style) {
				s.Font.Family = rich.Monospace
				s.Text.WhiteSpace = text.WhiteSpacePreWrap
			})
			d.AddBottomBar(func(bar *Frame) {
				NewButton(bar).SetText("Copy").SetIcon(icons.Copy).SetType(ButtonOutlined).
					OnClick(func(e events.Event) {
						d.Clipboard().Write(mimedata.NewText(body))
					})
				d.AddOK(bar)
			})
			d.RunFullDialog(b)
		})
		NewButton(bar).SetText("Quit").OnClick(func(e events.Event) {
			quit <- struct{}{}
		})
	})
	b.RunWindow()
	<-quit
	panic(r)
}
