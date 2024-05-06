// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"path/filepath"
	"runtime/debug"

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

// HandleRecover is the core value of [system.HandleRecover]. If r is not nil,
// it makes a window displaying information about the panic. [system.HandleRecover]
// is initialized to this in init.
func HandleRecover(r any) {
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

	b := NewBody("app-stopped-unexpectedly").AddTitle(system.TheApp.Name() + " stopped unexpectedly").
		AddText("There was an unexpected error and " + system.TheApp.Name() + " stopped running.")
	b.AddBottomBar(func(parent Widget) {
		NewButton(parent).SetText("Details").SetType(ButtonOutlined).OnClick(func(e events.Event) {
			clpath := filepath.Join(TheApp.AppDataDir(), "crash-logs")
			txt := fmt.Sprintf("Crash log saved in %s\n\n%s", clpath, system.CrashLogText(r, stack))
			d := NewBody("crash-details").AddTitle("Crash details")
			NewText(d).SetText(txt).Style(func(s *styles.Style) {
				s.SetMono(true)
				s.Text.WhiteSpace = styles.WhiteSpacePreWrap
			})
			d.AddBottomBar(func(parent Widget) {
				NewButton(parent).SetText("Copy").SetIcon(icons.Copy).SetType(ButtonOutlined).
					OnClick(func(e events.Event) {
						d.Clipboard().Write(mimedata.NewText(txt))
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
