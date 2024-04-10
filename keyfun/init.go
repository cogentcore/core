// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package keyfun

import "cogentcore.org/core/system"

func init() {
	AvailableMaps.CopyFrom(StandardMaps)
	switch system.TheApp.SystemPlatform() {
	case system.MacOS:
		DefaultMap = "MacStandard"
	case system.Windows:
		DefaultMap = "WindowsStandard"
	default:
		DefaultMap = "LinuxStandard"
	}
	SetActiveMapName(DefaultMap)
}
