// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package gimain provides a Main function that encapsulates the standard
// oswin driver main function, and also ensures that standard sub-packages
// that are required for typical gi gui usage are automatically included
package gimain

import (
	"sync/atomic"

	_ "goki.dev/gi/v2/giv"

	"goki.dev/gi/v2/gi"
	"goki.dev/goosi"
	"goki.dev/goosi/driver"
	_ "goki.dev/grog"
	// _ "goki.dev/vgpu/v2/vphong" // TODO(kai/web): do we need this bar import on other platforms?
)

// Main is run in a main package to start the GUI driver / event loop,
// and call given function as the effective "main" function.
func Run(mainrun func()) {
	DebugEnumSizes()
	driver.Main(func(app goosi.App) {
		gi.Init()
		mainrun()
	})
}

var quit = make(chan struct{})

var started int32

// Start is called via a library to start the driver -- dynamic libraries
// in Go do not run in the main thread, so this needs to be called after
// loading the library.  This call will never return, so another thread
// must be launched prior to calling this in the main thread.  That thread
// can call gimain.Quit() to close this main thread.
func Start() {
	driver.Main(func(app goosi.App) {
		atomic.AddInt32(&started, 1)
		<-quit
	})
}

func HasStarted() bool {
	return atomic.LoadInt32(&started) > 0
}

// Quit can be called to close the main thread started by Start()
func Quit() {
	close(quit)
}
