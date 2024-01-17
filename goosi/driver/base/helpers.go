// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package base

// FuncRun is a simple helper type that contains a function to call and a channel
// to send a signal on when the function is finished running.
type FuncRun struct {
	F    func()
	Done chan struct{}
}
