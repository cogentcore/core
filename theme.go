// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package goosi

import "log"

// LogIsDarkMonitor handles the given error channel returned
// by [IsDarkMonitor], in the context of the given done channel
// passed to [IsDarkMonitor]. If LogIsDarkMonitor gets any
// error on the error channel, it logs it and returns, and
// if the done channel is closed, it returns. It is a blocking
// call, so it should typically be called in a separate goroutine.
func LogIsDarkMonitor(ec chan error, done chan struct{}) {
	select {
	case err := <-ec:
		log.Println(err)
		return
	case _, ok := <-done:
		// if done is closed, we return
		if !ok {
			return
		}
	}
}
