// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"runtime/debug"

	"cogentcore.org/core/events"
	"cogentcore.org/core/goosi"
	"cogentcore.org/core/styles"
)

// HandleRecover is the gi value of [goosi.HandleRecover]. If r is not nil,
// it makes a window displaying information about the panic. [goosi.HandleRecover]
// is initialized to this in init.
func HandleRecover(r any) {
	if r == nil {
		return
	}
	goosi.HandleRecoverBase(r)

	stack := string(debug.Stack())

	// we have to handle the quit button indirectly so that it has the
	// right stack for debugging when panicking
	quit := make(chan struct{})

	b := NewBody("app-exited-unexpectedly").AddTitle(goosi.TheApp.Name() + " exited unexpectedly")
	b.AddBottomBar(func(pw Widget) {
		NewButton(pw).SetText("Details").SetType(ButtonOutlined).OnClick(func(e events.Event) {
			d := NewBody("crash-details").AddTitle("Crash details")
			NewLabel(d).SetText(stack).Style(func(s *styles.Style) {
				s.Font.Family = string(AppearanceSettings.MonoFont)
			})
			d.AddOkOnly().NewDialog(b).Run()
		})
		NewButton(pw).SetText("Quit").OnClick(func(e events.Event) {
			quit <- struct{}{}
		})
	})
	b.NewWindow().Run()
	<-quit
	panic(r)
}
