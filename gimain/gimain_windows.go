// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package gimain

import "github.com/goki/gi"

func init() {
	gi.DefaultKeyMap = &gi.WindowsKeyMap
	gi.ActiveKeyMap = gi.DefaultKeyMap
}
