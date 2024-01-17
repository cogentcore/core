// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package offscreen

import (
	"cogentcore.org/core/goosi"
	"cogentcore.org/core/goosi/driver/base"
)

// Window is the implementation of [goosi.Window] for the offscreen platform.
type Window struct { //gti:add
	base.WindowSingle[*App]
}

var _ goosi.Window = &Window{}

func (w *Window) Handle() any {
	return nil
}
