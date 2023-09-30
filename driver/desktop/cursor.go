// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package desktop

import (
	"sync"

	"github.com/go-gl/glfw/v3.3/glfw"
	"goki.dev/cursors/cursorimg"
	"goki.dev/enums"
	"goki.dev/goosi/cursor"
)

var theCursor = cursorImpl{CursorBase: cursor.CursorBase{Vis: true}, cursors: map[enums.Enum]map[int]*glfw.Cursor{}}

type cursorImpl struct {
	cursor.CursorBase
	cursors map[enums.Enum]map[int]*glfw.Cursor // cached cursors
	mu      sync.Mutex
}

func (c *cursorImpl) Set(cursor enums.Enum) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	sm := c.cursors[cursor]
	if sm == nil {
		sm = map[int]*glfw.Cursor{}
		c.cursors[cursor] = sm
	}
	if cur, ok := sm[c.Size]; ok {
		theApp.ctxtwin.glw.SetCursor(cur)
	}

	ci, err := cursorimg.Get(cursor, c.Size)
	if err != nil {
		return err
	}
	gc := glfw.CreateCursor(ci.Image, ci.Hotspot.X, ci.Hotspot.Y)
	sm[c.Size] = gc
	theApp.ctxtwin.glw.SetCursor(gc)
	return nil
}
