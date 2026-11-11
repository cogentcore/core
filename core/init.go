// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"cogentcore.org/core/base/dirsort"
	"cogentcore.org/core/system"
)

func init() {
	system.HandleRecover = handleRecover
	system.UpdateLogicalDPIScaleFunc = AppearanceSettings.updateLogicalDPI
	TheApp.CogentCoreDataDir()            // ensure it exists
	theWindowGeometrySaver.needToReload() // gets time stamp associated with open, so it doesn't re-open
	theWindowGeometrySaver.open()
	if TheApp.Platform() != system.Offscreen { // (not in tests)
		// the sort orders of directories are shared by all apps
		dirsort.DataDir = TheApp.CogentCoreDataDir()
	}
}
