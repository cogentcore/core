// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package base

import (
	"log"
	"runtime/debug"
)

// FuncRun is a simple helper type that contains a function to call and a channel
// to send a signal on when the function is finished running.
type FuncRun struct {
	F    func()
	Done chan struct{}
}

// HandleRecover takes the given value of recover, and, if it is not nil,
// prints a panic message and a stack trace, using a string-based log
// method that guarantees that the stack trace will be printed before
// the program exits. This is needed because, without this, the program
// will exit before it can print the stack trace, which makes debugging
// nearly impossible. The correct usage of HandleRecover is:
//
//	func myFunc() {
//		defer func() { HandleRecover(recover()) }()
//		...
//	}
func HandleRecover(r any) {
	if r == nil {
		return
	}
	log.Println("panic:", r)
	log.Println("")
	log.Println("----- START OF STACK TRACE: -----")
	log.Println(string(debug.Stack()))
	log.Fatalln("----- END OF STACK TRACE -----")
}
