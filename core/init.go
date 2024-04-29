// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/system"
)

func init() {
	system.HandleRecover = HandleRecover
	system.InitScreenLogicalDPIFunc = AppearanceSettings.ApplyDPI // called when screens are initialized
	TheApp.SetWebOnUpdate(webOnUpdate)
	TheApp.AppBarConfig = StandardAppBarConfig
	TheApp.CogentCoreDataDir()            // ensure it exists
	TheWindowGeometrySaver.NeedToReload() // gets time stamp associated with open, so it doesn't re-open
	TheWindowGeometrySaver.Open()

	if testing.Testing() {
		TheApp.AppBarConfig = nil
		// needed to prevent app from quitting prematurely
		NewBody().RunWindow()
	}
}

func webOnUpdate() {
	NewBody("web-update-available").
		AddSnackbarText("A new version of " + TheApp.Name() + " is available").
		AddSnackbarButton("Reload").NewSnackbar(nil).SetTimeout(0).Run()
}
