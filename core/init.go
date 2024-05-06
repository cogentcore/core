// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"

	"cogentcore.org/core/styles"
	"cogentcore.org/core/system"
)

func init() {
	system.HandleRecover = HandleRecover
	system.InitScreenLogicalDPIFunc = AppearanceSettings.ApplyDPI // called when screens are initialized
	TheApp.AppBarConfig = StandardAppBarConfig
	TheApp.CogentCoreDataDir()            // ensure it exists
	TheWindowGeometrySaver.NeedToReload() // gets time stamp associated with open, so it doesn't re-open
	TheWindowGeometrySaver.Open()
	styles.SettingsFont = (*string)(&AppearanceSettings.Font)
	styles.SettingsMonoFont = (*string)(&AppearanceSettings.MonoFont)

	if testing.Testing() {
		TheApp.AppBarConfig = nil
		// needed to prevent app from quitting prematurely
		NewBody().RunWindow()
	}
}
