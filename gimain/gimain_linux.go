// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build linux,!android dragonfly openbsd

package gimain

import "github.com/goki/gi/gi"

func init() {
	gi.DefaultKeyMap = gi.KeyMapName("LinuxStd")
	gi.SetActiveKeyMapName(gi.DefaultKeyMap)
	gi.Prefs.FontFamily = "Liberation Sans"
}
