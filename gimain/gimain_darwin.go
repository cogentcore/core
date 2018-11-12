// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build darwin

package gimain

import "github.com/goki/gi/gi"

func init() {
	gi.DefaultKeyMap = gi.KeyMapName("MacStd")
	gi.SetActiveKeyMapName(gi.DefaultKeyMap)
}
