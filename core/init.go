// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"testing"

	"cogentcore.org/core/styles"
	"cogentcore.org/core/system"
)

func init() {
	fmt.Println("core base init")
	system.HandleRecover = handleRecover
	system.InitScreenLogicalDPIFunc = AppearanceSettings.applyDPI // called when screens are initialized
	TheApp.CogentCoreDataDir()                                    // ensure it exists
	theWindowGeometrySaver.needToReload()                         // gets time stamp associated with open, so it doesn't re-open
	theWindowGeometrySaver.open()
	styles.SettingsFont = (*string)(&AppearanceSettings.Font)
	styles.SettingsMonoFont = (*string)(&AppearanceSettings.MonoFont)

	if testing.Testing() {
		// needed to prevent app from quitting prematurely
		NewBody().RunWindow()
	}
}
