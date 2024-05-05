// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ios

package ios

import (
	"cogentcore.org/core/system/driver/base"
)

// Window is the implementation of [system.Window] for the iOS platform.
type Window struct {
	base.WindowSingle[*App]
}
