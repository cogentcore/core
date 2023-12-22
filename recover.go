// Copyright 2023 The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package goosi

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	"goki.dev/grr"
)

// HandleRecover takes the given value of recover, and, if it is not nil,
// handles it. The behavior of HandleRecover can be customized by changing
// it to a custom function, but by default it is [HandleRecoverBase], which
// safely prints a stack trace and saves a crash log. It is set in gi to
// a function that does the aforementioned things in addition to creating a
// GUI error dialog. HandleRecover should be called at the start of every
// goroutine whenever possible. The correct usage of HandleRecover is:
//
//	func myFunc() {
//		defer func() { goosi.HandleRecover(recover()) }()
//		...
//	}
var HandleRecover = HandleRecoverBase

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

	cfnm := filepath.Join(TheApp.GokiDataDir(), "crash-logs", TheApp.Name(), "crash_"+time.Now().Format("2006-01-02_15-04-05"))
	cf, err := os.Create(cfnm)
	if grr.Log(err) == nil {
		print(cf)
		cf.Close()
	}
}
