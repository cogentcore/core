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

var theCursor = cursorImpl{CursorBase: cursor.CursorBase{Vis: true}, cursors: map[string]map[int]*glfw.Cursor{}}

type cursorImpl struct {
	cursor.CursorBase
	cursors map[string]map[int]*glfw.Cursor // cached cursors
	mu      sync.Mutex
}

func (c *cursorImpl) Set(cursor enums.Enum) error {
	nm := cursor.String()
	sm := c.cursors[nm]
	if sm == nil {
		c.cursors[nm] = map[int]*glfw.Cursor{}
	}
	if cur, ok := sm[c.Size]; ok {
		theApp.ctxtwin.glw.SetCursor(cur)
	}

	ci, err := cursorimg.Get(nm, c.Size)
	if err != nil {
		return err
	}
	gc := glfw.CreateCursor(ci.Image, ci.HotSpot.X, ci.HotSpot.Y)
	sm[c.Size] = gc
	theApp.ctxtwin.glw.SetCursor(gc)
	return nil
}
