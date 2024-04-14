// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build offscreen

package driver

import (
	"cogentcore.org/core/system/driver/offscreen"
)

func init() {
	offscreen.Init()
}
