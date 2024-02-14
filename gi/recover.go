// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"path/filepath"
	"runtime/debug"
	"time"

	"cogentcore.org/core/events"
	"cogentcore.org/core/goosi"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/mimedata"
	"cogentcore.org/core/styles"
)

// timesCrashed is the number of times that the program has
// crashed. It is used to prevent an infinite crash loop
// when rendering the crash window.
var timesCrashed int

// HandleRecover is the gi value of [goosi.HandleRecover]. If r is not nil,
// it makes a window displaying information about the panic. [goosi.HandleRecover]
// is initialized to this in init.
func HandleRecover(r any) {
	if r == nil {
		return
	}
	timesCrashed++
	goosi.HandleRecoverBase(r)
	if timesCrashed > 1 {
		return
	}

	stack := string(debug.Stack())

	// we have to handle the quit button indirectly so that it has the
	// right stack for debugging when panicking
	quit := make(chan struct{})

	b := NewBody("app-stopped-unexpectedly").AddTitle(goosi.TheApp.Name() + " stopped unexpectedly").
		AddText("There was an unexpected error and " + goosi.TheApp.Name() + " stopped running.")
	b.AddBottomBar(func(pw Widget) {
		NewButton(pw).SetText("Details").SetType(ButtonOutlined).OnClick(func(e events.Event) {
			clpath := filepath.Join(TheApp.AppDataDir(), "crash-logs")
			txt := fmt.Sprintf("Crash log saved in %s\n\nPlatform: %v\nSystem platform: %v\nApp version: %s\nCore version: %s\nTime: %s\n\npanic: %v\n\n%s", clpath, TheApp.Platform(), TheApp.SystemPlatform(), AppVersion, CoreVersion, time.Now().Format(time.DateTime), r, stack)
			d := NewBody("crash-details").AddTitle("Crash details")
			NewLabel(d).SetText(txt).Style(func(s *styles.Style) {
				s.Font.Family = string(AppearanceSettings.MonoFont)
				s.Text.WhiteSpace = styles.WhiteSpacePreWrap
			})
			d.AddBottomBar(func(pw Widget) {
				NewButton(pw).SetText("Copy").SetIcon(icons.Copy).SetType(ButtonOutlined).
					OnClick(func(e events.Event) {
						d.Clipboard().Write(mimedata.NewText(txt))
					})
				d.AddOk(pw)
			})
			d.NewFullDialog(b).Run()
		})
		NewButton(pw).SetText("Quit").OnClick(func(e events.Event) {
			quit <- struct{}{}
		})
	})
	b.NewWindow().Run()
	<-quit
	panic(r)
}
