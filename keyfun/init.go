// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package keyfun

import "cogentcore.org/core/goosi"

func init() {
	AvailMaps.CopyFrom(StandardMaps)
	switch goosi.TheApp.SystemPlatform() {
	case goosi.MacOS:
		DefaultMap = "MacStandard"
	case goosi.Windows:
		DefaultMap = "WindowsStandard"
	default:
		DefaultMap = "LinuxStandard"
	}
	SetActiveMapName(DefaultMap)
}
