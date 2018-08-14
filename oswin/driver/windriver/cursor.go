// Copyright 2018 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package windriver

import (
	"syscall"

	"github.com/goki/gi/oswin/cursor"
)

var cursorMap = map[cursor.Shapes]_LPCTSTR{
	Arrow:        _IDC_ARROW,
	Cross:        _IDC_CROSS,
	DragCopy:     _IDC_UPARROW, // todo: needs custom cursor
	DragMove:     _IDC_ARROW,   //  todo: needs custom cursor
	DragLink:     _IDC_ARROW,   //  todo: needs custom cursor
	HandPointing: _IDC_HAND,
	HandOpen:     _IDC_HAND, // todo: needs custom cursor
	HandClosed:   _IDC_HAND, // todo: needs custom cursor
	Help:         _IDC_HELP,
	IBeam:        _IDC_IBEAM,
	Not:          _IDC_NO,
	UpDown:       _IDC_SIZENS,
	LeftRight:    _IDC_SIZEWE,
	UpRight:      _IDC_SIZENESW,
	UpLeft:       _IDC_SIZENWSE,
	AllArrows:    _IDC_SIZEALL,
	Wait:         IDC_WAIT,
}

type cursorImpl struct {
	cursor.CursorBase
	cursors map[cursor.Shapes]syscall.Handle
}

var theCursor = cursorImpl{CursorBase: cursor.CursorBase{Vis: true}}

func (c *cursorImpl) cursorHandle(sh cursor.Shapes) syscall.Handle {
	if c.cursors == nil {
		c.cursors = make(map[cursor.Shapes]syscall.Handle, cursor.ShapesN)
	}
	ch, ok := c.cursors[sh]
	if !ok {
		ch = _LoadCursor(nil, cursorMap[sh])
		// todo: load custom
		c.cursors[sh] = ch
	}
	return ch
}

func (c *cursorImpl) Set(sh cursor.Shapes) {
	c.Cur = sh
	_SetCursor(cursorHandle(sh))
}

func (c *cursorImpl) Push(sh cursor.Shapes) {
	c.PushStack(sh)
	c.Set(sh)
}

func (c *cursorImpl) Pop() {
	sh, _ := c.PopStack()
	c.Set(sh)
}

func (c *cursorImpl) Hide() {
	if c.Vis == false {
		return
	}
	c.Vis = false
	_ShowCursor(false)
}

func (c *cursorImpl) Show() {
	if c.Vis {
		return
	}
	c.Vis = true
	_ShowCursor(true)
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
