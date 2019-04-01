// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows
// +build !3d

package windriver

import (
	"log"
	"runtime"

	"github.com/goki/gi/oswin"
)

// Main is called by the program's main function to run the graphical
// application.
//
// It calls f on the App, possibly in a separate goroutine, as some OS-
// specific libraries require being on 'the main thread'. It returns when f
// returns.
func Main(f func(oswin.App)) {
	oswin.TheApp = theApp
	theApp.initScreens()

	// It does not matter which OS thread we are on.
	// All that matters is that we confine all UI operations
	// to the thread that created the respective window.
	runtime.LockOSThread()

	if err := initCommon(); err != nil {
		return
	}

	if err := initAppWindow(); err != nil {
		return
	}
	defer func() {
		// TODO(andlabs): log an error if this fails?
		_DestroyWindow(appHWND)
		// TODO(andlabs): unregister window class
	}()

	if err := initWindowClass(); err != nil {
		return
	}

	// Prime the pump.
	mainCallback = f
	_PostMessage(appHWND, msgMainCallback, 0, 0)

	// Main message pump.
	var m _MSG
	for {
		done, err := _GetMessage(&m, 0, 0, 0)
		if err != nil {
			log.Printf("win32 GetMessage failed: %v", err)
			return
		}
		if done == 0 { // WM_QUIT
			break
		}
		_TranslateMessage(&m)
		_DispatchMessage(&m)
	}
}
