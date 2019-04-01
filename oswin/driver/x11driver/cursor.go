// Copyright 2018 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build linux,!android dragonfly openbsd
// +build !3d

package x11driver

import (
	"log"
	"sync"

	"github.com/BurntSushi/xgb/xproto"
	"github.com/goki/gi/oswin/cursor"
)

// https://xcb.freedesktop.org/tutorial/mousecursors/
// https://tronche.com/gui/x/xlib/appendix/b/

var cursorMap = map[cursor.Shapes]int{
	cursor.Arrow:        68, // XC_arrow is 2 but std seems to be left-up = 68
	cursor.Cross:        34,
	cursor.DragCopy:     90, // XC_plus
	cursor.DragMove:     50, // XC_exchange
	cursor.DragLink:     68, // todo: not special
	cursor.HandPointing: 58, // XC_hand1
	cursor.HandOpen:     60, // XC_hand2
	cursor.HandClosed:   60, // XC_hand2
	cursor.Help:         92,
	cursor.IBeam:        152,
	cursor.Not:          0, // X cursor?
	cursor.UpDown:       116,
	cursor.LeftRight:    108,
	cursor.UpRight:      136,
	cursor.UpLeft:       134,
	cursor.AllArrows:    52,
	cursor.Wait:         150,
}

type cursorImpl struct {
	cursor.CursorBase
	cursors  map[cursor.Shapes]xproto.Cursor
	cursFont xproto.Font
	mu       sync.Mutex
}

var theCursor = cursorImpl{CursorBase: cursor.CursorBase{Vis: true}}

func (c *cursorImpl) openCursorFont() {
	var err error
	c.cursFont, err = xproto.NewFontId(theApp.xc)
	if err != nil {
		log.Printf("x11driver: xproto.NewFontId failed: %v", err)
		return
	}
	fnm := "cursor"
	err = xproto.OpenFontChecked(theApp.xc, c.cursFont, uint16(len(fnm)), fnm).Check()
	if err != nil {
		log.Printf("x11driver: xproto.OpenFont for cursor font failed: %v", err)
		return
	}
}

func (c *cursorImpl) createCursor(curid int) xproto.Cursor {
	cur, err := xproto.NewCursorId(theApp.xc)
	if err != nil {
		log.Printf("x11driver: xproto.NewCursorId failed: %v", err)
		return 0
	}

	// 0's are all colors -- black by default
	err = xproto.CreateGlyphCursorChecked(theApp.xc, cur, c.cursFont, c.cursFont, uint16(curid), uint16(curid+1), 65535, 65535, 65535, 0, 0, 0).Check()

	if err != nil {
		log.Printf("x11driver: xproto.CreateGlyphCursor for cursor id %v failed: %v", curid, err)
		return 0
	}

	return cur
}

func (c *cursorImpl) setCursor(cur xproto.Cursor) {
	fw := theApp.ctxtwin
	vallist := []uint32{uint32(cur)}
	err := xproto.ChangeWindowAttributesChecked(theApp.xc, fw.xw, xproto.CwCursor, vallist).Check()
	if err != nil {
		log.Printf("x11driver: xproto.ChangeWindowAttributes for cursor failed: %v", err)
	}
}

func (c *cursorImpl) cursorHandle(sh cursor.Shapes) xproto.Cursor {
	c.mu.Lock()
	if c.cursors == nil {
		c.cursors = make(map[cursor.Shapes]xproto.Cursor, cursor.ShapesN)
		c.openCursorFont()
	}
	ch, ok := c.cursors[sh]
	if !ok {
		curid := cursorMap[sh]
		ch = c.createCursor(curid)
		c.cursors[sh] = ch
	}
	c.mu.Unlock()
	return ch
}

func (c *cursorImpl) setImpl(sh cursor.Shapes) {
	c.setCursor(c.cursorHandle(sh))
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
	c.mu.Unlock()
	// _ShowCursor(false) // todo: create blank cursor
}

func (c *cursorImpl) Show() {
	c.mu.Lock()
	if c.Vis {
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()
	c.Vis = true
	// _ShowCursor(true)
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
