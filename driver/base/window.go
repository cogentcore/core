// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package base

import (
	"image"

	"goki.dev/goosi"
	"goki.dev/goosi/events"
)

// Window contains the data and logic common to all implementations of [goosi.Window].
type Window struct {
	events.Deque

	Nm          string
	Titl        string
	Pos         image.Point
	WnSize      image.Point // window-manager coords
	PxSize      image.Point // pixel size
	DevPixRatio float32
	PhysDPI     float32
	LogDPI      float32
	Par         any
	Flag        goosi.WindowFlags
	FPS         int
	EvMgr       events.Mgr

	// set this to a function that will destroy GPU resources
	// in the main thread prior to destroying the drawer
	// and the surface -- otherwise it is difficult to
	// ensure that the proper ordering of destruction applies.
	DestroyGPUfunc func()
}

func (w *Window) Name() string {
	return w.Nm
}

func (w *Window) SetName(name string) {
	w.Nm = name
}

func (w *Window) Title() string {
	return w.Titl
}

func (w *Window) Parent() any {
	return w.Par
}

func (w *Window) SetParent(parent any) {
	w.Par = parent
}

func (w *Window) Flags() goosi.WindowFlags {
	return w.Flag
}

func (w *Window) Is(flag goosi.WindowFlags) bool {
	return w.Flag.HasFlag(flag)
}

func (w *Window) SetFPS(fps int) {
	w.FPS = fps
}

func (w *Window) EventMgr() *events.Mgr {
	return &w.EvMgr
}

func (w *Window) SetDestroyGPUResourcesFunc(f func()) {
	w.DestroyGPUfunc = f
}
