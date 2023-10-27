// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build (linux && !android) || dragonfly || openbsd
// +build linux,!android dragonfly openbsd

package gimain

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/keyfun"
)

func init() {
	keyfun.DefaultMap = keyfun.MapName("LinuxStd")
	keyfun.SetActiveMapName(keyfun.DefaultMap)
	gi.Prefs.FontFamily = "Liberation Sans"
}
