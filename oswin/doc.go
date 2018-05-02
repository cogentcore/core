// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
//
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package oswin provides interfaces for OS-specific GUI hardware for portable
// two-dimensional graphics and input events.
//
// The App interface provides a top-level, single-instance struct that knows
// about specific hardware, and can create new Window, Image, and Texture
// objects that are hardware-specific and provide the primary GUI interface.
// It is aways available as oswin.TheApp.
//
// Events are commuinicated through the Window -- see EventType and Event in
// event.go for all the different types.
//
// The driver package creates the App, via its Main function, which is
// designed to be called by the program's main function. The driver package
// provides the default driver for the system, such as the X11 driver for
// desktop Linux, but other drivers, such as the OpenGL driver, can be
// explicitly invoked by calling that driver's Main function. To use the
// default driver:
//
//
//	package main
//
//	import (
//		"github.com/goki/goki/gi"
//		"github.com/goki/goki/gi/oswin/driver"
//		"github.com/goki/goki/gi/oswin"
//	)
//
//	func main() {
//		driver.Main(func(app oswin.App) {
// 		mainrun()
// 	})
// }
//
// func mainrun() {
// 	width := 1024
// 	height := 768
//
// 	// turn these on to see a traces of various stages of processing..
// 	// gi.Update2DTrace = true
// 	// gi.Render2DTrace = true
// 	// gi.Layout2DTrace = true
// 	// ki.SignalTrace = true
//
// 	win := gi.NewWindow2D("GoGi Widgets Window", width, height, true) // pixel sizes
//  ...
//
// Complete examples can be found in the gi/examples directory.
//
// Each driver package provides App, Screen, Image, Texture and Window
// implementations that work together. Such types are interface types because
// this package is driver-independent, but those interfaces aren't expected to
// be implemented outside of drivers. For example, a driver's Window
// implementation will generally work only with that driver's Image
// implementation, and will not work with an arbitrary type that happens to
// implement the Image methods.
package oswin
