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
)

// timesCrashed is the number of times that the program has
// crashed. It is used to prevent an infinite crash loop
// when rendering the crash window.
var timesCrashed int

// webCrashDialog is the function used to display the crash dialog on web.
// It cannot be displayed normally due to threading and single-window issues.
var webCrashDialog func(title, text, body string)

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
	text := "There was an unexpected error and " + TheApp.Name() + " stopped running."

	clpath := filepath.Join(TheApp.AppDataDir(), "crash-logs")
	clpath = strings.ReplaceAll(clpath, " ", `\ `) // escape spaces
	body := fmt.Sprintf("Crash log saved in %s\n\n%s", clpath, system.CrashLogText(r, stack))

	if webCrashDialog != nil {
		webCrashDialog(title, text, body)
		return
	}

	b := NewBody("app-stopped-unexpectedly").AddTitle(title).AddText(text)
	b.AddBottomBar(func(parent Widget) {
		NewButton(parent).SetText("Details").SetType(ButtonOutlined).OnClick(func(e events.Event) {
			d := NewBody("crash-details").AddTitle("Crash details")
			NewText(d).SetText(body).Styler(func(s *styles.Style) {
				s.SetMono(true)
				s.Text.WhiteSpace = styles.WhiteSpacePreWrap
			})
			d.AddBottomBar(func(parent Widget) {
				NewButton(parent).SetText("Copy").SetIcon(icons.Copy).SetType(ButtonOutlined).
					OnClick(func(e events.Event) {
						d.Clipboard().Write(mimedata.NewText(body))
					})
				d.AddOK(parent)
			})
			d.RunFullDialog(b)
		})
		NewButton(parent).SetText("Quit").OnClick(func(e events.Event) {
			quit <- struct{}{}
		})
	})
	b.RunWindow()
	<-quit
	panic(r)
}
