// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"cogentcore.org/core/system"
)

func init() {
	system.HandleRecover = handleRecover
	system.InitScreenLogicalDPIFunc = AppearanceSettings.applyDPI // called when screens are initialized
	TheApp.CogentCoreDataDir()                                    // ensure it exists
	theWindowGeometrySaver.needToReload()                         // gets time stamp associated with open, so it doesn't re-open
	theWindowGeometrySaver.open()
}
