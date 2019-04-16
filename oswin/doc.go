// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// originally based on golang.org/x/exp/shiny:
//
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package oswin provides interfaces for OS-specific GUI hardware for portable
// two-dimensional graphics and input events.
//
// The App interface provides a top-level, single-instance struct that knows
// about specific hardware, and can create new Window and Texture
// objects that are hardware-specific and provide the primary GUI interface.
// It is always available as oswin.TheApp.
//
// Events are communicated through the Window -- see EventType and Event in
// event.go for all the different types.
//
// The driver package creates the App, via its Main function, which is
// designed to be called by the program's main function.  There can
// be multiple different drivers, but currently OpenGL on top of
// the glfw cross-platform library (i.e., the glos driver) is
// the only one supported.  See internal/*driver for older
// shiny-based drivers that are completely OS-specific and do not
// require cgo for Windows and X11 platforms (but do require it for mac).
// These older drivers are no longer compatible with the current GPU-based
// 3D rendering system in gi and gi3d.
//
// package gi/gimain provides the Main method used to initialize the
// oswin driver that implements the oswin interfaces, and start the
// main event processing loop.
//
// See examples in gi/examples directory for current example code.
//
package oswin
