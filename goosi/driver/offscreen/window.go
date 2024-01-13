// Copyright 2023 The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package offscreen

import (
	"goki.dev/goosi"
	"goki.dev/goosi/driver/base"
)

// Window is the implementation of [goosi.Window] for the offscreen platform.
type Window struct { //gti:add
	base.WindowSingle[*App]
}

func (w *Window) SetDropCallback(callback func(w goosi.Window, names []string)) {
	//TODO implement me
	panic("implement me")
}

var _ goosi.Window = &Window{}

func (w *Window) Handle() any {
	return nil
}
