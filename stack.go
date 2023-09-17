// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grr

import (
	"runtime"
	"strings"
)

// Stack returns the stack trace up to this caller as a slice of frames.
func Stack() []runtime.Frame {
	callers := make([]uintptr, 10)
	n := runtime.Callers(4, callers)
	// Return now to avoid processing the zero Frame that would
	// otherwise be returned by frames.Next below.
	if n == 0 {
		return nil
	}

	frames := runtime.CallersFrames(callers)
	res := []runtime.Frame{}
	for {
		frame, more := frames.Next()
		// Stop unwinding when we enter package runtime or test,
		// as we only care about errors in the program, not the
		// low-level language code.
		if strings.Contains(frame.File, "runtime/") || strings.Contains(frame.File, "testing/") {
			break
		}
		res = append(res, frame)
		if !more {
			break
		}
	}
	return res
}
