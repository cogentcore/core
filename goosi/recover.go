// Copyright 2023 The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package goosi

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	"goki.dev/grr"
)

// HandleRecover takes the given value of recover, and, if it is not nil,
// handles it. The behavior of HandleRecover can be customized by changing
// it to a custom function, but by default it calls [HandleRecoverBase] and
// [HandleRecoverPanic], which safely prints a stack trace, saves a crash log,
// and panics on non-mobile platforms. It is set in gi to a function that does
// the aforementioned things in addition to creating a GUI error dialog.
// HandleRecover should be called at the start of every
// goroutine whenever possible. The correct usage of HandleRecover is:
//
//	func myFunc() {
//		defer func() { goosi.HandleRecover(recover()) }()
//		...
//	}
var HandleRecover = func(r any) {
	HandleRecoverBase(r)
	HandleRecoverPanic(r)
}

// HandleRecoverBase is the default base value of [HandleRecover].
// It can be extended to form a different value of [HandleRecover].
// It prints a panic message and a stack trace, using a string-based log
// method that guarantees that the stack trace will be printed before
// the program exits. This is needed because, without this, the program
// may exit before it can print the stack trace on some systems like Android,
// which makes debugging nearly impossible. It also saves a crash log to
// [TheApp.GokiDataDir].
func HandleRecoverBase(r any) {
	if r == nil {
		return
	}

	stack := string(debug.Stack())

	print := func(w io.Writer) {
		fmt.Fprintln(w, "panic:", r)
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "----- START OF STACK TRACE: -----")
		fmt.Fprintln(w, stack)
		fmt.Fprintln(w, "----- END OF STACK TRACE -----")
	}

	print(os.Stderr)

	dnm := filepath.Join(TheApp.GokiDataDir(), "crash-logs", TheApp.Name())
	err := os.MkdirAll(dnm, 0755)
	if grr.Log(err) != nil {
		return
	}
	cfnm := filepath.Join(dnm, "crash_"+time.Now().Format("2006-01-02_15-04-05"))
	cf, err := os.Create(cfnm)
	if grr.Log(err) != nil {
		return
	}
	print(cf)
	cf.Close()
	log.Println("SAVED CRASH LOG TO", cfnm)
}

// HandleRecoverPanic panics on r if r is non-nil and [TheApp.Platform] is not mobile.
// This is because panicking screws up logging on mobile, but is necessary for debugging
// on desktop.
func HandleRecoverPanic(r any) {
	if r == nil {
		return
	}
	if !TheApp.Platform().IsMobile() || TheApp.Platform() == Web {
		panic(r)
	}
}
