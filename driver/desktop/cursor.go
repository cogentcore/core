// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package desktop

import (
	"github.com/go-gl/glfw/v3.3/glfw"
	"goki.dev/cursors/cursorimg"
	"goki.dev/enums"
	"goki.dev/goosi/cursor"
)

var theCursor = &cursorImpl{}

type cursorImpl struct {
	cursor.CursorBase
}

func (c *cursorImpl) Set(cursor enums.Enum) error {
	ci, err := cursorimg.Get(cursor.String(), c.Size)
	if err != nil {
		return err
	}
	gc := glfw.CreateCursor(ci.Image, ci.HotSpot.X, ci.HotSpot.Y)
}
