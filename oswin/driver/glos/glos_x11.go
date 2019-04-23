// Copyright 2018 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build linux,!android dragonfly openbsd

package glos

import (
	"sync"

	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/cursor"
	"github.com/goki/gi/oswin/mimedata"
)

// Notes on intermixing glfw and xgb: bottom line, can't do:
//
// can include Xlib-xcb.h to get XGetXCBConnection method, which can get the
// xcb_connection from the Display: https://xcb.freedesktop.org/MixingCalls/
// which glfw can return, in GetX11Display().
// BUT, BurntSushi/xgb does NOT seem to directly use the xcb_connection
// and instead is a complete ground-up rewrite using net.Conn connection protocol
// not sure if we can have 2 separate connections..
// and really, maybe we don't need it after all!?  just use the
// text-based clipboard mechanisms to write mime-encoded content, and
// cursor impl has full support for creating new cursors, so..

/////////////////////////////////////////////////////////////////
// OS-specific methods

// this is the main call to create the main menu if not exist
func (w *windowImpl) MainMenu() oswin.MainMenu {
	return nil
}

func (w *windowImpl) OSHandle() uintptr {
	return uintptr(w.glw.GetX11Window())
}

/////////////////////////////////////////////////////////////////
//   Clipboard

type clipImpl struct {
	lastWrite mimedata.Mimes
}

var theClip = clipImpl{}

func (ci *clipImpl) IsEmpty() bool {
	w := theApp.ctxtwin
	str, err := w.glw.GetClipboardString()
	if err != nil || len(str) == 0 {
		return true
	}
	return false
}

func (ci *clipImpl) Read(types []string) mimedata.Mimes {
	w := theApp.ctxtwin
	str, err := w.glw.GetClipboardString()
	if err != nil || len(str) == 0 {
		return nil
	}
	wantText := mimedata.IsText(types[0])
	if wantText {
		str, err := w.glw.GetClipboardString()
		if err != nil || len(str) == 0 {
			return nil
		}
		isMulti, mediaType, boundary, body := mimedata.IsMultipart(str)
		if isMulti {
			return mimedata.FromMultipart(body, boundary)
		} else {
			if mediaType != "" { // found a mime type encoding
				return mimedata.NewMime(mediaType, str)
			} else {
				// we can't really figure out type, so just assume..
				return mimedata.NewMime(types[0], str)
			}
		}
	} else {
		// todo: deal with image formats etc
	}
	return nil
}

func (ci *clipImpl) Write(data mimedata.Mimes) error {
	if len(data) == 0 {
		return nil
	}
	w := theApp.ctxtwin
	if len(data) > 1 { // multipart
		mpd := data.ToMultipart()
		w.wgl.SetClipboardString(string(mpd))
	} else {
		d := data[0]
		if mimedata.IsText(d.Type) {
			w.wgl.SetClipboardString(string(d.Data))
		}
	}
	return nil
}

func (ci *clipImpl) Clear() {
	// nop
}

//////////////////////////////////////////////////////
//  Cursor

// todo: glfw has a seriously impoverished set of standard cursors..
// need to find a good collection and install

var cursorMap = map[cursor.Shapes]glfw.StandardCursor{
	cursor.Arrow:        glfw.ArrowCursor,
	cursor.Cross:        glfw.CRosshairCursor,
	cursor.DragCopy:     glfw.HandCursor,
	cursor.DragMove:     glfw.HandCursor,
	cursor.DragLink:     glfw.HandCursor,
	cursor.HandPointing: glfw.HandCursor,
	cursor.HandOpen:     glfw.HandCursor,
	cursor.HandClosed:   glfw.HandCursor,
	cursor.Help:         glfw.HandCursor,
	cursor.IBeam:        glfw.IBeamCursor,
	cursor.Not:          glfw.HandCursor,
	cursor.UpDown:       glfw.VResizeCursor,
	cursor.LeftRight:    glfw.HResizeCursor,
	cursor.UpRight:      glfw.HResizeCursor,
	cursor.UpLeft:       glfw.HResizeCursor,
	cursor.AllArrows:    glfw.VResizeCursor,
	cursor.Wait:         glfw.VResizeCursor,
}

type cursorImpl struct {
	cursor.CursorBase
	cursors map[cursor.Shapes]*glfw.Cursor
	mu      sync.Mutex
}

var theCursor = cursorImpl{CursorBase: cursor.CursorBase{Vis: true}}

func (c *cursorImpl) createCursors() {
	if c.cursors != nil {
		return
	}
	c.cursors = make(map[cursor.Shapes]*glfw.Cursor)
	for cs, sc := range cursorMap {
		cur := glfw.CreateStandardCursor(sc)
		c.cursors[cs] = cur
	}
}

func (c *cursorImpl) setImpl(sh cursor.Shapes) {
	c.createCursors()
	cur, ok := c.cursors[sh]
	if !ok || cur == nil {
		return
	}
	w := theApp.ctxtwin
	w.glw.SetCursor(cur)
}

func (c *cursorImpl) Set(sh cursor.Shapes) {
	c.mu.Lock()
	c.Cur = sh
	c.mu.Unlock()
	c.setImpl(sh)
}

func (c *cursorImpl) Push(sh cursor.Shapes) {
	c.mu.Lock()
	c.PushStack(sh)
	c.mu.Unlock()
	c.setImpl(sh)
}

func (c *cursorImpl) Pop() {
	c.mu.Lock()
	sh, _ := c.PopStack()
	c.mu.Unlock()
	c.setImpl(sh)
}

func (c *cursorImpl) Hide() {
	c.mu.Lock()
	if c.Vis == false {
		c.mu.Unlock()
		return
	}
	c.Vis = false
	w := theApp.ctxtwin
	w.glw.SetInputMode(glfw.CursorHidden)
	c.mu.Unlock()
}

func (c *cursorImpl) Show() {
	c.mu.Lock()
	if c.Vis {
		c.mu.Unlock()
		return
	}
	c.Vis = true
	w := theApp.ctxtwin
	w.glw.SetInputMode(glfw.CursorNormal)
	c.mu.Unlock()
}

func (c *cursorImpl) PushIfNot(sh cursor.Shapes) bool {
	c.mu.Lock()
	if c.Cur == sh {
		c.mu.Unlock()
		return false
	}
	c.mu.Unlock()
	c.Push(sh)
	return true
}

func (c *cursorImpl) PopIf(sh cursor.Shapes) bool {
	c.mu.Lock()
	if c.Cur == sh {
		c.mu.Unlock()
		c.Pop()
		return true
	}
	c.mu.Unlock()
	return false
}
