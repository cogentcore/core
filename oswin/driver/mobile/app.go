// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package mobile implements oswin interfaces on mobile devices
package mobile

// type appImpl struct {
// 	mu            sync.Mutex
// 	mainQueue     chan funcRun
// 	mainDone      chan struct{}
// 	gpu           *vgpu.GPU
// 	shareWin      *glfw.Window // a non-visible, always-present window that all windows share gl context with
// 	windows       map[*glfw.Window]*windowImpl
// 	oswindows     map[uintptr]*windowImpl
// 	winlist       []*windowImpl
// 	screens       []*oswin.Screen
// 	screensAll    []*oswin.Screen // unique list of all screens ever seen -- get info from here if fails
// 	noScreens     bool            // if all screens have been disconnected, don't do anything..
// 	ctxtwin       *windowImpl     // context window, dynamically set, for e.g., pointer and other methods
// 	name          string
// 	about         string
// 	openFiles     []string
// 	quitting      bool          // set to true when quitting and closing windows
// 	quitCloseCnt  chan struct{} // counts windows to make sure all are closed before done
// 	quitReqFunc   func()
// 	quitCleanFunc func()
// }
