// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"github.com/skelterjohn/go.wde"
	// "image"
	"reflect"
)

// Window provides an OS window using go.wde package
type Window struct {
	GiNode
	Win wde.Window
}

// create a new window with given name and sizing
func NewWindow(name string, width, height int) *Window {
	win := &Window{}
	win.SetThisName(win, name)
	var err error
	win.Win, err = wde.NewWindow(width, height)
	if err != nil {
		fmt.Printf("gogi NewWindow error: %v \n", err)
		return nil
	}
	win.Win.SetTitle(name)
	return win
}

// create a new window with given name and sizing, and initialize a 2D viewport within it
func NewWindow2D(name string, width, height int) *Window {
	win := NewWindow(name, width, height)
	vp := NewViewport2D(width, height)
	win.AddChild(vp)
	return win
}

func (w *Window) WinViewport2D() *Viewport2D {
	vpi := w.FindChildByType(reflect.TypeOf(Viewport2D{}))
	vp, _ := vpi.(*Viewport2D)
	return vp
}
