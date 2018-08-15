// Copyright 2018 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build linux,!android dragonfly openbsd

package x11driver

import (
	"log"

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
	err = xproto.CreateGlyphCursorChecked(theApp.xc, cur, c.cursFont, c.cursFont, uint16(curid), uint16(curid+1), 0, 0, 0, 0, 0, 0).Check()

	if err != nil {
		log.Printf("x11driver: xproto.CreateGlyphCursor for cursor id %v failed: %v", curid, err)
		return 0
	}

	return cur
}

func (c *cursorImpl) setCursor(cur xproto.Cursor) {
	focwin := theApp.WindowInFocus()
	if focwin == nil {
		return
	}
	fw := focwin.(*windowImpl)
	vallist := []uint32{uint32(cur)}
	err := xproto.ChangeWindowAttributesChecked(theApp.xc, fw.xw, xproto.CwCursor, vallist).Check()
	if err != nil {
		log.Printf("x11driver: xproto.ChangeWindowAttributes for cursor failed: %v", err)
	}
}

func (c *cursorImpl) cursorHandle(sh cursor.Shapes) xproto.Cursor {
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
	return ch
}

func (c *cursorImpl) setImpl(sh cursor.Shapes) {
	c.setCursor(c.cursorHandle(sh))
}

func (c *cursorImpl) Set(sh cursor.Shapes) {
	c.Cur = sh
	c.setImpl(sh)
}

func (c *cursorImpl) Push(sh cursor.Shapes) {
	c.PushStack(sh)
	c.setImpl(sh)
}

func (c *cursorImpl) Pop() {
	sh, _ := c.PopStack()
	c.setImpl(sh)
}

func (c *cursorImpl) Hide() {
	if c.Vis == false {
		return
	}
	c.Vis = false
	// _ShowCursor(false) // todo: create blank cursor
}

func (c *cursorImpl) Show() {
	if c.Vis {
		return
	}
	c.Vis = true
	// _ShowCursor(true)
}

func (c *cursorImpl) PushIfNot(sh cursor.Shapes) bool {
	if c.Cur == sh {
		return false
	}
	c.Push(sh)
	return true
}

func (c *cursorImpl) PopIf(sh cursor.Shapes) bool {
	if c.Cur == sh {
		c.Pop()
		return true
	}
	return false
}
