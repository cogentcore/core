// Copyright 2018 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows
// +build !3d

package windriver

import (
	"sync"
	"syscall"

	"github.com/goki/gi/oswin/cursor"
)

var cursorMap = map[cursor.Shapes]int{
	cursor.Arrow:        _IDC_ARROW,
	cursor.Cross:        _IDC_CROSS,
	cursor.DragCopy:     _IDC_UPARROW, // todo: needs custom cursor
	cursor.DragMove:     _IDC_ARROW,   //  todo: needs custom cursor
	cursor.DragLink:     _IDC_ARROW,   //  todo: needs custom cursor
	cursor.HandPointing: _IDC_HAND,
	cursor.HandOpen:     _IDC_HAND, // todo: needs custom cursor
	cursor.HandClosed:   _IDC_HAND, // todo: needs custom cursor
	cursor.Help:         _IDC_HELP,
	cursor.IBeam:        _IDC_IBEAM,
	cursor.Not:          _IDC_NO,
	cursor.UpDown:       _IDC_SIZENS,
	cursor.LeftRight:    _IDC_SIZEWE,
	cursor.UpRight:      _IDC_SIZENESW,
	cursor.UpLeft:       _IDC_SIZENWSE,
	cursor.AllArrows:    _IDC_SIZEALL,
	cursor.Wait:         _IDC_WAIT,
}

type cursorImpl struct {
	cursor.CursorBase
	cursors map[cursor.Shapes]syscall.Handle
	mu      sync.Mutex
}

var theCursor = cursorImpl{CursorBase: cursor.CursorBase{Vis: true}}

func (c *cursorImpl) cursorHandle(sh cursor.Shapes) syscall.Handle {
	c.mu.Lock()
	if c.cursors == nil {
		c.cursors = make(map[cursor.Shapes]syscall.Handle, cursor.ShapesN)
	}
	ch, ok := c.cursors[sh]
	if !ok {
		idc := cursorMap[sh]
		ch = _LoadCursor(0, uintptr(idc))
		c.cursors[sh] = ch
	}
	c.mu.Unlock()
	return ch
}

func (c *cursorImpl) setImpl(sh cursor.Shapes) {
	_SetCursor(c.cursorHandle(sh))
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
	if c.Vis == false {
		return
	}
	c.mu.Lock()
	c.Vis = false
	c.mu.Unlock()
	_ShowCursor(false)
}

func (c *cursorImpl) Show() {
	if c.Vis {
		return
	}
	c.mu.Lock()
	c.Vis = true
	c.mu.Unlock()
	_ShowCursor(true)
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

// silly windows resets the cursor every time the mouse moves.. convince it to
// not do so
func resetCursor(hwnd syscall.Handle, uMsg uint32, wParam, lParam uintptr) (lResult uintptr) {
	theCursor.mu.Lock()
	if theCursor.Cur != cursor.Arrow {
		cc := theCursor.Cur
		theCursor.mu.Unlock()
		theCursor.setImpl(cc)
		return 1
	}
	theCursor.mu.Unlock()
	return _DefWindowProc(hwnd, uMsg, wParam, lParam)
}
