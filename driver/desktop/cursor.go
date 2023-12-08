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

var TheCursor = Cursor{CursorBase: cursor.CursorBase{Vis: true, Size: 32}, cursors: map[enums.Enum]map[int]*glfw.Cursor{}}

type Cursor struct {
	cursor.CursorBase
	cursors  map[enums.Enum]map[int]*glfw.Cursor // cached cursors
	mu       sync.Mutex
	prevSize int // cached previous size
}

func (c *Cursor) Set(cursor enums.Enum) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if cursor == c.Cur && c.Size == c.prevSize { // we already have, so we don't need to set again
		return nil
	}
	sm := c.cursors[cursor]
	if sm == nil {
		sm = map[int]*glfw.Cursor{}
		c.cursors[cursor] = sm
	}
	if cur, ok := sm[c.Size]; ok {
		TheApp.CtxWindow.glw.SetCursor(cur)
		c.prevSize = c.Size
		c.Cur = cursor
		return nil
	}

	ci, err := cursorimg.Get(cursor, c.Size)
	if err != nil {
		return err
	}
	h := ci.Hotspot
	gc := glfw.CreateCursor(ci.Image, h.X, h.Y)
	sm[c.Size] = gc
	TheApp.CtxWindow.glw.SetCursor(gc)
	c.prevSize = c.Size
	c.Cur = cursor
	return nil
}
