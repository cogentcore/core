// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package system

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"cogentcore.org/core/base/errors"
)

// HandleRecover takes the given value of recover, and, if it is not nil,
// handles it. The behavior of HandleRecover can be customized by changing
// it to a custom function, but by default it calls [HandleRecoverBase] and
// [HandleRecoverPanic], which safely prints a stack trace, saves a crash log,
// and panics on non-mobile platforms. It is set in core to a function that does
// the aforementioned things in addition to creating a GUI error dialog.
// HandleRecover should be called at the start of every
// goroutine whenever possible. The correct usage of HandleRecover is:
//
//	func myFunc() {
//		defer func() { system.HandleRecover(recover()) }()
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
// [App.AppDataDir].
func HandleRecoverBase(r any) {
	if r == nil {
		return
	}

	stack := string(debug.Stack())

	log.Println("panic:", r)
	log.Println("")
	log.Println("----- START OF STACK TRACE: -----")
	log.Println(stack)
	log.Println("----- END OF STACK TRACE -----")

	dnm := filepath.Join(TheApp.AppDataDir(), "crash-logs")
	err := os.MkdirAll(dnm, 0755)
	if errors.Log(err) != nil {
		return
	}
	cfnm := filepath.Join(dnm, "crash_"+time.Now().Format("2006-01-02_15-04-05"))
	err = os.WriteFile(cfnm, []byte(CrashLogText(r, stack)), 0666)
	if errors.Log(err) != nil {
		return
	}
	cfnm = strings.ReplaceAll(cfnm, " ", `\ `) // escape spaces
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

// CrashLogText returns an appropriate crash log string for the given recover value and stack trace.
func CrashLogText(r any, stack string) string {
	info := TheApp.SystemInfo()
	if info != "" {
		info += "\n"
	}
	return fmt.Sprintf("Platform: %v\nSystem platform: %v\nApp version: %s\nCore version: %s\nTime: %s\n%s\npanic: %v\n\n%s", TheApp.Platform(), TheApp.SystemPlatform(), AppVersion, CoreVersion, time.Now().Format(time.DateTime), info, r, stack)
}
