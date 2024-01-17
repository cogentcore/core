// Copyright (c) 2023, The Goki Authors. All rights reserved.
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
	goosi.InitScreenLogicalDPIFunc = AppearanceSettings.ApplyDPI // called when screens are initialized
	GokiDataDir()                                                // ensure it exists
	WinGeomMgr.NeedToReload()                                    // gets time stamp associated with open, so it doesn't re-open
	WinGeomMgr.Open()

	// needed to prevent app from quitting prematurely
	if testing.Testing() {
		b := NewAppBody("__test-base")
		b.App().AppBarConfig = nil
		b.NewWindow().Run()
	}
}
