// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build darwin
// +build darwin

package gimain

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/keyfuns"
)

func init() {
	gi.DefaultKeyMap = keyfuns.KeyMapName("MacStd")
	gi.SetActiveKeyMapName(gi.DefaultKeyMap)
}
