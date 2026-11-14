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
	TheApp.CogentCoreDataDir() // ensure it exists
	theWindowGeometrySaver.open()
	if TheApp.Platform() != system.Offscreen {
		// the sort orders of directories are shared by all apps
		dirsort.DataDir = TheApp.CogentCoreDataDir()
	} else {
		// tests do not depend on the file manager of the machine
		dirsort.Native = nil
	}
}
