// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package gimain provides a Main function that encapsulates the standard
// oswin driver main function, and also ensures that standard sub-packages
// that are required for typical gi gui usage are automatically included
package gimain

import (
	"sync/atomic"

	"github.com/goki/gi/gi"
	_ "github.com/goki/gi/gi3d/io/obj"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/driver"
	"github.com/goki/gi/svg"
)

// these dummy variables force inclusion of relevant packages
var dummyGi gi.Node
var dummSvg svg.Line
var dummyVV giv.ValueViewBase

// Main is run in a main package to start the GUI driver / event loop,
// and call given function as the effective "main" function.
func Main(mainrun func()) {
	DebugEnumSizes()
	driver.Main(func(app oswin.App) {
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
	driver.Main(func(app oswin.App) {
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
