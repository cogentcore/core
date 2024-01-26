// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"testing"

	"cogentcore.org/core/goosi"
	_ "cogentcore.org/core/goosi/driver"
	_ "cogentcore.org/core/grog"
)

func init() {
	goosi.HandleRecover = HandleRecover
	goosi.InitScreenLogicalDPIFunc = AppearanceSettings.ApplyDPI // called when screens are initialized
	TheApp.AppBarConfig = StdAppBarConfig
	TheApp.CogentCoreDataDir() // ensure it exists
	WinGeomMgr.NeedToReload()  // gets time stamp associated with open, so it doesn't re-open
	WinGeomMgr.Open()

	if testing.Testing() {
		TheApp.AppBarConfig = nil
		// needed to prevent app from quitting prematurely
		NewBody().NewWindow().Run()
	}
}
