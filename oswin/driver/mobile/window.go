// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package mobile

import (
	"sync"

	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/driver/internal/event"
	"github.com/goki/mobile/gl"
	"github.com/goki/vgpu/vdraw"
	"github.com/goki/vgpu/vgpu"
)

type windowImpl struct {
	oswin.WindowBase
	event.Deque
	// app            *appImpl
	ctx      *gl.Context
	Surface  *vgpu.Surface
	Draw     vdraw.Drawer
	scrnName string // last known screen name
	// runQueue       chan funcRun
	publish        chan struct{}
	publishDone    chan struct{}
	winClose       chan struct{}
	mu             sync.Mutex
	mainMenu       oswin.MainMenu
	closeReqFunc   func(win oswin.Window)
	closeCleanFunc func(win oswin.Window)
	mouseDisabled  bool
	resettingPos   bool
}

func (w *windowImpl) Handle() any {
	return w.ctx
}
