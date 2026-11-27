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

// crashOutputPrefix is the prefix of the files of the output of the Go
// runtime for crashes (see [SaveCrashOutput]).
const crashOutputPrefix = "runtime_"

// SaveCrashOutput makes the Go runtime write the output of a crash that
// [HandleRecover] does not handle, a panic in another goroutine or a fatal
// error (as a concurrent map read and write), with the stacks of all of the
// goroutines, to a file in the crash-logs of [App.AppDataDir], named for
// the time the app started. Otherwise that output goes to the standard
// error, which is lost for an app that is not run from a terminal, as an
// app bundle on macOS, which just exits. The files of earlier runs that
// did not crash, which are empty, are removed. It does nothing on mobile,
// on the web, and offscreen (as in tests).
func SaveCrashOutput() {
	p := TheApp.Platform()
	if p.IsMobile() || p == Web || p == Offscreen {
		return
	}
	dnm := filepath.Join(TheApp.AppDataDir(), "crash-logs")
	if errors.Log(os.MkdirAll(dnm, 0755)) != nil {
		return
	}
	if old, err := filepath.Glob(filepath.Join(dnm, crashOutputPrefix+"*")); err == nil {
		for _, f := range old {
			if fi, err := os.Stat(f); err == nil && fi.Size() == 0 {
				os.Remove(f)
			}
		}
	}
	fnm := filepath.Join(dnm, crashOutputPrefix+time.Now().Format("2006-01-02_15-04-05")+"_"+fmt.Sprint(os.Getpid()))
	f, err := os.Create(fnm)
	if errors.Log(err) != nil {
		return
	}
	// with the stacks of all of the goroutines, as for a concurrent map
	// write, whose other goroutine is the one to find
	debug.SetTraceback("all")
	errors.Log(debug.SetCrashOutput(f, debug.CrashOptions{}))
	f.Close() // SetCrashOutput has its own copy
}
