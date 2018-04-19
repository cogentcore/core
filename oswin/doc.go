// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
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
// Screens are not created directly. Instead, driver packages provide access
// to the screen through a Main function that is designed to be called by the
// program's main function. The driver package provides the default driver for
// the system, such as the X11 driver for desktop Linux, but other drivers,
// such as the OpenGL driver, can be explicitly invoked by calling that
// driver's Main function. To use the default driver:
//
//	package main
//
//	import (
//		"github.com/rcoreilly/goki/gi/oswin/driver"
//		"github.com/rcoreilly/goki/gi/oswin"
//		"golang.org/x/mobile/event/lifecycle"
//	)
//
//	func main() {
//		driver.Main(func(s oswin.Screen) {
//			w, err := s.NewWindow(nil)
//			if err != nil {
//				handleError(err)
//				return
//			}
//			defer w.Release()
//
//			for {
//				switch e := w.NextEvent().(type) {
//				case lifecycle.Event:
//					if e.To == lifecycle.StageDead {
//						return
//					}
//					etc
//				case etc:
//					etc
//				}
//			}
//		})
//	}
//
// Complete examples can be found in the gi/example directory.
//
// Each driver package provides Screen, Image, Texture and Window
// implementations that work together. Such types are interface types because
// this package is driver-independent, but those interfaces aren't expected to
// be implemented outside of drivers. For example, a driver's Window
// implementation will generally work only with that driver's Image
// implementation, and will not work with an arbitrary type that happens to
// implement the Image methods.
package oswin
